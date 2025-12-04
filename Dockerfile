# Lux CLI - Multi-stage Docker Build
ARG GO_VERSION=1.25

FROM golang:${GO_VERSION}-bookworm AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build CLI binary
RUN CGO_ENABLED=0 GOOS=linux go build -o lux -ldflags="-s -w" main.go

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy CLI binary from builder
COPY --from=builder /build/lux /usr/local/bin/lux

# Create directories for CLI data
RUN mkdir -p /root/.lux-cli

# Set environment
ENV PATH="/usr/local/bin:${PATH}"

# Default command
ENTRYPOINT ["lux"]
CMD ["--help"]
