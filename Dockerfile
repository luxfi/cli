# Lux CLI - Multi-stage Docker Build
ARG GO_VERSION=1.23.4

FROM golang:${GO_VERSION}-bookworm AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build CLI binary
RUN CGO_ENABLED=0 GOOS=linux go build -o lux -ldflags="-s -w" main.go

# Runtime stage - use distroless for minimal size and security
# The static variant includes ca-certificates
FROM gcr.io/distroless/static-debian12:nonroot

# Copy CLI binary from builder
COPY --from=builder /build/lux /usr/local/bin/lux

# Run as nonroot user (uid: 65532)
USER nonroot:nonroot

# Default command
ENTRYPOINT ["lux"]
CMD ["--help"]
