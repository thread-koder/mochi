FROM golang:1.26 AS builder

ARG VERSION=dev
ARG BUILD_TIME=unknown
ARG GIT_COMMIT=unknown

WORKDIR /src
COPY agent/go.mod agent/go.sum ./
RUN go mod download

COPY agent/ ./
RUN CGO_ENABLED=0 GOOS=linux go build \
	-ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT}" \
	-o /mochi-agent ./cmd/agent

# Privileged node agent (hostPID/hostNetwork in DaemonSet).
FROM gcr.io/distroless/static-debian13

COPY --from=builder /mochi-agent /mochi-agent

EXPOSE 9800
ENTRYPOINT ["/mochi-agent"]
