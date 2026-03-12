FROM node:24-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

FROM golang:1.26-alpine AS go-builder
WORKDIR /app/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
COPY --from=frontend-builder /app/frontend/dist ./internal/api/web
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /out/rtspanda ./cmd/rtspanda

FROM alpine:3.22 AS mediamtx-downloader
ARG MEDIAMTX_VERSION=v1.12.3
RUN apk add --no-cache curl tar && \
    curl -fsSL "https://github.com/bluenviron/mediamtx/releases/download/${MEDIAMTX_VERSION}/mediamtx_${MEDIAMTX_VERSION}_linux_amd64.tar.gz" \
    | tar -xz -C /usr/local/bin mediamtx && \
    chmod +x /usr/local/bin/mediamtx

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata libc6-compat ffmpeg && \
    mkdir -p /data

COPY --from=go-builder /out/rtspanda /usr/local/bin/rtspanda
COPY --from=mediamtx-downloader /usr/local/bin/mediamtx /usr/local/bin/mediamtx

ENV DATA_DIR=/data
ENV PORT=8080
ENV MEDIAMTX_BIN=/usr/local/bin/mediamtx

VOLUME ["/data"]
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --retries=5 CMD wget -qO- http://127.0.0.1:8080/api/v1/health || exit 1

ENTRYPOINT ["/usr/local/bin/rtspanda"]
