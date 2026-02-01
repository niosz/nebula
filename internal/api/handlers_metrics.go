package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nebula/nebula/internal/metrics"
)

// MetricsHandler handles metrics endpoints
type MetricsHandler struct {
	collector *metrics.Collector
}

// NewMetricsHandler creates a new metrics handler
func NewMetricsHandler(collector *metrics.Collector) *MetricsHandler {
	return &MetricsHandler{collector: collector}
}

// GetCPU godoc
// @Summary Get CPU metrics
// @Description Returns current CPU usage information
// @Tags metrics
// @Produce json
// @Success 200 {object} metrics.CPUInfo
// @Router /api/v1/metrics/cpu [get]
func (h *MetricsHandler) GetCPU(c *gin.Context) {
	cpu, err := h.collector.GetCPUInfo()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, cpu)
}

// GetMemory godoc
// @Summary Get memory metrics
// @Description Returns current memory usage information
// @Tags metrics
// @Produce json
// @Success 200 {object} metrics.MemoryInfo
// @Router /api/v1/metrics/memory [get]
func (h *MetricsHandler) GetMemory(c *gin.Context) {
	mem, err := h.collector.GetMemoryInfo()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, mem)
}

// GetDisk godoc
// @Summary Get disk metrics
// @Description Returns disk usage information for all mounted partitions
// @Tags metrics
// @Produce json
// @Success 200 {array} metrics.DiskInfo
// @Router /api/v1/metrics/disk [get]
func (h *MetricsHandler) GetDisk(c *gin.Context) {
	disks, err := h.collector.GetDiskInfo()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, disks)
}

// GetNetwork godoc
// @Summary Get network metrics
// @Description Returns network interface statistics
// @Tags metrics
// @Produce json
// @Success 200 {array} metrics.NetworkInfo
// @Router /api/v1/metrics/network [get]
func (h *MetricsHandler) GetNetwork(c *gin.Context) {
	net, err := h.collector.GetNetworkInfo()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, net)
}

// GetAll godoc
// @Summary Get all metrics
// @Description Returns all system metrics
// @Tags metrics
// @Produce json
// @Success 200 {object} metrics.AllMetrics
// @Router /api/v1/metrics/all [get]
func (h *MetricsHandler) GetAll(c *gin.Context) {
	metrics := h.collector.GetLatest()
	c.JSON(http.StatusOK, metrics)
}

// GetHistory godoc
// @Summary Get metrics history
// @Description Returns historical metrics data
// @Tags metrics
// @Produce json
// @Success 200 {array} metrics.AllMetrics
// @Router /api/v1/metrics/history [get]
func (h *MetricsHandler) GetHistory(c *gin.Context) {
	history := h.collector.GetHistory()
	c.JSON(http.StatusOK, history)
}
