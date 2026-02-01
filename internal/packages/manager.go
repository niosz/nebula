package packages

import (
	"os/exec"
	"runtime"
)

// PackageInfo contains package information
type PackageInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	Installed   bool   `json:"installed"`
	CanUpgrade  bool   `json:"can_upgrade,omitempty"`
	NewVersion  string `json:"new_version,omitempty"`
}

// Manager interface for package management
type Manager interface {
	// List returns installed packages
	List() ([]PackageInfo, error)
	
	// Search searches for packages
	Search(query string) ([]PackageInfo, error)
	
	// Install installs a package
	Install(name string) error
	
	// Remove removes a package
	Remove(name string) error
	
	// Update updates a package
	Update(name string) error
	
	// UpgradeAll upgrades all packages
	UpgradeAll() error
	
	// Info returns package information
	Info(name string) (PackageInfo, error)
	
	// Type returns the package manager type
	Type() string
}

// DetectManager detects and returns the appropriate package manager
func DetectManager() (Manager, error) {
	switch runtime.GOOS {
	case "darwin":
		if _, err := exec.LookPath("brew"); err == nil {
			return NewBrewManager()
		}
	case "linux":
		if _, err := exec.LookPath("apt"); err == nil {
			return NewAptManager()
		}
		if _, err := exec.LookPath("yum"); err == nil {
			return NewYumManager()
		}
		if _, err := exec.LookPath("dnf"); err == nil {
			return NewDnfManager()
		}
	case "windows":
		if _, err := exec.LookPath("choco"); err == nil {
			return NewChocoManager()
		}
		if _, err := exec.LookPath("winget"); err == nil {
			return NewWingetManager()
		}
	}
	
	return &NullManager{}, nil
}

// NullManager is a no-op package manager
type NullManager struct{}

func (m *NullManager) List() ([]PackageInfo, error)         { return nil, nil }
func (m *NullManager) Search(query string) ([]PackageInfo, error) { return nil, nil }
func (m *NullManager) Install(name string) error            { return nil }
func (m *NullManager) Remove(name string) error             { return nil }
func (m *NullManager) Update(name string) error             { return nil }
func (m *NullManager) UpgradeAll() error                    { return nil }
func (m *NullManager) Info(name string) (PackageInfo, error) { return PackageInfo{}, nil }
func (m *NullManager) Type() string                         { return "none" }
