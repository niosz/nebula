package metrics

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/nebula/nebula/internal/storage"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

// SystemInfo contains general system information
type SystemInfo struct {
	Hostname        string `json:"hostname"`
	OS              string `json:"os"`
	Platform        string `json:"platform"`
	PlatformVersion string `json:"platform_version"`
	KernelVersion   string `json:"kernel_version"`
	KernelArch      string `json:"kernel_arch"`
	Uptime          uint64 `json:"uptime"`
	BootTime        uint64 `json:"boot_time"`
	NumCPU          int    `json:"num_cpu"`
}

// CPUInfo contains CPU information
type CPUInfo struct {
	Cores        int       `json:"cores"`
	ModelName    string    `json:"model_name"`
	Mhz          float64   `json:"mhz"`
	UsagePercent []float64 `json:"usage_percent"`
	TotalPercent float64   `json:"total_percent"`
}

// MemoryInfo contains memory information
type MemoryInfo struct {
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	Available   uint64  `json:"available"`
	UsedPercent float64 `json:"used_percent"`
	SwapTotal   uint64  `json:"swap_total"`
	SwapUsed    uint64  `json:"swap_used"`
	SwapFree    uint64  `json:"swap_free"`
}

// DiskInfo contains disk information
type DiskInfo struct {
	Device      string  `json:"device"`
	Mountpoint  string  `json:"mountpoint"`
	Fstype      string  `json:"fstype"`
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsedPercent float64 `json:"used_percent"`
}

// NetworkInfo contains network interface information
type NetworkInfo struct {
	Name        string `json:"name"`
	BytesSent   uint64 `json:"bytes_sent"`
	BytesRecv   uint64 `json:"bytes_recv"`
	PacketsSent uint64 `json:"packets_sent"`
	PacketsRecv uint64 `json:"packets_recv"`
	Errin       uint64 `json:"errin"`
	Errout      uint64 `json:"errout"`
}

// AllMetrics contains all system metrics
type AllMetrics struct {
	Timestamp time.Time     `json:"timestamp"`
	System    SystemInfo    `json:"system"`
	CPU       CPUInfo       `json:"cpu"`
	Memory    MemoryInfo    `json:"memory"`
	Disks     []DiskInfo    `json:"disks"`
	Network   []NetworkInfo `json:"network"`
}

// Collector collects system metrics
type Collector struct {
	storage  *storage.Storage
	interval time.Duration
	history  []AllMetrics
	histSize int
	mu       sync.RWMutex

	subscribers []chan AllMetrics
	subMu       sync.RWMutex
}

// NewCollector creates a new metrics collector
func NewCollector(store *storage.Storage, interval time.Duration, historySize int) *Collector {
	return &Collector{
		storage:  store,
		interval: interval,
		histSize: historySize,
		history:  make([]AllMetrics, 0, historySize),
	}
}

// Start begins collecting metrics
func (c *Collector) Start(ctx context.Context) {
	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	// Collect immediately
	c.collect()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.collect()
		}
	}
}

// collect gathers all metrics
func (c *Collector) collect() {
	metrics := AllMetrics{
		Timestamp: time.Now(),
	}

	// Collect system info
	if info, err := c.GetSystemInfo(); err == nil {
		metrics.System = info
	}

	// Collect CPU info
	if cpu, err := c.GetCPUInfo(); err == nil {
		metrics.CPU = cpu
	}

	// Collect memory info
	if mem, err := c.GetMemoryInfo(); err == nil {
		metrics.Memory = mem
	}

	// Collect disk info
	if disks, err := c.GetDiskInfo(); err == nil {
		metrics.Disks = disks
	}

	// Collect network info
	if net, err := c.GetNetworkInfo(); err == nil {
		metrics.Network = net
	}

	// Store in history
	c.mu.Lock()
	c.history = append(c.history, metrics)
	if len(c.history) > c.histSize {
		c.history = c.history[1:]
	}
	c.mu.Unlock()

	// Store in database
	if c.storage != nil {
		entry := storage.MetricsEntry{
			Timestamp: metrics.Timestamp,
			CPU: storage.CPUMetrics{
				UsagePercent: metrics.CPU.UsagePercent,
				TotalPercent: metrics.CPU.TotalPercent,
			},
			Memory: storage.MemMetrics{
				Total:       metrics.Memory.Total,
				Used:        metrics.Memory.Used,
				Free:        metrics.Memory.Free,
				UsedPercent: metrics.Memory.UsedPercent,
				SwapTotal:   metrics.Memory.SwapTotal,
				SwapUsed:    metrics.Memory.SwapUsed,
				SwapFree:    metrics.Memory.SwapFree,
			},
		}
		for _, d := range metrics.Disks {
			entry.Disk = append(entry.Disk, storage.DiskInfo{
				Device:      d.Device,
				Mountpoint:  d.Mountpoint,
				Fstype:      d.Fstype,
				Total:       d.Total,
				Used:        d.Used,
				Free:        d.Free,
				UsedPercent: d.UsedPercent,
			})
		}
		for _, n := range metrics.Network {
			entry.Network = append(entry.Network, storage.NetInfo{
				Name:        n.Name,
				BytesSent:   n.BytesSent,
				BytesRecv:   n.BytesRecv,
				PacketsSent: n.PacketsSent,
				PacketsRecv: n.PacketsRecv,
			})
		}
		c.storage.AddMetricsEntry(entry)
	}

	// Notify subscribers
	c.notifySubscribers(metrics)
}

// Subscribe returns a channel that receives metrics updates
func (c *Collector) Subscribe() chan AllMetrics {
	ch := make(chan AllMetrics, 10)
	c.subMu.Lock()
	c.subscribers = append(c.subscribers, ch)
	c.subMu.Unlock()
	return ch
}

// Unsubscribe removes a subscriber channel
func (c *Collector) Unsubscribe(ch chan AllMetrics) {
	c.subMu.Lock()
	defer c.subMu.Unlock()

	for i, sub := range c.subscribers {
		if sub == ch {
			c.subscribers = append(c.subscribers[:i], c.subscribers[i+1:]...)
			close(ch)
			return
		}
	}
}

// notifySubscribers sends metrics to all subscribers
func (c *Collector) notifySubscribers(metrics AllMetrics) {
	c.subMu.RLock()
	defer c.subMu.RUnlock()

	for _, ch := range c.subscribers {
		select {
		case ch <- metrics:
		default:
			// Channel full, skip
		}
	}
}

// GetLatest returns the latest metrics
func (c *Collector) GetLatest() AllMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.history) == 0 {
		return AllMetrics{}
	}
	return c.history[len(c.history)-1]
}

// GetHistory returns the metrics history
func (c *Collector) GetHistory() []AllMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]AllMetrics, len(c.history))
	copy(result, c.history)
	return result
}

// GetSystemInfo returns system information
func (c *Collector) GetSystemInfo() (SystemInfo, error) {
	info := SystemInfo{
		NumCPU: runtime.NumCPU(),
	}

	hostInfo, err := host.Info()
	if err != nil {
		return info, err
	}

	info.Hostname = hostInfo.Hostname
	info.OS = hostInfo.OS
	info.Platform = hostInfo.Platform
	info.PlatformVersion = hostInfo.PlatformVersion
	info.KernelVersion = hostInfo.KernelVersion
	info.KernelArch = hostInfo.KernelArch
	info.Uptime = hostInfo.Uptime
	info.BootTime = hostInfo.BootTime

	return info, nil
}

// GetCPUInfo returns CPU information
func (c *Collector) GetCPUInfo() (CPUInfo, error) {
	info := CPUInfo{}

	// Get CPU info
	cpuInfo, err := cpu.Info()
	if err == nil && len(cpuInfo) > 0 {
		info.Cores = int(cpuInfo[0].Cores)
		info.ModelName = cpuInfo[0].ModelName
		info.Mhz = cpuInfo[0].Mhz
	}

	// Get per-CPU usage
	perCPU, err := cpu.Percent(0, true)
	if err == nil {
		info.UsagePercent = perCPU
	}

	// Get total CPU usage
	total, err := cpu.Percent(0, false)
	if err == nil && len(total) > 0 {
		info.TotalPercent = total[0]
	}

	return info, nil
}

// GetMemoryInfo returns memory information
func (c *Collector) GetMemoryInfo() (MemoryInfo, error) {
	info := MemoryInfo{}

	vmem, err := mem.VirtualMemory()
	if err != nil {
		return info, err
	}

	info.Total = vmem.Total
	info.Used = vmem.Used
	info.Free = vmem.Free
	info.Available = vmem.Available
	info.UsedPercent = vmem.UsedPercent

	swap, err := mem.SwapMemory()
	if err == nil {
		info.SwapTotal = swap.Total
		info.SwapUsed = swap.Used
		info.SwapFree = swap.Free
	}

	return info, nil
}

// GetDiskInfo returns disk information
func (c *Collector) GetDiskInfo() ([]DiskInfo, error) {
	var disks []DiskInfo

	partitions, err := disk.Partitions(false)
	if err != nil {
		return disks, err
	}

	for _, p := range partitions {
		usage, err := disk.Usage(p.Mountpoint)
		if err != nil {
			continue
		}

		disks = append(disks, DiskInfo{
			Device:      p.Device,
			Mountpoint:  p.Mountpoint,
			Fstype:      p.Fstype,
			Total:       usage.Total,
			Used:        usage.Used,
			Free:        usage.Free,
			UsedPercent: usage.UsedPercent,
		})
	}

	return disks, nil
}

// GetNetworkInfo returns network information
func (c *Collector) GetNetworkInfo() ([]NetworkInfo, error) {
	var networks []NetworkInfo

	counters, err := net.IOCounters(true)
	if err != nil {
		return networks, err
	}

	for _, counter := range counters {
		networks = append(networks, NetworkInfo{
			Name:        counter.Name,
			BytesSent:   counter.BytesSent,
			BytesRecv:   counter.BytesRecv,
			PacketsSent: counter.PacketsSent,
			PacketsRecv: counter.PacketsRecv,
			Errin:       counter.Errin,
			Errout:      counter.Errout,
		})
	}

	return networks, nil
}
