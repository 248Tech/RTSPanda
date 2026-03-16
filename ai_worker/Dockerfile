# =============================================================================
# Stage 1 — model exporter
# =============================================================================
# Exports a YOLOv8 .pt model to ONNX format.
# This stage is LARGE (~2 GB during build) because it needs PyTorch.
# It is NOT shipped in the final image.
#
# Build arg: YOLO_MODEL_NAME (default: yolov8n)
# Produces:  /model.onnx  (copied to stage 2)
# =============================================================================
FROM python:3.12-slim AS exporter

ARG YOLO_MODEL_NAME=yolov8n
ENV YOLO_MODEL_NAME=${YOLO_MODEL_NAME}

RUN apt-get update && apt-get install -y --no-install-recommends \
    libxcb1 \
    libgl1 \
    libglib2.0-0 \
    && rm -rf /var/lib/apt/lists/*

RUN pip install --no-cache-dir "ultralytics>=8.3"

RUN python - <<'EOF'
import os, shutil
from ultralytics import YOLO

name = os.environ["YOLO_MODEL_NAME"]
print(f"Exporting {name}.pt → {name}.onnx ...")
m = YOLO(f"{name}.pt")
m.export(format="onnx", imgsz=640, simplify=True, dynamic=False)
shutil.copy(f"{name}.onnx", "/model.onnx")
print("Export done.")
EOF


# =============================================================================
# Stage 2 — lean runtime (~200 MB)
# =============================================================================
# No PyTorch, no ultralytics. Only onnxruntime + tiny Python deps.
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

COPY --from=exporter /model.onnx /model/yolov8n.onnx
COPY ai_worker/app /app/app

EXPOSE 8090

HEALTHCHECK --interval=30s --timeout=5s --retries=5 \
    CMD python -c "import urllib.request; urllib.request.urlopen('http://127.0.0.1:8090/health', timeout=3)"

# Single worker — the model is not thread-safe to reload; one process is enough
# for the detection throughput RTSPanda needs.
CMD ["uvicorn", "app.main:app", "--host", "0.0.0.0", "--port", "8090", "--workers", "1"]
