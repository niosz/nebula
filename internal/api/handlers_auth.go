package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nebula/nebula/internal/auth"
)

// AuthHandler handles authentication and privilege endpoints
type AuthHandler struct {
	privilegeManager *auth.PrivilegeManager
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(pm *auth.PrivilegeManager) *AuthHandler {
	return &AuthHandler{privilegeManager: pm}
}

// GetPrivilegeStatus godoc
// @Summary Get privilege status
// @Description Returns current privilege status and whether credentials are stored
// @Tags auth
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/auth/status [get]
func (h *AuthHandler) GetPrivilegeStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"is_elevated":       h.privilegeManager.IsElevated(),
		"has_credentials":   h.privilegeManager.HasCredentials(),
		"requires_password": !h.privilegeManager.IsElevated() && !h.privilegeManager.HasCredentials(),
	})
}

// SetCredentials godoc
// @Summary Set sudo credentials
// @Description Stores sudo password for privileged operations
// @Tags auth
// @Accept json
// @Produce json
// @Param body body map[string]string true "Password"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /api/v1/auth/credentials [post]
func (h *AuthHandler) SetCredentials(c *gin.Context) {
	var req struct {
		Password string `json:"password"`
	}

	if err := c.BindJSON(&req); err != nil || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password required"})
		return
	}

	// Validate credentials
	if !h.privilegeManager.ValidateCredentials(req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	// Store credentials
	if err := h.privilegeManager.SetCredentials(req.Password); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to store credentials"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "credentials stored successfully"})
}

// ClearCredentials godoc
// @Summary Clear stored credentials
// @Description Removes stored sudo credentials
// @Tags auth
// @Produce json
// @Success 200 {object} map[string]string
// @Router /api/v1/auth/credentials [delete]
func (h *AuthHandler) ClearCredentials(c *gin.Context) {
	if err := h.privilegeManager.ClearCredentials(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to clear credentials"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "credentials cleared"})
}

// ValidateCredentials godoc
// @Summary Validate credentials
// @Description Tests if provided credentials are valid
// @Tags auth
// @Accept json
// @Produce json
// @Param body body map[string]string true "Password"
// @Success 200 {object} map[string]bool
// @Router /api/v1/auth/validate [post]
func (h *AuthHandler) ValidateCredentials(c *gin.Context) {
	var req struct {
		Password string `json:"password"`
	}

	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "password required"})
		return
	}

	valid := h.privilegeManager.ValidateCredentials(req.Password)
	c.JSON(http.StatusOK, gin.H{"valid": valid})
}
