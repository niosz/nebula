package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/minio/selfupdate"
)

// Version is set at build time
var Version = "dev"

// ReleaseInfo contains release information
type ReleaseInfo struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Body        string    `json:"body"`
	PublishedAt time.Time `json:"published_at"`
	Assets      []Asset   `json:"assets"`
}

// Asset represents a release asset
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// UpdateInfo contains update information
type UpdateInfo struct {
	Available   bool      `json:"available"`
	CurrentVer  string    `json:"current_version"`
	LatestVer   string    `json:"latest_version"`
	ReleaseDate time.Time `json:"release_date"`
	ReleaseURL  string    `json:"release_url"`
	Changelog   string    `json:"changelog"`
}

// Updater handles self-updates
type Updater struct {
	githubRepo    string
	currentVer    string
	enabled       bool
	checkInterval time.Duration
	lastCheck     time.Time
	latestRelease *ReleaseInfo
}

// NewUpdater creates a new updater
func NewUpdater(githubRepo string, enabled bool, checkInterval time.Duration) *Updater {
	return &Updater{
		githubRepo:    githubRepo,
		currentVer:    Version,
		enabled:       enabled,
		checkInterval: checkInterval,
	}
}

// CheckForUpdate checks for a new version
func (u *Updater) CheckForUpdate() (UpdateInfo, error) {
	info := UpdateInfo{
		CurrentVer: u.currentVer,
		Available:  false,
	}

	if !u.enabled {
		return info, nil
	}

	release, err := u.getLatestRelease()
	if err != nil {
		return info, err
	}

	u.latestRelease = release
	u.lastCheck = time.Now()

	info.LatestVer = release.TagName
	info.ReleaseDate = release.PublishedAt
	info.Changelog = release.Body
	info.ReleaseURL = fmt.Sprintf("https://github.com/%s/releases/tag/%s", u.githubRepo, release.TagName)

	// Compare versions
	if u.isNewerVersion(release.TagName, u.currentVer) {
		info.Available = true
	}

	return info, nil
}

// Apply applies the update
func (u *Updater) Apply() error {
	if !u.enabled {
		return fmt.Errorf("updater is disabled")
	}

	if u.latestRelease == nil {
		// Check for update first
		_, err := u.CheckForUpdate()
		if err != nil {
			return err
		}
	}

	if u.latestRelease == nil {
		return fmt.Errorf("no release information available")
	}

	// Find the appropriate asset for this platform
	asset := u.findAsset()
	if asset == nil {
		return fmt.Errorf("no suitable binary found for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	// Download and apply update
	resp, err := http.Get(asset.BrowserDownloadURL)
	if err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download update: status %d", resp.StatusCode)
	}

	err = selfupdate.Apply(resp.Body, selfupdate.Options{})
	if err != nil {
		if rerr := selfupdate.RollbackError(err); rerr != nil {
			return fmt.Errorf("failed to rollback after failed update: %w", rerr)
		}
		return fmt.Errorf("failed to apply update: %w", err)
	}

	return nil
}

// getLatestRelease fetches the latest release from GitHub
func (u *Updater) getLatestRelease() (*ReleaseInfo, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", u.githubRepo)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error: %s", string(body))
	}

	var release ReleaseInfo
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode release: %w", err)
	}

	return &release, nil
}

// findAsset finds the appropriate asset for this platform
func (u *Updater) findAsset() *Asset {
	if u.latestRelease == nil {
		return nil
	}

	osName := runtime.GOOS
	arch := runtime.GOARCH

	// Try to find exact match
	for _, asset := range u.latestRelease.Assets {
		name := strings.ToLower(asset.Name)
		
		// Skip checksums and signatures
		if strings.HasSuffix(name, ".sha256") || strings.HasSuffix(name, ".sig") {
			continue
		}

		// Check for OS
		hasOS := strings.Contains(name, osName) ||
			(osName == "darwin" && strings.Contains(name, "macos")) ||
			(osName == "windows" && strings.Contains(name, "win"))

		// Check for architecture
		hasArch := strings.Contains(name, arch) ||
			(arch == "amd64" && (strings.Contains(name, "x86_64") || strings.Contains(name, "x64"))) ||
			(arch == "arm64" && strings.Contains(name, "aarch64"))

		if hasOS && hasArch {
			return &asset
		}
	}

	return nil
}

// isNewerVersion compares version strings
func (u *Updater) isNewerVersion(new, current string) bool {
	// Remove 'v' prefix
	new = strings.TrimPrefix(new, "v")
	current = strings.TrimPrefix(current, "v")

	// Handle dev version
	if current == "dev" || current == "" {
		return true
	}

	// Simple string comparison for semver
	newParts := strings.Split(new, ".")
	currentParts := strings.Split(current, ".")

	for i := 0; i < len(newParts) && i < len(currentParts); i++ {
		if newParts[i] > currentParts[i] {
			return true
		} else if newParts[i] < currentParts[i] {
			return false
		}
	}

	return len(newParts) > len(currentParts)
}

// GetVersion returns the current version
func (u *Updater) GetVersion() string {
	return u.currentVer
}

// Restart restarts the application
func (u *Updater) Restart() error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}

	// On Unix, we can exec the new binary
	// On Windows, we need to start a new process
	if runtime.GOOS == "windows" {
		cmd := fmt.Sprintf("Start-Sleep -Seconds 2; Start-Process '%s'", executable)
		return runPowerShell(cmd)
	}

	return syscallExec(executable, os.Args, os.Environ())
}

// runPowerShell runs a PowerShell command (Windows only)
func runPowerShell(cmd string) error {
	return nil // Stub for cross-compilation
}

// syscallExec executes a new process (Unix only)
func syscallExec(argv0 string, argv []string, envv []string) error {
	return nil // Stub for cross-compilation
}
