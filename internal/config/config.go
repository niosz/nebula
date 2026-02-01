package config

import (
	"fmt"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/nebula/nebula/internal/storage"
	"github.com/spf13/viper"
)

// Config holds all configuration values
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Storage  StorageConfig  `mapstructure:"storage"`
	Auth     AuthConfig     `mapstructure:"auth"`
	Metrics  MetricsConfig  `mapstructure:"metrics"`
	Terminal TerminalConfig `mapstructure:"terminal"`
	Files    FilesConfig    `mapstructure:"files"`
	Packages PackagesConfig `mapstructure:"packages"`
	Updater  UpdaterConfig  `mapstructure:"updater"`
	Logging  LoggingConfig  `mapstructure:"logging"`
}

// ServerConfig holds server configuration
type ServerConfig struct {
	Host            string        `mapstructure:"host"`
	Port            int           `mapstructure:"port"`
	ReadTimeout     time.Duration `mapstructure:"read_timeout"`
	WriteTimeout    time.Duration `mapstructure:"write_timeout"`
	ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
}

// StorageConfig holds storage configuration
type StorageConfig struct {
	Path             string        `mapstructure:"path"`
	MetricsRetention time.Duration `mapstructure:"metrics_retention"`
	AuditRetention   time.Duration `mapstructure:"audit_retention"`
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	Enabled  bool   `mapstructure:"enabled"`
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

// MetricsConfig holds metrics configuration
type MetricsConfig struct {
	Interval    time.Duration `mapstructure:"interval"`
	HistorySize int           `mapstructure:"history_size"`
}

// TerminalConfig holds terminal configuration
type TerminalConfig struct {
	DefaultShell  string   `mapstructure:"default_shell"`
	AllowedShells []string `mapstructure:"allowed_shells"`
	MaxSessions   int      `mapstructure:"max_sessions"`
}

// FilesConfig holds file manager configuration
type FilesConfig struct {
	RootPath          string   `mapstructure:"root_path"`
	MaxUploadSize     int64    `mapstructure:"max_upload_size"`
	AllowedExtensions []string `mapstructure:"allowed_extensions"`
}

// PackagesConfig holds packages configuration
type PackagesConfig struct {
	AutoDetect bool `mapstructure:"auto_detect"`
}

// UpdaterConfig holds updater configuration
type UpdaterConfig struct {
	Enabled       bool          `mapstructure:"enabled"`
	CheckInterval time.Duration `mapstructure:"check_interval"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// Manager manages configuration with hot reload support
type Manager struct {
	config  *Config
	storage *storage.Storage
	viper   *viper.Viper
	mu      sync.RWMutex

	onReload []func(*Config)
}

// NewManager creates a new configuration manager
func NewManager(configPath string, store *storage.Storage) (*Manager, error) {
	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType("yaml")

	// Set defaults
	setDefaults(v)

	// Read config file
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	m := &Manager{
		config:  &Config{},
		storage: store,
		viper:   v,
	}

	// Unmarshal config
	if err := v.Unmarshal(m.config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Apply overrides from storage
	m.applyStorageOverrides()

	// Watch for config changes
	v.WatchConfig()
	v.OnConfigChange(func(e fsnotify.Event) {
		m.reload()
	})

	return m, nil
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	// Server defaults
	v.SetDefault("server.host", "0.0.0.0")
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.read_timeout", "10s")
	v.SetDefault("server.write_timeout", "10s")
	v.SetDefault("server.shutdown_timeout", "30s")

	// Storage defaults
	v.SetDefault("storage.path", "./nebula.db")
	v.SetDefault("storage.metrics_retention", "1h")
	v.SetDefault("storage.audit_retention", "168h")

	// Auth defaults
	v.SetDefault("auth.enabled", false)
	v.SetDefault("auth.username", "admin")
	v.SetDefault("auth.password", "changeme")

	// Metrics defaults
	v.SetDefault("metrics.interval", "1s")
	v.SetDefault("metrics.history_size", 60)

	// Terminal defaults
	v.SetDefault("terminal.default_shell", "")
	v.SetDefault("terminal.allowed_shells", []string{"bash", "zsh", "sh", "ksh", "cmd", "powershell"})
	v.SetDefault("terminal.max_sessions", 10)

	// Files defaults
	v.SetDefault("files.root_path", "/")
	v.SetDefault("files.max_upload_size", 104857600) // 100MB
	v.SetDefault("files.allowed_extensions", []string{})

	// Packages defaults
	v.SetDefault("packages.auto_detect", true)

	// Updater defaults
	v.SetDefault("updater.enabled", true)
	v.SetDefault("updater.check_interval", "24h")

	// Logging defaults
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")
}

// Get returns the current configuration
func (m *Manager) Get() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config
}

// reload reloads the configuration
func (m *Manager) reload() {
	m.mu.Lock()
	defer m.mu.Unlock()

	newConfig := &Config{}
	if err := m.viper.Unmarshal(newConfig); err != nil {
		return
	}

	m.config = newConfig
	m.applyStorageOverrides()

	// Notify listeners
	for _, fn := range m.onReload {
		go fn(m.config)
	}
}

// Reload forces a configuration reload
func (m *Manager) Reload() error {
	if err := m.viper.ReadInConfig(); err != nil {
		return err
	}
	m.reload()
	return nil
}

// OnReload registers a callback for configuration changes
func (m *Manager) OnReload(fn func(*Config)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onReload = append(m.onReload, fn)
}

// applyStorageOverrides applies configuration overrides from storage
func (m *Manager) applyStorageOverrides() {
	if m.storage == nil {
		return
	}

	// Check for port override
	var port int
	if err := m.storage.GetJSON(storage.BucketConfig, "server.port", &port); err == nil && port > 0 {
		m.config.Server.Port = port
	}

	// Check for auth override
	var authEnabled bool
	if err := m.storage.GetJSON(storage.BucketConfig, "auth.enabled", &authEnabled); err == nil {
		m.config.Auth.Enabled = authEnabled
	}

	// Add more overrides as needed
}

// SetOverride sets a configuration override in storage
func (m *Manager) SetOverride(key string, value interface{}) error {
	if m.storage == nil {
		return fmt.Errorf("storage not available")
	}
	return m.storage.SetJSON(storage.BucketConfig, key, value)
}

// GetOverride gets a configuration override from storage
func (m *Manager) GetOverride(key string, value interface{}) error {
	if m.storage == nil {
		return fmt.Errorf("storage not available")
	}
	return m.storage.GetJSON(storage.BucketConfig, key, value)
}

// Address returns the server address string
func (c *Config) Address() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}
