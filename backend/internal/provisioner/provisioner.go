package provisioner

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// ServerInfo holds the connection details for a server
type ServerInfo struct {
	IPAddress   string
	Port        int
	SSHUser     string
	SSHKey      string
	SSHPassword string
	AuthMethod  string // "password", "key", or "local"
	IsLocal     bool   // true = run via nsenter on host, no SSH needed

	// Cloudflare Access SSH tunnel fields (connect_type == "cloudflare")
	ConnectType    string // "ssh" | "cloudflare" | "local"
	CFHostname     string // e.g. "ssh.gigaza.org"
	CFClientID     string // Cloudflare Access Service Token client ID
	CFClientSecret string // Cloudflare Access Service Token client secret
}

// Result holds the output of a provisioning step
type Result struct {
	Module    string `json:"module"`
	Status    string `json:"status"` // completed, failed
	Output    string `json:"output"`
	Duration  string `json:"duration"`
}

// RunScript runs a shell script on the server, routing to local nsenter,
// Cloudflare Access SSH tunnel, or direct SSH as appropriate.
func RunScript(server ServerInfo, script string) (string, error) {
	if server.IsLocal || server.ConnectType == "local" {
		return RunScriptLocally(script)
	}
	if server.ConnectType == "cloudflare" {
		needsSudo := server.SSHUser != "" && server.SSHUser != "root"
		return runCloudflareSSH(server, script, needsSudo)
	}
	needsSudo := server.SSHUser != "" && server.SSHUser != "root"
	return runSSH(server, script, needsSudo)
}

// RunScriptAsUser runs a script as the logged-in user (no sudo).
func RunScriptAsUser(server ServerInfo, script string) (string, error) {
	if server.IsLocal || server.ConnectType == "local" {
		return RunScriptLocally(script)
	}
	if server.ConnectType == "cloudflare" {
		return runCloudflareSSH(server, script, false)
	}
	return runSSH(server, script, false)
}

// RunScriptAsSystemUser runs a script as a specific system user.
func RunScriptAsSystemUser(server ServerInfo, systemUser string, script string) (string, error) {
	wrapSudo := func(s string) string {
		if systemUser == "" || systemUser == "root" {
			return s
		}
		return fmt.Sprintf("sudo -u '%s' bash -c '%s'",
			strings.ReplaceAll(systemUser, "'", "'\"'\"'"),
			strings.ReplaceAll(s, "'", "'\"'\"'"))
	}
	if server.IsLocal || server.ConnectType == "local" {
		return RunScriptLocally(wrapSudo(script))
	}
	if server.ConnectType == "cloudflare" {
		return runCloudflareSSH(server, wrapSudo(script), false)
	}
	if systemUser == "" || systemUser == "root" {
		return RunScript(server, script)
	}
	return runSSH(server, wrapSudo(script), false)
}

// procConn wraps a subprocess stdin/stdout as a net.Conn so the Go SSH client
// can use cloudflared as a transparent ProxyCommand.
type procConn struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
}

func (c *procConn) Read(b []byte) (int, error)  { return c.stdout.Read(b) }
func (c *procConn) Write(b []byte) (int, error) { return c.stdin.Write(b) }
func (c *procConn) Close() error {
	c.stdin.Close()
	c.stdout.Close()
	_ = c.cmd.Process.Kill()
	_ = c.cmd.Wait()
	return nil
}
func (c *procConn) LocalAddr() net.Addr             { return &net.TCPAddr{} }
func (c *procConn) RemoteAddr() net.Addr            { return &net.TCPAddr{} }
func (c *procConn) SetDeadline(_ time.Time) error      { return nil }
func (c *procConn) SetReadDeadline(_ time.Time) error  { return nil }
func (c *procConn) SetWriteDeadline(_ time.Time) error { return nil }

// runCloudflareSSH connects via `cloudflared access ssh` as a ProxyCommand,
// authenticating with a Cloudflare Access Service Token (non-interactive).
// The SSH layer on top uses the usual key/password credentials.
func runCloudflareSSH(server ServerInfo, script string, useSudo bool) (string, error) {
	if server.CFHostname == "" {
		return "", fmt.Errorf("cloudflare access: cf_hostname is not configured for this server")
	}

	cmd := exec.Command("cloudflared", "access", "ssh", "--hostname", server.CFHostname)
	cmd.Env = append(os.Environ(),
		"CF_ACCESS_CLIENT_ID="+server.CFClientID,
		"CF_ACCESS_CLIENT_SECRET="+server.CFClientSecret,
	)
	cmd.Stderr = io.Discard // keep stderr away from SSH traffic

	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("cloudflared stdin pipe: %w", err)
	}
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("cloudflared stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("cloudflared start: %w — is cloudflared installed?", err)
	}

	conn := &procConn{cmd: cmd, stdin: stdinPipe, stdout: stdoutPipe}
	defer conn.Close()

	sshCfg, err := buildSSHConfig(server)
	if err != nil {
		return "", err
	}

	sshConn, chans, reqs, err := ssh.NewClientConn(conn, server.CFHostname, sshCfg)
	if err != nil {
		return "", fmt.Errorf("SSH over Cloudflare tunnel to %s: %w", server.CFHostname, err)
	}
	client := ssh.NewClient(sshConn, chans, reqs)
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("ssh session: %w", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	innerScript := fmt.Sprintf("set -e\nexport DEBIAN_FRONTEND=noninteractive\n%s", script)
	wrappedScript := innerScript
	if useSudo {
		escaped := strings.ReplaceAll(innerScript, "'", "'\"'\"'")
		if server.SSHPassword != "" {
			escapedPass := strings.ReplaceAll(server.SSHPassword, "'", "'\"'\"'")
			wrappedScript = fmt.Sprintf("echo '%s' | sudo -S bash -c '%s'", escapedPass, escaped)
		} else {
			wrappedScript = fmt.Sprintf("sudo -n bash -c '%s'", escaped)
		}
	}

	if err := session.Run(wrappedScript); err != nil {
		return stderr.String() + "\n" + stdout.String(),
			fmt.Errorf("script failed: %w\nstderr: %s", err, stderr.String())
	}
	return stdout.String(), nil
}

// RunScriptLocally runs a script on the host system by entering host namespaces
// via nsenter. Requires the container to run with pid:host and SYS_PTRACE cap.
func RunScriptLocally(script string) (string, error) {
	innerScript := fmt.Sprintf("set -e\nexport DEBIAN_FRONTEND=noninteractive\n%s", script)
	cmd := exec.Command("nsenter", "-t", "1", "-m", "-u", "-i", "-n", "-p", "--", "bash", "-c", innerScript)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return stderr.String() + "\n" + stdout.String(),
			fmt.Errorf("local script failed: %w\nstderr: %s", err, stderr.String())
	}
	return stdout.String(), nil
}

func runSSH(server ServerInfo, script string, useSudo bool) (string, error) {
	config, err := buildSSHConfig(server)
	if err != nil {
		return "", fmt.Errorf("ssh config error: %w", err)
	}

	addr := fmt.Sprintf("%s:%d", server.IPAddress, server.Port)
	conn, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return "", fmt.Errorf("ssh dial error to %s: %w", addr, err)
	}
	defer conn.Close()

	session, err := conn.NewSession()
	if err != nil {
		return "", fmt.Errorf("ssh session error: %w", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	// Build the inner script with error handling
	innerScript := fmt.Sprintf("set -e\nexport DEBIAN_FRONTEND=noninteractive\n%s", script)

	var wrappedScript string
	if useSudo {
		// Escape single quotes in the script to safely embed in bash -c '...'
		escapedScript := strings.ReplaceAll(innerScript, "'", "'\"'\"'")

		if server.SSHPassword != "" {
			// Use sudo -S to read password from stdin (pipe it via echo)
			// Escape single quotes in password too
			escapedPass := strings.ReplaceAll(server.SSHPassword, "'", "'\"'\"'")
			wrappedScript = fmt.Sprintf("echo '%s' | sudo -S bash -c '%s'", escapedPass, escapedScript)
		} else {
			// No password available — try passwordless sudo (NOPASSWD in sudoers)
			wrappedScript = fmt.Sprintf("sudo -n bash -c '%s'", escapedScript)
		}
	} else {
		wrappedScript = innerScript
	}

	if err := session.Run(wrappedScript); err != nil {
		return stderr.String() + "\n" + stdout.String(), fmt.Errorf("script failed: %w\nstderr: %s", err, stderr.String())
	}

	return stdout.String(), nil
}

// ProvisionServer runs all module install scripts for a server
func ProvisionServer(server ServerInfo, modules []string, onProgress func(module, status, output string)) []Result {
	var results []Result

	for _, mod := range modules {
		script, ok := InstallScripts[mod]
		if !ok {
			log.Printf("  No install script for module: %s", mod)
			results = append(results, Result{Module: mod, Status: "failed", Output: "no install script available"})
			if onProgress != nil {
				onProgress(mod, "failed", "no install script available")
			}
			continue
		}

		if onProgress != nil {
			onProgress(mod, "running", "")
		}

		start := time.Now()
		output, err := RunScript(server, script)
		duration := time.Since(start).Round(time.Second).String()

		if err != nil {
			log.Printf("  Module %s failed: %v", mod, err)
			results = append(results, Result{Module: mod, Status: "failed", Output: output, Duration: duration})
			if onProgress != nil {
				onProgress(mod, "failed", output)
			}
		} else {
			log.Printf("  Module %s installed in %s", mod, duration)
			results = append(results, Result{Module: mod, Status: "completed", Output: output, Duration: duration})
			if onProgress != nil {
				onProgress(mod, "completed", output)
			}
		}
	}

	return results
}

func buildSSHConfig(server ServerInfo) (*ssh.ClientConfig, error) {
	config := &ssh.ClientConfig{
		User:            server.SSHUser,
		Timeout:         30 * time.Second,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // For panel use
	}

	if server.AuthMethod == "key" && server.SSHKey != "" {
		signer, err := ssh.ParsePrivateKey([]byte(server.SSHKey))
		if err != nil {
			// Try with empty passphrase first, then treat as unencrypted
			return nil, fmt.Errorf("failed to parse SSH key: %w", err)
		}
		config.Auth = []ssh.AuthMethod{ssh.PublicKeys(signer)}
	} else if server.SSHPassword != "" {
		config.Auth = []ssh.AuthMethod{ssh.Password(server.SSHPassword)}
	} else {
		return nil, fmt.Errorf("no valid SSH auth method available")
	}

	// Add keyboard-interactive as fallback
	if server.SSHPassword != "" {
		config.Auth = append(config.Auth, ssh.KeyboardInteractive(
			func(user, instruction string, questions []string, echos []bool) (answers []string, err error) {
				answers = make([]string, len(questions))
				for i := range answers {
					answers[i] = server.SSHPassword
				}
				return answers, nil
			},
		))
	}

	// Also handle hostkey verification for known hosts
	config.HostKeyCallback = func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		return nil // Trust all hosts in panel context
	}

	return config, nil
}
