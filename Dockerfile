# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
ARG VERSION=dev
ARG BUILD_TIME
ARG GIT_COMMIT
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT}" \
    -o mochi ./cmd/mochi

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/mochi .

# Create non-root user
RUN addgroup -g 1000 mochi && \
    adduser -D -u 1000 -G mochi mochi && \
    chown -R mochi:mochi /app

USER mochi

EXPOSE 8080

ENTRYPOINT ["./mochi"]
