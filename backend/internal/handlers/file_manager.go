package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/novapanel/novapanel/internal/models"
	"github.com/novapanel/novapanel/internal/services"
)

type FileManagerHandler struct {
	svc *services.FileManagerService
}

func NewFileManagerHandler(svc *services.FileManagerService) *FileManagerHandler {
	return &FileManagerHandler{svc: svc}
}

// POST /files/list
func (h *FileManagerHandler) List(c *gin.Context) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
		Path     string `json:"path"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	if req.Path == "" {
		req.Path = "/"
	}
	output, err := h.svc.ListFiles(c.Request.Context(), req.ServerID, req.Path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	files, _ := services.ParseFileList(output)
	c.JSON(http.StatusOK, gin.H{"files": files, "path": req.Path})
}

// POST /files/read
func (h *FileManagerHandler) Read(c *gin.Context) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
		Path     string `json:"path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	output, err := h.svc.ReadFile(c.Request.Context(), req.ServerID, req.Path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	if strings.HasPrefix(output, "ERROR:") {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: strings.TrimPrefix(output, "ERROR:")})
		return
	}
	isBinary := strings.HasPrefix(output, "BINARY:")
	c.JSON(http.StatusOK, gin.H{
		"path":      req.Path,
		"content":   output,
		"is_binary": isBinary,
	})
}

// POST /files/write
func (h *FileManagerHandler) Write(c *gin.Context) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
		Path     string `json:"path" binding:"required"`
		Content  string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	output, err := h.svc.WriteFile(c.Request.Context(), req.ServerID, req.Path, req.Content)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": output})
}

// POST /files/create
func (h *FileManagerHandler) Create(c *gin.Context) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
		Path     string `json:"path" binding:"required"`
		IsDir    bool   `json:"is_dir"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	var output string
	var err error
	if req.IsDir {
		output, err = h.svc.CreateDir(c.Request.Context(), req.ServerID, req.Path)
	} else {
		output, err = h.svc.CreateFile(c.Request.Context(), req.ServerID, req.Path)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": output})
}

// POST /files/rename
func (h *FileManagerHandler) Rename(c *gin.Context) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
		OldPath  string `json:"old_path" binding:"required"`
		NewPath  string `json:"new_path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	output, err := h.svc.Rename(c.Request.Context(), req.ServerID, req.OldPath, req.NewPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": output})
}

// POST /files/copy
func (h *FileManagerHandler) Copy(c *gin.Context) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
		Src      string `json:"src" binding:"required"`
		Dest     string `json:"dest" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	output, err := h.svc.Copy(c.Request.Context(), req.ServerID, req.Src, req.Dest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": output})
}

// POST /files/move
func (h *FileManagerHandler) Move(c *gin.Context) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
		Src      string `json:"src" binding:"required"`
		Dest     string `json:"dest" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	output, err := h.svc.Move(c.Request.Context(), req.ServerID, req.Src, req.Dest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": output})
}

// POST /files/delete
func (h *FileManagerHandler) Delete(c *gin.Context) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
		Path     string `json:"path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	output, err := h.svc.Delete(c.Request.Context(), req.ServerID, req.Path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": output})
}

// POST /files/chmod
func (h *FileManagerHandler) Chmod(c *gin.Context) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
		Path     string `json:"path" binding:"required"`
		Mode     string `json:"mode" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	output, err := h.svc.Chmod(c.Request.Context(), req.ServerID, req.Path, req.Mode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": output})
}

// POST /files/chown
func (h *FileManagerHandler) Chown(c *gin.Context) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
		Path     string `json:"path" binding:"required"`
		Owner    string `json:"owner" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	output, err := h.svc.Chown(c.Request.Context(), req.ServerID, req.Path, req.Owner)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": output})
}

// POST /files/search
func (h *FileManagerHandler) Search(c *gin.Context) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
		Path     string `json:"path"`
		Query    string `json:"query" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	if req.Path == "" {
		req.Path = "/"
	}
	output, err := h.svc.Search(c.Request.Context(), req.ServerID, req.Path, req.Query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	results := strings.Split(strings.TrimSpace(output), "\n")
	if len(results) == 1 && results[0] == "" {
		results = []string{}
	}
	c.JSON(http.StatusOK, gin.H{"results": results})
}

// POST /files/info
func (h *FileManagerHandler) Info(c *gin.Context) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
		Path     string `json:"path" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	output, err := h.svc.FileInfo(c.Request.Context(), req.ServerID, req.Path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"info": output})
}

// POST /files/extract
func (h *FileManagerHandler) Extract(c *gin.Context) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
		Path     string `json:"path" binding:"required"`
		Dest     string `json:"dest"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	output, err := h.svc.Extract(c.Request.Context(), req.ServerID, req.Path, req.Dest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": output})
}

// POST /files/compress
func (h *FileManagerHandler) Compress(c *gin.Context) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
		Path     string `json:"path" binding:"required"`
		Dest     string `json:"dest"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	output, err := h.svc.Compress(c.Request.Context(), req.ServerID, req.Path, req.Dest)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": output})
}

// POST /files/grep
func (h *FileManagerHandler) Grep(c *gin.Context) {
	var req struct {
		ServerID string `json:"server_id" binding:"required"`
		Path     string `json:"path"`
		Pattern  string `json:"pattern" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}
	if req.Path == "" {
		req.Path = "/"
	}
	output, err := h.svc.GrepInFiles(c.Request.Context(), req.ServerID, req.Path, req.Pattern)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: err.Error()})
		return
	}
	results := strings.Split(strings.TrimSpace(output), "\n")
	if len(results) == 1 && results[0] == "" {
		results = []string{}
	}
	c.JSON(http.StatusOK, gin.H{"results": results})
}
