# Stage 1: Build
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo \
    -ldflags "-s -w -X main.version=v1.0.0" \
    -o evm_rpc ./cmd/rpc

# Stage 2: Runtime
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/evm_rpc .

# Copy config file
COPY config/config.yaml /app/config/config.yaml

# Create non-root user
RUN addgroup -g 1000 rpc && \
    adduser -D -u 1000 -G rpc rpc && \
    chown -R rpc:rpc /app

USER rpc

# Expose ports
EXPOSE 8545 8546 8080 9092

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/evm_rpc"]
CMD ["-config", "/app/config/config.yaml"]
