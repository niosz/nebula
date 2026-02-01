package api

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/nebula/nebula/internal/terminal"
	ws "github.com/nebula/nebula/internal/websocket"
)

// TerminalHandler handles terminal endpoints
type TerminalHandler struct {
	manager     *terminal.Manager
	terminalHub *ws.TerminalHub
}

// NewTerminalHandler creates a new terminal handler
func NewTerminalHandler(manager *terminal.Manager, hub *ws.TerminalHub) *TerminalHandler {
	return &TerminalHandler{
		manager:     manager,
		terminalHub: hub,
	}
}

// GetShells godoc
// @Summary Get available shells
// @Description Returns a list of available shells
// @Tags terminal
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/terminal/shells [get]
func (h *TerminalHandler) GetShells(c *gin.Context) {
	shells := h.manager.GetAvailableShells()
	defaultShell := h.manager.GetDefaultShell()

	c.JSON(http.StatusOK, gin.H{
		"shells":        shells,
		"default_shell": defaultShell,
	})
}

// GetSessions godoc
// @Summary Get active sessions
// @Description Returns a list of active terminal sessions
// @Tags terminal
// @Produce json
// @Success 200 {array} string
// @Router /api/v1/terminal/sessions [get]
func (h *TerminalHandler) GetSessions(c *gin.Context) {
	sessions := h.manager.ListSessions()
	c.JSON(http.StatusOK, sessions)
}

// HandleWebSocket handles the terminal WebSocket connection
func (h *TerminalHandler) HandleWebSocket(c *gin.Context) {
	sessionID := c.Query("session")
	shell := c.Query("shell")
	
	// Default terminal size
	cols := uint16(80)
	rows := uint16(24)

	// Create terminal session
	session, err := h.manager.CreateSession(sessionID, shell, cols, rows)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Upgrade to WebSocket
	client, err := h.terminalHub.HandleTerminalWebSocket(c.Writer, c.Request, sessionID)
	if err != nil {
		h.manager.CloseSession(sessionID)
		return
	}

	// Handle terminal I/O
	go h.handleTerminalInput(client, session)
	go h.handleTerminalOutput(client, session)
}

// handleTerminalInput reads from WebSocket and writes to PTY
func (h *TerminalHandler) handleTerminalInput(client *ws.TerminalClient, session *terminal.Session) {
	defer func() {
		h.manager.CloseSession(session.ID)
		h.terminalHub.RemoveClient(session.ID)
	}()

	for {
		msgType, data, err := client.ReadMessage()
		if err != nil {
			return
		}

		if msgType == websocket.TextMessage {
			// Check for resize message
			var msg struct {
				Type string `json:"type"`
				Cols uint16 `json:"cols"`
				Rows uint16 `json:"rows"`
			}
			if err := json.Unmarshal(data, &msg); err == nil && msg.Type == "resize" {
				session.Resize(msg.Cols, msg.Rows)
				continue
			}
		}

		// Write to PTY
		if msgType == websocket.BinaryMessage || msgType == websocket.TextMessage {
			session.Write(data)
		}
	}
}

// handleTerminalOutput reads from PTY and writes to WebSocket
func (h *TerminalHandler) handleTerminalOutput(client *ws.TerminalClient, session *terminal.Session) {
	buf := make([]byte, 4096)
	for {
		n, err := session.Read(buf)
		if err != nil {
			if err != io.EOF {
				// Terminal closed
			}
			return
		}

		if n > 0 {
			if err := client.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
				return
			}
		}
	}
}
