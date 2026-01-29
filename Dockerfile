FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache gcc musl-dev

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build
RUN CGO_ENABLED=1 go build -o cxtv-alerts .

# Runtime image
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Copy binary
COPY --from=builder /app/cxtv-alerts .

# Copy web files
COPY --from=builder /app/web ./web

# Create directories for mounted volumes
RUN mkdir -p /app/config /app/data /app/web/avatars

# Default config (will be overwritten by mount)
COPY config/settings.json /app/config/settings.json
COPY config/streamers.json /app/config/streamers.json

EXPOSE 8080

CMD ["./cxtv-alerts"]
