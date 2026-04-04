# om-scrum-poker — Deployment Guide

## Prerequisites

| Tool | Version | Check |
|------|---------|-------|
| Go | 1.22+ | `go version` |
| Node.js | 18+ | `node --version` |
| npm | 9+ | `npm --version` |
| Docker (optional) | 20+ | `docker --version` |

---

## Local Development (macOS)

### 1. Clone and install dependencies

```bash
git clone <your-repo-url> om-scrum-poker
cd om-scrum-poker

# Install frontend dependencies
cd web && npm install && cd ..

# Download Go modules
go mod download
```

### 2. Run in development mode

Open **two terminals**:

**Terminal 1 — Backend (Go)**
```bash
go run ./cmd/server
# Server starts on http://localhost:8080
```

**Terminal 2 — Frontend (Vite dev server)**
```bash
cd web
npm run dev
# Vite starts on http://localhost:5173 with proxy to :8080
```

Open `http://localhost:5173` in your browser. Vite proxies `/ws/*` and `/health` to the Go backend automatically.

### 3. Build a self-contained binary

```bash
# Build frontend first
cd web && npm run build && cd ..

# Build Go binary with embedded frontend
go build -o om-scrum-poker ./cmd/server

# Run it
./om-scrum-poker
# Open http://localhost:8080
```

The binary includes all frontend assets — no external files needed.

### 4. Using Make

```bash
make build          # Build frontend + backend
make dev-backend    # Run Go server
make dev-frontend   # Run Vite dev server
make test           # Run all tests
make clean          # Remove build artifacts
```

---

## Production Deployment (Ubuntu Server)

### Option A: Binary Deployment

#### 1. Install dependencies on the server

```bash
# Install Go 1.22+
wget https://go.dev/dl/go1.22.5.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.22.5.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Install Node.js 20 LTS
curl -fsSL https://deb.nodesource.com/setup_20.x | sudo -E bash -
sudo apt-get install -y nodejs
```

#### 2. Build on the server

```bash
git clone <your-repo-url> /opt/om-scrum-poker
cd /opt/om-scrum-poker

# Build frontend
cd web && npm ci && npm run build && cd ..

# Build binary
CGO_ENABLED=0 go build -o om-scrum-poker ./cmd/server
```

#### 3. Create a systemd service

```bash
sudo tee /etc/systemd/system/om-scrum-poker.service > /dev/null <<'EOF'
[Unit]
Description=OM Scrum Poker
After=network.target

[Service]
Type=simple
User=www-data
Group=www-data
WorkingDirectory=/opt/om-scrum-poker
ExecStart=/opt/om-scrum-poker/om-scrum-poker
Restart=on-failure
RestartSec=5
Environment=PORT=8080
Environment=HOST=127.0.0.1
Environment=TRUST_PROXY=true

# Security hardening
NoNewPrivileges=yes
ProtectSystem=strict
ProtectHome=yes
ReadWritePaths=/opt/om-scrum-poker

[Install]
WantedBy=multi-user.target
EOF
```

#### 4. Start the service

```bash
sudo systemctl daemon-reload
sudo systemctl enable om-scrum-poker
sudo systemctl start om-scrum-poker

# Check status
sudo systemctl status om-scrum-poker

# View logs
sudo journalctl -u om-scrum-poker -f
```

#### 5. Configure Nginx as reverse proxy

```bash
sudo apt-get install -y nginx

sudo tee /etc/nginx/sites-available/om-scrum-poker > /dev/null <<'EOF'
server {
    listen 80;
    server_name your-domain.com;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location /ws/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_read_timeout 86400s;
        proxy_send_timeout 86400s;
    }
}
EOF

sudo ln -sf /etc/nginx/sites-available/om-scrum-poker /etc/nginx/sites-enabled/
sudo rm -f /etc/nginx/sites-enabled/default
sudo nginx -t && sudo systemctl reload nginx
```

#### 6. Add HTTPS with Let's Encrypt

```bash
sudo apt-get install -y certbot python3-certbot-nginx
sudo certbot --nginx -d your-domain.com
```

Certbot will automatically configure SSL in the Nginx config.

---

### Option B: Docker Deployment

#### 1. Build the Docker image

```bash
cd /opt/om-scrum-poker
docker build -t om-scrum-poker .
```

#### 2. Run with Docker

```bash
docker run -d \
  --name om-scrum-poker \
  --restart unless-stopped \
  -p 8080:8080 \
  -e PORT=8080 \
  om-scrum-poker
```

#### 3. Run with Docker Compose

```yaml
# docker-compose.yml
version: '3.8'
services:
  app:
    build: .
    ports:
      - "8080:8080"
    environment:
      - PORT=8080
      - HOST=0.0.0.0
      - TRUST_PROXY=true
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--spider", "-q", "http://localhost:8080/health"]
      interval: 30s
      timeout: 5s
      retries: 3
```

```bash
docker compose up -d
```

Use the same Nginx reverse proxy config (Option A, step 5) in front of Docker.

---

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP port |
| `HOST` | `0.0.0.0` | Bind address |
| `TRUST_PROXY` | `false` | Trust `X-Forwarded-For` header (set `true` behind Nginx) |

---

## Health Check

```bash
curl http://localhost:8080/health
# {"status":"ok","rooms":0,"connections":0,"uptime":"5m30s"}
```

---

## Cross-build (build on macOS, deploy to Linux)

```bash
# Build frontend
cd web && npm run build && cd ..

# Cross-compile for Linux
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o om-scrum-poker-linux ./cmd/server

# Copy to server
scp om-scrum-poker-linux user@server:/opt/om-scrum-poker/om-scrum-poker
```

This works because the Go binary embeds all frontend assets via `go:embed`.

---

## Troubleshooting

| Problem | Solution |
|---------|----------|
| WebSocket not connecting | Check Nginx `proxy_set_header Upgrade` and `Connection "upgrade"` |
| "Frontend not built" placeholder | Run `cd web && npm run build` before `go build` |
| Port already in use | `lsof -i :8080` to find the process, or change `PORT` env var |
| Permission denied on port 80 | Use Nginx as reverse proxy, don't run the app on port 80 directly |
| High memory usage | Rooms are in-memory; GC removes stale rooms after 24h with no connections |
