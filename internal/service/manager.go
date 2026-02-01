package service

// ServiceInfo contains service information
type ServiceInfo struct {
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	Status      string `json:"status"`
	PID         int    `json:"pid,omitempty"`
	StartType   string `json:"start_type"`
	User        string `json:"user,omitempty"`
	MainPID     int    `json:"main_pid,omitempty"`
}

// ServiceLog contains service log entry
type ServiceLog struct {
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`
	Priority  string `json:"priority,omitempty"`
}

// Manager interface for service management
type Manager interface {
	// List returns all services
	List() ([]ServiceInfo, error)
	
	// Get returns information about a specific service
	Get(name string) (ServiceInfo, error)
	
	// Start starts a service
	Start(name string) error
	
	// Stop stops a service
	Stop(name string) error
	
	// Restart restarts a service
	Restart(name string) error
	
	// Enable enables a service to start at boot
	Enable(name string) error
	
	// Disable disables a service from starting at boot
	Disable(name string) error
	
	// Logs returns recent logs for a service
	Logs(name string, lines int) ([]ServiceLog, error)
	
	// Status returns the status of a service
	Status(name string) (string, error)
}

// NewManager creates a new service manager for the current OS
func NewManager() (Manager, error) {
	return newPlatformManager()
}

// StatusRunning indicates a running service
const StatusRunning = "running"

// StatusStopped indicates a stopped service
const StatusStopped = "stopped"

// StatusFailed indicates a failed service
const StatusFailed = "failed"

// StatusUnknown indicates unknown status
const StatusUnknown = "unknown"

// StartTypeAuto indicates automatic start
const StartTypeAuto = "auto"

// StartTypeManual indicates manual start
const StartTypeManual = "manual"

// StartTypeDisabled indicates disabled service
const StartTypeDisabled = "disabled"
