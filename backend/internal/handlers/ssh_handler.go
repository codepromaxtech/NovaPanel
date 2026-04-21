package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	ws "github.com/gorilla/websocket"
	"github.com/novapanel/novapanel/internal/services"
	"golang.org/x/crypto/ssh"
)

var termUpgrader = ws.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type SSHHandler struct {
	serverService *services.ServerService
}

func NewSSHHandler(serverService *services.ServerService) *SSHHandler {
	return &SSHHandler{serverService: serverService}
}

// terminalMsg is a message from/to the browser terminal.
type terminalMsg struct {
	Type string `json:"type"` // "input", "resize", "output"
	Data string `json:"data,omitempty"`
	Cols int    `json:"cols,omitempty"`
	Rows int    `json:"rows,omitempty"`
}

// resolveSSHTarget determines the SSH host, user, and auth methods for a server.
func (h *SSHHandler) resolveSSHTarget(server *services.ServerForSSH) (string, *ssh.ClientConfig) {
	host := server.IPAddress
	port := server.Port
	if port == 0 {
		port = 22
	}

	// For master/local server, connect via Docker gateway to reach the host
	if server.Role == "master" {
		gateway := getDockerGatewayIP()
		if gateway != "" {
			host = gateway
		}
	}

	// Prefer SSH user from DB; fall back to env var then root
	sshUser := server.SSHUser
	if sshUser == "" {
		sshUser = os.Getenv("SSH_USER")
	}
	if sshUser == "" {
		sshUser = "root"
	}

	config := &ssh.ClientConfig{
		User:            sshUser,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	var authMethods []ssh.AuthMethod

	// 1. DB-stored SSH key (decrypted)
	if server.SSHKey != "" {
		if signer, err := ssh.ParsePrivateKey([]byte(server.SSHKey)); err == nil {
			authMethods = append(authMethods, ssh.PublicKeys(signer))
		}
	}

	// 2. DB-stored password
	if server.SSHPassword != "" {
		authMethods = append(authMethods, ssh.Password(server.SSHPassword))
	}

	// 3. Mounted host SSH keys (fallback for local/master server)
	keyPaths := []string{
		"/root/.ssh/id_rsa",
		"/root/.ssh/id_ed25519",
		"/host/ssh/id_rsa",
		"/host/ssh/id_ed25519",
	}
	for _, kp := range keyPaths {
		keyData, err := os.ReadFile(kp)
		if err != nil {
			continue
		}
		if signer, err := ssh.ParsePrivateKey(keyData); err == nil {
			authMethods = append(authMethods, ssh.PublicKeys(signer))
		}
	}

	// 4. Env var password fallback
	if envPass := os.Getenv("SSH_PASSWORD"); envPass != "" {
		authMethods = append(authMethods, ssh.Password(envPass))
	}

	config.Auth = authMethods

	addr := fmt.Sprintf("%s:%d", host, port)
	return addr, config
}

func getDockerGatewayIP() string {
	data, err := os.ReadFile("/proc/net/route")
	if err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) >= 3 && fields[1] == "00000000" {
				gw := fields[2]
				if len(gw) == 8 {
					var a, b, c, d uint64
					fmt.Sscanf(gw[6:8], "%x", &a)
					fmt.Sscanf(gw[4:6], "%x", &b)
					fmt.Sscanf(gw[2:4], "%x", &c)
					fmt.Sscanf(gw[0:2], "%x", &d)
					return fmt.Sprintf("%d.%d.%d.%d", a, b, c, d)
				}
			}
		}
	}
	return "172.17.0.1"
}

// HandleTerminal upgrades to WebSocket and bridges it to an SSH session.
func (h *SSHHandler) HandleTerminal(c *gin.Context) {
	serverID := c.Param("id")

	// Look up the server with decrypted SSH credentials
	decryptedServer, err := h.serverService.GetDecryptedSSH(c.Request.Context(), serverID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "server not found"})
		return
	}

	// Resolve SSH target using decrypted credentials
	sshTarget := &services.ServerForSSH{
		IPAddress:   decryptedServer.IPAddress,
		Port:        decryptedServer.Port,
		Role:        "worker",
		SSHUser:     decryptedServer.SSHUser,
		SSHKey:      decryptedServer.SSHKey,
		SSHPassword: decryptedServer.SSHPassword,
		AuthMethod:  decryptedServer.AuthMethod,
	}
	addr, sshConfig := h.resolveSSHTarget(sshTarget)

	// Upgrade to WebSocket FIRST so we can send status messages
	conn, err := termUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// Send connecting message
	connectMsg, _ := json.Marshal(terminalMsg{Type: "output", Data: fmt.Sprintf("\r\n\x1b[33mConnecting to %s as %s...\x1b[0m\r\n", addr, sshConfig.User)})
	conn.WriteMessage(ws.TextMessage, connectMsg)

	// Connect to SSH
	sshClient, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		log.Printf("SSH connection failed to %s: %v", addr, err)
		errMsg, _ := json.Marshal(terminalMsg{Type: "output", Data: fmt.Sprintf("\r\n\x1b[31m✕ SSH connection failed: %s\x1b[0m\r\n", err.Error())})
		conn.WriteMessage(ws.TextMessage, errMsg)
		conn.Close()
		return
	}

	// Open session
	session, err := sshClient.NewSession()
	if err != nil {
		sshClient.Close()
		errMsg, _ := json.Marshal(terminalMsg{Type: "output", Data: "\r\n\x1b[31m✕ Failed to create SSH session\x1b[0m\r\n"})
		conn.WriteMessage(ws.TextMessage, errMsg)
		conn.Close()
		return
	}

	// Request PTY
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if err := session.RequestPty("xterm-256color", 24, 80, modes); err != nil {
		session.Close()
		sshClient.Close()
		errMsg, _ := json.Marshal(terminalMsg{Type: "output", Data: "\r\n\x1b[31m✕ PTY request failed\x1b[0m\r\n"})
		conn.WriteMessage(ws.TextMessage, errMsg)
		conn.Close()
		return
	}

	// Get stdin/stdout pipes
	stdinPipe, err := session.StdinPipe()
	if err != nil {
		session.Close()
		sshClient.Close()
		conn.Close()
		return
	}
	stdoutPipe, err := session.StdoutPipe()
	if err != nil {
		session.Close()
		sshClient.Close()
		conn.Close()
		return
	}
	stderrPipe, err := session.StderrPipe()
	if err != nil {
		session.Close()
		sshClient.Close()
		conn.Close()
		return
	}

	// Start shell
	if err := session.Shell(); err != nil {
		session.Close()
		sshClient.Close()
		errMsg, _ := json.Marshal(terminalMsg{Type: "output", Data: "\r\n\x1b[31m✕ Shell start failed\x1b[0m\r\n"})
		conn.WriteMessage(ws.TextMessage, errMsg)
		conn.Close()
		return
	}

	// Send connected message
	okMsg, _ := json.Marshal(terminalMsg{Type: "output", Data: fmt.Sprintf("\x1b[32m✓ Connected to %s\x1b[0m\r\n\r\n", addr)})
	conn.WriteMessage(ws.TextMessage, okMsg)

	log.Printf("Terminal session opened for server %s (%s)", serverID, addr)

	// SSH stdout → WebSocket
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stdoutPipe.Read(buf)
			if n > 0 {
				msg := terminalMsg{Type: "output", Data: string(buf[:n])}
				data, _ := json.Marshal(msg)
				if writeErr := conn.WriteMessage(ws.TextMessage, data); writeErr != nil {
					break
				}
			}
			if err != nil {
				break
			}
		}
		conn.Close()
	}()

	// SSH stderr → WebSocket
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stderrPipe.Read(buf)
			if n > 0 {
				msg := terminalMsg{Type: "output", Data: string(buf[:n])}
				data, _ := json.Marshal(msg)
				conn.WriteMessage(ws.TextMessage, data)
			}
			if err != nil {
				break
			}
		}
	}()

	// WebSocket → SSH stdin
	go func() {
		defer func() {
			stdinPipe.Close()
			session.Close()
			sshClient.Close()
			conn.Close()
			log.Printf("Terminal session closed for server %s", serverID)
		}()

		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				break
			}

			var msg terminalMsg
			if err := json.Unmarshal(message, &msg); err != nil {
				// Raw text input fallback
				io.WriteString(stdinPipe, string(message))
				continue
			}

			switch msg.Type {
			case "input":
				io.WriteString(stdinPipe, msg.Data)
			case "resize":
				if msg.Cols > 0 && msg.Rows > 0 {
					session.WindowChange(msg.Rows, msg.Cols)
				}
			}
		}
	}()

	// Wait for session to end
	session.Wait()
}

func itoa(n int) string {
	if n == 0 {
		return "22"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
