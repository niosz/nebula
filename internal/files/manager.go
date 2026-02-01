package files

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// FileInfo contains file information
type FileInfo struct {
	Name        string    `json:"name"`
	Path        string    `json:"path"`
	Size        int64     `json:"size"`
	Mode        string    `json:"mode"`
	ModTime     time.Time `json:"mod_time"`
	IsDir       bool      `json:"is_dir"`
	IsSymlink   bool      `json:"is_symlink"`
	Extension   string    `json:"extension,omitempty"`
	MimeType    string    `json:"mime_type,omitempty"`
	Permissions string    `json:"permissions"`
}

// Manager manages file operations
type Manager struct {
	rootPath          string
	maxUploadSize     int64
	allowedExtensions []string
}

// NewManager creates a new file manager
func NewManager(rootPath string, maxUploadSize int64, allowedExtensions []string) *Manager {
	return &Manager{
		rootPath:          rootPath,
		maxUploadSize:     maxUploadSize,
		allowedExtensions: allowedExtensions,
	}
}

// List returns files in a directory
func (m *Manager) List(path string) ([]FileInfo, error) {
	fullPath, err := m.resolvePath(path)
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var files []FileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}

		file := FileInfo{
			Name:        entry.Name(),
			Path:        filepath.Join(path, entry.Name()),
			Size:        info.Size(),
			Mode:        info.Mode().String(),
			ModTime:     info.ModTime(),
			IsDir:       entry.IsDir(),
			IsSymlink:   info.Mode()&os.ModeSymlink != 0,
			Permissions: formatPermissions(info.Mode()),
		}

		if !entry.IsDir() {
			file.Extension = strings.TrimPrefix(filepath.Ext(entry.Name()), ".")
			file.MimeType = getMimeType(file.Extension)
		}

		files = append(files, file)
	}

	// Sort: directories first, then by name
	sort.Slice(files, func(i, j int) bool {
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir
		}
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})

	return files, nil
}

// Info returns information about a file or directory
func (m *Manager) Info(path string) (FileInfo, error) {
	fullPath, err := m.resolvePath(path)
	if err != nil {
		return FileInfo{}, err
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		return FileInfo{}, fmt.Errorf("failed to stat file: %w", err)
	}

	file := FileInfo{
		Name:        info.Name(),
		Path:        path,
		Size:        info.Size(),
		Mode:        info.Mode().String(),
		ModTime:     info.ModTime(),
		IsDir:       info.IsDir(),
		IsSymlink:   info.Mode()&os.ModeSymlink != 0,
		Permissions: formatPermissions(info.Mode()),
	}

	if !info.IsDir() {
		file.Extension = strings.TrimPrefix(filepath.Ext(info.Name()), ".")
		file.MimeType = getMimeType(file.Extension)
	}

	return file, nil
}

// Read reads the content of a file
func (m *Manager) Read(path string) ([]byte, error) {
	fullPath, err := m.resolvePath(path)
	if err != nil {
		return nil, err
	}

	// Check file size
	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	if info.IsDir() {
		return nil, fmt.Errorf("cannot read directory")
	}

	// Limit file size for reading
	if info.Size() > 10*1024*1024 { // 10MB limit
		return nil, fmt.Errorf("file too large to read")
	}

	return os.ReadFile(fullPath)
}

// Write writes content to a file
func (m *Manager) Write(path string, content []byte) error {
	fullPath, err := m.resolvePath(path)
	if err != nil {
		return err
	}

	if err := m.checkExtension(path); err != nil {
		return err
	}

	return os.WriteFile(fullPath, content, 0644)
}

// CreateDir creates a directory
func (m *Manager) CreateDir(path string) error {
	fullPath, err := m.resolvePath(path)
	if err != nil {
		return err
	}

	return os.MkdirAll(fullPath, 0755)
}

// Delete deletes a file or directory
func (m *Manager) Delete(path string) error {
	fullPath, err := m.resolvePath(path)
	if err != nil {
		return err
	}

	// Prevent deleting root
	if fullPath == m.rootPath || fullPath == "/" {
		return fmt.Errorf("cannot delete root directory")
	}

	return os.RemoveAll(fullPath)
}

// Rename renames a file or directory
func (m *Manager) Rename(oldPath, newPath string) error {
	oldFullPath, err := m.resolvePath(oldPath)
	if err != nil {
		return err
	}

	newFullPath, err := m.resolvePath(newPath)
	if err != nil {
		return err
	}

	return os.Rename(oldFullPath, newFullPath)
}

// Copy copies a file or directory
func (m *Manager) Copy(srcPath, dstPath string) error {
	srcFullPath, err := m.resolvePath(srcPath)
	if err != nil {
		return err
	}

	dstFullPath, err := m.resolvePath(dstPath)
	if err != nil {
		return err
	}

	info, err := os.Stat(srcFullPath)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return m.copyDir(srcFullPath, dstFullPath)
	}
	return m.copyFile(srcFullPath, dstFullPath)
}

// copyFile copies a single file
func (m *Manager) copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// copyDir copies a directory recursively
func (m *Manager) copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := m.copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			if err := m.copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
	}

	return nil
}

// Upload handles file upload
func (m *Manager) Upload(path string, reader io.Reader, filename string) error {
	fullPath, err := m.resolvePath(path)
	if err != nil {
		return err
	}

	if err := m.checkExtension(filename); err != nil {
		return err
	}

	targetPath := filepath.Join(fullPath, filename)
	
	file, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	// Limit upload size
	limitedReader := io.LimitReader(reader, m.maxUploadSize)
	
	_, err = io.Copy(file, limitedReader)
	return err
}

// Download prepares a file for download
func (m *Manager) Download(path string) (io.ReadCloser, int64, error) {
	fullPath, err := m.resolvePath(path)
	if err != nil {
		return nil, 0, err
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		return nil, 0, err
	}

	if info.IsDir() {
		// Create zip file for directory
		return m.zipDirectory(fullPath)
	}

	file, err := os.Open(fullPath)
	if err != nil {
		return nil, 0, err
	}

	return file, info.Size(), nil
}

// zipDirectory creates a zip archive of a directory
func (m *Manager) zipDirectory(path string) (io.ReadCloser, int64, error) {
	// Create temporary file for zip
	tmpFile, err := os.CreateTemp("", "nebula-download-*.zip")
	if err != nil {
		return nil, 0, err
	}

	zipWriter := zip.NewWriter(tmpFile)

	basePath := filepath.Dir(path)
	err = filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(basePath, filePath)
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		writer, err := zipWriter.Create(relPath)
		if err != nil {
			return err
		}

		file, err := os.Open(filePath)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})

	if err != nil {
		zipWriter.Close()
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return nil, 0, err
	}

	zipWriter.Close()

	// Get file size
	info, _ := tmpFile.Stat()
	size := info.Size()

	// Seek to beginning
	tmpFile.Seek(0, 0)

	return &tempFileReader{tmpFile}, size, nil
}

// tempFileReader wraps a temp file and deletes it on close
type tempFileReader struct {
	*os.File
}

func (t *tempFileReader) Close() error {
	name := t.File.Name()
	t.File.Close()
	return os.Remove(name)
}

// resolvePath resolves and validates a path
func (m *Manager) resolvePath(path string) (string, error) {
	// Clean the path
	cleanPath := filepath.Clean(path)
	
	// Make it relative to root
	if !filepath.IsAbs(cleanPath) {
		cleanPath = filepath.Join(m.rootPath, cleanPath)
	}

	// Resolve to absolute path
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return "", err
	}

	// Ensure path is within root
	if m.rootPath != "/" {
		if !strings.HasPrefix(absPath, m.rootPath) {
			return "", fmt.Errorf("path outside root directory")
		}
	}

	return absPath, nil
}

// checkExtension validates file extension
func (m *Manager) checkExtension(filename string) error {
	if len(m.allowedExtensions) == 0 {
		return nil // All extensions allowed
	}

	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), "."))
	for _, allowed := range m.allowedExtensions {
		if strings.ToLower(allowed) == ext {
			return nil
		}
	}

	return fmt.Errorf("file extension not allowed: %s", ext)
}

// formatPermissions formats file permissions as rwxrwxrwx
func formatPermissions(mode os.FileMode) string {
	perm := mode.Perm()
	result := make([]byte, 9)
	
	for i := 0; i < 9; i++ {
		if perm&(1<<uint(8-i)) != 0 {
			switch i % 3 {
			case 0:
				result[i] = 'r'
			case 1:
				result[i] = 'w'
			case 2:
				result[i] = 'x'
			}
		} else {
			result[i] = '-'
		}
	}
	
	return string(result)
}

// getMimeType returns MIME type for a file extension
func getMimeType(ext string) string {
	mimeTypes := map[string]string{
		"txt":  "text/plain",
		"html": "text/html",
		"css":  "text/css",
		"js":   "application/javascript",
		"json": "application/json",
		"xml":  "application/xml",
		"pdf":  "application/pdf",
		"zip":  "application/zip",
		"tar":  "application/x-tar",
		"gz":   "application/gzip",
		"jpg":  "image/jpeg",
		"jpeg": "image/jpeg",
		"png":  "image/png",
		"gif":  "image/gif",
		"svg":  "image/svg+xml",
		"ico":  "image/x-icon",
		"mp3":  "audio/mpeg",
		"mp4":  "video/mp4",
		"webm": "video/webm",
		"go":   "text/x-go",
		"py":   "text/x-python",
		"sh":   "application/x-sh",
		"md":   "text/markdown",
		"yaml": "text/yaml",
		"yml":  "text/yaml",
	}

	if mime, ok := mimeTypes[strings.ToLower(ext)]; ok {
		return mime
	}
	return "application/octet-stream"
}

// Search searches for files matching a pattern
func (m *Manager) Search(basePath, pattern string) ([]FileInfo, error) {
	fullPath, err := m.resolvePath(basePath)
	if err != nil {
		return nil, err
	}

	var results []FileInfo
	err = filepath.Walk(fullPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		matched, err := filepath.Match(pattern, info.Name())
		if err != nil {
			return nil
		}

		if matched {
			relPath, _ := filepath.Rel(m.rootPath, path)
			results = append(results, FileInfo{
				Name:        info.Name(),
				Path:        relPath,
				Size:        info.Size(),
				Mode:        info.Mode().String(),
				ModTime:     info.ModTime(),
				IsDir:       info.IsDir(),
				Permissions: formatPermissions(info.Mode()),
			})
		}

		return nil
	})

	return results, err
}
