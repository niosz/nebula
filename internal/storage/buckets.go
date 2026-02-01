package storage

import (
	"encoding/json"
	"time"

	bolt "go.etcd.io/bbolt"
)

// Bucket names
const (
	BucketConfig           = "config"
	BucketMetricsHistory   = "metrics_history"
	BucketSessions         = "sessions"
	BucketTerminalSessions = "terminal_sessions"
	BucketBookmarks        = "bookmarks"
	BucketPreferences      = "preferences"
	BucketAuditLog         = "audit_log"
)

// AllBuckets returns all bucket names
var AllBuckets = []string{
	BucketConfig,
	BucketMetricsHistory,
	BucketSessions,
	BucketTerminalSessions,
	BucketBookmarks,
	BucketPreferences,
	BucketAuditLog,
}

// initBuckets creates all required buckets
func (s *Storage) initBuckets() error {
	return s.db.Update(func(tx *bolt.Tx) error {
		for _, bucket := range AllBuckets {
			_, err := tx.CreateBucketIfNotExists([]byte(bucket))
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// AuditEntry represents an audit log entry
type AuditEntry struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Action    string    `json:"action"`
	Resource  string    `json:"resource"`
	Details   string    `json:"details"`
	User      string    `json:"user"`
	IP        string    `json:"ip"`
}

// MetricsEntry represents a metrics history entry
type MetricsEntry struct {
	Timestamp time.Time   `json:"timestamp"`
	CPU       CPUMetrics  `json:"cpu"`
	Memory    MemMetrics  `json:"memory"`
	Disk      []DiskInfo  `json:"disk"`
	Network   []NetInfo   `json:"network"`
}

// CPUMetrics represents CPU usage metrics
type CPUMetrics struct {
	UsagePercent []float64 `json:"usage_percent"`
	TotalPercent float64   `json:"total_percent"`
}

// MemMetrics represents memory usage metrics
type MemMetrics struct {
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsedPercent float64 `json:"used_percent"`
	SwapTotal   uint64  `json:"swap_total"`
	SwapUsed    uint64  `json:"swap_used"`
	SwapFree    uint64  `json:"swap_free"`
}

// DiskInfo represents disk usage information
type DiskInfo struct {
	Device      string  `json:"device"`
	Mountpoint  string  `json:"mountpoint"`
	Fstype      string  `json:"fstype"`
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsedPercent float64 `json:"used_percent"`
}

// NetInfo represents network interface information
type NetInfo struct {
	Name        string `json:"name"`
	BytesSent   uint64 `json:"bytes_sent"`
	BytesRecv   uint64 `json:"bytes_recv"`
	PacketsSent uint64 `json:"packets_sent"`
	PacketsRecv uint64 `json:"packets_recv"`
}

// Session represents a user session
type Session struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	IP        string    `json:"ip"`
}

// TerminalSession represents a terminal session state
type TerminalSession struct {
	ID        string    `json:"id"`
	Shell     string    `json:"shell"`
	CreatedAt time.Time `json:"created_at"`
	LastUsed  time.Time `json:"last_used"`
}

// Bookmark represents a file manager bookmark
type Bookmark struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Path string `json:"path"`
}

// Preferences represents user preferences
type Preferences struct {
	Theme       string `json:"theme"`
	Language    string `json:"language"`
	RefreshRate int    `json:"refresh_rate"`
}

// AddAuditLog adds an entry to the audit log
func (s *Storage) AddAuditLog(entry AuditEntry) error {
	return s.SetJSON(BucketAuditLog, entry.ID, entry)
}

// AddMetricsEntry adds a metrics entry to history
func (s *Storage) AddMetricsEntry(entry MetricsEntry) error {
	key := entry.Timestamp.Format(time.RFC3339Nano)
	return s.SetJSON(BucketMetricsHistory, key, entry)
}

// GetMetricsHistory retrieves metrics history
func (s *Storage) GetMetricsHistory(limit int) ([]MetricsEntry, error) {
	all, err := s.GetAll(BucketMetricsHistory)
	if err != nil {
		return nil, err
	}

	var entries []MetricsEntry
	for _, v := range all {
		var entry MetricsEntry
		if err := unmarshalJSON(v, &entry); err == nil {
			entries = append(entries, entry)
		}
	}

	// Sort by timestamp descending and limit
	sortMetricsByTimestamp(entries)
	if limit > 0 && len(entries) > limit {
		entries = entries[:limit]
	}

	return entries, nil
}

// Helper function to unmarshal JSON
func unmarshalJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

// sortMetricsByTimestamp sorts metrics entries by timestamp (newest first)
func sortMetricsByTimestamp(entries []MetricsEntry) {
	for i := 0; i < len(entries)-1; i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[j].Timestamp.After(entries[i].Timestamp) {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}
}
