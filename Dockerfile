# syntax=docker/dockerfile:1

# ---------- Build stage ----------
FROM golang:1.26-alpine AS builder

WORKDIR /app

# Cache go mod downloads separately from source changes
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/api ./cmd/api

# ---------- Run stage ----------
FROM alpine:3.20

RUN apk --no-cache add ca-certificates tzdata curl \
    && addgroup -S app && adduser -S app -G app

WORKDIR /app

COPY --from=builder /app/bin/api .

# Runtime dirs (real data is mounted as a volume in docker-compose/production)
RUN mkdir -p /app/uploads && chown -R app:app /app

USER app

# Real port comes from APP_PORT env var (see internal/config/config.go, default 8080).
# No .env file is baked into the image — configuration is injected at runtime
# via `docker run -e` / docker-compose `env_file` / GitLab CI deploy variables.
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:${APP_PORT:-8080}/health || exit 1

CMD ["./api"]
