package handlers

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/docker/docker/api/types/container"
	"github.com/gin-gonic/gin"
	ws "github.com/gorilla/websocket"
	"github.com/novapanel/novapanel/internal/services"
)

// DockerExecHandler manages WebSocket-based docker exec sessions.
type DockerExecHandler struct {
	dockerService *services.DockerService
}

func NewDockerExecHandler(dockerService *services.DockerService) *DockerExecHandler {
	return &DockerExecHandler{dockerService: dockerService}
}

var execUpgrader = ws.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true },
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
}

type execMsg struct {
	Type string `json:"type"`
	Data string `json:"data"`
	Cols int    `json:"cols,omitempty"`
	Rows int    `json:"rows,omitempty"`
}

// HandleExec opens an interactive exec session into a container via WebSocket.
func (h *DockerExecHandler) HandleExec(c *gin.Context) {
	containerID := c.Param("id")
	shell := c.DefaultQuery("shell", "/bin/sh")

	// Upgrade WebSocket
	conn, err := execUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("Exec WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	ctx := c.Request.Context()
	cli := h.dockerService.Client()

	// Create exec
	execConfig := container.ExecOptions{
		Cmd:          []string{shell},
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
	}
	execResp, err := cli.ContainerExecCreate(ctx, containerID, execConfig)
	if err != nil {
		log.Printf("Exec create error: %v", err)
		errData, _ := json.Marshal(execMsg{Type: "output", Data: "\r\n\x1b[31m✕ Exec failed: " + err.Error() + "\x1b[0m\r\n"})
		conn.WriteMessage(ws.TextMessage, errData)
		return
	}

	// Attach
	attach, err := cli.ContainerExecAttach(ctx, execResp.ID, container.ExecAttachOptions{Tty: true})
	if err != nil {
		log.Printf("Exec attach error: %v", err)
		errData, _ := json.Marshal(execMsg{Type: "output", Data: "\r\n\x1b[31m✕ Attach failed: " + err.Error() + "\x1b[0m\r\n"})
		conn.WriteMessage(ws.TextMessage, errData)
		return
	}
	defer attach.Close()

	log.Printf("Exec session opened for container %s (shell: %s)", containerID, shell)

	// Send connected message
	okData, _ := json.Marshal(execMsg{Type: "output", Data: "\x1b[32m✓ Connected to container " + containerID + "\x1b[0m\r\n\r\n"})
	conn.WriteMessage(ws.TextMessage, okData)

	// Docker stdout → WebSocket
	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := attach.Reader.Read(buf)
			if n > 0 {
				outData, _ := json.Marshal(execMsg{Type: "output", Data: string(buf[:n])})
				if writeErr := conn.WriteMessage(ws.TextMessage, outData); writeErr != nil {
					break
				}
			}
			if err != nil {
				break
			}
		}
		conn.Close()
	}()

	// WebSocket → Docker stdin
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var incoming execMsg
		if jsonErr := json.Unmarshal(msg, &incoming); jsonErr != nil {
			continue
		}

		switch incoming.Type {
		case "input":
			attach.Conn.Write([]byte(incoming.Data))
		case "resize":
			if incoming.Cols > 0 && incoming.Rows > 0 {
				cli.ContainerExecResize(ctx, execResp.ID, container.ResizeOptions{
					Height: uint(incoming.Rows),
					Width:  uint(incoming.Cols),
				})
			}
		}
	}

	log.Printf("Exec session closed for container %s", containerID)
}
