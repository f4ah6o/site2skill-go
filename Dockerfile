# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy project files
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o site2skillgo ./cmd/site2skillgo

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates wget git

WORKDIR /workspace

# Copy binary from builder
COPY --from=builder /app/site2skillgo /usr/local/bin/site2skillgo

# Default command shows help
CMD ["site2skillgo", "--help"]
