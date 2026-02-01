// Dashboard Module
const Dashboard = {
    cpuChart: null,
    memoryChart: null,
    networkChart: null,
    cpuHistory: [],
    memoryHistory: [],
    networkHistory: { sent: [], recv: [] },
    lastNetwork: null,

    init() {
        this.initCharts();
        this.loadSystemInfo();
        this.startPolling();
    },

    initCharts() {
        const chartOptions = {
            responsive: true,
            maintainAspectRatio: false,
            animation: { duration: 0 },
            scales: {
                y: {
                    beginAtZero: true,
                    max: 100,
                    grid: { color: 'rgba(255,255,255,0.1)' },
                    ticks: { color: '#94a3b8' }
                },
                x: {
                    display: false
                }
            },
            plugins: {
                legend: { display: false }
            }
        };

        // CPU Chart
        const cpuCtx = document.getElementById('cpu-chart');
        if (cpuCtx) {
            this.cpuChart = new Chart(cpuCtx, {
                type: 'line',
                data: {
                    labels: Array(60).fill(''),
                    datasets: [{
                        data: [],
                        borderColor: '#3b82f6',
                        backgroundColor: 'rgba(59, 130, 246, 0.1)',
                        fill: true,
                        tension: 0.4
                    }]
                },
                options: chartOptions
            });
        }

        // Memory Chart
        const memCtx = document.getElementById('memory-chart');
        if (memCtx) {
            this.memoryChart = new Chart(memCtx, {
                type: 'line',
                data: {
                    labels: Array(60).fill(''),
                    datasets: [{
                        data: [],
                        borderColor: '#22c55e',
                        backgroundColor: 'rgba(34, 197, 94, 0.1)',
                        fill: true,
                        tension: 0.4
                    }]
                },
                options: chartOptions
            });
        }

        // Network Chart
        const netCtx = document.getElementById('network-chart');
        if (netCtx) {
            this.networkChart = new Chart(netCtx, {
                type: 'line',
                data: {
                    labels: Array(60).fill(''),
                    datasets: [
                        {
                            label: 'Sent',
                            data: [],
                            borderColor: '#f59e0b',
                            backgroundColor: 'rgba(245, 158, 11, 0.1)',
                            fill: false,
                            tension: 0.4
                        },
                        {
                            label: 'Received',
                            data: [],
                            borderColor: '#8b5cf6',
                            backgroundColor: 'rgba(139, 92, 246, 0.1)',
                            fill: false,
                            tension: 0.4
                        }
                    ]
                },
                options: {
                    ...chartOptions,
                    scales: {
                        ...chartOptions.scales,
                        y: {
                            ...chartOptions.scales.y,
                            max: undefined
                        }
                    },
                    plugins: {
                        legend: {
                            display: true,
                            labels: { color: '#94a3b8' }
                        }
                    }
                }
            });
        }
    },

    async loadSystemInfo() {
        try {
            const response = await fetch('/api/v1/system/info');
            const info = await response.json();

            document.getElementById('hostname').textContent = info.hostname;
            document.getElementById('os-info').textContent = `${info.platform} ${info.platform_version}`;
            document.getElementById('uptime').textContent = this.formatUptime(info.uptime);
        } catch (error) {
            console.error('Failed to load system info:', error);
        }
    },

    async loadMetrics() {
        try {
            const response = await fetch('/api/v1/metrics/all');
            const metrics = await response.json();
            this.updateMetrics(metrics);
        } catch (error) {
            console.error('Failed to load metrics:', error);
        }
    },

    updateMetrics(metrics) {
        // Update CPU
        if (metrics.cpu) {
            const cpuTotal = metrics.cpu.total_percent.toFixed(1);
            document.getElementById('cpu-total').textContent = `${cpuTotal}%`;

            this.cpuHistory.push(parseFloat(cpuTotal));
            if (this.cpuHistory.length > 60) this.cpuHistory.shift();

            if (this.cpuChart) {
                this.cpuChart.data.datasets[0].data = [...this.cpuHistory];
                this.cpuChart.update();
            }

            // CPU cores
            const coresContainer = document.getElementById('cpu-cores');
            if (coresContainer && metrics.cpu.usage_percent) {
                coresContainer.innerHTML = metrics.cpu.usage_percent.map((usage, i) =>
                    `<div class="detail-row"><span>Core ${i}</span><span>${usage.toFixed(1)}%</span></div>`
                ).join('');
            }
        }

        // Update Memory
        if (metrics.memory) {
            const memPercent = metrics.memory.used_percent.toFixed(1);
            document.getElementById('memory-total').textContent = `${memPercent}%`;
            document.getElementById('memory-used').textContent = this.formatBytes(metrics.memory.used);
            document.getElementById('memory-free').textContent = this.formatBytes(metrics.memory.free);
            document.getElementById('memory-swap').textContent = this.formatBytes(metrics.memory.swap_used);

            this.memoryHistory.push(parseFloat(memPercent));
            if (this.memoryHistory.length > 60) this.memoryHistory.shift();

            if (this.memoryChart) {
                this.memoryChart.data.datasets[0].data = [...this.memoryHistory];
                this.memoryChart.update();
            }
        }

        // Update Disks
        if (metrics.disks) {
            const diskList = document.getElementById('disk-list');
            if (diskList) {
                diskList.innerHTML = metrics.disks.map(disk => {
                    const percent = disk.used_percent;
                    let barClass = '';
                    if (percent > 90) barClass = 'danger';
                    else if (percent > 70) barClass = 'warning';

                    return `
                        <div class="disk-item">
                            <div class="disk-header">
                                <span>${disk.mountpoint}</span>
                                <span>${this.formatBytes(disk.used)} / ${this.formatBytes(disk.total)}</span>
                            </div>
                            <div class="disk-bar">
                                <div class="disk-bar-fill ${barClass}" style="width: ${percent}%"></div>
                            </div>
                        </div>
                    `;
                }).join('');
            }
        }

        // Update Network
        if (metrics.network) {
            const networkList = document.getElementById('network-list');
            if (networkList) {
                let totalSent = 0, totalRecv = 0;

                networkList.innerHTML = metrics.network.map(net => {
                    totalSent += net.bytes_sent;
                    totalRecv += net.bytes_recv;
                    return `
                        <div class="network-item">
                            <span>${net.name}</span>
                            <span>↑ ${this.formatBytes(net.bytes_sent)} ↓ ${this.formatBytes(net.bytes_recv)}</span>
                        </div>
                    `;
                }).join('');

                // Calculate rate
                if (this.lastNetwork) {
                    const sentRate = (totalSent - this.lastNetwork.sent) / 1; // per second
                    const recvRate = (totalRecv - this.lastNetwork.recv) / 1;

                    this.networkHistory.sent.push(sentRate);
                    this.networkHistory.recv.push(recvRate);

                    if (this.networkHistory.sent.length > 60) {
                        this.networkHistory.sent.shift();
                        this.networkHistory.recv.shift();
                    }

                    if (this.networkChart) {
                        this.networkChart.data.datasets[0].data = [...this.networkHistory.sent];
                        this.networkChart.data.datasets[1].data = [...this.networkHistory.recv];
                        this.networkChart.update();
                    }
                }

                this.lastNetwork = { sent: totalSent, recv: totalRecv };
            }
        }
    },

    startPolling() {
        this.loadMetrics();
        setInterval(() => this.loadMetrics(), 1000);
    },

    formatBytes(bytes) {
        if (bytes === 0) return '0 B';
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
    },

    formatUptime(seconds) {
        const days = Math.floor(seconds / 86400);
        const hours = Math.floor((seconds % 86400) / 3600);
        const minutes = Math.floor((seconds % 3600) / 60);
        return `${days}d ${hours}h ${minutes}m`;
    }
};

window.Dashboard = Dashboard;
