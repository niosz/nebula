package auth

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"strings"
	"sync"

	"github.com/nebula/nebula/internal/storage"
)

const (
	credentialsKey = "sudo_credentials"
	encryptionKey  = "nebula_secret_key_32bytes_long!" // In production, use a secure key derivation
)

// PrivilegeManager manages elevated privileges and credentials
type PrivilegeManager struct {
	storage    *storage.Storage
	password   string
	mu         sync.RWMutex
	isElevated bool
}

// NewPrivilegeManager creates a new privilege manager
func NewPrivilegeManager(store *storage.Storage) *PrivilegeManager {
	pm := &PrivilegeManager{
		storage:    store,
		isElevated: IsRunningAsRoot(),
	}

	// Load saved credentials
	if store != nil {
		pm.loadCredentials()
	}

	return pm
}

// IsRunningAsRoot checks if the application is running with elevated privileges
func IsRunningAsRoot() bool {
	switch runtime.GOOS {
	case "windows":
		return isWindowsAdmin()
	default:
		return os.Getuid() == 0
	}
}

// isWindowsAdmin checks if running as admin on Windows
func isWindowsAdmin() bool {
	cmd := exec.Command("net", "session")
	err := cmd.Run()
	return err == nil
}

// RequireRoot returns an error if not running as root/admin
func RequireRoot() error {
	if !IsRunningAsRoot() {
		switch runtime.GOOS {
		case "windows":
			return fmt.Errorf("Nebula richiede privilegi di amministratore. Esegui come Amministratore")
		default:
			return fmt.Errorf("Nebula richiede privilegi di root. Esegui con sudo")
		}
	}
	return nil
}

// GetCurrentUser returns the current user info
func GetCurrentUser() (*user.User, error) {
	return user.Current()
}

// IsElevated returns whether the app is running elevated
func (pm *PrivilegeManager) IsElevated() bool {
	return pm.isElevated
}

// SetCredentials sets and saves sudo credentials
func (pm *PrivilegeManager) SetCredentials(password string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.password = password

	// Encrypt and save to storage
	if pm.storage != nil {
		encrypted, err := encrypt(password, encryptionKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt credentials: %w", err)
		}
		return pm.storage.Set(storage.BucketSessions, credentialsKey, []byte(encrypted))
	}

	return nil
}

// GetCredentials returns the stored credentials
func (pm *PrivilegeManager) GetCredentials() string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.password
}

// HasCredentials checks if credentials are stored
func (pm *PrivilegeManager) HasCredentials() bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.password != ""
}

// ClearCredentials removes stored credentials
func (pm *PrivilegeManager) ClearCredentials() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.password = ""

	if pm.storage != nil {
		return pm.storage.Delete(storage.BucketSessions, credentialsKey)
	}

	return nil
}

// loadCredentials loads credentials from storage
func (pm *PrivilegeManager) loadCredentials() {
	data, err := pm.storage.Get(storage.BucketSessions, credentialsKey)
	if err != nil || len(data) == 0 {
		return
	}

	decrypted, err := decrypt(string(data), encryptionKey)
	if err != nil {
		return
	}

	pm.password = decrypted
}

// ValidateCredentials validates sudo credentials
func (pm *PrivilegeManager) ValidateCredentials(password string) bool {
	if runtime.GOOS == "windows" {
		// On Windows, we don't use sudo
		return true
	}

	// Test credentials with sudo -S
	cmd := exec.Command("sudo", "-S", "-v")
	cmd.Stdin = strings.NewReader(password + "\n")
	err := cmd.Run()
	return err == nil
}

// RunWithPrivileges runs a command with elevated privileges
func (pm *PrivilegeManager) RunWithPrivileges(name string, args ...string) ([]byte, error) {
	if pm.isElevated {
		// Already running as root, execute directly
		cmd := exec.Command(name, args...)
		return cmd.CombinedOutput()
	}

	// Use stored credentials with sudo
	pm.mu.RLock()
	password := pm.password
	pm.mu.RUnlock()

	if password == "" {
		return nil, fmt.Errorf("no credentials stored, cannot run privileged command")
	}

	if runtime.GOOS == "windows" {
		// Windows doesn't use sudo
		cmd := exec.Command(name, args...)
		return cmd.CombinedOutput()
	}

	// Use sudo with password from stdin
	fullArgs := append([]string{"-S", name}, args...)
	cmd := exec.Command("sudo", fullArgs...)
	cmd.Stdin = strings.NewReader(password + "\n")
	return cmd.CombinedOutput()
}

// RunWithPrivilegesInteractive runs an interactive command with privileges
func (pm *PrivilegeManager) RunWithPrivilegesInteractive(name string, args ...string) *exec.Cmd {
	if pm.isElevated {
		return exec.Command(name, args...)
	}

	if runtime.GOOS == "windows" {
		return exec.Command(name, args...)
	}

	fullArgs := append([]string{"-S", name}, args...)
	cmd := exec.Command("sudo", fullArgs...)

	pm.mu.RLock()
	password := pm.password
	pm.mu.RUnlock()

	if password != "" {
		cmd.Stdin = strings.NewReader(password + "\n")
	}

	return cmd
}

// encrypt encrypts a string using AES-GCM
func encrypt(plaintext, key string) (string, error) {
	// Create a 32-byte key using SHA-256
	keyHash := sha256.Sum256([]byte(key))

	block, err := aes.NewCipher(keyHash[:])
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decrypt decrypts a string using AES-GCM
func decrypt(ciphertext, key string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	keyHash := sha256.Sum256([]byte(key))

	block, err := aes.NewCipher(keyHash[:])
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, cipherData := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, cipherData, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
