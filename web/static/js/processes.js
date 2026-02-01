// Processes Module
const Processes = {
    processes: [],
    sortColumn: 'cpu',
    sortAsc: false,
    searchQuery: '',

    init() {
        this.setupEventListeners();
        this.load();
    },

    setupEventListeners() {
        const searchInput = document.getElementById('process-search');
        if (searchInput) {
            searchInput.addEventListener('input', (e) => {
                this.searchQuery = e.target.value.toLowerCase();
                this.render();
            });
        }

        const table = document.getElementById('process-table');
        if (table) {
            table.querySelectorAll('th[data-sort]').forEach(th => {
                th.addEventListener('click', () => {
                    const column = th.dataset.sort;
                    if (this.sortColumn === column) {
                        this.sortAsc = !this.sortAsc;
                    } else {
                        this.sortColumn = column;
                        this.sortAsc = false;
                    }
                    this.render();
                });
            });
        }
    },

    async load() {
        try {
            const response = await fetch('/api/v1/processes');
            this.processes = await response.json();
            this.render();
        } catch (error) {
            console.error('Failed to load processes:', error);
            App.showToast('Failed to load processes', 'error');
        }
    },

    render() {
        const tbody = document.getElementById('process-list');
        if (!tbody) return;

        let filtered = this.processes;

        // Filter
        if (this.searchQuery) {
            filtered = filtered.filter(p =>
                p.name.toLowerCase().includes(this.searchQuery) ||
                p.username.toLowerCase().includes(this.searchQuery)
            );
        }

        // Sort
        filtered.sort((a, b) => {
            let aVal, bVal;
            switch (this.sortColumn) {
                case 'pid': aVal = a.pid; bVal = b.pid; break;
                case 'name': aVal = a.name.toLowerCase(); bVal = b.name.toLowerCase(); break;
                case 'cpu': aVal = a.cpu_percent; bVal = b.cpu_percent; break;
                case 'memory': aVal = a.mem_percent; bVal = b.mem_percent; break;
                case 'user': aVal = a.username.toLowerCase(); bVal = b.username.toLowerCase(); break;
                default: return 0;
            }

            if (aVal < bVal) return this.sortAsc ? -1 : 1;
            if (aVal > bVal) return this.sortAsc ? 1 : -1;
            return 0;
        });

        tbody.innerHTML = filtered.slice(0, 100).map(p => `
            <tr>
                <td>${p.pid}</td>
                <td>${this.escapeHtml(p.name)}</td>
                <td>${p.cpu_percent.toFixed(1)}%</td>
                <td>${p.mem_percent.toFixed(1)}%</td>
                <td>${this.escapeHtml(p.username)}</td>
                <td>
                    <button class="btn btn-sm btn-danger" onclick="Processes.kill(${p.pid})">Kill</button>
                    <button class="btn btn-sm" onclick="Processes.showDetails(${p.pid})">Details</button>
                </td>
            </tr>
        `).join('');
    },

    async kill(pid) {
        if (!confirm(`Kill process ${pid}?`)) return;

        try {
            const response = await fetch(`/api/v1/processes/${pid}/kill`, { method: 'POST' });
            if (response.ok) {
                App.showToast('Process terminated', 'success');
                this.load();
            } else {
                const data = await response.json();
                App.showToast(data.error || 'Failed to kill process', 'error');
            }
        } catch (error) {
            console.error('Failed to kill process:', error);
            App.showToast('Failed to kill process', 'error');
        }
    },

    async showDetails(pid) {
        try {
            const response = await fetch(`/api/v1/processes/${pid}`);
            const process = await response.json();

            const content = `
                <div class="detail-row"><span>PID:</span><span>${process.pid}</span></div>
                <div class="detail-row"><span>Name:</span><span>${this.escapeHtml(process.name)}</span></div>
                <div class="detail-row"><span>Status:</span><span>${process.status}</span></div>
                <div class="detail-row"><span>User:</span><span>${this.escapeHtml(process.username)}</span></div>
                <div class="detail-row"><span>CPU:</span><span>${process.cpu_percent.toFixed(2)}%</span></div>
                <div class="detail-row"><span>Memory:</span><span>${process.mem_percent.toFixed(2)}%</span></div>
                <div class="detail-row"><span>Threads:</span><span>${process.num_threads}</span></div>
                <div class="detail-row"><span>Created:</span><span>${new Date(process.create_time).toLocaleString()}</span></div>
                <div class="detail-row"><span>Command:</span><span style="word-break: break-all;">${this.escapeHtml(process.cmdline || 'N/A')}</span></div>
            `;

            App.showModal('Process Details', content, [
                { text: 'Kill', class: 'btn-danger', action: () => { App.closeModal(); this.kill(pid); } },
                { text: 'Close', class: '', action: () => App.closeModal() }
            ]);
        } catch (error) {
            console.error('Failed to load process details:', error);
            App.showToast('Failed to load details', 'error');
        }
    },

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
};

window.Processes = Processes;
