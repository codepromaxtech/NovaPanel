package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/novapanel/novapanel/internal/models"
	"github.com/novapanel/novapanel/internal/services"
)

type DBManagerHandler struct {
	svc *services.DBManagerService
}

func NewDBManagerHandler(svc *services.DBManagerService) *DBManagerHandler {
	return &DBManagerHandler{svc: svc}
}

// POST /db-manager/query
func (h *DBManagerHandler) RunQuery(c *gin.Context) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
		Engine   string `json:"engine" binding:"required"`
		Database string `json:"database" binding:"required"`
		Query    string `json:"query" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	output, err := h.svc.RunQuery(c.Request.Context(), req.ServerID, req.Engine, req.Database, req.Query)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"output": output, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"output": output})
}

// POST /db-manager/tables
func (h *DBManagerHandler) ListTables(c *gin.Context) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
		Engine   string `json:"engine" binding:"required"`
		Database string `json:"database" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	output, err := h.svc.ListTables(c.Request.Context(), req.ServerID, req.Engine, req.Database)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"output": output})
}

// POST /db-manager/describe
func (h *DBManagerHandler) DescribeTable(c *gin.Context) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
		Engine   string `json:"engine" binding:"required"`
		Database string `json:"database" binding:"required"`
		Table    string `json:"table" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	output, err := h.svc.DescribeTable(c.Request.Context(), req.ServerID, req.Engine, req.Database, req.Table)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"output": output})
}

// POST /db-manager/size
func (h *DBManagerHandler) GetDBSize(c *gin.Context) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
		Engine   string `json:"engine" binding:"required"`
		Database string `json:"database" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	output, err := h.svc.GetDBSize(c.Request.Context(), req.ServerID, req.Engine, req.Database)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"output": output})
}

// POST /db-manager/databases
func (h *DBManagerHandler) ListDBsOnServer(c *gin.Context) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
		Engine   string `json:"engine" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	output, err := h.svc.ListDatabasesOnServer(c.Request.Context(), req.ServerID, req.Engine)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"output": output})
}

// POST /db-manager/users
func (h *DBManagerHandler) ListUsers(c *gin.Context) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
		Engine   string `json:"engine" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	output, err := h.svc.ListUsers(c.Request.Context(), req.ServerID, req.Engine)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"output": output})
}

// POST /db-manager/users/create
func (h *DBManagerHandler) CreateUser(c *gin.Context) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
		Engine   string `json:"engine" binding:"required"`
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
		Database string `json:"database" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	output, err := h.svc.CreateDBUser(c.Request.Context(), req.ServerID, req.Engine, req.Username, req.Password, req.Database)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"output": output, "message": "User created"})
}

// POST /db-manager/export
func (h *DBManagerHandler) ExportDB(c *gin.Context) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
		Engine   string `json:"engine" binding:"required"`
		Database string `json:"database" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	output, err := h.svc.ExportDB(c.Request.Context(), req.ServerID, req.Engine, req.Database)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"output": output, "message": "Export completed"})
}

// POST /db-manager/import
func (h *DBManagerHandler) ImportDB(c *gin.Context) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
		Engine   string `json:"engine" binding:"required"`
		Database string `json:"database" binding:"required"`
		FilePath string `json:"file_path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	output, err := h.svc.ImportDB(c.Request.Context(), req.ServerID, req.Engine, req.Database, req.FilePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"output": output, "message": "Import completed"})
}

// POST /db-manager/tools/deploy
func (h *DBManagerHandler) DeployTool(c *gin.Context) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
		Engine   string `json:"engine" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	info, err := h.svc.DeployDBTool(c.Request.Context(), req.ServerID, req.Engine)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, info)
}

// POST /db-manager/tools/status
func (h *DBManagerHandler) ToolStatus(c *gin.Context) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
		Engine   string `json:"engine" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	info, err := h.svc.GetToolStatus(c.Request.Context(), req.ServerID, req.Engine)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, info)
}

// POST /db-manager/tools/stop
func (h *DBManagerHandler) StopTool(c *gin.Context) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
		Engine   string `json:"engine" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	if err := h.svc.StopTool(c.Request.Context(), req.ServerID, req.Engine); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Tool stopped"})
}
