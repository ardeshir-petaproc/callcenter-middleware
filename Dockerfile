# syntax=docker/dockerfile:1

# 1. Build the control panel.
FROM node:22-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN mkdir -p ../internal/web && npm run build

# 2. Build the Go binaries (pure Go, no cgo).
FROM golang:1.27.3-alpine AS build
WORKDIR /src
ENV CGO_ENABLED=0 GOTOOLCHAIN=local
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /src/internal/web/dist ./internal/web/dist
ARG VERSION=dev
RUN go build -trimpath -ldflags "-s -w -X callcenter-middleware/internal/server.Version=${VERSION}" -o /out/middleware ./cmd/middleware \
 && go build -trimpath -ldflags "-s -w" -o /out/ari-sim ./cmd/ari-sim \
 && go build -trimpath -ldflags "-s -w" -o /out/mail-sim ./cmd/mail-sim

# 3. Minimal runtime image.
FROM alpine:3.22
RUN adduser -D -u 10001 cc && mkdir -p /data && chown cc /data
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /out/middleware /out/ari-sim /out/mail-sim /usr/local/bin/
USER cc
ENV DB_PATH=/data/callcenter.db DATA_DIR=/data HTTP_PORT=8080
VOLUME ["/data"]
EXPOSE 8080 40000-40199/udp
HEALTHCHECK --interval=15s --timeout=3s CMD wget -qO- http://127.0.0.1:8080/healthz || exit 1
ENTRYPOINT ["/usr/local/bin/middleware"]
