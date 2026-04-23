package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/novapanel/novapanel/internal/models"
	"github.com/novapanel/novapanel/internal/queue"
	"github.com/novapanel/novapanel/internal/services"
)

type DomainHandler struct {
	service     *services.DomainService
	serverSvc   *services.ServerService
	taskQueue   *queue.TaskQueue
	pool        *pgxpool.Pool
}

func NewDomainHandler(service *services.DomainService, serverSvc *services.ServerService, taskQueue *queue.TaskQueue, pool *pgxpool.Pool) *DomainHandler {
	return &DomainHandler{
		service:   service,
		serverSvc: serverSvc,
		taskQueue: taskQueue,
		pool:      pool,
	}
}

// POST /api/v1/domains
func (h *DomainHandler) Create(c *gin.Context) {
	var req models.CreateDomainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Invalid request", Message: err.Error()})
		return
	}

	userID := c.MustGet("user_id").(uuid.UUID)
	domain, err := h.service.Create(c.Request.Context(), userID.String(), req)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	if domain.IsLoadBalancer {
		// Emit load balancer config task
		var targetIPs []string
		for _, sid := range domain.BackendServerIDs {
			if s, err := h.serverSvc.GetByID(c.Request.Context(), sid.String()); err == nil {
				targetIPs = append(targetIPs, s.IPAddress)
			}
		}
		serverID := ""
		if domain.ServerID != nil {
			serverID = domain.ServerID.String()
		}
		h.taskQueue.Enqueue(
			c.Request.Context(),
			"nginx:configure",
			map[string]interface{}{
				"domain":           domain.Name,
				"server_id":        serverID,
				"is_load_balancer": true,
				"target_ips":       targetIPs,
			},
			1,
			serverID,
			userID.String(),
		)
	}

	writeAudit(c.Request.Context(), h.pool, userID, "create", "domain", domain.ID.String(), c.ClientIP(),
		map[string]interface{}{"name": domain.Name})
	c.JSON(http.StatusCreated, domain)
}

// GET /api/v1/domains
func (h *DomainHandler) List(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)
	role := c.MustGet("user_role").(string)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))

	resp, err := h.service.List(c.Request.Context(), userID.String(), role, page, perPage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GET /api/v1/domains/:id
func (h *DomainHandler) GetByID(c *gin.Context) {
	id := c.Param("id")
	domain, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, domain)
}

// PUT /api/v1/domains/:id
func (h *DomainHandler) Update(c *gin.Context) {
	id := c.Param("id")
	var req models.UpdateDomainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "Invalid request", Message: err.Error()})
		return
	}

	domain, err := h.service.Update(c.Request.Context(), id, req)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, domain)
}

// DELETE /api/v1/domains/:id
func (h *DomainHandler) Delete(c *gin.Context) {
	id := c.Param("id")
	userID := c.MustGet("user_id").(uuid.UUID)
	role := c.MustGet("user_role").(string)
	if err := h.service.Delete(c.Request.Context(), id, userID, role); err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: err.Error()})
		return
	}
	writeAudit(c.Request.Context(), h.pool, userID, "delete", "domain", id, c.ClientIP(), nil)
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Domain deleted successfully"})
}

// POST /api/v1/domains/:id/ssl/wildcard
func (h *DomainHandler) WildcardSSL(c *gin.Context) {
	id := c.Param("id")
	if err := h.service.ProvisionWildcardSSL(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "wildcard SSL certificate issued successfully"})
}
