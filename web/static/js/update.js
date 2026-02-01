// Update Module - Manages application updates
const UpdateManager = {
    currentVersion: null,
    availableUpdate: null,
    lastCheck: null,

    init() {
        this.loadCurrentVersion();
        this.bindEvents();
        this.loadLastCheck();
    },

    bindEvents() {
        document.getElementById('btn-check-update')?.addEventListener('click', () => this.checkForUpdates());
        document.getElementById('btn-apply-update')?.addEventListener('click', () => this.applyUpdate());
    },

    async loadCurrentVersion() {
        try {
            const response = await fetch('/api/v1/version');
            const data = await response.json();
            
            this.currentVersion = data;
            
            document.getElementById('current-version').textContent = data.version || 'v0.0.0';
            document.getElementById('build-date').textContent = data.build_date || '-';
            document.getElementById('go-version').textContent = data.go_version || '-';
            
            // Set repo URL
            const repoUrl = data.repository || 'https://github.com/owner/nebula';
            document.getElementById('repo-url').href = repoUrl;
            document.getElementById('repo-url').textContent = repoUrl.replace('https://github.com/', '');
            
        } catch (error) {
            console.error('Failed to load version:', error);
            document.getElementById('current-version').textContent = 'Errore';
        }
    },

    loadLastCheck() {
        const lastCheck = localStorage.getItem('nebula_last_update_check');
        if (lastCheck) {
            this.lastCheck = new Date(lastCheck);
            document.getElementById('last-check').textContent = this.formatDate(this.lastCheck);
        }
    },

    async checkForUpdates() {
        const btn = document.getElementById('btn-check-update');
        const statusDiv = document.getElementById('update-status');
        const availableCard = document.getElementById('update-available-card');
        
        btn.disabled = true;
        btn.innerHTML = '<span class="btn-icon spinner">&#8635;</span> Controllo in corso...';
        statusDiv.innerHTML = '';
        availableCard.style.display = 'none';
        
        try {
            const response = await fetch('/api/v1/update/check');
            const data = await response.json();
            
            // Save last check time
            this.lastCheck = new Date();
            localStorage.setItem('nebula_last_update_check', this.lastCheck.toISOString());
            document.getElementById('last-check').textContent = this.formatDate(this.lastCheck);
            
            if (data.available) {
                this.availableUpdate = data;
                this.showAvailableUpdate(data);
                statusDiv.innerHTML = '<span class="status-success">&#10003; Nuova versione trovata!</span>';
            } else {
                statusDiv.innerHTML = '<span class="status-info">&#10003; Nebula è aggiornato all\'ultima versione.</span>';
            }
            
        } catch (error) {
            console.error('Failed to check for updates:', error);
            statusDiv.innerHTML = '<span class="status-error">&#10007; Errore durante il controllo: ' + error.message + '</span>';
        } finally {
            btn.disabled = false;
            btn.innerHTML = '<span class="btn-icon">&#128269;</span> Verifica Aggiornamenti';
        }
    },

    showAvailableUpdate(data) {
        const card = document.getElementById('update-available-card');
        
        document.getElementById('new-version').textContent = data.latest_version || data.version || '-';
        document.getElementById('new-version-date').textContent = data.published_at ? 
            'Rilasciato il ' + this.formatDate(new Date(data.published_at)) : '';
        
        // Show changelog/release notes
        const changelogDiv = document.getElementById('changelog');
        if (data.release_notes || data.body) {
            changelogDiv.innerHTML = '<h4>Note di Rilascio:</h4><pre>' + 
                this.escapeHtml(data.release_notes || data.body || '') + '</pre>';
        } else {
            changelogDiv.innerHTML = '';
        }
        
        // Set release link
        const releaseLink = document.getElementById('release-link');
        if (data.html_url || data.release_url) {
            releaseLink.href = data.html_url || data.release_url;
            releaseLink.style.display = 'inline-block';
        } else {
            releaseLink.style.display = 'none';
        }
        
        card.style.display = 'block';
    },

    async applyUpdate() {
        if (!this.availableUpdate) {
            App.showToast('Nessun aggiornamento disponibile', 'error');
            return;
        }
        
        const confirmed = await App.confirm(
            'Conferma Aggiornamento',
            `Vuoi installare la versione ${this.availableUpdate.latest_version || this.availableUpdate.version}?\n\nIl server verrà riavviato automaticamente.`
        );
        
        if (!confirmed) return;
        
        const progressCard = document.getElementById('update-progress-card');
        const availableCard = document.getElementById('update-available-card');
        const progressFill = document.getElementById('update-progress');
        const progressText = document.getElementById('update-progress-text');
        
        availableCard.style.display = 'none';
        progressCard.style.display = 'block';
        
        // Simulate progress
        let progress = 0;
        const progressInterval = setInterval(() => {
            if (progress < 90) {
                progress += Math.random() * 10;
                progressFill.style.width = Math.min(progress, 90) + '%';
                
                if (progress < 30) {
                    progressText.textContent = 'Download in corso...';
                } else if (progress < 60) {
                    progressText.textContent = 'Verifica integrità...';
                } else {
                    progressText.textContent = 'Applicazione aggiornamento...';
                }
            }
        }, 500);
        
        try {
            const response = await fetch('/api/v1/update/apply', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ version: this.availableUpdate.latest_version })
            });
            
            clearInterval(progressInterval);
            
            if (response.ok) {
                progressFill.style.width = '100%';
                progressText.textContent = 'Aggiornamento completato! Riavvio in corso...';
                App.showToast('Aggiornamento completato! Il server si sta riavviando...', 'success');
                
                // Try to reconnect after a delay
                setTimeout(() => {
                    this.waitForRestart();
                }, 3000);
            } else {
                const error = await response.json();
                throw new Error(error.error || 'Aggiornamento fallito');
            }
            
        } catch (error) {
            clearInterval(progressInterval);
            progressCard.style.display = 'none';
            availableCard.style.display = 'block';
            console.error('Failed to apply update:', error);
            App.showToast('Errore durante l\'aggiornamento: ' + error.message, 'error');
        }
    },

    async waitForRestart() {
        const progressText = document.getElementById('update-progress-text');
        let attempts = 0;
        const maxAttempts = 30;
        
        const checkServer = async () => {
            attempts++;
            progressText.textContent = `Attesa riavvio server... (${attempts}/${maxAttempts})`;
            
            try {
                const response = await fetch('/api/v1/version', { 
                    cache: 'no-store',
                    signal: AbortSignal.timeout(2000)
                });
                
                if (response.ok) {
                    // Server is back!
                    progressText.textContent = 'Server riavviato! Ricaricamento pagina...';
                    setTimeout(() => window.location.reload(), 1000);
                    return;
                }
            } catch (e) {
                // Server still down
            }
            
            if (attempts < maxAttempts) {
                setTimeout(checkServer, 2000);
            } else {
                progressText.textContent = 'Timeout attesa server. Ricarica la pagina manualmente.';
            }
        };
        
        checkServer();
    },

    formatDate(date) {
        return date.toLocaleDateString('it-IT', {
            day: '2-digit',
            month: '2-digit',
            year: 'numeric',
            hour: '2-digit',
            minute: '2-digit'
        });
    },

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }
};

window.UpdateManager = UpdateManager;
