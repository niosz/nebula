package packages

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// BrewManager manages packages using Homebrew
type BrewManager struct{}

// NewBrewManager creates a new Homebrew manager
func NewBrewManager() (*BrewManager, error) {
	return &BrewManager{}, nil
}

// Type returns the package manager type
func (m *BrewManager) Type() string {
	return "brew"
}

// List returns installed packages
func (m *BrewManager) List() ([]PackageInfo, error) {
	cmd := exec.Command("brew", "list", "--versions")
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

		packages = append(packages, PackageInfo{
			Name:      parts[0],
			Version:   parts[1],
			Installed: true,
		})
	}

	return packages, nil
}

// Search searches for packages
func (m *BrewManager) Search(query string) ([]PackageInfo, error) {
	cmd := exec.Command("brew", "search", query)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to search packages: %w", err)
	}

	var packages []PackageInfo
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		name := strings.TrimSpace(scanner.Text())
		if name == "" || strings.HasPrefix(name, "==>") {
			continue
		}

		packages = append(packages, PackageInfo{
			Name:      name,
			Installed: false,
		})
	}

	return packages, nil
}

// Install installs a package
func (m *BrewManager) Install(name string) error {
	cmd := exec.Command("brew", "install", name)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to install package: %s", string(output))
	}
	return nil
}

// Remove removes a package
func (m *BrewManager) Remove(name string) error {
	cmd := exec.Command("brew", "uninstall", name)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to remove package: %s", string(output))
	}
	return nil
}

// Update updates a package
func (m *BrewManager) Update(name string) error {
	cmd := exec.Command("brew", "upgrade", name)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to update package: %s", string(output))
	}
	return nil
}

// UpgradeAll upgrades all packages
func (m *BrewManager) UpgradeAll() error {
	// First update brew itself
	exec.Command("brew", "update").Run()
	
	cmd := exec.Command("brew", "upgrade")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to upgrade packages: %s", string(output))
	}
	return nil
}

// Info returns package information
func (m *BrewManager) Info(name string) (PackageInfo, error) {
	cmd := exec.Command("brew", "info", "--json=v2", name)
	output, err := cmd.Output()
	if err != nil {
		return PackageInfo{}, fmt.Errorf("failed to get package info: %w", err)
	}

	var result struct {
		Formulae []struct {
			Name     string `json:"name"`
			Versions struct {
				Stable string `json:"stable"`
			} `json:"versions"`
			Desc      string `json:"desc"`
			Installed []struct {
				Version string `json:"version"`
			} `json:"installed"`
		} `json:"formulae"`
		Casks []struct {
			Token   string `json:"token"`
			Version string `json:"version"`
			Desc    string `json:"desc"`
		} `json:"casks"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return PackageInfo{}, err
	}

	pkg := PackageInfo{Name: name}

	if len(result.Formulae) > 0 {
		f := result.Formulae[0]
		pkg.Version = f.Versions.Stable
		pkg.Description = f.Desc
		pkg.Installed = len(f.Installed) > 0
	} else if len(result.Casks) > 0 {
		c := result.Casks[0]
		pkg.Version = c.Version
		pkg.Description = c.Desc
	}

	return pkg, nil
}

// GetOutdated returns packages that can be upgraded
func (m *BrewManager) GetOutdated() ([]PackageInfo, error) {
	cmd := exec.Command("brew", "outdated", "--json=v2")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var result struct {
		Formulae []struct {
			Name               string `json:"name"`
			InstalledVersions  []string `json:"installed_versions"`
			CurrentVersion     string `json:"current_version"`
		} `json:"formulae"`
	}

	if err := json.Unmarshal(output, &result); err != nil {
		return nil, err
	}

	var packages []PackageInfo
	for _, f := range result.Formulae {
		packages = append(packages, PackageInfo{
			Name:       f.Name,
			Version:    strings.Join(f.InstalledVersions, ", "),
			NewVersion: f.CurrentVersion,
			Installed:  true,
			CanUpgrade: true,
		})
	}

	return packages, nil
}
