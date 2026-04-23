# Build context: project root (NovaPanel/)
# Stage 1: Build React frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --prefer-offline --legacy-peer-deps
COPY frontend/ .
RUN npm run build

# Stage 2: Build Go backend
FROM golang:alpine AS go-builder
WORKDIR /app
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ .
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-s -w -X github.com/novapanel/novapanel/internal/version.AppVersion=${VERSION}" \
    -o novapanel-api ./cmd/api

# Stage 3: Minimal runtime image
FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata util-linux curl && \
    ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/') && \
    curl -fsSL "https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-${ARCH}" \
      -o /usr/local/bin/cloudflared && \
    chmod +x /usr/local/bin/cloudflared
WORKDIR /app

COPY --from=go-builder /app/novapanel-api .
COPY --from=go-builder /app/migrations ./migrations
COPY --from=frontend-builder /app/frontend/dist ./frontend

EXPOSE 8080
CMD ["./novapanel-api", "serve"]
