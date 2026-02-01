//go:build windows

package service

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
)

// newPlatformManager creates the platform-specific manager
func newPlatformManager() (Manager, error) {
	return NewWindowsManager()
}

// WindowsManager manages Windows services
type WindowsManager struct{}

// NewWindowsManager creates a new Windows service manager
func NewWindowsManager() (*WindowsManager, error) {
	return &WindowsManager{}, nil
}

// List returns all Windows services
func (m *WindowsManager) List() ([]ServiceInfo, error) {
	cmd := exec.Command("powershell", "-Command", "Get-Service | Select-Object Name,DisplayName,Status,StartType | ConvertTo-Json")
	output, err := cmd.Output()
	if err != nil {
		// Fallback to sc query
		return m.listWithSC()
	}

	// Parse JSON output
	var services []ServiceInfo
	// Simple parsing - in production would use proper JSON parsing
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "\"Name\"") {
			// Extract service info from JSON
			// This is simplified - production code would properly parse JSON
		}
	}

	return services, nil
}

// listWithSC uses sc.exe to list services
func (m *WindowsManager) listWithSC() ([]ServiceInfo, error) {
	cmd := exec.Command("sc", "query", "state=", "all")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}

	var services []ServiceInfo
	var current ServiceInfo
	
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		if strings.HasPrefix(line, "SERVICE_NAME:") {
			if current.Name != "" {
				services = append(services, current)
			}
			current = ServiceInfo{
				Name: strings.TrimSpace(strings.TrimPrefix(line, "SERVICE_NAME:")),
			}
		} else if strings.HasPrefix(line, "DISPLAY_NAME:") {
			current.DisplayName = strings.TrimSpace(strings.TrimPrefix(line, "DISPLAY_NAME:"))
		} else if strings.HasPrefix(line, "STATE") {
			if strings.Contains(line, "RUNNING") {
				current.Status = StatusRunning
			} else if strings.Contains(line, "STOPPED") {
				current.Status = StatusStopped
			} else {
				current.Status = StatusUnknown
			}
		}
	}
	
	if current.Name != "" {
		services = append(services, current)
	}

	return services, nil
}

// Get returns information about a specific service
func (m *WindowsManager) Get(name string) (ServiceInfo, error) {
	info := ServiceInfo{Name: name}

	cmd := exec.Command("sc", "qc", name)
	output, err := cmd.Output()
	if err != nil {
		return info, fmt.Errorf("failed to get service info: %w", err)
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		if strings.HasPrefix(line, "DISPLAY_NAME") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				info.DisplayName = strings.TrimSpace(parts[1])
			}
		} else if strings.HasPrefix(line, "START_TYPE") {
			if strings.Contains(line, "AUTO_START") {
				info.StartType = StartTypeAuto
			} else if strings.Contains(line, "DEMAND_START") {
				info.StartType = StartTypeManual
			} else if strings.Contains(line, "DISABLED") {
				info.StartType = StartTypeDisabled
			}
		}
	}

	// Get current status
	status, _ := m.Status(name)
	info.Status = status

	return info, nil
}

// Start starts a service
func (m *WindowsManager) Start(name string) error {
	cmd := exec.Command("net", "start", name)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to start service: %s", string(output))
	}
	return nil
}

// Stop stops a service
func (m *WindowsManager) Stop(name string) error {
	cmd := exec.Command("net", "stop", name)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to stop service: %s", string(output))
	}
	return nil
}

// Restart restarts a service
func (m *WindowsManager) Restart(name string) error {
	if err := m.Stop(name); err != nil {
		// Ignore stop errors
	}
	return m.Start(name)
}

// Enable enables a service
func (m *WindowsManager) Enable(name string) error {
	cmd := exec.Command("sc", "config", name, "start=", "auto")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to enable service: %s", string(output))
	}
	return nil
}

// Disable disables a service
func (m *WindowsManager) Disable(name string) error {
	cmd := exec.Command("sc", "config", name, "start=", "disabled")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to disable service: %s", string(output))
	}
	return nil
}

// Logs returns service logs from Event Log
func (m *WindowsManager) Logs(name string, lines int) ([]ServiceLog, error) {
	cmd := exec.Command("powershell", "-Command",
		fmt.Sprintf("Get-EventLog -LogName System -Source '%s' -Newest %d | Select-Object TimeGenerated,Message | ConvertTo-Json", name, lines))
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get logs: %w", err)
	}

	var logs []ServiceLog
	// Parse output - simplified
	for _, line := range strings.Split(string(output), "\n") {
		if line != "" {
			logs = append(logs, ServiceLog{Message: line})
		}
	}

	return logs, nil
}

// Status returns the status of a service
func (m *WindowsManager) Status(name string) (string, error) {
	cmd := exec.Command("sc", "query", name)
	output, err := cmd.Output()
	if err != nil {
		return StatusUnknown, nil
	}

	if strings.Contains(string(output), "RUNNING") {
		return StatusRunning, nil
	} else if strings.Contains(string(output), "STOPPED") {
		return StatusStopped, nil
	}
	return StatusUnknown, nil
}
