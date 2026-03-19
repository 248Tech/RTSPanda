# =============================================================================
# Stage 1 — model downloader
# =============================================================================
# Downloads a pre-exported YOLOv8 ONNX model.
# This avoids PyTorch export during image build, which is too heavy/slow for Pi.
#
# Build args:
#   - YOLO_MODEL_NAME (default: yolov8n)
#   - YOLO_MODEL_RELEASE (default: v8.3.0)
#   - YOLO_MODEL_URL (optional explicit URL override)
#
# Produces: /model/model.onnx
# =============================================================================
FROM python:3.12-slim AS model-downloader

ARG YOLO_MODEL_NAME=yolov8n
ARG YOLO_MODEL_RELEASE=v8.3.0
ARG YOLO_MODEL_URL

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    && rm -rf /var/lib/apt/lists/*

RUN set -eux; \
    model_file="${YOLO_MODEL_NAME}.onnx"; \
    mkdir -p /model; \
    downloaded=""; \
    if [ -n "${YOLO_MODEL_URL:-}" ]; then \
      curl -fsSL "${YOLO_MODEL_URL}" -o /model/model.onnx; \
      downloaded="yes"; \
    else \
      for release in "${YOLO_MODEL_RELEASE}" v8.2.0 v8.1.0 v8.0.0; do \
        url="https://github.com/ultralytics/assets/releases/download/${release}/${model_file}"; \
        if curl -fsSL "${url}" -o /model/model.onnx; then \
          downloaded="yes"; \
          break; \
        fi; \
      done; \
    fi; \
    if [ "${downloaded}" != "yes" ]; then \
      echo "download not found for ${model_file}; exporting from ${YOLO_MODEL_NAME}.pt via ultralytics"; \
      apt-get update && apt-get install -y --no-install-recommends \
        libxcb1 \
        libgl1 \
        libglib2.0-0 \
        && rm -rf /var/lib/apt/lists/*; \
      pip install --no-cache-dir "ultralytics>=8.3"; \
      export YOLO_MODEL_NAME="${YOLO_MODEL_NAME}"; \
      python -c "import os, shutil; from ultralytics import YOLO; name = os.environ['YOLO_MODEL_NAME']; print(f'Exporting {name}.pt -> {name}.onnx ...'); model = YOLO(f'{name}.pt'); model.export(format='onnx', imgsz=640, simplify=True, dynamic=False); shutil.copy(f'{name}.onnx', '/model/model.onnx'); print('Export complete.')"; \
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
    YOLO_MODEL_PATH=/model/yolov8n.onnx

RUN apt-get update && apt-get install -y --no-install-recommends \
    libgomp1 \
    && rm -rf /var/lib/apt/lists/*

COPY ai_worker/requirements.txt /tmp/requirements.txt
RUN pip install --no-cache-dir -r /tmp/requirements.txt

COPY --from=model-downloader /model/model.onnx /model/yolov8n.onnx
COPY ai_worker/app /app/app

EXPOSE 8090

HEALTHCHECK --interval=30s --timeout=5s --retries=5 \
    CMD python -c "import urllib.request; urllib.request.urlopen('http://127.0.0.1:8090/health', timeout=3)"

# Single worker — the model is not thread-safe to reload; one process is enough
# for the detection throughput RTSPanda needs.
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8090", "--workers", "1"]
