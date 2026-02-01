//go:build windows

package terminal

import (
	"io"
	"os"
	"os/exec"
)

// pipeReadWriteCloser wraps stdin/stdout pipes
type pipeReadWriteCloser struct {
	stdin  io.WriteCloser
	stdout io.ReadCloser
}

func (p *pipeReadWriteCloser) Read(b []byte) (int, error) {
	return p.stdout.Read(b)
}

func (p *pipeReadWriteCloser) Write(b []byte) (int, error) {
	return p.stdin.Write(b)
}

func (p *pipeReadWriteCloser) Close() error {
	p.stdin.Close()
	return p.stdout.Close()
}

// newPlatformSession creates a new terminal session for Windows
func newPlatformSession(id, shell string, cols, rows uint16) (*Session, error) {
	var cmd *exec.Cmd
	
	switch shell {
	case "powershell":
		cmd = exec.Command("powershell", "-NoLogo", "-NoProfile")
	case "cmd":
		cmd = exec.Command("cmd")
	default:
		cmd = exec.Command(shell)
	}

	cmd.Env = append(os.Environ(), "TERM=xterm-256color")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, err
	}

	// Redirect stderr to stdout
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		return nil, err
	}

	pty := &pipeReadWriteCloser{
		stdin:  stdin,
		stdout: stdout,
	}

	session := &Session{
		ID:    id,
		Shell: shell,
		Cmd:   cmd,
		Pty:   pty,
		OnResize: func(cols, rows uint16) error {
			// Windows pipes don't support resize
			return nil
		},
	}

	return session, nil
}
