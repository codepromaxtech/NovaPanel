package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/novapanel/novapanel/internal/models"
	"github.com/novapanel/novapanel/internal/services"
)

type CronHandler struct {
	svc *services.CronService
}

func NewCronHandler(svc *services.CronService) *CronHandler {
	return &CronHandler{svc: svc}
}

// POST /cron/list
func (h *CronHandler) List(c *gin.Context) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
		User     string `json:"user"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	output, err := h.svc.ListCronJobs(c.Request.Context(), req.ServerID, req.User)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"output": output})
}

// POST /cron/add
func (h *CronHandler) Add(c *gin.Context) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
		User     string `json:"user"`
		Schedule string `json:"schedule" binding:"required"`
		Command  string `json:"command" binding:"required"`
		Comment  string `json:"comment"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	output, err := h.svc.AddCronJob(c.Request.Context(), req.ServerID, req.User, req.Schedule, req.Command, req.Comment)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"output": output, "message": "Cron job added"})
}

// POST /cron/delete
func (h *CronHandler) Delete(c *gin.Context) {
	var req struct {
		ServerID   string `json:"server_id" binding:"required"`
		User       string `json:"user"`
		LineNumber int    `json:"line_number" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	output, err := h.svc.DeleteCronJob(c.Request.Context(), req.ServerID, req.User, req.LineNumber)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"output": output, "message": "Cron job deleted"})
}

// POST /cron/update
func (h *CronHandler) Update(c *gin.Context) {
	var req struct {
		ServerID   string `json:"server_id" binding:"required"`
		User       string `json:"user"`
		LineNumber int    `json:"line_number" binding:"required"`
		NewLine    string `json:"new_line" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	output, err := h.svc.UpdateCronJob(c.Request.Context(), req.ServerID, req.User, req.LineNumber, req.NewLine)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"output": output, "message": "Cron job updated"})
}

// POST /cron/users
func (h *CronHandler) ListUsers(c *gin.Context) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	output, err := h.svc.ListCronUsers(c.Request.Context(), req.ServerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"output": output})
}

// POST /cron/logs
func (h *CronHandler) Logs(c *gin.Context) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
		Lines    int    `json:"lines"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	lines := req.Lines
	if lines <= 0 {
		lines = 50
	}
	output, err := h.svc.GetCronLog(c.Request.Context(), req.ServerID, lines)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"output": output})
}

// ──── Systemctl Handler ────

type SystemctlHandler struct {
	svc *services.SystemctlService
}

func NewSystemctlHandler(svc *services.SystemctlService) *SystemctlHandler {
	return &SystemctlHandler{svc: svc}
}

// POST /systemctl/list
func (h *SystemctlHandler) List(c *gin.Context) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
		Filter   string `json:"filter"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	output, err := h.svc.ListServices(c.Request.Context(), req.ServerID, req.Filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"output": output})
}

// POST /systemctl/status
func (h *SystemctlHandler) Status(c *gin.Context) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
		Service  string `json:"service" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	output, err := h.svc.GetServiceStatus(c.Request.Context(), req.ServerID, req.Service)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"output": output})
		return
	}
	c.JSON(http.StatusOK, gin.H{"output": output})
}

func (h *SystemctlHandler) action(c *gin.Context, fn func(ctx gin.Context, serverID, service string) (string, error)) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
		Service  string `json:"service" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	_ = req
}

// POST /systemctl/start
func (h *SystemctlHandler) Start(c *gin.Context) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
		Service  string `json:"service" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	output, err := h.svc.StartService(c.Request.Context(), req.ServerID, req.Service)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"output": output, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"output": output, "message": "Service started"})
}

// POST /systemctl/stop
func (h *SystemctlHandler) Stop(c *gin.Context) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
		Service  string `json:"service" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	output, err := h.svc.StopService(c.Request.Context(), req.ServerID, req.Service)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"output": output, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"output": output, "message": "Service stopped"})
}

// POST /systemctl/restart
func (h *SystemctlHandler) Restart(c *gin.Context) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
		Service  string `json:"service" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	output, err := h.svc.RestartService(c.Request.Context(), req.ServerID, req.Service)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"output": output, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"output": output, "message": "Service restarted"})
}

// POST /systemctl/reload
func (h *SystemctlHandler) Reload(c *gin.Context) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
		Service  string `json:"service" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	output, err := h.svc.ReloadService(c.Request.Context(), req.ServerID, req.Service)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"output": output, "error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"output": output, "message": "Service reloaded"})
}

// POST /systemctl/enable
func (h *SystemctlHandler) Enable(c *gin.Context) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
		Service  string `json:"service" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	output, err := h.svc.EnableService(c.Request.Context(), req.ServerID, req.Service)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"output": output, "message": "Service enabled"})
}

// POST /systemctl/disable
func (h *SystemctlHandler) Disable(c *gin.Context) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
		Service  string `json:"service" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	output, err := h.svc.DisableService(c.Request.Context(), req.ServerID, req.Service)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"output": output, "message": "Service disabled"})
}

// POST /systemctl/logs
func (h *SystemctlHandler) Logs(c *gin.Context) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
		Service  string `json:"service" binding:"required"`
		Lines    int    `json:"lines"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	lines := req.Lines
	if lines <= 0 {
		lines = 50
	}
	output, err := h.svc.GetServiceLogs(c.Request.Context(), req.ServerID, req.Service, lines)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"output": output})
		return
	}
	c.JSON(http.StatusOK, gin.H{"output": output})
}

// POST /systemctl/failed
func (h *SystemctlHandler) Failed(c *gin.Context) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	output, err := h.svc.ListFailedServices(c.Request.Context(), req.ServerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"output": output})
}

// POST /systemctl/timers
func (h *SystemctlHandler) Timers(c *gin.Context) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	output, err := h.svc.ListTimers(c.Request.Context(), req.ServerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"output": output})
}

// POST /systemctl/daemon-reload
func (h *SystemctlHandler) DaemonReload(c *gin.Context) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	output, err := h.svc.DaemonReload(c.Request.Context(), req.ServerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"output": output})
}

// ──── Backup Manager Handler ────

type BackupManagerHandler struct {
	svc *services.BackupManager
}

func NewBackupManagerHandler(svc *services.BackupManager) *BackupManagerHandler {
	return &BackupManagerHandler{svc: svc}
}

// POST /backup-manager/database
func (h *BackupManagerHandler) BackupDatabase(c *gin.Context) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
		Engine   string `json:"engine" binding:"required"`
		Database string `json:"database" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	userID := c.MustGet("user_id").(uuid.UUID)
	result, err := h.svc.BackupDatabase(c.Request.Context(), userID, req.ServerID, req.Engine, req.Database)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// POST /backup-manager/site
func (h *BackupManagerHandler) BackupSite(c *gin.Context) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
		SitePath string `json:"site_path" binding:"required"`
		SiteName string `json:"site_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	userID := c.MustGet("user_id").(uuid.UUID)
	result, err := h.svc.BackupSite(c.Request.Context(), userID, req.ServerID, req.SitePath, req.SiteName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// POST /backup-manager/full
func (h *BackupManagerHandler) BackupFull(c *gin.Context) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	userID := c.MustGet("user_id").(uuid.UUID)
	result, err := h.svc.BackupFull(c.Request.Context(), userID, req.ServerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// POST /backup-manager/restore/database
func (h *BackupManagerHandler) RestoreDatabase(c *gin.Context) {
	var req struct {
		ServerID   string `json:"server_id" binding:"required"`
		Engine     string `json:"engine" binding:"required"`
		Database   string `json:"database" binding:"required"`
		BackupPath string `json:"backup_path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	output, err := h.svc.RestoreDatabase(c.Request.Context(), req.ServerID, req.Engine, req.Database, req.BackupPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"output": output, "message": "Restore completed"})
}

// POST /backup-manager/restore/site
func (h *BackupManagerHandler) RestoreSite(c *gin.Context) {
	var req struct {
		ServerID   string `json:"server_id" binding:"required"`
		SitePath   string `json:"site_path" binding:"required"`
		BackupPath string `json:"backup_path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	output, err := h.svc.RestoreSite(c.Request.Context(), req.ServerID, req.SitePath, req.BackupPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"output": output, "message": "Site restored"})
}

// POST /backup-manager/files
func (h *BackupManagerHandler) ListFiles(c *gin.Context) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	output, err := h.svc.ListBackupFiles(c.Request.Context(), req.ServerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"output": output})
}

// POST /backup-manager/files/delete
func (h *BackupManagerHandler) DeleteFile(c *gin.Context) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
		FilePath string `json:"file_path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	output, err := h.svc.DeleteBackupFile(c.Request.Context(), req.ServerID, req.FilePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"output": output, "message": "File deleted"})
}

// Need uuid import for BackupManagerHandler
func init() {
	_ = strconv.Atoi
}
