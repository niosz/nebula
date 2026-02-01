package packages

import (
	"bufio"
	"fmt"
	"os/exec"
	"strings"
)

// AptManager manages packages using apt
type AptManager struct{}

// NewAptManager creates a new apt manager
func NewAptManager() (*AptManager, error) {
	return &AptManager{}, nil
}

// Type returns the package manager type
func (m *AptManager) Type() string {
	return "apt"
}

// List returns installed packages
func (m *AptManager) List() ([]PackageInfo, error) {
	cmd := exec.Command("dpkg-query", "-W", "-f", "${Package}\t${Version}\t${Description}\n")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list packages: %w", err)
	}

	var packages []PackageInfo
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 2 {
			continue
		}

		pkg := PackageInfo{
			Name:      parts[0],
			Version:   parts[1],
			Installed: true,
		}
		if len(parts) > 2 {
			pkg.Description = parts[2]
		}
		packages = append(packages, pkg)
	}

	return packages, nil
}

// Search searches for packages
func (m *AptManager) Search(query string) ([]PackageInfo, error) {
	cmd := exec.Command("apt-cache", "search", query)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to search packages: %w", err)
	}

	var packages []PackageInfo
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, " - ", 2)
		if len(parts) < 1 {
			continue
		}

		pkg := PackageInfo{
			Name:      parts[0],
			Installed: false,
		}
		if len(parts) > 1 {
			pkg.Description = parts[1]
		}
		packages = append(packages, pkg)
	}

	return packages, nil
}

// Install installs a package
func (m *AptManager) Install(name string) error {
	cmd := exec.Command("apt-get", "install", "-y", name)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to install package: %s", string(output))
	}
	return nil
}

// Remove removes a package
func (m *AptManager) Remove(name string) error {
	cmd := exec.Command("apt-get", "remove", "-y", name)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to remove package: %s", string(output))
	}
	return nil
}

// Update updates a package
func (m *AptManager) Update(name string) error {
	// First update package list
	exec.Command("apt-get", "update").Run()
	
	cmd := exec.Command("apt-get", "install", "--only-upgrade", "-y", name)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to update package: %s", string(output))
	}
	return nil
}

// UpgradeAll upgrades all packages
func (m *AptManager) UpgradeAll() error {
	// First update package list
	exec.Command("apt-get", "update").Run()
	
	cmd := exec.Command("apt-get", "upgrade", "-y")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to upgrade packages: %s", string(output))
	}
	return nil
}

// Info returns package information
func (m *AptManager) Info(name string) (PackageInfo, error) {
	cmd := exec.Command("apt-cache", "show", name)
	output, err := cmd.Output()
	if err != nil {
		return PackageInfo{}, fmt.Errorf("failed to get package info: %w", err)
	}

	pkg := PackageInfo{Name: name}
	for _, line := range strings.Split(string(output), "\n") {
		if strings.HasPrefix(line, "Version:") {
			pkg.Version = strings.TrimSpace(strings.TrimPrefix(line, "Version:"))
		} else if strings.HasPrefix(line, "Description:") {
			pkg.Description = strings.TrimSpace(strings.TrimPrefix(line, "Description:"))
		}
	}

	// Check if installed
	checkCmd := exec.Command("dpkg", "-s", name)
	if checkCmd.Run() == nil {
		pkg.Installed = true
	}

	return pkg, nil
}

// YumManager manages packages using yum
type YumManager struct{}

func NewYumManager() (*YumManager, error) {
	return &YumManager{}, nil
}

func (m *YumManager) Type() string { return "yum" }

func (m *YumManager) List() ([]PackageInfo, error) {
	cmd := exec.Command("yum", "list", "installed")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var packages []PackageInfo
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 {
			packages = append(packages, PackageInfo{
				Name:      strings.Split(fields[0], ".")[0],
				Version:   fields[1],
				Installed: true,
			})
		}
	}
	return packages, nil
}

func (m *YumManager) Search(query string) ([]PackageInfo, error) {
	cmd := exec.Command("yum", "search", query)
	output, _ := cmd.Output()

	var packages []PackageInfo
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, " : ") {
			parts := strings.SplitN(line, " : ", 2)
			packages = append(packages, PackageInfo{
				Name:        strings.Split(parts[0], ".")[0],
				Description: parts[1],
			})
		}
	}
	return packages, nil
}

func (m *YumManager) Install(name string) error {
	cmd := exec.Command("yum", "install", "-y", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed: %s", string(output))
	}
	return nil
}

func (m *YumManager) Remove(name string) error {
	cmd := exec.Command("yum", "remove", "-y", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed: %s", string(output))
	}
	return nil
}

func (m *YumManager) Update(name string) error {
	cmd := exec.Command("yum", "update", "-y", name)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed: %s", string(output))
	}
	return nil
}

func (m *YumManager) UpgradeAll() error {
	cmd := exec.Command("yum", "update", "-y")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed: %s", string(output))
	}
	return nil
}

func (m *YumManager) Info(name string) (PackageInfo, error) {
	cmd := exec.Command("yum", "info", name)
	output, _ := cmd.Output()

	pkg := PackageInfo{Name: name}
	for _, line := range strings.Split(string(output), "\n") {
		if strings.HasPrefix(line, "Version") {
			pkg.Version = strings.TrimSpace(strings.Split(line, ":")[1])
		} else if strings.HasPrefix(line, "Description") {
			pkg.Description = strings.TrimSpace(strings.Split(line, ":")[1])
		}
	}
	return pkg, nil
}

// DnfManager manages packages using dnf
type DnfManager struct {
	YumManager // dnf is compatible with yum
}

func NewDnfManager() (*DnfManager, error) {
	return &DnfManager{}, nil
}

func (m *DnfManager) Type() string { return "dnf" }
