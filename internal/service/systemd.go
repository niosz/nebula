//go:build linux

package service

import (
	"bufio"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// newPlatformManager creates the platform-specific manager
func newPlatformManager() (Manager, error) {
	return NewSystemdManager()
}

// SystemdManager manages systemd services on Linux
type SystemdManager struct{}

// NewSystemdManager creates a new systemd manager
func NewSystemdManager() (*SystemdManager, error) {
	// Check if systemctl is available
	if _, err := exec.LookPath("systemctl"); err != nil {
		return nil, fmt.Errorf("systemctl not found: %w", err)
	}
	return &SystemdManager{}, nil
}

// List returns all systemd services
func (m *SystemdManager) List() ([]ServiceInfo, error) {
	cmd := exec.Command("systemctl", "list-units", "--type=service", "--all", "--no-pager", "--no-legend")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}

	var services []ServiceInfo
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		name := strings.TrimSuffix(fields[0], ".service")
		status := StatusUnknown
		
		switch fields[3] {
		case "running":
			status = StatusRunning
		case "exited", "dead":
			status = StatusStopped
		case "failed":
			status = StatusFailed
		}

		services = append(services, ServiceInfo{
			Name:   name,
			Status: status,
		})
	}

	return services, nil
}

// Get returns information about a specific service
func (m *SystemdManager) Get(name string) (ServiceInfo, error) {
	info := ServiceInfo{Name: name}

	// Get service status
	cmd := exec.Command("systemctl", "show", name+".service",
		"--property=Description,LoadState,ActiveState,SubState,MainPID,UnitFileState")
	output, err := cmd.Output()
	if err != nil {
		return info, fmt.Errorf("failed to get service info: %w", err)
	}

	for _, line := range strings.Split(string(output), "\n") {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key, value := parts[0], parts[1]

		switch key {
		case "Description":
			info.Description = value
			info.DisplayName = value
		case "ActiveState":
			switch value {
			case "active":
				info.Status = StatusRunning
			case "inactive":
				info.Status = StatusStopped
			case "failed":
				info.Status = StatusFailed
			default:
				info.Status = StatusUnknown
			}
		case "MainPID":
			if pid, err := strconv.Atoi(value); err == nil {
				info.MainPID = pid
				info.PID = pid
			}
		case "UnitFileState":
			switch value {
			case "enabled":
				info.StartType = StartTypeAuto
			case "disabled":
				info.StartType = StartTypeDisabled
			default:
				info.StartType = StartTypeManual
			}
		}
	}

	return info, nil
}

// Start starts a service
func (m *SystemdManager) Start(name string) error {
	cmd := exec.Command("systemctl", "start", name+".service")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to start service: %s", string(output))
	}
	return nil
}

// Stop stops a service
func (m *SystemdManager) Stop(name string) error {
	cmd := exec.Command("systemctl", "stop", name+".service")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to stop service: %s", string(output))
	}
	return nil
}

// Restart restarts a service
func (m *SystemdManager) Restart(name string) error {
	cmd := exec.Command("systemctl", "restart", name+".service")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to restart service: %s", string(output))
	}
	return nil
}

// Enable enables a service
func (m *SystemdManager) Enable(name string) error {
	cmd := exec.Command("systemctl", "enable", name+".service")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to enable service: %s", string(output))
	}
	return nil
}

// Disable disables a service
func (m *SystemdManager) Disable(name string) error {
	cmd := exec.Command("systemctl", "disable", name+".service")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to disable service: %s", string(output))
	}
	return nil
}

// Logs returns service logs
func (m *SystemdManager) Logs(name string, lines int) ([]ServiceLog, error) {
	cmd := exec.Command("journalctl", "-u", name+".service", "-n", strconv.Itoa(lines), "--no-pager", "-o", "short-iso")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get logs: %w", err)
	}

	var logs []ServiceLog
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// Parse timestamp and message
		// Format: 2024-01-15T10:30:45+0000 hostname service[pid]: message
		parts := strings.SplitN(line, " ", 4)
		if len(parts) >= 4 {
			logs = append(logs, ServiceLog{
				Timestamp: parts[0],
				Message:   parts[3],
			})
		} else {
			logs = append(logs, ServiceLog{
				Message: line,
			})
		}
	}

	return logs, nil
}

// Status returns the status of a service
func (m *SystemdManager) Status(name string) (string, error) {
	cmd := exec.Command("systemctl", "is-active", name+".service")
	output, _ := cmd.Output()
	
	status := strings.TrimSpace(string(output))
	switch status {
	case "active":
		return StatusRunning, nil
	case "inactive":
		return StatusStopped, nil
	case "failed":
		return StatusFailed, nil
	default:
		return StatusUnknown, nil
	}
}
