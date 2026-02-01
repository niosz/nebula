package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nebula/nebula/internal/config"
	"github.com/nebula/nebula/internal/metrics"
	"github.com/nebula/nebula/internal/updater"
)

// SystemHandler handles system endpoints
type SystemHandler struct {
	configManager   *config.Manager
	metricsCollector *metrics.Collector
	updater         *updater.Updater
}

// NewSystemHandler creates a new system handler
func NewSystemHandler(cfg *config.Manager, mc *metrics.Collector, upd *updater.Updater) *SystemHandler {
	return &SystemHandler{
		configManager:    cfg,
		metricsCollector: mc,
		updater:          upd,
	}
}

// GetSystemInfo godoc
// @Summary Get system information
// @Description Returns general system information
// @Tags system
// @Produce json
// @Success 200 {object} metrics.SystemInfo
// @Router /api/v1/system/info [get]
func (h *SystemHandler) GetSystemInfo(c *gin.Context) {
	info, err := h.metricsCollector.GetSystemInfo()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, info)
}

// GetConfig godoc
// @Summary Get current configuration
// @Description Returns the current server configuration
// @Tags system
// @Produce json
// @Success 200 {object} config.Config
// @Router /api/v1/config [get]
func (h *SystemHandler) GetConfig(c *gin.Context) {
	cfg := h.configManager.Get()
	
	// Mask sensitive data
	safeCfg := *cfg
	safeCfg.Auth.Password = "********"
	
	c.JSON(http.StatusOK, safeCfg)
}

// ReloadConfig godoc
// @Summary Reload configuration
// @Description Reloads the configuration from file
// @Tags system
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/config/reload [post]
func (h *SystemHandler) ReloadConfig(c *gin.Context) {
	if err := h.configManager.Reload(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "configuration reloaded"})
}

// CheckUpdate godoc
// @Summary Check for updates
// @Description Checks if a new version is available
// @Tags system
// @Produce json
// @Success 200 {object} updater.UpdateInfo
// @Failure 500 {object} map[string]string
// @Router /api/v1/update/check [get]
func (h *SystemHandler) CheckUpdate(c *gin.Context) {
	info, err := h.updater.CheckForUpdate()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, info)
}

// ApplyUpdate godoc
// @Summary Apply update
// @Description Downloads and applies the latest update
// @Tags system
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/update/apply [post]
func (h *SystemHandler) ApplyUpdate(c *gin.Context) {
	if err := h.updater.Apply(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "update applied, restart required"})
}

// GetVersion godoc
// @Summary Get version
// @Description Returns the current version
// @Tags system
// @Produce json
// @Success 200 {object} map[string]string
// @Router /api/v1/version [get]
func (h *SystemHandler) GetVersion(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"version": h.updater.GetVersion(),
	})
}
