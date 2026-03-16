"""
RTSPanda AI Worker — ONNX Runtime inference.

Replaces the ultralytics/PyTorch stack with onnxruntime + numpy.
Runtime memory drops from ~600-1500 MB to ~150-250 MB.

The ONNX model is exported at build time (see Dockerfile stage 1).
Expected model output: float32[1, 84, N] (YOLOv8 raw predictions).
  - [:, :4, :] = cx, cy, w, h  (input-image pixel coordinates)
  - [:, 4:, :] = 80 COCO class probabilities
"""

import io
import logging
import os
import time
from datetime import datetime, timezone
from typing import Optional

import numpy as np
import onnxruntime as ort
from fastapi import FastAPI, File, Form, HTTPException, UploadFile
from PIL import Image

# ---------------------------------------------------------------------------
# Config
# ---------------------------------------------------------------------------

MODEL_PATH = os.getenv("YOLO_MODEL_PATH", "/model/yolov8n.onnx")
CONFIDENCE = float(os.getenv("YOLO_CONFIDENCE", "0.25"))
IOU_THRESHOLD = float(os.getenv("YOLO_IOU", "0.45"))
MAX_DETECTIONS = int(os.getenv("YOLO_MAX_DETECTIONS", "100"))
LOG_LEVEL = os.getenv("YOLO_LOG_LEVEL", "INFO").upper()

logging.basicConfig(
    level=getattr(logging, LOG_LEVEL, logging.INFO),
    format="%(asctime)s %(levelname)s %(name)s: %(message)s",
)
logger = logging.getLogger("rtspanda.ai_worker")

# ---------------------------------------------------------------------------
# COCO class names (80 classes)
# ---------------------------------------------------------------------------

COCO_NAMES = [
    "person", "bicycle", "car", "motorcycle", "airplane", "bus", "train",
    "truck", "boat", "traffic light", "fire hydrant", "stop sign",
    "parking meter", "bench", "bird", "cat", "dog", "horse", "sheep", "cow",
    "elephant", "bear", "zebra", "giraffe", "backpack", "umbrella", "handbag",
    "tie", "suitcase", "frisbee", "skis", "snowboard", "sports ball", "kite",
    "baseball bat", "baseball glove", "skateboard", "surfboard",
    "tennis racket", "bottle", "wine glass", "cup", "fork", "knife", "spoon",
    "bowl", "banana", "apple", "sandwich", "orange", "broccoli", "carrot",
    "hot dog", "pizza", "donut", "cake", "chair", "couch", "potted plant",
    "bed", "dining table", "toilet", "tv", "laptop", "mouse", "remote",
    "keyboard", "cell phone", "microwave", "oven", "toaster", "sink",
    "refrigerator", "book", "clock", "vase", "scissors", "teddy bear",
    "hair drier", "toothbrush",
]

# ---------------------------------------------------------------------------
# ONNX session options
# ---------------------------------------------------------------------------


def _make_session_options() -> ort.SessionOptions:
    """Limit internal thread pools so the process doesn't over-spawn OS threads."""
    opts = ort.SessionOptions()
    opts.intra_op_num_threads = 2
    opts.inter_op_num_threads = 1
    opts.graph_optimization_level = ort.GraphOptimizationLevel.ORT_ENABLE_ALL
    return opts


# ---------------------------------------------------------------------------
# Load model once at startup
# ---------------------------------------------------------------------------

logger.info("ai_worker: loading model path=%s", MODEL_PATH)
_session = ort.InferenceSession(
    MODEL_PATH,
    providers=["CPUExecutionProvider"],
    sess_options=_make_session_options(),
)
_input_name: str = _session.get_inputs()[0].name
_infer_size: int = int(_session.get_inputs()[0].shape[2])  # e.g. 640

logger.info(
    "ai_worker: model ready input=%s infer_size=%d confidence=%.2f iou=%.2f max_det=%d",
    _input_name,
    _infer_size,
    CONFIDENCE,
    IOU_THRESHOLD,
    MAX_DETECTIONS,
)

# ---------------------------------------------------------------------------
# Inference helpers
# ---------------------------------------------------------------------------


def _letterbox(img: Image.Image, size: int) -> tuple[Image.Image, float, int, int]:
    """Resize + pad image to a square canvas (letterbox). Returns (canvas, scale, pad_x, pad_y)."""
    iw, ih = img.size
    scale = min(size / iw, size / ih)
    nw, nh = round(iw * scale), round(ih * scale)
    pad_x = (size - nw) // 2
    pad_y = (size - nh) // 2
    resized = img.resize((nw, nh), Image.BILINEAR)
    canvas = Image.new("RGB", (size, size), (114, 114, 114))
    canvas.paste(resized, (pad_x, pad_y))
    return canvas, scale, pad_x, pad_y


def _nms(boxes: np.ndarray, scores: np.ndarray, iou_threshold: float) -> list[int]:
    """Greedy NMS. Returns kept indices ordered by descending score."""
    x1, y1, x2, y2 = boxes[:, 0], boxes[:, 1], boxes[:, 2], boxes[:, 3]
    areas = np.maximum(0.0, x2 - x1) * np.maximum(0.0, y2 - y1)
    order = scores.argsort()[::-1]
    keep: list[int] = []
    while order.size > 0:
        i = int(order[0])
        keep.append(i)
        if order.size == 1:
            break
        xx1 = np.maximum(x1[i], x1[order[1:]])
        yy1 = np.maximum(y1[i], y1[order[1:]])
        xx2 = np.minimum(x2[i], x2[order[1:]])
        yy2 = np.minimum(y2[i], y2[order[1:]])
        inter = np.maximum(0.0, xx2 - xx1) * np.maximum(0.0, yy2 - yy1)
        union = areas[i] + areas[order[1:]] - inter
        iou = inter / np.where(union > 0.0, union, 1e-9)
        order = order[np.where(iou <= iou_threshold)[0] + 1]
    return keep


def _run_detection(
    pil_image: Image.Image,
    conf_threshold: float,
    iou_threshold: float,
    max_det: int,
) -> list[dict]:
    orig_w, orig_h = pil_image.size
    padded, scale, pad_x, pad_y = _letterbox(pil_image, _infer_size)

    # Normalize to [0,1], convert to NCHW float32
    img_np = np.array(padded, dtype=np.float32) / 255.0
    img_np = img_np.transpose(2, 0, 1)[np.newaxis]  # [1, 3, H, W]

    raw = _session.run(None, {_input_name: img_np})[0]  # [1, 84, N]
    pred = raw[0].T  # [N, 84]

    boxes_xywh = pred[:, :4]
    class_scores = pred[:, 4:]

    class_ids = class_scores.argmax(axis=1)
    confidences = class_scores.max(axis=1)

    mask = confidences >= conf_threshold
    if not mask.any():
        return []

    boxes_xywh = boxes_xywh[mask]
    confidences = confidences[mask]
    class_ids = class_ids[mask]

    # cx,cy,w,h → x1,y1,x2,y2
    cx, cy, w, h = boxes_xywh[:, 0], boxes_xywh[:, 1], boxes_xywh[:, 2], boxes_xywh[:, 3]
    boxes_xyxy = np.stack([cx - w / 2, cy - h / 2, cx + w / 2, cy + h / 2], axis=1)

    keep = _nms(boxes_xyxy, confidences, iou_threshold)
    if len(keep) > max_det:
        keep = keep[:max_det]

    results = []
    for i in keep:
        bx1 = max(0.0, (float(boxes_xyxy[i, 0]) - pad_x) / scale)
        by1 = max(0.0, (float(boxes_xyxy[i, 1]) - pad_y) / scale)
        bx2 = min(float(orig_w), (float(boxes_xyxy[i, 2]) - pad_x) / scale)
        by2 = min(float(orig_h), (float(boxes_xyxy[i, 3]) - pad_y) / scale)

        cls_id = int(class_ids[i])
        label = COCO_NAMES[cls_id] if cls_id < len(COCO_NAMES) else str(cls_id)

        results.append(
            {
                "label": label,
                "confidence": round(float(confidences[i]), 6),
                "bbox": {
                    "x": int(round(bx1)),
                    "y": int(round(by1)),
                    "width": max(0, int(round(bx2 - bx1))),
                    "height": max(0, int(round(by2 - by1))),
                },
            }
        )
    return results


# ---------------------------------------------------------------------------
# FastAPI app
# ---------------------------------------------------------------------------

app = FastAPI(title="RTSPanda AI Worker", version="0.2.0")


@app.get("/health")
def health() -> dict:
    return {
        "status": "ok",
        "model_path": MODEL_PATH,
        "infer_size": _infer_size,
        "confidence_threshold": CONFIDENCE,
        "iou_threshold": IOU_THRESHOLD,
        "max_detections": MAX_DETECTIONS,
    }


@app.post("/detect")
async def detect(
    image: UploadFile = File(...),
    camera_id: Optional[str] = Form(default=None),
    timestamp: Optional[str] = Form(default=None),
) -> dict:
    started = time.perf_counter()
    request_camera = camera_id or "unknown"

    if image.content_type and not image.content_type.startswith("image/"):
        raise HTTPException(status_code=400, detail="image must be an image/* upload")

    data = await image.read()
    if not data:
        raise HTTPException(status_code=400, detail="empty image upload")

    try:
        pil_image = Image.open(io.BytesIO(data)).convert("RGB")
    except Exception as exc:
        raise HTTPException(status_code=400, detail=f"invalid image: {exc}") from exc

    image_width, image_height = pil_image.size
    logger.info(
        "detect start camera_id=%s size=%dx%d",
        request_camera,
        image_width,
        image_height,
    )

    try:
        detections = _run_detection(pil_image, CONFIDENCE, IOU_THRESHOLD, MAX_DETECTIONS)
    except Exception as exc:
        logger.exception("detect error camera_id=%s err=%s", request_camera, exc)
        raise HTTPException(status_code=500, detail=f"inference error: {exc}") from exc

    response_ts = timestamp or datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    elapsed_ms = int((time.perf_counter() - started) * 1000)
    labels = ", ".join(d["label"] for d in detections[:8]) or "none"
    if len(detections) > 8:
        labels += ", ..."

    logger.info(
        "detect done camera_id=%s detections=%d labels=%s elapsed_ms=%d",
        request_camera,
        len(detections),
        labels,
        elapsed_ms,
    )

    return {
        "camera_id": camera_id,
        "timestamp": response_ts,
        "image_width": image_width,
        "image_height": image_height,
        "detections": detections,
    }
