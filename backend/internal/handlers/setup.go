package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/novapanel/novapanel/internal/models"
	"github.com/novapanel/novapanel/internal/queue"
)

type SetupHandler struct {
	db        *pgxpool.Pool
	taskQueue *queue.TaskQueue
}

func NewSetupHandler(db *pgxpool.Pool, taskQueue *queue.TaskQueue) *SetupHandler {
	return &SetupHandler{db: db, taskQueue: taskQueue}
}

// GET /servers/:id/setup — get setup logs for a server
func (h *SetupHandler) GetSetupLogs(c *gin.Context) {
	serverID := c.Param("id")

	rows, err := h.db.Query(c.Request.Context(),
		`SELECT id, server_id, module, status, COALESCE(output, ''), COALESCE(duration, ''),
			started_at, completed_at, created_at
		 FROM setup_logs WHERE server_id = $1 ORDER BY created_at`,
		serverID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	defer rows.Close()

	type SetupLog struct {
		ID          string  `json:"id"`
		ServerID    string  `json:"server_id"`
		Module      string  `json:"module"`
		Status      string  `json:"status"`
		Output      string  `json:"output"`
		Duration    string  `json:"duration"`
		StartedAt   *string `json:"started_at"`
		CompletedAt *string `json:"completed_at"`
		CreatedAt   string  `json:"created_at"`
	}

	var logs []SetupLog
	for rows.Next() {
		var l SetupLog
		rows.Scan(&l.ID, &l.ServerID, &l.Module, &l.Status, &l.Output, &l.Duration,
			&l.StartedAt, &l.CompletedAt, &l.CreatedAt)
		logs = append(logs, l)
	}

	// Also get server setup_status
	var setupStatus string
	h.db.QueryRow(c.Request.Context(), `SELECT COALESCE(setup_status, 'none') FROM servers WHERE id = $1`, serverID).Scan(&setupStatus)

	c.JSON(http.StatusOK, gin.H{
		"setup_status": setupStatus,
		"logs":         logs,
	})
}

// POST /servers/:id/setup — manually trigger provisioning
func (h *SetupHandler) TriggerSetup(c *gin.Context) {
	serverID := c.Param("id")
	var req struct {
		Modules []string `json:"modules" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "modules array required"})
		return
	}

	payload := map[string]interface{}{
		"server_id": serverID,
		"modules":   req.Modules,
	}
	taskID, err := h.taskQueue.Enqueue(c.Request.Context(), "server_setup", payload, 1, serverID, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message": "Server setup enqueued",
		"task_id": taskID,
	})
}
