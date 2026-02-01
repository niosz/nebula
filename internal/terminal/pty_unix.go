//go:build !windows

package terminal

import (
	"os"
	"os/exec"

	"github.com/creack/pty"
)

// newPlatformSession creates a new terminal session for Unix systems
func newPlatformSession(id, shell string, cols, rows uint16) (*Session, error) {
	cmd := exec.Command(shell)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	// Start the command with a pty
	ptmx, err := pty.StartWithSize(cmd, &pty.Winsize{
		Cols: cols,
		Rows: rows,
	})
	if err != nil {
		return nil, err
	}

	session := &Session{
		ID:    id,
		Shell: shell,
		Cmd:   cmd,
		Pty:   ptmx,
		OnResize: func(cols, rows uint16) error {
			return pty.Setsize(ptmx, &pty.Winsize{
				Cols: cols,
				Rows: rows,
			})
		},
	}

	return session, nil
}
