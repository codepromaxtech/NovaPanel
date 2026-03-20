package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/novapanel/novapanel/internal/models"
)

type FileHandler struct{}

func NewFileHandler() *FileHandler {
	return &FileHandler{}
}

func sanitizePath(base, requested string) (string, error) {
	clean := filepath.Clean(requested)
	full := filepath.Join(base, clean)
	if !strings.HasPrefix(full, base) {
		return "", fmt.Errorf("path traversal attempt blocked")
	}
	return full, nil
}

// GET /api/v1/files?path=/var/www
func (h *FileHandler) List(c *gin.Context) {
	reqPath := c.DefaultQuery("path", "/var/www")
	base := "/var/www"
	fullPath, err := sanitizePath(base, reqPath)
	if err != nil {
		c.JSON(http.StatusForbidden, models.ErrorResponse{Error: err.Error()})
		return
	}

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Directory not found"})
		return
	}

	var files []models.FileEntry
	for _, entry := range entries {
		info, _ := entry.Info()
		fe := models.FileEntry{
			Name:  entry.Name(),
			Path:  filepath.Join(reqPath, entry.Name()),
			IsDir: entry.IsDir(),
		}
		if info != nil {
			fe.Size = info.Size()
			fe.Permissions = info.Mode().String()
			fe.ModifiedAt = info.ModTime().Format("2006-01-02 15:04:05")
		}
		files = append(files, fe)
	}

	c.JSON(http.StatusOK, files)
}

// GET /api/v1/files/content?path=/var/www/index.html
func (h *FileHandler) Read(c *gin.Context) {
	reqPath := c.Query("path")
	if reqPath == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "path is required"})
		return
	}

	base := "/var/www"
	fullPath, err := sanitizePath(base, reqPath)
	if err != nil {
		c.JSON(http.StatusForbidden, models.ErrorResponse{Error: err.Error()})
		return
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "File not found"})
		return
	}

	if info.Size() > 5*1024*1024 {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "File too large (> 5MB)"})
		return
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to read file"})
		return
	}

	c.JSON(http.StatusOK, models.FileContentResponse{
		Path:    reqPath,
		Content: string(content),
		Size:    info.Size(),
	})
}

// PUT /api/v1/files/content
func (h *FileHandler) Write(c *gin.Context) {
	var req struct {
		Path    string `json:"path" binding:"required"`
		Content string `json:"content"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	base := "/var/www"
	fullPath, err := sanitizePath(base, req.Path)
	if err != nil {
		c.JSON(http.StatusForbidden, models.ErrorResponse{Error: err.Error()})
		return
	}

	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to create directory"})
		return
	}

	if err := os.WriteFile(fullPath, []byte(req.Content), 0644); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to write file"})
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse{Message: "File saved"})
}

// POST /api/v1/files/upload
func (h *FileHandler) Upload(c *gin.Context) {
	targetDir := c.PostForm("path")
	if targetDir == "" {
		targetDir = "/var/www"
	}

	base := "/var/www"
	fullDir, err := sanitizePath(base, targetDir)
	if err != nil {
		c.JSON(http.StatusForbidden, models.ErrorResponse{Error: err.Error()})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "No file provided"})
		return
	}
	defer file.Close()

	if err := os.MkdirAll(fullDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to create directory"})
		return
	}

	destPath := filepath.Join(fullDir, header.Filename)
	out, err := os.Create(destPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to create file"})
		return
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to write upload"})
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse{Message: fmt.Sprintf("Uploaded %s", header.Filename)})
}

// DELETE /api/v1/files?path=/var/www/old.html
func (h *FileHandler) Delete(c *gin.Context) {
	reqPath := c.Query("path")
	if reqPath == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "path is required"})
		return
	}

	base := "/var/www"
	fullPath, err := sanitizePath(base, reqPath)
	if err != nil {
		c.JSON(http.StatusForbidden, models.ErrorResponse{Error: err.Error()})
		return
	}

	if err := os.RemoveAll(fullPath); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Failed to delete"})
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse{Message: "Deleted successfully"})
}
