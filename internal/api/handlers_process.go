package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/nebula/nebula/internal/process"
)

// ProcessHandler handles process endpoints
type ProcessHandler struct {
	manager *process.Manager
}

// NewProcessHandler creates a new process handler
func NewProcessHandler(manager *process.Manager) *ProcessHandler {
	return &ProcessHandler{manager: manager}
}

// List godoc
// @Summary List all processes
// @Description Returns a list of all running processes
// @Tags processes
// @Produce json
// @Success 200 {array} process.ProcessInfo
// @Router /api/v1/processes [get]
func (h *ProcessHandler) List(c *gin.Context) {
	procs, err := h.manager.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, procs)
}

// Get godoc
// @Summary Get process details
// @Description Returns detailed information about a specific process
// @Tags processes
// @Produce json
// @Param pid path int true "Process ID"
// @Success 200 {object} process.ProcessInfo
// @Failure 404 {object} map[string]string
// @Router /api/v1/processes/{pid} [get]
func (h *ProcessHandler) Get(c *gin.Context) {
	pidStr := c.Param("pid")
	pid, err := strconv.ParseInt(pidStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid PID"})
		return
	}

	proc, err := h.manager.Get(int32(pid))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, proc)
}

// Kill godoc
// @Summary Kill a process
// @Description Terminates a process by PID
// @Tags processes
// @Produce json
// @Param pid path int true "Process ID"
// @Param force query bool false "Force kill (SIGKILL)"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/processes/{pid}/kill [post]
func (h *ProcessHandler) Kill(c *gin.Context) {
	pidStr := c.Param("pid")
	pid, err := strconv.ParseInt(pidStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid PID"})
		return
	}

	force := c.Query("force") == "true"

	if err := h.manager.Kill(int32(pid), force); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "process terminated"})
}

// Tree godoc
// @Summary Get process tree
// @Description Returns the process tree starting from a specific PID
// @Tags processes
// @Produce json
// @Param pid path int true "Process ID"
// @Success 200 {object} process.TreeNode
// @Failure 404 {object} map[string]string
// @Router /api/v1/processes/{pid}/tree [get]
func (h *ProcessHandler) Tree(c *gin.Context) {
	pidStr := c.Param("pid")
	pid, err := strconv.ParseInt(pidStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid PID"})
		return
	}

	tree, err := h.manager.Tree(int32(pid))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, tree)
}

// Search godoc
// @Summary Search processes
// @Description Search for processes by name
// @Tags processes
// @Produce json
// @Param q query string true "Search query"
// @Success 200 {array} process.ProcessInfo
// @Router /api/v1/processes/search [get]
func (h *ProcessHandler) Search(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query parameter required"})
		return
	}

	procs, err := h.manager.Search(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, procs)
}
