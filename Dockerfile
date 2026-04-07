# Stage 1: Build frontend
FROM node:20-alpine AS frontend
WORKDIR /app/web
COPY web/package*.json ./
RUN npm ci
COPY web/ .
RUN npm run build

# Stage 2: Build Go binary with embedded frontend
FROM golang:1.22-alpine AS backend
WORKDIR /app
COPY go.* ./
RUN go mod download
COPY . .
COPY --from=frontend /app/web/dist web/dist
RUN CGO_ENABLED=0 go build -ldflags "-X main.buildTime=$(date -u '+%Y-%m-%dT%H:%M:%SZ')" -o /om-scrum-poker ./cmd/server

# Stage 3: Minimal runtime image
FROM scratch
COPY --from=backend /om-scrum-poker /om-scrum-poker
EXPOSE 8080
ENTRYPOINT ["/om-scrum-poker"]
