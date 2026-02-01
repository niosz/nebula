package process

import (
	"fmt"
	"os"
	"sort"
	"syscall"

	"github.com/shirou/gopsutil/v3/process"
)

// ProcessInfo contains process information
type ProcessInfo struct {
	PID         int32    `json:"pid"`
	PPID        int32    `json:"ppid"`
	Name        string   `json:"name"`
	Status      string   `json:"status"`
	Username    string   `json:"username"`
	CPUPercent  float64  `json:"cpu_percent"`
	MemPercent  float32  `json:"mem_percent"`
	MemRSS      uint64   `json:"mem_rss"`
	MemVMS      uint64   `json:"mem_vms"`
	NumThreads  int32    `json:"num_threads"`
	CreateTime  int64    `json:"create_time"`
	Cmdline     string   `json:"cmdline"`
	Exe         string   `json:"exe"`
	Cwd         string   `json:"cwd"`
	Nice        int32    `json:"nice"`
	IOCounters  *IOInfo  `json:"io_counters,omitempty"`
	Connections []ConnInfo `json:"connections,omitempty"`
}

// IOInfo contains process I/O information
type IOInfo struct {
	ReadCount  uint64 `json:"read_count"`
	WriteCount uint64 `json:"write_count"`
	ReadBytes  uint64 `json:"read_bytes"`
	WriteBytes uint64 `json:"write_bytes"`
}

// ConnInfo contains connection information
type ConnInfo struct {
	Fd     uint32 `json:"fd"`
	Family uint32 `json:"family"`
	Type   uint32 `json:"type"`
	Laddr  string `json:"laddr"`
	Raddr  string `json:"raddr"`
	Status string `json:"status"`
}

// TreeNode represents a process in the tree
type TreeNode struct {
	Process  ProcessInfo `json:"process"`
	Children []TreeNode  `json:"children"`
}

// Manager manages system processes
type Manager struct{}

// NewManager creates a new process manager
func NewManager() *Manager {
	return &Manager{}
}

// List returns all running processes
func (m *Manager) List() ([]ProcessInfo, error) {
	procs, err := process.Processes()
	if err != nil {
		return nil, fmt.Errorf("failed to get processes: %w", err)
	}

	var result []ProcessInfo
	for _, p := range procs {
		info := m.getBasicInfo(p)
		result = append(result, info)
	}

	// Sort by CPU usage descending
	sort.Slice(result, func(i, j int) bool {
		return result[i].CPUPercent > result[j].CPUPercent
	})

	return result, nil
}

// Get returns detailed information about a specific process
func (m *Manager) Get(pid int32) (ProcessInfo, error) {
	p, err := process.NewProcess(pid)
	if err != nil {
		return ProcessInfo{}, fmt.Errorf("process not found: %w", err)
	}

	info := m.getBasicInfo(p)
	
	// Get additional details
	if cmdline, err := p.Cmdline(); err == nil {
		info.Cmdline = cmdline
	}
	if exe, err := p.Exe(); err == nil {
		info.Exe = exe
	}
	if cwd, err := p.Cwd(); err == nil {
		info.Cwd = cwd
	}
	if nice, err := p.Nice(); err == nil {
		info.Nice = nice
	}
	
	// Get I/O counters
	if io, err := p.IOCounters(); err == nil && io != nil {
		info.IOCounters = &IOInfo{
			ReadCount:  io.ReadCount,
			WriteCount: io.WriteCount,
			ReadBytes:  io.ReadBytes,
			WriteBytes: io.WriteBytes,
		}
	}
	
	// Get connections
	if conns, err := p.Connections(); err == nil {
		for _, c := range conns {
			info.Connections = append(info.Connections, ConnInfo{
				Fd:     c.Fd,
				Family: c.Family,
				Type:   c.Type,
				Laddr:  fmt.Sprintf("%s:%d", c.Laddr.IP, c.Laddr.Port),
				Raddr:  fmt.Sprintf("%s:%d", c.Raddr.IP, c.Raddr.Port),
				Status: c.Status,
			})
		}
	}

	return info, nil
}

// getBasicInfo extracts basic process information
func (m *Manager) getBasicInfo(p *process.Process) ProcessInfo {
	info := ProcessInfo{
		PID: p.Pid,
	}

	if ppid, err := p.Ppid(); err == nil {
		info.PPID = ppid
	}
	if name, err := p.Name(); err == nil {
		info.Name = name
	}
	if status, err := p.Status(); err == nil && len(status) > 0 {
		info.Status = status[0]
	}
	if username, err := p.Username(); err == nil {
		info.Username = username
	}
	if cpuPercent, err := p.CPUPercent(); err == nil {
		info.CPUPercent = cpuPercent
	}
	if memPercent, err := p.MemoryPercent(); err == nil {
		info.MemPercent = memPercent
	}
	if memInfo, err := p.MemoryInfo(); err == nil && memInfo != nil {
		info.MemRSS = memInfo.RSS
		info.MemVMS = memInfo.VMS
	}
	if numThreads, err := p.NumThreads(); err == nil {
		info.NumThreads = numThreads
	}
	if createTime, err := p.CreateTime(); err == nil {
		info.CreateTime = createTime
	}

	return info
}

// Kill terminates a process
func (m *Manager) Kill(pid int32, force bool) error {
	p, err := process.NewProcess(pid)
	if err != nil {
		return fmt.Errorf("process not found: %w", err)
	}

	// Check if it's the current process or init
	if pid == 1 || pid == int32(os.Getpid()) {
		return fmt.Errorf("cannot kill protected process")
	}

	if force {
		return p.Kill()
	}
	return p.Terminate()
}

// Signal sends a signal to a process
func (m *Manager) Signal(pid int32, sig syscall.Signal) error {
	p, err := process.NewProcess(pid)
	if err != nil {
		return fmt.Errorf("process not found: %w", err)
	}

	return p.SendSignal(sig)
}

// Tree returns the process tree for a given PID
func (m *Manager) Tree(pid int32) (TreeNode, error) {
	procs, err := process.Processes()
	if err != nil {
		return TreeNode{}, err
	}

	// Build a map of processes
	procMap := make(map[int32]*process.Process)
	for _, p := range procs {
		procMap[p.Pid] = p
	}

	// Check if the root process exists
	if _, exists := procMap[pid]; !exists {
		return TreeNode{}, fmt.Errorf("process %d not found", pid)
	}

	return m.buildTree(pid, procMap), nil
}

// buildTree recursively builds the process tree
func (m *Manager) buildTree(pid int32, procMap map[int32]*process.Process) TreeNode {
	p := procMap[pid]
	if p == nil {
		return TreeNode{}
	}

	node := TreeNode{
		Process: m.getBasicInfo(p),
	}

	// Find children
	for childPID, childProc := range procMap {
		if ppid, err := childProc.Ppid(); err == nil && ppid == pid {
			node.Children = append(node.Children, m.buildTree(childPID, procMap))
		}
	}

	// Sort children by PID
	sort.Slice(node.Children, func(i, j int) bool {
		return node.Children[i].Process.PID < node.Children[j].Process.PID
	})

	return node
}

// Search searches for processes by name
func (m *Manager) Search(query string) ([]ProcessInfo, error) {
	procs, err := m.List()
	if err != nil {
		return nil, err
	}

	var result []ProcessInfo
	for _, p := range procs {
		if containsIgnoreCase(p.Name, query) || containsIgnoreCase(p.Cmdline, query) {
			result = append(result, p)
		}
	}

	return result, nil
}

// containsIgnoreCase checks if s contains substr (case insensitive)
func containsIgnoreCase(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	
	sLower := toLower(s)
	substrLower := toLower(substr)
	
	for i := 0; i <= len(sLower)-len(substrLower); i++ {
		if sLower[i:i+len(substrLower)] == substrLower {
			return true
		}
	}
	return false
}

// toLower converts a string to lowercase
func toLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}
