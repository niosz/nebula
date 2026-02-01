package packages

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
)

// ChocoManager manages packages using Chocolatey
type ChocoManager struct{}

// NewChocoManager creates a new Chocolatey manager
func NewChocoManager() (*ChocoManager, error) {
	return &ChocoManager{}, nil
}

// Type returns the package manager type
func (m *ChocoManager) Type() string {
	return "choco"
}

// List returns installed packages
func (m *ChocoManager) List() ([]PackageInfo, error) {
	cmd := exec.Command("choco", "list", "--local-only", "--no-color")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list packages: %w", err)
	}

	var packages []PackageInfo
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		// Skip summary line
		if strings.Contains(line, "packages installed") {
			continue
		}

		packages = append(packages, PackageInfo{
			Name:      parts[0],
			Version:   parts[1],
			Installed: true,
		})
	}

	return packages, nil
}

// Search searches for packages
func (m *ChocoManager) Search(query string) ([]PackageInfo, error) {
	cmd := exec.Command("choco", "search", query, "--no-color")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to search packages: %w", err)
	}

	var packages []PackageInfo
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		// Skip summary line
		if strings.Contains(line, "packages found") {
			continue
		}

		packages = append(packages, PackageInfo{
			Name:      parts[0],
			Version:   parts[1],
			Installed: false,
		})
	}

	return packages, nil
}

// Install installs a package
func (m *ChocoManager) Install(name string) error {
	cmd := exec.Command("choco", "install", name, "-y", "--no-color")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to install package: %s", string(output))
	}
	return nil
}

// Remove removes a package
func (m *ChocoManager) Remove(name string) error {
	cmd := exec.Command("choco", "uninstall", name, "-y", "--no-color")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to remove package: %s", string(output))
	}
	return nil
}

// Update updates a package
func (m *ChocoManager) Update(name string) error {
	cmd := exec.Command("choco", "upgrade", name, "-y", "--no-color")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to update package: %s", string(output))
	}
	return nil
}

// UpgradeAll upgrades all packages
func (m *ChocoManager) UpgradeAll() error {
	cmd := exec.Command("choco", "upgrade", "all", "-y", "--no-color")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to upgrade packages: %s", string(output))
	}
	return nil
}

// Info returns package information
func (m *ChocoManager) Info(name string) (PackageInfo, error) {
	cmd := exec.Command("choco", "info", name, "--no-color")
	output, err := cmd.Output()
	if err != nil {
		return PackageInfo{}, fmt.Errorf("failed to get package info: %w", err)
	}

	pkg := PackageInfo{Name: name}
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Title:") {
			pkg.Description = strings.TrimSpace(strings.TrimPrefix(line, "Title:"))
		} else if strings.HasPrefix(line, "Version:") {
			pkg.Version = strings.TrimSpace(strings.TrimPrefix(line, "Version:"))
		}
	}

	// Check if installed
	installed, _ := m.List()
	for _, p := range installed {
		if p.Name == name {
			pkg.Installed = true
			break
		}
	}

	return pkg, nil
}

// WingetManager manages packages using winget
type WingetManager struct{}

// NewWingetManager creates a new winget manager
func NewWingetManager() (*WingetManager, error) {
	return &WingetManager{}, nil
}

// Type returns the package manager type
func (m *WingetManager) Type() string {
	return "winget"
}

// List returns installed packages
func (m *WingetManager) List() ([]PackageInfo, error) {
	cmd := exec.Command("winget", "list")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list packages: %w", err)
	}

	var packages []PackageInfo
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	lineCount := 0
	for scanner.Scan() {
		lineCount++
		if lineCount <= 2 { // Skip header
			continue
		}
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			packages = append(packages, PackageInfo{
				Name:      parts[0],
				Version:   parts[len(parts)-1],
				Installed: true,
			})
		}
	}

	return packages, nil
}

// Search searches for packages
func (m *WingetManager) Search(query string) ([]PackageInfo, error) {
	cmd := exec.Command("winget", "search", query)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to search packages: %w", err)
	}

	var packages []PackageInfo
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	lineCount := 0
	for scanner.Scan() {
		lineCount++
		if lineCount <= 2 { // Skip header
			continue
		}
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			packages = append(packages, PackageInfo{
				Name:    parts[0],
				Version: parts[len(parts)-1],
			})
		}
	}

	return packages, nil
}

// Install installs a package
func (m *WingetManager) Install(name string) error {
	cmd := exec.Command("winget", "install", name, "--silent", "--accept-package-agreements", "--accept-source-agreements")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to install package: %s", string(output))
	}
	return nil
}

// Remove removes a package
func (m *WingetManager) Remove(name string) error {
	cmd := exec.Command("winget", "uninstall", name, "--silent")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to remove package: %s", string(output))
	}
	return nil
}

// Update updates a package
func (m *WingetManager) Update(name string) error {
	cmd := exec.Command("winget", "upgrade", name, "--silent")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to update package: %s", string(output))
	}
	return nil
}

// UpgradeAll upgrades all packages
func (m *WingetManager) UpgradeAll() error {
	cmd := exec.Command("winget", "upgrade", "--all", "--silent")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to upgrade packages: %s", string(output))
	}
	return nil
}

// Info returns package information
func (m *WingetManager) Info(name string) (PackageInfo, error) {
	cmd := exec.Command("winget", "show", name)
	output, err := cmd.Output()
	if err != nil {
		return PackageInfo{}, fmt.Errorf("failed to get package info: %w", err)
	}

	pkg := PackageInfo{Name: name}
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Description:") {
			pkg.Description = strings.TrimSpace(strings.TrimPrefix(line, "Description:"))
		} else if strings.HasPrefix(line, "Version:") {
			pkg.Version = strings.TrimSpace(strings.TrimPrefix(line, "Version:"))
		}
	}

	return pkg, nil
}
