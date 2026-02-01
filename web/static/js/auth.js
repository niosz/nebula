// Auth Module - Manages privileges and credentials
const Auth = {
    isElevated: false,
    hasCredentials: false,
    pendingCallback: null,

    async init() {
        await this.checkStatus();
    },

    async checkStatus() {
        try {
            const response = await fetch('/api/v1/auth/status');
            const status = await response.json();
            
            this.isElevated = status.is_elevated;
            this.hasCredentials = status.has_credentials;
            
            // Update UI indicators if needed
            this.updateStatusIndicator();
            
            return status;
        } catch (error) {
            console.error('Failed to check auth status:', error);
            return null;
        }
    },

    updateStatusIndicator() {
        // Could add a visual indicator showing privilege status
        const indicator = document.getElementById('privilege-status');
        if (indicator) {
            if (this.isElevated) {
                indicator.textContent = '🔓 Root';
                indicator.title = 'Running as root/admin';
            } else if (this.hasCredentials) {
                indicator.textContent = '🔑 Sudo';
                indicator.title = 'Credentials stored';
            } else {
                indicator.textContent = '🔒';
                indicator.title = 'No credentials';
            }
        }
    },

    // Request credentials from user
    requestCredentials(callback) {
        this.pendingCallback = callback;
        document.getElementById('credentials-modal').classList.add('active');
        document.getElementById('sudo-password').focus();
        
        // Handle Enter key
        document.getElementById('sudo-password').onkeypress = (e) => {
            if (e.key === 'Enter') {
                this.submitCredentials();
            }
        };
    },

    closeCredentialsModal() {
        document.getElementById('credentials-modal').classList.remove('active');
        document.getElementById('sudo-password').value = '';
        this.pendingCallback = null;
    },

    async submitCredentials() {
        const password = document.getElementById('sudo-password').value;
        const remember = document.getElementById('remember-credentials').checked;
        
        if (!password) {
            App.showToast('Password richiesta', 'error');
            return;
        }

        try {
            // Validate credentials
            const validateResponse = await fetch('/api/v1/auth/validate', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ password })
            });
            
            const validateResult = await validateResponse.json();
            
            if (!validateResult.valid) {
                App.showToast('Password non valida', 'error');
                return;
            }

            // Store credentials if remember is checked
            if (remember) {
                const storeResponse = await fetch('/api/v1/auth/credentials', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ password })
                });
                
                if (storeResponse.ok) {
                    this.hasCredentials = true;
                    App.showToast('Credenziali salvate', 'success');
                }
            }

            this.closeCredentialsModal();
            this.updateStatusIndicator();

            // Execute pending callback
            if (this.pendingCallback) {
                this.pendingCallback();
                this.pendingCallback = null;
            }
        } catch (error) {
            console.error('Failed to submit credentials:', error);
            App.showToast('Errore durante la verifica', 'error');
        }
    },

    async clearCredentials() {
        try {
            const response = await fetch('/api/v1/auth/credentials', { method: 'DELETE' });
            if (response.ok) {
                this.hasCredentials = false;
                this.updateStatusIndicator();
                App.showToast('Credenziali rimosse', 'success');
            }
        } catch (error) {
            console.error('Failed to clear credentials:', error);
        }
    },

    // Helper to run a privileged action
    async runPrivileged(action) {
        // If already elevated or has credentials, just run the action
        if (this.isElevated || this.hasCredentials) {
            return action();
        }

        // Otherwise, request credentials first
        return new Promise((resolve) => {
            this.requestCredentials(async () => {
                const result = await action();
                resolve(result);
            });
        });
    }
};

window.Auth = Auth;
