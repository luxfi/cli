# Lux CLI - Multi-stage Docker Build
# Stage 1: Install Go 1.25.5 from source
FROM debian:bookworm-slim AS go-builder

RUN apt-get update && apt-get install -y --no-install-recommends \
    wget ca-certificates \
    && rm -rf /var/lib/apt/lists/*

ARG GO_VERSION=1.26.1
ARG TARGETARCH

RUN wget -q "https://go.dev/dl/go${GO_VERSION}.linux-${TARGETARCH}.tar.gz" \
    && tar -C /usr/local -xzf "go${GO_VERSION}.linux-${TARGETARCH}.tar.gz" \
    && rm "go${GO_VERSION}.linux-${TARGETARCH}.tar.gz"

# Stage 2: Build the CLI
FROM debian:bookworm-slim AS builder

# Install ca-certificates + tzdata; create nonroot user (uid 65532) for scratch runtime.
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates tzdata git \
    && rm -rf /var/lib/apt/lists/* \
    && echo 'nonroot:x:65532:65532:nonroot:/home/nonroot:/sbin/nologin' >> /etc/passwd \
    && echo 'nonroot:x:65532:' >> /etc/group

COPY --from=go-builder /usr/local/go /usr/local/go
ENV PATH="/usr/local/go/bin:${PATH}"

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build CLI binary
RUN CGO_ENABLED=0 GOOS=linux go build -o lux -ldflags="-s -w" main.go

# Runtime stage - scratch for minimal size and security.
FROM scratch

# Copy ca-certificates, timezone data, and passwd/group for the nonroot user.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

# Copy CLI binary from builder
COPY --from=builder /build/lux /usr/local/bin/lux

# Run as nonroot user (uid: 65532)
USER 65532:65532

# Default command
ENTRYPOINT ["lux"]
CMD ["--help"]
