package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/novapanel/novapanel/internal/services"
)

type ServerModulesHandler struct {
	svc *services.ServerModulesService
}

func NewServerModulesHandler(svc *services.ServerModulesService) *ServerModulesHandler {
	return &ServerModulesHandler{svc: svc}
}

// GET /servers/:id/modules
func (h *ServerModulesHandler) ListModules(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid server id"})
		return
	}
	modules, err := h.svc.ListModules(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": modules})
}

// POST /servers/:id/modules
func (h *ServerModulesHandler) EnableModule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid server id"})
		return
	}
	var req struct {
		Module string `json:"module"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Module == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "module required"})
		return
	}
	if err := h.svc.EnableModule(c.Request.Context(), id, req.Module); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "module enabled"})
}

// DELETE /servers/:id/modules/:module
func (h *ServerModulesHandler) DisableModule(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid server id"})
		return
	}
	module := c.Param("module")
	if err := h.svc.RemoveModule(c.Request.Context(), id, module); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "module disabled"})
}

// PUT /servers/:id/modules — set all modules at once
func (h *ServerModulesHandler) SetModules(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid server id"})
		return
	}
	var req struct {
		Modules []string `json:"modules"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "modules array required"})
		return
	}
	if err := h.svc.SetModules(c.Request.Context(), id, req.Modules); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "modules updated"})
}

// GET /modules/available — list all available module types
func (h *ServerModulesHandler) ListAvailable(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"data": services.AvailableModules})
}

// GET /modules/active — list globally active modules (across all servers)
func (h *ServerModulesHandler) ListActive(c *gin.Context) {
	modules, err := h.svc.GetActiveModulesGlobal(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": modules})
}

// GET /modules/counts — module usage counts
func (h *ServerModulesHandler) ModuleCounts(c *gin.Context) {
	counts, err := h.svc.GetModuleCounts(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": counts})
}
