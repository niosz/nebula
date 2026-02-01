# Nebula - System Administration Panel

Un pannello di amministrazione sistema completo scritto in Go con interfaccia web moderna.

## Funzionalita

- **Dashboard Real-time**: Monitoraggio CPU, RAM, Disco, Rete con grafici storici
- **Gestione Processi**: Lista, dettagli, kill processi
- **Gestione Servizi**: Start/Stop/Restart servizi di sistema (systemd, launchctl, Windows Services)
- **File Manager**: Browse, upload, download, crea/rinomina/elimina file e cartelle
- **Package Manager**: Gestione pacchetti (apt, brew, chocolatey)
- **Terminal Web**: Terminale interattivo con supporto multi-shell (bash, zsh, cmd, PowerShell)
- **API REST**: Tutte le funzionalita esposte via API REST con Swagger
- **Self-Update**: Aggiornamento automatico da GitHub Releases

## Requisiti

- Go 1.21 o superiore

## Installazione

```bash
# Clona il repository
git clone https://github.com/nebula/nebula.git
cd nebula

# Compila
go build -o nebula ./cmd/server

# Esegui
./nebula
```

## Configurazione

Crea un file `config.yaml` nella stessa directory del binario:

```yaml
server:
  host: "0.0.0.0"      # Bind su tutte le interfacce
  port: 8080
  read_timeout: 10s
  write_timeout: 10s
  shutdown_timeout: 30s

storage:
  path: "./nebula.db"
  metrics_retention: 1h
  audit_retention: 168h  # 7 giorni

auth:
  enabled: false         # Abilita per produzione!
  username: "admin"
  password: "changeme"

metrics:
  interval: 1s
  history_size: 60

terminal:
  default_shell: ""      # Auto-detect
  allowed_shells:
    - bash
    - zsh
    - sh
    - cmd
    - powershell
  max_sessions: 10

files:
  root_path: "/"
  max_upload_size: 104857600  # 100MB
  allowed_extensions: []      # Vuoto = tutti

packages:
  auto_detect: true

updater:
  enabled: true
  github_repo: "nebula/nebula"
  check_interval: 24h

logging:
  level: "info"
  format: "json"
```

## Utilizzo

**IMPORTANTE**: Nebula richiede privilegi di root/amministratore per funzionare.

```bash
# Linux/macOS
sudo ./nebula

# Windows (esegui come Amministratore)
.\nebula.exe
```

Dopo aver avviato il server, accedi a:

- **Dashboard**: `http://localhost:8080/`
- **Swagger API**: `http://localhost:8080/swagger/index.html`

### Gestione Credenziali

Quando esegui Nebula come root, le operazioni privilegiate vengono eseguite direttamente.
Se necessario, puoi salvare le credenziali sudo per operazioni future tramite l'interfaccia web
o l'API `/api/v1/auth/credentials`.

Le credenziali vengono salvate in modo sicuro (crittografate AES-256) nel database BoltDB.

### Variabili d'Ambiente

- `NEBULA_CONFIG`: Path del file di configurazione (default: `config.yaml`)
- `NEBULA_NO_ROOT`: Imposta a `1` per disabilitare il controllo root (solo sviluppo)

## API REST

### Metriche
- `GET /api/v1/metrics/cpu` - Utilizzo CPU
- `GET /api/v1/metrics/memory` - Utilizzo memoria
- `GET /api/v1/metrics/disk` - Spazio dischi
- `GET /api/v1/metrics/network` - Statistiche rete
- `GET /api/v1/metrics/all` - Tutte le metriche

### Processi
- `GET /api/v1/processes` - Lista processi
- `GET /api/v1/processes/:pid` - Dettagli processo
- `POST /api/v1/processes/:pid/kill` - Termina processo
- `GET /api/v1/processes/:pid/tree` - Albero processo

### Servizi
- `GET /api/v1/services` - Lista servizi
- `GET /api/v1/services/:name` - Dettagli servizio
- `POST /api/v1/services/:name/start` - Avvia servizio
- `POST /api/v1/services/:name/stop` - Ferma servizio
- `POST /api/v1/services/:name/restart` - Riavvia servizio
- `GET /api/v1/services/:name/logs` - Log servizio

### File Manager
- `GET /api/v1/files/list?path=` - Lista directory
- `GET /api/v1/files/download?path=` - Download file
- `POST /api/v1/files/upload?path=` - Upload file
- `POST /api/v1/files/mkdir` - Crea directory
- `DELETE /api/v1/files/delete?path=` - Elimina file/directory

### Pacchetti
- `GET /api/v1/packages` - Lista pacchetti installati
- `GET /api/v1/packages/search?q=` - Cerca pacchetti
- `POST /api/v1/packages/install` - Installa pacchetto
- `DELETE /api/v1/packages/remove?name=` - Rimuovi pacchetto

### Terminal
- `GET /api/v1/terminal/shells` - Shell disponibili
- `WebSocket /ws/terminal` - Connessione terminal

### Sistema
- `GET /api/v1/system/info` - Info sistema
- `GET /api/v1/config` - Configurazione
- `POST /api/v1/config/reload` - Ricarica config
- `GET /api/v1/update/check` - Verifica aggiornamenti
- `POST /api/v1/update/apply` - Applica aggiornamento

### WebSocket
- `/ws/metrics` - Stream metriche real-time
- `/ws/terminal` - Connessione terminal

## Sicurezza

Per l'uso in produzione:

1. **Abilita l'autenticazione** in `config.yaml`:
   ```yaml
   auth:
     enabled: true
     username: "admin"
     password: "password-sicura"
   ```

2. **Usa HTTPS** con un reverse proxy (nginx, caddy)

3. **Limita l'accesso ai file**:
   ```yaml
   files:
     root_path: "/home/user"  # Non /
   ```

4. **Non esporre direttamente su Internet** senza protezione

## Sviluppo

```bash
# Sviluppo con hot reload (richiede air)
go install github.com/cosmtrek/air@latest
air

# Test
go test ./...

# Genera documentazione Swagger
go install github.com/swaggo/swag/cmd/swag@latest
swag init -g cmd/server/main.go
```

## Struttura Progetto

```
nebula/
├── cmd/server/main.go       # Entry point
├── internal/
│   ├── api/                 # Handler REST
│   ├── config/              # Gestione configurazione
│   ├── files/               # File manager
│   ├── metrics/             # Raccolta metriche
│   ├── packages/            # Package manager
│   ├── process/             # Gestione processi
│   ├── service/             # Gestione servizi
│   ├── storage/             # BoltDB storage
│   ├── terminal/            # PTY terminal
│   ├── updater/             # Self-update
│   └── websocket/           # WebSocket hub
├── web/
│   ├── static/              # Frontend (HTML/CSS/JS)
│   └── embed.go             # File embedding
├── config.yaml              # Configurazione
├── go.mod
└── README.md
```

## Licenza

MIT License
