# om-scrum-poker — Deployment Guide

## Production Setup (AWS EC2 + GitHub Actions)

The project is deployed on a single EC2 instance. Push to `main` triggers automatic build and deploy via GitHub Actions.

### Architecture

```
User → HTTPS → Caddy (ports 80/443, auto-SSL) → localhost:8080 → poker container
```

- **EC2**: t3.micro, Amazon Linux 2023, Elastic IP
- **Docker**: poker app runs as a container, Caddy runs as a reverse proxy container
- **GitHub Actions**: builds Docker image, pushes to GHCR, deploys to EC2 via SSH
- **GHCR** (GitHub Container Registry): stores Docker images
- **Caddy**: reverse proxy, automatic Let's Encrypt HTTPS certificates

### How deploy works

1. Developer pushes to `main`
2. GitHub Actions (`.github/workflows/deploy.yml`) triggers:
   - Builds Docker image using multi-stage Dockerfile
   - Pushes image to `ghcr.io/<repo>:latest`
   - SSHes into EC2, pulls new image, restarts `poker` container
3. Caddy proxies traffic to the new container — no restart needed

Deploy takes ~35 seconds.

### GitHub Secrets (Settings → Secrets → Actions)

| Secret | Description |
|--------|-------------|
| `EC2_HOST` | Elastic IP of the EC2 instance |
| `EC2_SSH_KEY` | Private SSH key (`poker-key`) for ec2-user |
| `ALLOWED_ORIGINS` | Allowed WebSocket origins (the domain with https://) |

`GITHUB_TOKEN` is built-in and provides access to GHCR.

### EC2 instance setup

Instance was created with AWS CLI:

```bash
# Security group with ports 22, 80, 443
aws ec2 create-security-group --group-name poker-sg --description "Scrum Poker"
aws ec2 authorize-security-group-ingress --group-name poker-sg --protocol tcp --port 22 --cidr 0.0.0.0/0
aws ec2 authorize-security-group-ingress --group-name poker-sg --protocol tcp --port 80 --cidr 0.0.0.0/0
aws ec2 authorize-security-group-ingress --group-name poker-sg --protocol tcp --port 443 --cidr 0.0.0.0/0

# SSH key pair
aws ec2 create-key-pair --key-name poker-key --query 'KeyMaterial' --output text > ~/.ssh/poker-key.pem
chmod 400 ~/.ssh/poker-key.pem

# Launch instance (Amazon Linux 2023, t3.micro)
aws ec2 run-instances \
  --image-id <ami-id> \
  --instance-type t3.micro \
  --key-name poker-key \
  --security-group-ids <sg-id> \
  --tag-specifications 'ResourceType=instance,Tags=[{Key=Name,Value=poker}]'

# Elastic IP
aws ec2 allocate-address
aws ec2 associate-address --instance-id <id> --allocation-id <eipalloc-id>
```

Docker installed on the instance:

```bash
ssh -i ~/.ssh/poker-key.pem ec2-user@<ELASTIC_IP>
sudo yum install -y docker
sudo systemctl enable --now docker
sudo usermod -aG docker ec2-user
```

### Caddy (HTTPS reverse proxy)

Caddy runs as a Docker container with `--network host`. Configuration:

```
# ~/Caddyfile on EC2
your-domain.com {
    reverse_proxy localhost:8080
}
```

Started with:

```bash
docker run -d --name caddy --restart unless-stopped \
  --network host \
  -v /home/ec2-user/Caddyfile:/etc/caddy/Caddyfile \
  -v caddy_data:/data \
  -v caddy_config:/config \
  caddy:2-alpine
```

Caddy automatically obtains and renews Let's Encrypt certificates. No manual SSL setup needed.

### DNS

A-record pointing the domain to the Elastic IP. Managed in Route 53 (separate AWS account).

### Containers on EC2

| Container | Image | Ports | Purpose |
|-----------|-------|-------|---------|
| `poker` | `ghcr.io/<repo>:latest` | `127.0.0.1:8080→8080` | Application |
| `caddy` | `caddy:2-alpine` | `80, 443` (host network) | HTTPS reverse proxy |

Both have `--restart unless-stopped`.

### Manual operations

```bash
# SSH into server
ssh -i ~/.ssh/poker-key.pem ec2-user@<ELASTIC_IP>

# View running containers
docker ps

# View app logs
docker logs poker
docker logs caddy

# Manual redeploy
docker pull ghcr.io/<repo>:latest
docker stop poker && docker rm poker
docker run -d --name poker --restart unless-stopped \
  -p 127.0.0.1:8080:8080 \
  -e ALLOWED_ORIGINS='https://your-domain.com' \
  -e TRUST_PROXY=true \
  ghcr.io/<repo>:latest

# Restart Caddy (e.g. after editing Caddyfile)
docker restart caddy
```

### Cost

- EC2 t3.micro: free tier eligible (750 hrs/month for 12 months), then ~$8/month
- Elastic IP: free while associated with a running instance
- GitHub Actions: free for public repositories
- GHCR: free for public repositories

---

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
