//go:build darwin

package service

import (
	"bufio"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// newPlatformManager creates the platform-specific manager
func newPlatformManager() (Manager, error) {
	return NewLaunchctlManager()
}

// LaunchctlManager manages launchd services on macOS
type LaunchctlManager struct{}

// NewLaunchctlManager creates a new launchctl manager
func NewLaunchctlManager() (*LaunchctlManager, error) {
	return &LaunchctlManager{}, nil
}

// List returns all launchd services
func (m *LaunchctlManager) List() ([]ServiceInfo, error) {
	cmd := exec.Command("launchctl", "list")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list services: %w", err)
	}

	var services []ServiceInfo
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	
	// Skip header
	scanner.Scan()
	
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		pid, _ := strconv.Atoi(fields[0])
		name := fields[2]
		
		status := StatusStopped
		if pid > 0 {
			status = StatusRunning
		} else if fields[1] != "0" && fields[1] != "-" {
			status = StatusFailed
		}

		services = append(services, ServiceInfo{
			Name:   name,
			PID:    pid,
			Status: status,
		})
	}

	return services, nil
}

// Get returns information about a specific service
func (m *LaunchctlManager) Get(name string) (ServiceInfo, error) {
	info := ServiceInfo{
		Name:        name,
		DisplayName: name,
	}

	// Try to find the plist file
	plistPath := m.findPlist(name)
	if plistPath != "" {
		info.Description = fmt.Sprintf("Plist: %s", plistPath)
	}

	// Get service info from launchctl
	cmd := exec.Command("launchctl", "list", name)
	output, err := cmd.Output()
	if err != nil {
		info.Status = StatusStopped
		return info, nil
	}

	// Parse output
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "\"PID\"") {
			parts := strings.Split(line, "=")
			if len(parts) == 2 {
				pidStr := strings.TrimSpace(strings.Trim(parts[1], ";"))
				if pid, err := strconv.Atoi(pidStr); err == nil {
					info.PID = pid
					info.MainPID = pid
				}
			}
		}
	}

	if info.PID > 0 {
		info.Status = StatusRunning
	} else {
		info.Status = StatusStopped
	}

	return info, nil
}

// findPlist finds the plist file for a service
func (m *LaunchctlManager) findPlist(name string) string {
	searchPaths := []string{
		"/Library/LaunchDaemons",
		"/Library/LaunchAgents",
		"/System/Library/LaunchDaemons",
		"/System/Library/LaunchAgents",
	}

	for _, path := range searchPaths {
		plistPath := filepath.Join(path, name+".plist")
		if _, err := exec.Command("test", "-f", plistPath).Output(); err == nil {
			return plistPath
		}
	}

	return ""
}

// Start starts a service
func (m *LaunchctlManager) Start(name string) error {
	// Try to find and load the plist
	plistPath := m.findPlist(name)
	if plistPath != "" {
		cmd := exec.Command("launchctl", "load", plistPath)
		if output, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to load service: %s", string(output))
		}
	}

	cmd := exec.Command("launchctl", "start", name)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to start service: %s", string(output))
	}
	return nil
}

// Stop stops a service
func (m *LaunchctlManager) Stop(name string) error {
	cmd := exec.Command("launchctl", "stop", name)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to stop service: %s", string(output))
	}
	return nil
}

// Restart restarts a service
func (m *LaunchctlManager) Restart(name string) error {
	if err := m.Stop(name); err != nil {
		// Ignore stop errors
	}
	return m.Start(name)
}

// Enable enables a service (load)
func (m *LaunchctlManager) Enable(name string) error {
	plistPath := m.findPlist(name)
	if plistPath == "" {
		return fmt.Errorf("plist not found for service: %s", name)
	}

	cmd := exec.Command("launchctl", "load", "-w", plistPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to enable service: %s", string(output))
	}
	return nil
}

// Disable disables a service (unload)
func (m *LaunchctlManager) Disable(name string) error {
	plistPath := m.findPlist(name)
	if plistPath == "" {
		return fmt.Errorf("plist not found for service: %s", name)
	}

	cmd := exec.Command("launchctl", "unload", "-w", plistPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to disable service: %s", string(output))
	}
	return nil
}

// Logs returns service logs from system log
func (m *LaunchctlManager) Logs(name string, lines int) ([]ServiceLog, error) {
	cmd := exec.Command("log", "show", "--predicate", fmt.Sprintf("subsystem == '%s'", name),
		"--last", "1h", "--style", "syslog")
	output, err := cmd.Output()
	if err != nil {
		// Fallback to grep in system.log
		cmd = exec.Command("grep", "-i", name, "/var/log/system.log")
		output, _ = cmd.Output()
	}

	var logs []ServiceLog
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	count := 0
	for scanner.Scan() && count < lines {
		line := scanner.Text()
		if line == "" {
			continue
		}
		logs = append(logs, ServiceLog{
			Message: line,
		})
		count++
	}

	return logs, nil
}

// Status returns the status of a service
func (m *LaunchctlManager) Status(name string) (string, error) {
	cmd := exec.Command("launchctl", "list", name)
	output, err := cmd.Output()
	if err != nil {
		return StatusStopped, nil
	}

	if strings.Contains(string(output), "\"PID\"") {
		return StatusRunning, nil
	}
	return StatusStopped, nil
}
