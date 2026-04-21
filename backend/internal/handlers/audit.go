package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/novapanel/novapanel/internal/models"
)

type AuditHandler struct {
	pool *pgxpool.Pool
}

func NewAuditHandler(pool *pgxpool.Pool) *AuditHandler {
	return &AuditHandler{pool: pool}
}

func (h *AuditHandler) List(c *gin.Context) {
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}

	rows, err := h.pool.Query(c.Request.Context(),
		`SELECT id::text, COALESCE(user_id::text,''), action, COALESCE(resource,''),
		        COALESCE(resource_id::text,''), COALESCE(host(ip_address),''),
		        COALESCE(details::text,'{}'), created_at::text
		 FROM audit_logs ORDER BY created_at DESC LIMIT $1`, perPage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	defer rows.Close()

	type AuditEntry struct {
		ID           string `json:"id"`
		UserID       string `json:"user_id"`
		Action       string `json:"action"`
		ResourceType string `json:"resource_type"`
		ResourceID   string `json:"resource_id"`
		ResourceName string `json:"resource_name"`
		IPAddress    string `json:"ip_address"`
		Details      string `json:"details"`
		CreatedAt    string `json:"created_at"`
	}

	var logs []AuditEntry
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.ID, &e.UserID, &e.Action, &e.ResourceType,
			&e.ResourceID, &e.IPAddress, &e.Details, &e.CreatedAt); err != nil {
			continue
		}
		e.ResourceName = e.ResourceID
		logs = append(logs, e)
	}

	if logs == nil {
		logs = []AuditEntry{}
	}

	c.JSON(http.StatusOK, gin.H{"data": logs})
}
