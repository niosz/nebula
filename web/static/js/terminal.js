// Terminal Module
const TerminalManager = {
    terminals: new Map(),
    activeTerminal: null,
    shells: [],
    terminalCounter: 0,

    init() {
        this.loadShells();
        this.setupEventListeners();
    },

    setupEventListeners() {
        document.getElementById('btn-new-terminal')?.addEventListener('click', () => {
            const shell = document.getElementById('shell-select').value;
            this.createTerminal(shell);
        });
    },

    async loadShells() {
        try {
            const response = await fetch('/api/v1/terminal/shells');
            const data = await response.json();
            this.shells = data.shells || [];

            const select = document.getElementById('shell-select');
            if (select) {
                select.innerHTML = this.shells.map(shell => 
                    `<option value="${shell}">${shell.split('/').pop()}</option>`
                ).join('');

                if (data.default_shell) {
                    select.value = data.default_shell;
                }
            }
        } catch (error) {
            console.error('Failed to load shells:', error);
        }
    },

    createTerminal(shell) {
        const id = `term-${++this.terminalCounter}`;

        // Create tab
        const tabsContainer = document.getElementById('terminal-tabs');
        const tab = document.createElement('div');
        tab.className = 'terminal-tab';
        tab.dataset.id = id;
        tab.innerHTML = `
            <span>${shell ? shell.split('/').pop() : 'Terminal'} #${this.terminalCounter}</span>
            <span class="terminal-tab-close" onclick="event.stopPropagation(); TerminalManager.closeTerminal('${id}')">×</span>
        `;
        tab.onclick = () => this.activateTerminal(id);
        tabsContainer.appendChild(tab);

        // Create terminal container
        const container = document.getElementById('terminal-container');
        const termDiv = document.createElement('div');
        termDiv.id = id;
        termDiv.style.height = '100%';
        termDiv.style.display = 'none';
        container.appendChild(termDiv);

        // Initialize xterm.js with MesloLGS NF font
        const term = new Terminal({
            cursorBlink: true,
            fontSize: 14,
            fontFamily: '"MesloLGS NF", Menlo, Monaco, "Courier New", monospace',
            fontWeight: 'normal',
            fontWeightBold: 'bold',
            letterSpacing: 0,
            lineHeight: 1.1,
            theme: {
                background: '#1e1e2e',
                foreground: '#cdd6f4',
                cursor: '#f5e0dc',
                cursorAccent: '#1e1e2e',
                selectionBackground: '#585b70',
                selectionForeground: '#cdd6f4',
                black: '#45475a',
                red: '#f38ba8',
                green: '#a6e3a1',
                yellow: '#f9e2af',
                blue: '#89b4fa',
                magenta: '#f5c2e7',
                cyan: '#94e2d5',
                white: '#bac2de',
                brightBlack: '#585b70',
                brightRed: '#f38ba8',
                brightGreen: '#a6e3a1',
                brightYellow: '#f9e2af',
                brightBlue: '#89b4fa',
                brightMagenta: '#f5c2e7',
                brightCyan: '#94e2d5',
                brightWhite: '#a6adc8'
            }
        });

        const fitAddon = new FitAddon.FitAddon();
        term.loadAddon(fitAddon);
        term.open(termDiv);
        fitAddon.fit();

        // Connect WebSocket
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/ws/terminal?session=${id}&shell=${encodeURIComponent(shell)}`;
        const ws = new WebSocket(wsUrl);

        ws.binaryType = 'arraybuffer';

        ws.onopen = () => {
            console.log(`Terminal ${id} connected`);
            
            // Send initial size
            ws.send(JSON.stringify({
                type: 'resize',
                cols: term.cols,
                rows: term.rows
            }));
        };

        ws.onmessage = (event) => {
            if (event.data instanceof ArrayBuffer) {
                term.write(new Uint8Array(event.data));
            } else {
                term.write(event.data);
            }
        };

        ws.onclose = () => {
            console.log(`Terminal ${id} disconnected`);
            term.write('\r\n\x1b[31m[Connection closed]\x1b[0m\r\n');
        };

        ws.onerror = (error) => {
            console.error(`Terminal ${id} error:`, error);
        };

        // Handle input
        term.onData((data) => {
            if (ws.readyState === WebSocket.OPEN) {
                ws.send(data);
            }
        });

        // Handle resize
        term.onResize(({ cols, rows }) => {
            if (ws.readyState === WebSocket.OPEN) {
                ws.send(JSON.stringify({
                    type: 'resize',
                    cols,
                    rows
                }));
            }
        });

        // Store terminal
        this.terminals.set(id, {
            term,
            ws,
            fitAddon,
            tab
        });

        // Activate this terminal
        this.activateTerminal(id);

        // Handle window resize
        window.addEventListener('resize', () => {
            if (this.activeTerminal === id) {
                fitAddon.fit();
            }
        });
    },

    activateTerminal(id) {
        // Deactivate all
        this.terminals.forEach((terminal, termId) => {
            terminal.tab.classList.remove('active');
            document.getElementById(termId).style.display = 'none';
        });

        // Activate selected
        const terminal = this.terminals.get(id);
        if (terminal) {
            terminal.tab.classList.add('active');
            document.getElementById(id).style.display = 'block';
            terminal.fitAddon.fit();
            terminal.term.focus();
            this.activeTerminal = id;
        }
    },

    closeTerminal(id) {
        const terminal = this.terminals.get(id);
        if (terminal) {
            terminal.ws.close();
            terminal.term.dispose();
            terminal.tab.remove();
            document.getElementById(id).remove();
            this.terminals.delete(id);

            // Activate another terminal if exists
            if (this.terminals.size > 0) {
                const firstId = this.terminals.keys().next().value;
                this.activateTerminal(firstId);
            } else {
                this.activeTerminal = null;
            }
        }
    },

    closeAll() {
        this.terminals.forEach((_, id) => this.closeTerminal(id));
    }
};

window.TerminalManager = TerminalManager;
