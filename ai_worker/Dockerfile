# syntax=docker/dockerfile:1.7

# =============================================================================
# Stage 1 — deterministic model resolver
# =============================================================================
# Resolution order:
#   1. Local build-context model (`./model.onnx` or `./ai_worker/model/model.onnx`)
#   2. Explicit remote URL (`YOLO_MODEL_URL`)
#   3. Named Ultralytics ONNX asset (`YOLO_MODEL_NAME` + `YOLO_MODEL_RELEASE`)
#
# No PyTorch, no ultralytics export, and no runtime conversion path.
# =============================================================================
FROM python:3.12-slim AS model-resolver

ARG MODEL_SOURCE=remote
ARG YOLO_MODEL_NAME=yolov8n
ARG YOLO_MODEL_RELEASE=v8.3.0
ARG YOLO_MODEL_URL

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    && rm -rf /var/lib/apt/lists/*

RUN --mount=type=bind,source=.,target=/context,readonly \
    set -eux; \
    model_source="$(printf '%s' "${MODEL_SOURCE:-remote}" | tr '[:upper:]' '[:lower:]')"; \
    case "$model_source" in \
      local|remote) ;; \
      *) echo "unsupported MODEL_SOURCE=${MODEL_SOURCE:-}" >&2; exit 1 ;; \
    esac; \
    mkdir -p /model; \
    if [ -s /context/model.onnx ]; then \
      cp /context/model.onnx /model/model.onnx; \
      echo "using local build-context model ./model.onnx"; \
    elif [ -s /context/ai_worker/model/model.onnx ]; then \
      cp /context/ai_worker/model/model.onnx /model/model.onnx; \
      echo "using local build-context model ./ai_worker/model/model.onnx"; \
    elif [ "$model_source" = "local" ]; then \
      echo "MODEL_SOURCE=local but no local model was found at ./model.onnx or ./ai_worker/model/model.onnx" >&2; \
      exit 1; \
    elif [ -n "${YOLO_MODEL_URL:-}" ]; then \
      curl -fsSL "${YOLO_MODEL_URL}" -o /model/model.onnx || { \
        echo "failed to download model from YOLO_MODEL_URL=${YOLO_MODEL_URL}" >&2; exit 1; \
      }; \
    else \
      model_file="${YOLO_MODEL_NAME}.onnx"; \
      model_url="https://github.com/ultralytics/assets/releases/download/${YOLO_MODEL_RELEASE}/${model_file}"; \
      curl -fsSL "${model_url}" -o /model/model.onnx || { \
        echo "failed to download prebuilt ONNX model ${model_file} from ${model_url}" >&2; exit 1; \
      }; \
    fi; \
    test -s /model/model.onnx


# =============================================================================
# Stage 2 — lean runtime
# =============================================================================
FROM python:3.12-slim

WORKDIR /app

ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PORT=8090 \
    MODEL_SOURCE=remote \
    MODEL_PATH=/model/model.onnx \
    YOLO_MODEL_PATH=/model/model.onnx

RUN apt-get update && apt-get install -y --no-install-recommends \
    libgomp1 \
    && rm -rf /var/lib/apt/lists/*

COPY ai_worker/requirements.txt /tmp/requirements.txt
RUN pip install --no-cache-dir -r /tmp/requirements.txt

COPY --from=model-resolver /model/model.onnx /model/model.onnx
COPY ai_worker/app /app/app

EXPOSE 8090

HEALTHCHECK --interval=30s --timeout=5s --retries=5 \
    CMD python -c "import urllib.request; urllib.request.urlopen('http://127.0.0.1:8090/health', timeout=3)"

# Single worker — the model is loaded once and detection throughput is limited by
# CPU-bound ONNX inference, not process fan-out.
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8090", "--workers", "1"]
