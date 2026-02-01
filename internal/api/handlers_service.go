package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/nebula/nebula/internal/service"
)

// ServiceHandler handles service endpoints
type ServiceHandler struct {
	manager service.Manager
}

// NewServiceHandler creates a new service handler
func NewServiceHandler(manager service.Manager) *ServiceHandler {
	return &ServiceHandler{manager: manager}
}

// List godoc
// @Summary List all services
// @Description Returns a list of all system services
// @Tags services
// @Produce json
// @Success 200 {array} service.ServiceInfo
// @Router /api/v1/services [get]
func (h *ServiceHandler) List(c *gin.Context) {
	services, err := h.manager.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, services)
}

// Get godoc
// @Summary Get service details
// @Description Returns detailed information about a specific service
// @Tags services
// @Produce json
// @Param name path string true "Service name"
// @Success 200 {object} service.ServiceInfo
// @Failure 404 {object} map[string]string
// @Router /api/v1/services/{name} [get]
func (h *ServiceHandler) Get(c *gin.Context) {
	name := c.Param("name")

	svc, err := h.manager.Get(name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, svc)
}

// Start godoc
// @Summary Start a service
// @Description Starts a system service
// @Tags services
// @Produce json
// @Param name path string true "Service name"
// @Success 200 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/services/{name}/start [post]
func (h *ServiceHandler) Start(c *gin.Context) {
	name := c.Param("name")

	if err := h.manager.Start(name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "service started"})
}

// Stop godoc
// @Summary Stop a service
// @Description Stops a system service
// @Tags services
// @Produce json
// @Param name path string true "Service name"
// @Success 200 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/services/{name}/stop [post]
func (h *ServiceHandler) Stop(c *gin.Context) {
	name := c.Param("name")

	if err := h.manager.Stop(name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "service stopped"})
}

// Restart godoc
// @Summary Restart a service
// @Description Restarts a system service
// @Tags services
// @Produce json
// @Param name path string true "Service name"
// @Success 200 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/services/{name}/restart [post]
func (h *ServiceHandler) Restart(c *gin.Context) {
	name := c.Param("name")

	if err := h.manager.Restart(name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "service restarted"})
}

// Enable godoc
// @Summary Enable a service
// @Description Enables a service to start at boot
// @Tags services
// @Produce json
// @Param name path string true "Service name"
// @Success 200 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/services/{name}/enable [post]
func (h *ServiceHandler) Enable(c *gin.Context) {
	name := c.Param("name")

	if err := h.manager.Enable(name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "service enabled"})
}

// Disable godoc
// @Summary Disable a service
// @Description Disables a service from starting at boot
// @Tags services
// @Produce json
// @Param name path string true "Service name"
// @Success 200 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/services/{name}/disable [post]
func (h *ServiceHandler) Disable(c *gin.Context) {
	name := c.Param("name")

	if err := h.manager.Disable(name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "service disabled"})
}

// Logs godoc
// @Summary Get service logs
// @Description Returns recent logs for a service
// @Tags services
// @Produce json
// @Param name path string true "Service name"
// @Param lines query int false "Number of lines" default(100)
// @Success 200 {array} service.ServiceLog
// @Failure 500 {object} map[string]string
// @Router /api/v1/services/{name}/logs [get]
func (h *ServiceHandler) Logs(c *gin.Context) {
	name := c.Param("name")
	lines := 100
	if l := c.Query("lines"); l != "" {
		if n, err := strconv.Atoi(l); err == nil {
			lines = n
		}
	}

	logs, err := h.manager.Logs(name, lines)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, logs)
}
