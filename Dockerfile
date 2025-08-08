# Build stage
FROM golang:1.24-alpine AS builder
ARG VERSION="dev"

# Install build dependencies
RUN --mount=type=cache,target=/var/cache/apk \
    apk add git ca-certificates

WORKDIR /build

# Copy go module files first for better caching
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Copy source code
COPY . .

# Build the server with caching
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -ldflags="-s -w -X main.version=${VERSION}" \
    -o /bin/mcp-dap-server .

# Install Delve debugger
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    go install github.com/go-delve/delve/cmd/dlv@latest

# Runtime stage - using distroless for minimal attack surface
FROM gcr.io/distroless/static-debian12

# Copy the binaries from builder stage
COPY --from=builder /bin/mcp-dap-server /usr/local/bin/mcp-dap-server
COPY --from=builder /go/bin/dlv /usr/local/bin/dlv

# Add metadata labels
LABEL org.opencontainers.image.title="MCP DAP Server"
LABEL org.opencontainers.image.description="Model Context Protocol server for Debug Adapter Protocol integration"
LABEL org.opencontainers.image.source="https://github.com/go-delve/mcp-dap-server"
LABEL org.opencontainers.image.licenses="MIT"

# Expose the default port (hardcoded in the app)
EXPOSE 8080

# Start the server
ENTRYPOINT ["/usr/local/bin/mcp-dap-server"]
