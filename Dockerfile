# syntax=docker/dockerfile:1.7

FROM --platform=$BUILDPLATFORM node:24-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
# Keep builds resilient when package-lock.json is temporarily out of sync.
RUN npm ci || npm install
COPY frontend/ ./
RUN npm run build

FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS go-builder
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
WORKDIR /app/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
COPY --from=frontend-builder /app/frontend/dist ./internal/api/web
RUN set -eux; \
    goos="${TARGETOS:-$(go env GOOS)}"; \
    goarch="${TARGETARCH:-$(go env GOARCH)}"; \
    govariant="${TARGETVARIANT:-}"; \
    if [ "$goarch" = "arm" ] && [ -n "$govariant" ]; then \
      export GOARM="${govariant#v}"; \
    fi; \
    CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" go build -ldflags="-s -w" -o /out/rtspanda ./cmd/rtspanda

FROM --platform=$BUILDPLATFORM alpine:3.22 AS mediamtx-downloader
ARG MEDIAMTX_VERSION=v1.12.3
ARG TARGETARCH
ARG TARGETVARIANT
RUN apk add --no-cache curl tar && \
    set -eux; \
    target_arch="${TARGETARCH:-}"; \
    target_variant="${TARGETVARIANT:-}"; \
    if [ -z "$target_arch" ]; then \
      case "$(apk --print-arch)" in \
        x86_64) target_arch="amd64" ;; \
        aarch64) target_arch="arm64" ;; \
        armv7|armhf) target_arch="arm"; target_variant="v7" ;; \
        armv6) target_arch="arm"; target_variant="v6" ;; \
        *) echo "unsupported build architecture: $(apk --print-arch)" >&2; exit 1 ;; \
      esac; \
    fi; \
    candidates=""; \
    case "$target_arch" in \
      amd64) candidates="amd64" ;; \
      arm64) candidates="arm64v8 arm64" ;; \
      arm) \
        case "$target_variant" in \
          v6) candidates="armv6" ;; \
          v7|"") candidates="armv7 armhf armv6" ;; \
          *) candidates="armv7 armhf armv6" ;; \
        esac \
        ;; \
      *) echo "unsupported TARGETARCH: $target_arch" >&2; exit 1 ;; \
    esac; \
    mkdir -p /tmp/mediamtx /usr/local/bin; \
    found=""; \
    for mediamtx_arch in $candidates; do \
      url="https://github.com/bluenviron/mediamtx/releases/download/${MEDIAMTX_VERSION}/mediamtx_${MEDIAMTX_VERSION}_linux_${mediamtx_arch}.tar.gz"; \
      if curl -fsSL "$url" -o /tmp/mediamtx.tar.gz; then \
        tar -xzf /tmp/mediamtx.tar.gz -C /tmp/mediamtx mediamtx; \
        install -m 0755 /tmp/mediamtx/mediamtx /usr/local/bin/mediamtx; \
        found="$mediamtx_arch"; \
        break; \
      fi; \
    done; \
    if [ -z "$found" ]; then \
      echo "failed to download mediamtx ${MEDIAMTX_VERSION} for arch=${target_arch} variant=${target_variant}" >&2; \
      exit 1; \
    fi

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
