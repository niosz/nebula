// Files Module
const Files = {
    currentPath: '/',
    files: [],

    init() {
        this.setupEventListeners();
        this.load('/');
    },

    setupEventListeners() {
        document.getElementById('btn-upload')?.addEventListener('click', () => {
            document.getElementById('file-upload-input').click();
        });

        document.getElementById('file-upload-input')?.addEventListener('change', (e) => {
            this.uploadFiles(e.target.files);
        });

        document.getElementById('btn-new-folder')?.addEventListener('click', () => {
            this.promptNewFolder();
        });

        // Drag and drop
        const container = document.getElementById('page-files');
        if (container) {
            container.addEventListener('dragover', (e) => {
                e.preventDefault();
                container.classList.add('dragover');
            });
            container.addEventListener('dragleave', () => {
                container.classList.remove('dragover');
            });
            container.addEventListener('drop', (e) => {
                e.preventDefault();
                container.classList.remove('dragover');
                this.uploadFiles(e.dataTransfer.files);
            });
        }
    },

    async load(path) {
        try {
            const response = await fetch(`/api/v1/files/list?path=${encodeURIComponent(path)}`);
            this.files = await response.json();
            this.currentPath = path;
            this.render();
        } catch (error) {
            console.error('Failed to load files:', error);
            App.showToast('Failed to load files', 'error');
        }
    },

    render() {
        this.renderBreadcrumb();
        this.renderFiles();
    },

    renderBreadcrumb() {
        const breadcrumb = document.getElementById('file-breadcrumb');
        if (!breadcrumb) return;

        const parts = this.currentPath.split('/').filter(p => p);
        let path = '';

        breadcrumb.innerHTML = `
            <span class="breadcrumb-item" onclick="Files.load('/')">Root</span>
            ${parts.map(part => {
                path += '/' + part;
                const fullPath = path;
                return `<span class="breadcrumb-item" onclick="Files.load('${fullPath}')">${this.escapeHtml(part)}</span>`;
            }).join('')}
        `;
    },

    renderFiles() {
        const list = document.getElementById('file-list');
        if (!list) return;

        if (this.files.length === 0) {
            list.innerHTML = '<div class="file-item"><span>Empty directory</span></div>';
            return;
        }

        list.innerHTML = this.files.map(file => {
            const icon = file.is_dir ? '📁' : this.getFileIcon(file.extension);
            const size = file.is_dir ? '' : this.formatSize(file.size);

            return `
                <div class="file-item" ondblclick="Files.openItem('${this.escapeAttr(file.path)}', ${file.is_dir})">
                    <span class="file-icon">${icon}</span>
                    <div class="file-info">
                        <div class="file-name">${this.escapeHtml(file.name)}</div>
                        <div class="file-size">${size}</div>
                    </div>
                    <div class="file-item-actions">
                        ${!file.is_dir ? `<button class="btn btn-sm" onclick="event.stopPropagation(); Files.download('${this.escapeAttr(file.path)}')">↓</button>` : ''}
                        <button class="btn btn-sm btn-danger" onclick="event.stopPropagation(); Files.delete('${this.escapeAttr(file.path)}')">×</button>
                    </div>
                </div>
            `;
        }).join('');
    },

    openItem(path, isDir) {
        if (isDir) {
            this.load(path);
        } else {
            this.viewFile(path);
        }
    },

    async viewFile(path) {
        try {
            const response = await fetch(`/api/v1/files/read?path=${encodeURIComponent(path)}`);
            const data = await response.json();

            const content = `
                <pre style="max-height: 400px; overflow: auto; background: var(--bg-primary); padding: 1rem; border-radius: 0.5rem; white-space: pre-wrap; word-wrap: break-word;">${this.escapeHtml(data.content)}</pre>
            `;

            App.showModal(`File: ${path.split('/').pop()}`, content, [
                { text: 'Download', class: 'btn-primary', action: () => this.download(path) },
                { text: 'Close', class: '', action: () => App.closeModal() }
            ]);
        } catch (error) {
            App.showToast('Failed to read file', 'error');
        }
    },

    download(path) {
        window.open(`/api/v1/files/download?path=${encodeURIComponent(path)}`, '_blank');
    },

    async uploadFiles(files) {
        for (const file of files) {
            const formData = new FormData();
            formData.append('file', file);

            try {
                const response = await fetch(`/api/v1/files/upload?path=${encodeURIComponent(this.currentPath)}`, {
                    method: 'POST',
                    body: formData
                });

                if (response.ok) {
                    App.showToast(`Uploaded: ${file.name}`, 'success');
                } else {
                    const data = await response.json();
                    App.showToast(data.error || 'Upload failed', 'error');
                }
            } catch (error) {
                App.showToast('Upload failed', 'error');
            }
        }
        this.load(this.currentPath);
    },

    promptNewFolder() {
        const content = `
            <input type="text" id="new-folder-name" placeholder="Folder name" style="width: 100%;">
        `;

        App.showModal('New Folder', content, [
            {
                text: 'Create',
                class: 'btn-primary',
                action: async () => {
                    const name = document.getElementById('new-folder-name').value;
                    if (name) {
                        await this.createFolder(name);
                        App.closeModal();
                    }
                }
            },
            { text: 'Cancel', class: '', action: () => App.closeModal() }
        ]);
    },

    async createFolder(name) {
        const path = this.currentPath === '/' ? `/${name}` : `${this.currentPath}/${name}`;

        try {
            const response = await fetch('/api/v1/files/mkdir', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ path })
            });

            if (response.ok) {
                App.showToast('Folder created', 'success');
                this.load(this.currentPath);
            } else {
                const data = await response.json();
                App.showToast(data.error || 'Failed to create folder', 'error');
            }
        } catch (error) {
            App.showToast('Failed to create folder', 'error');
        }
    },

    async delete(path) {
        if (!confirm(`Delete ${path}?`)) return;

        try {
            const response = await fetch(`/api/v1/files/delete?path=${encodeURIComponent(path)}`, {
                method: 'DELETE'
            });

            if (response.ok) {
                App.showToast('Deleted', 'success');
                this.load(this.currentPath);
            } else {
                const data = await response.json();
                App.showToast(data.error || 'Failed to delete', 'error');
            }
        } catch (error) {
            App.showToast('Failed to delete', 'error');
        }
    },

    getFileIcon(ext) {
        const icons = {
            'js': '📜', 'ts': '📜', 'py': '🐍', 'go': '🔷',
            'html': '🌐', 'css': '🎨', 'json': '📋', 'md': '📝',
            'txt': '📄', 'pdf': '📕', 'zip': '📦', 'tar': '📦',
            'jpg': '🖼️', 'jpeg': '🖼️', 'png': '🖼️', 'gif': '🖼️',
            'mp3': '🎵', 'mp4': '🎬', 'sh': '⚡', 'yml': '⚙️', 'yaml': '⚙️'
        };
        return icons[ext?.toLowerCase()] || '📄';
    },

    formatSize(bytes) {
        if (bytes === 0) return '0 B';
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
    },

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    },

    escapeAttr(text) {
        return text.replace(/'/g, "\\'").replace(/"/g, '\\"');
    }
};

window.Files = Files;
