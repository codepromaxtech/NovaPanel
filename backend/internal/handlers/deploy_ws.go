package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/novapanel/novapanel/internal/services"
)

var deployUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// DeployWSHandler handles WebSocket connections for streaming deployment logs
type DeployWSHandler struct {
	deploySvc *services.DeployService
}

func NewDeployWSHandler(deploySvc *services.DeployService) *DeployWSHandler {
	return &DeployWSHandler{deploySvc: deploySvc}
}

// HandleDeployLogs upgrades to WebSocket and streams build logs in real-time
// GET /api/v1/deployments/:id/ws
func (h *DeployWSHandler) HandleDeployLogs(c *gin.Context) {
	deploymentID := c.Param("id")

	conn, err := deployUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	// Set read deadline and handle pong
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Read messages in background (to detect client disconnect)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	// Poll deployment logs and send updates
	var lastLogLen int
	ticker := time.NewTicker(500 * time.Millisecond)
	pingTicker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	defer pingTicker.Stop()

	for {
		select {
		case <-done:
			return
		case <-pingTicker.C:
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case <-ticker.C:
			logs, status, err := h.deploySvc.GetLogs(c.Request.Context(), deploymentID)
			if err != nil {
				msg, _ := json.Marshal(map[string]string{
					"type":  "error",
					"error": "deployment not found",
				})
				conn.WriteMessage(websocket.TextMessage, msg)
				return
			}

			// Only send new log content
			if len(logs) > lastLogLen {
				newContent := logs[lastLogLen:]
				lastLogLen = len(logs)

				msg, _ := json.Marshal(map[string]interface{}{
					"type":   "log",
					"data":   newContent,
					"status": status,
				})
				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					return
				}
			}

			// Send status update
			if status == "success" || status == "failed" {
				msg, _ := json.Marshal(map[string]interface{}{
					"type":   "complete",
					"status": status,
					"logs":   logs,
				})
				conn.WriteMessage(websocket.TextMessage, msg)
				return
			}
		}
	}
}
