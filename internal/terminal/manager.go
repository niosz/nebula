package terminal

import (
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"sync"
)

// Session represents a terminal session
type Session struct {
	ID       string
	Shell    string
	Cmd      *exec.Cmd
	Pty      io.ReadWriteCloser
	mu       sync.Mutex
	closed   bool
	OnResize func(cols, rows uint16) error
}

// IsClosed returns whether the session is closed
func (s *Session) IsClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

// Manager manages terminal sessions
type Manager struct {
	sessions      map[string]*Session
	mu            sync.RWMutex
	maxSessions   int
	allowedShells []string
	defaultShell  string
}

// NewManager creates a new terminal manager
func NewManager(maxSessions int, allowedShells []string, defaultShell string) *Manager {
	return &Manager{
		sessions:      make(map[string]*Session),
		maxSessions:   maxSessions,
		allowedShells: allowedShells,
		defaultShell:  defaultShell,
	}
}

// GetAvailableShells returns available shells on the system
func (m *Manager) GetAvailableShells() []string {
	var shells []string
	
	for _, shell := range m.allowedShells {
		if path, err := exec.LookPath(shell); err == nil {
			shells = append(shells, path)
		}
	}
	
	return shells
}

// GetDefaultShell returns the default shell (full path)
func (m *Manager) GetDefaultShell() string {
	if m.defaultShell != "" {
		if path, err := exec.LookPath(m.defaultShell); err == nil {
			return path
		}
	}
	
	// Auto-detect
	switch runtime.GOOS {
	case "windows":
		if path, err := exec.LookPath("powershell"); err == nil {
			return path
		}
		if path, err := exec.LookPath("cmd"); err == nil {
			return path
		}
	default:
		if path, err := exec.LookPath("bash"); err == nil {
			return path
		}
		if path, err := exec.LookPath("sh"); err == nil {
			return path
		}
	}
	
	return ""
}

// IsShellAllowed checks if a shell is allowed
func (m *Manager) IsShellAllowed(shell string) bool {
	// Extract base name from path
	baseName := shell
	if idx := lastIndex(shell, '/'); idx >= 0 {
		baseName = shell[idx+1:]
	}
	if idx := lastIndex(shell, '\\'); idx >= 0 {
		baseName = shell[idx+1:]
	}
	
	for _, allowed := range m.allowedShells {
		if shell == allowed || baseName == allowed {
			return true
		}
	}
	return false
}

func lastIndex(s string, c byte) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == c {
			return i
		}
	}
	return -1
}

// CreateSession creates a new terminal session
func (m *Manager) CreateSession(id, shell string, cols, rows uint16) (*Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if len(m.sessions) >= m.maxSessions {
		return nil, fmt.Errorf("maximum sessions reached")
	}
	
	if _, exists := m.sessions[id]; exists {
		return nil, fmt.Errorf("session already exists")
	}
	
	if shell == "" {
		shell = m.GetDefaultShell()
	}
	
	if !m.IsShellAllowed(shell) {
		return nil, fmt.Errorf("shell not allowed: %s", shell)
	}
	
	session, err := newPlatformSession(id, shell, cols, rows)
	if err != nil {
		return nil, err
	}
	
	m.sessions[id] = session
	return session, nil
}

// GetSession returns a session by ID
func (m *Manager) GetSession(id string) (*Session, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, ok := m.sessions[id]
	return session, ok
}

// CloseSession closes a session
func (m *Manager) CloseSession(id string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	session, ok := m.sessions[id]
	if !ok {
		return fmt.Errorf("session not found")
	}
	
	session.Close()
	delete(m.sessions, id)
	return nil
}

// ListSessions returns all active sessions
func (m *Manager) ListSessions() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var ids []string
	for id := range m.sessions {
		ids = append(ids, id)
	}
	return ids
}

// Close closes all sessions
func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	for id, session := range m.sessions {
		session.Close()
		delete(m.sessions, id)
	}
}

// Read reads from the session
func (s *Session) Read(p []byte) (int, error) {
	if s.IsClosed() {
		return 0, io.EOF
	}
	return s.Pty.Read(p)
}

// Write writes to the session
func (s *Session) Write(p []byte) (int, error) {
	if s.IsClosed() {
		return 0, io.EOF
	}
	return s.Pty.Write(p)
}

// Close closes the session
func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.closed {
		return nil
	}
	
	s.closed = true
	
	if s.Cmd != nil && s.Cmd.Process != nil {
		s.Cmd.Process.Kill()
	}
	
	if s.Pty != nil {
		return s.Pty.Close()
	}
	
	return nil
}

// Resize resizes the terminal
func (s *Session) Resize(cols, rows uint16) error {
	if s.OnResize != nil {
		return s.OnResize(cols, rows)
	}
	return nil
}
