// Packages Module
const Packages = {
    installedPackages: [],
    searchResults: [],
    activeTab: 'installed',

    init() {
        this.setupEventListeners();
        this.loadInstalled();
    },

    setupEventListeners() {
        document.getElementById('btn-search-packages')?.addEventListener('click', () => {
            const query = document.getElementById('package-search').value;
            if (query) this.search(query);
        });

        document.getElementById('package-search')?.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') {
                const query = e.target.value;
                if (query) this.search(query);
            }
        });

        document.getElementById('btn-upgrade-all')?.addEventListener('click', () => {
            this.upgradeAll();
        });

        document.querySelectorAll('.tab').forEach(tab => {
            tab.addEventListener('click', (e) => {
                document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
                e.target.classList.add('active');
                this.activeTab = e.target.dataset.tab;
                this.render();
            });
        });
    },

    async loadInstalled() {
        try {
            const response = await fetch('/api/v1/packages');
            this.installedPackages = await response.json() || [];
            this.render();
        } catch (error) {
            console.error('Failed to load packages:', error);
            App.showToast('Failed to load packages', 'error');
        }
    },

    async search(query) {
        try {
            const response = await fetch(`/api/v1/packages/search?q=${encodeURIComponent(query)}`);
            this.searchResults = await response.json() || [];
            this.activeTab = 'search-results';
            
            document.querySelectorAll('.tab').forEach(t => {
                t.classList.toggle('active', t.dataset.tab === 'search-results');
            });
            
            this.render();
        } catch (error) {
            console.error('Failed to search packages:', error);
            App.showToast('Search failed', 'error');
        }
    },

    render() {
        const tbody = document.getElementById('package-list');
        if (!tbody) return;

        const packages = this.activeTab === 'installed' ? this.installedPackages : this.searchResults;

        if (!packages || packages.length === 0) {
            tbody.innerHTML = '<tr><td colspan="4">No packages found</td></tr>';
            return;
        }

        tbody.innerHTML = packages.map(pkg => `
            <tr>
                <td>${this.escapeHtml(pkg.name)}</td>
                <td>${this.escapeHtml(pkg.version || 'N/A')}</td>
                <td>${this.escapeHtml(pkg.description || '')}</td>
                <td>
                    ${pkg.installed ?
                        `<button class="btn btn-sm btn-danger" onclick="Packages.remove('${this.escapeAttr(pkg.name)}')">Remove</button>
                         <button class="btn btn-sm" onclick="Packages.update('${this.escapeAttr(pkg.name)}')">Update</button>` :
                        `<button class="btn btn-sm btn-success" onclick="Packages.install('${this.escapeAttr(pkg.name)}')">Install</button>`
                    }
                </td>
            </tr>
        `).join('');
    },

    async install(name) {
        App.showToast(`Installing ${name}...`, 'info');
        try {
            const response = await fetch('/api/v1/packages/install', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ name })
            });

            if (response.ok) {
                App.showToast(`${name} installed`, 'success');
                this.loadInstalled();
            } else {
                const data = await response.json();
                App.showToast(data.error || 'Installation failed', 'error');
            }
        } catch (error) {
            App.showToast('Installation failed', 'error');
        }
    },

    async remove(name) {
        if (!confirm(`Remove ${name}?`)) return;

        try {
            const response = await fetch(`/api/v1/packages/remove?name=${encodeURIComponent(name)}`, {
                method: 'DELETE'
            });

            if (response.ok) {
                App.showToast(`${name} removed`, 'success');
                this.loadInstalled();
            } else {
                const data = await response.json();
                App.showToast(data.error || 'Removal failed', 'error');
            }
        } catch (error) {
            App.showToast('Removal failed', 'error');
        }
    },

    async update(name) {
        App.showToast(`Updating ${name}...`, 'info');
        try {
            const response = await fetch('/api/v1/packages/update', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ name })
            });

            if (response.ok) {
                App.showToast(`${name} updated`, 'success');
                this.loadInstalled();
            } else {
                const data = await response.json();
                App.showToast(data.error || 'Update failed', 'error');
            }
        } catch (error) {
            App.showToast('Update failed', 'error');
        }
    },

    async upgradeAll() {
        if (!confirm('Upgrade all packages?')) return;

        App.showToast('Upgrading all packages...', 'info');
        try {
            const response = await fetch('/api/v1/packages/upgrade-all', { method: 'POST' });

            if (response.ok) {
                App.showToast('All packages upgraded', 'success');
                this.loadInstalled();
            } else {
                const data = await response.json();
                App.showToast(data.error || 'Upgrade failed', 'error');
            }
        } catch (error) {
            App.showToast('Upgrade failed', 'error');
        }
    },

    escapeHtml(text) {
        if (!text) return '';
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    },

    escapeAttr(text) {
        return text.replace(/'/g, "\\'").replace(/"/g, '\\"');
    }
};

window.Packages = Packages;
