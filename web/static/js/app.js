// Main Application
const App = {
    currentPage: 'dashboard',

    init() {
        this.setupNavigation();
        this.setupTheme();
        this.initModules();
        
        // Connect WebSocket
        window.wsManager.connect();

        // Listen for metrics updates
        window.wsManager.on('metrics', (data) => {
            if (Dashboard && Dashboard.updateMetrics) {
                Dashboard.updateMetrics(data);
            }
        });
    },

    setupNavigation() {
        document.querySelectorAll('.navbar-menu a').forEach(link => {
            link.addEventListener('click', (e) => {
                e.preventDefault();
                const page = link.dataset.page;
                this.navigateTo(page);
            });
        });

        // Handle browser back/forward
        window.addEventListener('popstate', (e) => {
            if (e.state && e.state.page) {
                this.navigateTo(e.state.page, false);
            }
        });

        // Initial page from URL hash
        const hash = window.location.hash.slice(1);
        if (hash) {
            this.navigateTo(hash, false);
        }
    },

    navigateTo(page, pushState = true) {
        // Hide all pages
        document.querySelectorAll('.page').forEach(p => p.classList.remove('active'));

        // Show requested page
        const pageEl = document.getElementById(`page-${page}`);
        if (pageEl) {
            pageEl.classList.add('active');
            this.currentPage = page;

            // Update nav
            document.querySelectorAll('.navbar-menu a').forEach(a => {
                a.classList.toggle('active', a.dataset.page === page);
            });

            // Update URL
            if (pushState) {
                history.pushState({ page }, '', `#${page}`);
            }

            // Initialize page-specific content
            this.initPage(page);
        }
    },

    initPage(page) {
        switch (page) {
            case 'dashboard':
                // Dashboard auto-updates
                break;
            case 'processes':
                Processes.load();
                break;
            case 'services':
                Services.load();
                break;
            case 'files':
                Files.load(Files.currentPath);
                break;
            case 'packages':
                Packages.loadInstalled();
                break;
            case 'terminal':
                // Terminal is ready
                break;
        }
    },

    initModules() {
        Auth.init();
        Dashboard.init();
        Processes.init();
        Services.init();
        Files.init();
        Packages.init();
        TerminalManager.init();
    },

    setupTheme() {
        const toggle = document.getElementById('theme-toggle');
        const savedTheme = localStorage.getItem('theme') || 'dark';
        
        document.documentElement.setAttribute('data-theme', savedTheme);
        
        toggle?.addEventListener('click', () => {
            const current = document.documentElement.getAttribute('data-theme');
            const newTheme = current === 'dark' ? 'light' : 'dark';
            document.documentElement.setAttribute('data-theme', newTheme);
            localStorage.setItem('theme', newTheme);
        });
    },

    showModal(title, content, buttons = []) {
        const modal = document.getElementById('modal');
        const modalTitle = document.getElementById('modal-title');
        const modalBody = document.getElementById('modal-body');
        const modalFooter = document.getElementById('modal-footer');

        modalTitle.textContent = title;
        modalBody.innerHTML = content;
        
        modalFooter.innerHTML = buttons.map(btn => 
            `<button class="btn ${btn.class}" onclick="(${btn.action.toString()})()">${btn.text}</button>`
        ).join('');

        modal.classList.add('active');
        
        // Close on click outside
        modal.onclick = (e) => {
            if (e.target === modal) this.closeModal();
        };

        // Close button
        modal.querySelector('.modal-close').onclick = () => this.closeModal();
    },

    closeModal() {
        const modal = document.getElementById('modal');
        modal.classList.remove('active');
    },

    showToast(message, type = 'info') {
        const container = document.getElementById('toast-container');
        const toast = document.createElement('div');
        toast.className = `toast ${type}`;
        toast.textContent = message;
        container.appendChild(toast);

        setTimeout(() => {
            toast.style.opacity = '0';
            setTimeout(() => toast.remove(), 300);
        }, 3000);
    }
};

// Initialize when DOM is ready
document.addEventListener('DOMContentLoaded', () => App.init());
