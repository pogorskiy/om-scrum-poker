.PHONY: dev dev-backend dev-frontend build build-frontend build-backend test docker-build docker-run clean

# Development — run backend and frontend separately
dev: dev-backend

dev-backend:
	go run ./cmd/server

dev-frontend:
	cd web && npm run dev

# Production build
build: build-frontend build-backend

build-frontend:
	cd web && npm ci && npm run build

build-backend:
	go build -o om-scrum-poker ./cmd/server

# Tests
test:
	go test ./...

# Docker
docker-build:
	docker build -t om-scrum-poker .

docker-run:
	docker run -p 8080:8080 om-scrum-poker

# Cleanup
clean:
	rm -f om-scrum-poker
	rm -rf web/dist/assets web/dist/index.html
