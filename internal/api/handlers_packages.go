package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/nebula/nebula/internal/packages"
)

// PackagesHandler handles package manager endpoints
type PackagesHandler struct {
	manager packages.Manager
}

// NewPackagesHandler creates a new packages handler
func NewPackagesHandler(manager packages.Manager) *PackagesHandler {
	return &PackagesHandler{manager: manager}
}

// List godoc
// @Summary List installed packages
// @Description Returns a list of installed packages
// @Tags packages
// @Produce json
// @Success 200 {array} packages.PackageInfo
// @Router /api/v1/packages [get]
func (h *PackagesHandler) List(c *gin.Context) {
	pkgs, err := h.manager.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, pkgs)
}

// Search godoc
// @Summary Search packages
// @Description Searches for packages in the repository
// @Tags packages
// @Produce json
// @Param q query string true "Search query"
// @Success 200 {array} packages.PackageInfo
// @Failure 400 {object} map[string]string
// @Router /api/v1/packages/search [get]
func (h *PackagesHandler) Search(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "query required"})
		return
	}

	pkgs, err := h.manager.Search(query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, pkgs)
}

// Install godoc
// @Summary Install a package
// @Description Installs a package
// @Tags packages
// @Accept json
// @Produce json
// @Param body body map[string]string true "Package name"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/packages/install [post]
func (h *PackagesHandler) Install(c *gin.Context) {
	var req struct {
		Name string `json:"name"`
	}
	if err := c.BindJSON(&req); err != nil || req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "package name required"})
		return
	}

	if err := h.manager.Install(req.Name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "package installed"})
}

// Remove godoc
// @Summary Remove a package
// @Description Removes an installed package
// @Tags packages
// @Produce json
// @Param name query string true "Package name"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/packages/remove [delete]
func (h *PackagesHandler) Remove(c *gin.Context) {
	name := c.Query("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "package name required"})
		return
	}

	if err := h.manager.Remove(name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "package removed"})
}

// Update godoc
// @Summary Update a package
// @Description Updates a package to the latest version
// @Tags packages
// @Accept json
// @Produce json
// @Param body body map[string]string true "Package name"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/packages/update [post]
func (h *PackagesHandler) Update(c *gin.Context) {
	var req struct {
		Name string `json:"name"`
	}
	if err := c.BindJSON(&req); err != nil || req.Name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "package name required"})
		return
	}

	if err := h.manager.Update(req.Name); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "package updated"})
}

// UpgradeAll godoc
// @Summary Upgrade all packages
// @Description Upgrades all installed packages
// @Tags packages
// @Produce json
// @Success 200 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/packages/upgrade-all [post]
func (h *PackagesHandler) UpgradeAll(c *gin.Context) {
	if err := h.manager.UpgradeAll(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "all packages upgraded"})
}

// Info godoc
// @Summary Get package info
// @Description Returns detailed information about a package
// @Tags packages
// @Produce json
// @Param name query string true "Package name"
// @Success 200 {object} packages.PackageInfo
// @Failure 400 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/v1/packages/info [get]
func (h *PackagesHandler) Info(c *gin.Context) {
	name := c.Query("name")
	if name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "package name required"})
		return
	}

	pkg, err := h.manager.Info(name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, pkg)
}

// GetType godoc
// @Summary Get package manager type
// @Description Returns the detected package manager type
// @Tags packages
// @Produce json
// @Success 200 {object} map[string]string
// @Router /api/v1/packages/type [get]
func (h *PackagesHandler) GetType(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"type": h.manager.Type()})
}
