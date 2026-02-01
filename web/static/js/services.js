// Services Module
const Services = {
    services: [],
    searchQuery: '',

    init() {
        this.setupEventListeners();
        this.load();
    },

    setupEventListeners() {
        const searchInput = document.getElementById('service-search');
        if (searchInput) {
            searchInput.addEventListener('input', (e) => {
                this.searchQuery = e.target.value.toLowerCase();
                this.render();
            });
        }
    },

    async load() {
        try {
            const response = await fetch('/api/v1/services');
            this.services = await response.json();
            this.render();
        } catch (error) {
            console.error('Failed to load services:', error);
            App.showToast('Failed to load services', 'error');
        }
    },

    render() {
        const tbody = document.getElementById('service-list');
        if (!tbody) return;

        let filtered = this.services;

        if (this.searchQuery) {
            filtered = filtered.filter(s =>
                s.name.toLowerCase().includes(this.searchQuery)
            );
        }

        tbody.innerHTML = filtered.map(s => {
            const statusClass = s.status === 'running' ? 'badge-success' :
                               s.status === 'failed' ? 'badge-danger' : 'badge-warning';

            return `
                <tr>
                    <td>${this.escapeHtml(s.name)}</td>
                    <td><span class="badge ${statusClass}">${s.status}</span></td>
                    <td>${s.start_type || 'N/A'}</td>
                    <td>
                        ${s.status === 'running' ? 
                            `<button class="btn btn-sm" onclick="Services.stop('${s.name}')">Stop</button>
                             <button class="btn btn-sm" onclick="Services.restart('${s.name}')">Restart</button>` :
                            `<button class="btn btn-sm btn-success" onclick="Services.start('${s.name}')">Start</button>`
                        }
                        <button class="btn btn-sm" onclick="Services.showLogs('${s.name}')">Logs</button>
                    </td>
                </tr>
            `;
        }).join('');
    },

    async start(name) {
        try {
            const response = await fetch(`/api/v1/services/${name}/start`, { method: 'POST' });
            if (response.ok) {
                App.showToast(`Service ${name} started`, 'success');
                this.load();
            } else {
                const data = await response.json();
                App.showToast(data.error || 'Failed to start service', 'error');
            }
        } catch (error) {
            App.showToast('Failed to start service', 'error');
        }
    },

    async stop(name) {
        try {
            const response = await fetch(`/api/v1/services/${name}/stop`, { method: 'POST' });
            if (response.ok) {
                App.showToast(`Service ${name} stopped`, 'success');
                this.load();
            } else {
                const data = await response.json();
                App.showToast(data.error || 'Failed to stop service', 'error');
            }
        } catch (error) {
            App.showToast('Failed to stop service', 'error');
        }
    },

    async restart(name) {
        try {
            const response = await fetch(`/api/v1/services/${name}/restart`, { method: 'POST' });
            if (response.ok) {
                App.showToast(`Service ${name} restarted`, 'success');
                this.load();
            } else {
                const data = await response.json();
                App.showToast(data.error || 'Failed to restart service', 'error');
            }
        } catch (error) {
            App.showToast('Failed to restart service', 'error');
        }
    },

    async showLogs(name) {
        try {
            const response = await fetch(`/api/v1/services/${name}/logs?lines=50`);
            const logs = await response.json();

            const content = `
                <div style="max-height: 400px; overflow-y: auto; font-family: monospace; font-size: 12px; background: var(--bg-primary); padding: 1rem; border-radius: 0.5rem;">
                    ${logs.map(log => `<div>${this.escapeHtml(log.message)}</div>`).join('')}
                </div>
            `;

            App.showModal(`Logs: ${name}`, content, [
                { text: 'Close', class: '', action: () => App.closeModal() }
            ]);
        } catch (error) {
            App.showToast('Failed to load logs', 'error');
        }
    },

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
};

window.Services = Services;
