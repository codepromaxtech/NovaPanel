FROM golang:alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o novapanel-api ./cmd/api

FROM alpine:3.19
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app

COPY --from=builder /app/novapanel-api .
COPY --from=builder /app/migrations ./migrations

# Copy pre-built frontend assets
COPY frontend/ ./frontend/

EXPOSE 8080
CMD ["./novapanel-api", "serve"]
