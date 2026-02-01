package api

import (
	"io"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/nebula/nebula/internal/files"
)

// FilesHandler handles file manager endpoints
type FilesHandler struct {
	manager *files.Manager
}

// NewFilesHandler creates a new files handler
func NewFilesHandler(manager *files.Manager) *FilesHandler {
	return &FilesHandler{manager: manager}
}

// List godoc
// @Summary List directory contents
// @Description Returns files and directories in a path
// @Tags files
// @Produce json
// @Param path query string true "Directory path"
// @Success 200 {array} files.FileInfo
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/files/list [get]
func (h *FilesHandler) List(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		path = "/"
	}

	list, err := h.manager.List(path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, list)
}

// Info godoc
// @Summary Get file/directory info
// @Description Returns information about a file or directory
// @Tags files
// @Produce json
// @Param path query string true "File or directory path"
// @Success 200 {object} files.FileInfo
// @Failure 404 {object} map[string]string
// @Router /api/v1/files/info [get]
func (h *FilesHandler) Info(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path required"})
		return
	}

	info, err := h.manager.Info(path)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, info)
}

// Download godoc
// @Summary Download a file
// @Description Downloads a file or directory (as zip)
// @Tags files
// @Produce octet-stream
// @Param path query string true "File path"
// @Success 200 {file} binary
// @Failure 404 {object} map[string]string
// @Router /api/v1/files/download [get]
func (h *FilesHandler) Download(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path required"})
		return
	}

	reader, size, err := h.manager.Download(path)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	defer reader.Close()

	filename := filepath.Base(path)
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Header("Content-Length", string(rune(size)))
	c.Header("Content-Type", "application/octet-stream")

	io.Copy(c.Writer, reader)
}

// Upload godoc
// @Summary Upload a file
// @Description Uploads a file to a directory
// @Tags files
// @Accept multipart/form-data
// @Produce json
// @Param path query string true "Destination directory"
// @Param file formData file true "File to upload"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/files/upload [post]
func (h *FilesHandler) Upload(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		path = "/"
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file required"})
		return
	}
	defer file.Close()

	if err := h.manager.Upload(path, file, header.Filename); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "file uploaded", "filename": header.Filename})
}

// Mkdir godoc
// @Summary Create directory
// @Description Creates a new directory
// @Tags files
// @Accept json
// @Produce json
// @Param body body map[string]string true "Path"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/files/mkdir [post]
func (h *FilesHandler) Mkdir(c *gin.Context) {
	var req struct {
		Path string `json:"path"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if err := h.manager.CreateDir(req.Path); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "directory created"})
}

// Delete godoc
// @Summary Delete file or directory
// @Description Deletes a file or directory
// @Tags files
// @Produce json
// @Param path query string true "Path to delete"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/files/delete [delete]
func (h *FilesHandler) Delete(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path required"})
		return
	}

	if err := h.manager.Delete(path); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// Rename godoc
// @Summary Rename file or directory
// @Description Renames a file or directory
// @Tags files
// @Accept json
// @Produce json
// @Param body body map[string]string true "Old and new paths"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/files/rename [put]
func (h *FilesHandler) Rename(c *gin.Context) {
	var req struct {
		OldPath string `json:"old_path"`
		NewPath string `json:"new_path"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if err := h.manager.Rename(req.OldPath, req.NewPath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "renamed"})
}

// Read godoc
// @Summary Read file content
// @Description Returns the content of a text file
// @Tags files
// @Produce json
// @Param path query string true "File path"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/files/read [get]
func (h *FilesHandler) Read(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path required"})
		return
	}

	content, err := h.manager.Read(path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"content": string(content)})
}

// Write godoc
// @Summary Write file content
// @Description Writes content to a file
// @Tags files
// @Accept json
// @Produce json
// @Param body body map[string]string true "Path and content"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/files/write [put]
func (h *FilesHandler) Write(c *gin.Context) {
	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	if err := h.manager.Write(req.Path, []byte(req.Content)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "file written"})
}
