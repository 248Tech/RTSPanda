"""
RTSPanda AI Worker - ONNX Runtime inference.

Uses onnxruntime + numpy with CPU execution provider. This module includes
Pi-friendly defaults, explicit request throttling, and fallback behavior for
low-power ARM hosts.
"""

from __future__ import annotations

import io
import logging
import os
import platform
import threading
import time
from datetime import datetime, timezone
from typing import Optional

import numpy as np
from fastapi import FastAPI, File, Form, HTTPException, UploadFile
from PIL import Image

try:
    import onnxruntime as ort
except Exception as exc:  # pragma: no cover - exercised via env-driven tests
    ort = None
    _ORT_IMPORT_ERROR = str(exc)
else:
    _ORT_IMPORT_ERROR = None


# ---------------------------------------------------------------------------
# Logging
# ---------------------------------------------------------------------------

LOG_LEVEL = os.getenv("YOLO_LOG_LEVEL", "INFO").upper()
logging.basicConfig(
    level=getattr(logging, LOG_LEVEL, logging.INFO),
    format="%(asctime)s %(levelname)s %(name)s: %(message)s",
)
logger = logging.getLogger("rtspanda.ai_worker")


# ---------------------------------------------------------------------------
# Env parsing helpers
# ---------------------------------------------------------------------------


def _env_bool(name: str, default: bool) -> bool:
    raw = os.getenv(name)
    if raw is None:
        return default
    return raw.strip().lower() in {"1", "true", "yes", "on", "y"}


def _env_int(name: str, default: int, minimum: Optional[int] = None) -> int:
    raw = os.getenv(name)
    if raw is None or raw.strip() == "":
        value = default
    else:
        try:
            value = int(raw)
        except ValueError:
            logger.warning("invalid %s=%r, using %d", name, raw, default)
            value = default
    if minimum is not None and value < minimum:
        logger.warning("%s=%d below minimum %d, clamping", name, value, minimum)
        value = minimum
    return value


def _env_float(name: str, default: float, minimum: Optional[float] = None) -> float:
    raw = os.getenv(name)
    if raw is None or raw.strip() == "":
        value = default
    else:
        try:
            value = float(raw)
        except ValueError:
            logger.warning("invalid %s=%r, using %.3f", name, raw, default)
            value = default
    if minimum is not None and value < minimum:
        logger.warning("%s=%.3f below minimum %.3f, clamping", name, value, minimum)
        value = minimum
    return value


def _normalize_choice(name: str, raw_value: str, default: str, allowed: set[str]) -> str:
    value = (raw_value or "").strip().lower()
    if value in allowed:
        return value
    if value:
        logger.warning("invalid %s=%r, using %s", name, raw_value, default)
    return default


def _is_arm_machine(machine: str) -> bool:
    lowered = machine.lower()
    return lowered.startswith("arm") or lowered.startswith("aarch")


def _resolve_pi_mode(machine: str) -> bool:
    raw = os.getenv("YOLO_PI_MODE", "auto")
    normalized = raw.strip().lower()
    if normalized in {"1", "true", "yes", "on"}:
        return True
    if normalized in {"0", "false", "no", "off"}:
        return False
    if normalized not in {"", "auto"}:
        logger.warning("invalid YOLO_PI_MODE=%r, using auto", raw)
    return _is_arm_machine(machine)


# ---------------------------------------------------------------------------
# Config
# ---------------------------------------------------------------------------

MACHINE = platform.machine() or "unknown"
PI_MODE = _resolve_pi_mode(MACHINE)

MODEL_PATH = os.getenv("YOLO_MODEL_PATH", "/model/yolov8n.onnx")
CONFIDENCE = _env_float("YOLO_CONFIDENCE", 0.25, minimum=0.0)
IOU_THRESHOLD = _env_float("YOLO_IOU", 0.45, minimum=0.0)
MAX_DETECTIONS = _env_int(
    "YOLO_MAX_DETECTIONS",
    25 if PI_MODE else 100,
    minimum=1,
)

ORT_INTRA_THREADS = _env_int(
    "YOLO_ORT_INTRA_THREADS",
    1 if PI_MODE else 2,
    minimum=1,
)
ORT_INTER_THREADS = _env_int("YOLO_ORT_INTER_THREADS", 1, minimum=1)

INFER_SIZE_FALLBACK = _env_int("YOLO_INFER_SIZE", 640, minimum=64)
MODEL_REQUIRED = _env_bool("YOLO_MODEL_REQUIRED", not PI_MODE)
MODEL_ENABLED = _env_bool("YOLO_ENABLE_MODEL", True)
MAX_MODEL_MB = _env_float("YOLO_MAX_MODEL_MB", 0.0, minimum=0.0)

FALLBACK_MODE = _normalize_choice(
    "YOLO_FALLBACK_MODE",
    os.getenv("YOLO_FALLBACK_MODE", "empty" if PI_MODE else "error"),
    "empty" if PI_MODE else "error",
    {"empty", "error"},
)
BUSY_POLICY = _normalize_choice(
    "YOLO_BUSY_POLICY",
    os.getenv("YOLO_BUSY_POLICY", "drop" if PI_MODE else "wait"),
    "drop" if PI_MODE else "wait",
    {"drop", "wait"},
)

MIN_REQUEST_INTERVAL_MS = _env_int(
    "YOLO_MIN_REQUEST_INTERVAL_MS",
    750 if PI_MODE else 0,
    minimum=0,
)
MAX_UPLOAD_BYTES = _env_int(
    "YOLO_MAX_UPLOAD_BYTES",
    4 * 1024 * 1024 if PI_MODE else 8 * 1024 * 1024,
    minimum=1024,
)
MAX_SOURCE_SIDE = _env_int("YOLO_MAX_SOURCE_SIDE", 1280 if PI_MODE else 0, minimum=0)
MAX_SOURCE_PIXELS = _env_int(
    "YOLO_MAX_SOURCE_PIXELS",
    0,
    minimum=0,
)
ALLOW_DOWNSCALE = _env_bool("YOLO_ALLOW_DOWNSCALE", True)

SLOW_INFER_MS = _env_int("YOLO_SLOW_INFER_MS", 1500 if PI_MODE else 0, minimum=0)
SLOW_COOLDOWN_MS = _env_int(
    "YOLO_SLOW_COOLDOWN_MS",
    2000 if PI_MODE else 0,
    minimum=0,
)


logger.info(
    "ai_worker: config pi_mode=%s arch=%s model_required=%s fallback=%s busy=%s intra=%d inter=%d min_interval_ms=%d",
    PI_MODE,
    MACHINE,
    MODEL_REQUIRED,
    FALLBACK_MODE,
    BUSY_POLICY,
    ORT_INTRA_THREADS,
    ORT_INTER_THREADS,
    MIN_REQUEST_INTERVAL_MS,
)


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
# Runtime state
# ---------------------------------------------------------------------------

_session: Optional[object] = None
_model_error: Optional[str] = None
_input_name: str = "images"
_infer_size: int = INFER_SIZE_FALLBACK
_provider_names: list[str] = ["CPUExecutionProvider"]

_inference_lock = threading.Lock()
_state_lock = threading.Lock()
_last_inference_monotonic = 0.0
_last_inference_ms: Optional[int] = None
_cooldown_until_monotonic = 0.0


# ---------------------------------------------------------------------------
# ONNX session options
# ---------------------------------------------------------------------------


def _make_session_options():
    """Limit ORT thread pools for lower memory pressure on small devices."""
    if ort is None:
        raise RuntimeError(f"onnxruntime import failed: {_ORT_IMPORT_ERROR}")
    opts = ort.SessionOptions()
    opts.intra_op_num_threads = ORT_INTRA_THREADS
    opts.inter_op_num_threads = ORT_INTER_THREADS
    opts.graph_optimization_level = ort.GraphOptimizationLevel.ORT_ENABLE_ALL
    return opts


def _infer_size_from_shape(shape: list[object]) -> int:
    if len(shape) >= 4:
        for idx in (2, 3):
            dim = shape[idx]
            if isinstance(dim, (int, np.integer)) and int(dim) > 0:
                return int(dim)
    return INFER_SIZE_FALLBACK


def _model_size_error(path: str) -> Optional[str]:
    if MAX_MODEL_MB <= 0:
        return None
    try:
        size_bytes = os.path.getsize(path)
    except OSError as exc:
        return f"unable to stat model at {path}: {exc}"
    size_mb = size_bytes / (1024 * 1024)
    if size_mb > MAX_MODEL_MB:
        return f"model size {size_mb:.1f} MB exceeds YOLO_MAX_MODEL_MB={MAX_MODEL_MB:.1f}"
    return None


def _load_model() -> None:
    global _session, _input_name, _infer_size, _provider_names, _model_error

    if not MODEL_ENABLED:
        _model_error = "model disabled by YOLO_ENABLE_MODEL"
        logger.warning("ai_worker: %s", _model_error)
        return

    if ort is None:
        _model_error = f"onnxruntime unavailable: {_ORT_IMPORT_ERROR}"
        if MODEL_REQUIRED:
            raise RuntimeError(_model_error)
        logger.warning("ai_worker: %s (degraded mode)", _model_error)
        return

    size_error = _model_size_error(MODEL_PATH)
    if size_error is not None:
        _model_error = size_error
        if MODEL_REQUIRED:
            raise RuntimeError(size_error)
        logger.warning("ai_worker: %s (degraded mode)", size_error)
        return

    logger.info("ai_worker: loading model path=%s", MODEL_PATH)
    try:
        session = ort.InferenceSession(
            MODEL_PATH,
            providers=["CPUExecutionProvider"],
            sess_options=_make_session_options(),
        )
        inputs = session.get_inputs()
        input_shape = list(inputs[0].shape) if inputs else []
        _input_name = inputs[0].name if inputs else "images"
        _infer_size = _infer_size_from_shape(input_shape)
        _provider_names = (
            list(session.get_providers()) if hasattr(session, "get_providers") else ["CPUExecutionProvider"]
        )
        _session = session
        _model_error = None
    except Exception as exc:
        _session = None
        _model_error = str(exc)
        if MODEL_REQUIRED:
            logger.exception("ai_worker: failed to load model")
            raise
        logger.warning("ai_worker: model load failed (%s), continuing degraded", exc)


_load_model()


logger.info(
    "ai_worker: runtime ready loaded=%s infer_size=%d confidence=%.2f iou=%.2f max_det=%d",
    _session is not None,
    _infer_size,
    CONFIDENCE,
    IOU_THRESHOLD,
    MAX_DETECTIONS,
)


# ---------------------------------------------------------------------------
# Inference helpers
# ---------------------------------------------------------------------------


def _letterbox(img: Image.Image, size: int) -> tuple[Image.Image, float, int, int]:
    """Resize + pad image to a square canvas. Returns (canvas, scale, pad_x, pad_y)."""
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
    if _session is None:
        raise RuntimeError("model session is unavailable")

    orig_w, orig_h = pil_image.size
    padded, scale, pad_x, pad_y = _letterbox(pil_image, _infer_size)

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


def _apply_source_limits(image: Image.Image) -> tuple[Image.Image, float]:
    """Optionally downscale oversized source frames. Returns (image, scale)."""
    width, height = image.size
    scale = 1.0

    if MAX_SOURCE_SIDE > 0:
        max_side = max(width, height)
        if max_side > MAX_SOURCE_SIDE:
            scale = min(scale, MAX_SOURCE_SIDE / max_side)

    if MAX_SOURCE_PIXELS > 0:
        pixels = width * height
        if pixels > MAX_SOURCE_PIXELS:
            scale = min(scale, (MAX_SOURCE_PIXELS / float(pixels)) ** 0.5)

    if scale >= 1.0:
        return image, 1.0

    if not ALLOW_DOWNSCALE:
        raise HTTPException(status_code=413, detail="image dimensions exceed configured limits")

    new_width = max(1, int(round(width * scale)))
    new_height = max(1, int(round(height * scale)))
    return image.resize((new_width, new_height), Image.BILINEAR), scale


def _rescale_detections(
    detections: list[dict],
    source_scale: float,
    target_width: int,
    target_height: int,
) -> list[dict]:
    if source_scale >= 1.0:
        return detections

    inv = 1.0 / source_scale
    out: list[dict] = []
    for detection in detections:
        bbox = detection.get("bbox", {})
        x = max(0, int(round(int(bbox.get("x", 0)) * inv)))
        y = max(0, int(round(int(bbox.get("y", 0)) * inv)))
        width = max(0, int(round(int(bbox.get("width", 0)) * inv)))
        height = max(0, int(round(int(bbox.get("height", 0)) * inv)))

        x = min(x, target_width)
        y = min(y, target_height)
        width = min(width, max(0, target_width - x))
        height = min(height, max(0, target_height - y))

        out.append(
            {
                "label": detection.get("label"),
                "confidence": detection.get("confidence"),
                "bbox": {"x": x, "y": y, "width": width, "height": height},
            }
        )
    return out


def _status_code_for_skip(reason: str) -> int:
    if reason in {"busy", "throttled", "cooldown"}:
        return 429
    return 503


def _skip_reason(now_monotonic: float) -> Optional[str]:
    if _session is None:
        return "model_unavailable"
    with _state_lock:
        if _cooldown_until_monotonic > now_monotonic:
            return "cooldown"
        if MIN_REQUEST_INTERVAL_MS > 0 and _last_inference_monotonic > 0.0:
            elapsed_ms = (now_monotonic - _last_inference_monotonic) * 1000.0
            if elapsed_ms < MIN_REQUEST_INTERVAL_MS:
                return "throttled"
    return None


# ---------------------------------------------------------------------------
# FastAPI app
# ---------------------------------------------------------------------------

app = FastAPI(title="RTSPanda AI Worker", version="0.3.0")


@app.get("/health")
def health() -> dict:
    with _state_lock:
        cooldown_remaining_ms = max(0, int((_cooldown_until_monotonic - time.monotonic()) * 1000))
        last_inference_ms = _last_inference_ms
    return {
        "status": "ok" if _session is not None else "degraded",
        "model_path": MODEL_PATH,
        "infer_size": _infer_size,
        "confidence_threshold": CONFIDENCE,
        "iou_threshold": IOU_THRESHOLD,
        "max_detections": MAX_DETECTIONS,
        "model_loaded": _session is not None,
        "model_required": MODEL_REQUIRED,
        "model_error": _model_error,
        "pi_mode": PI_MODE,
        "machine": MACHINE,
        "onnxruntime_available": ort is not None,
        "onnxruntime_version": getattr(ort, "__version__", None) if ort is not None else None,
        "providers": _provider_names,
        "fallback_mode": FALLBACK_MODE,
        "busy_policy": BUSY_POLICY,
        "min_request_interval_ms": MIN_REQUEST_INTERVAL_MS,
        "slow_infer_ms": SLOW_INFER_MS,
        "slow_cooldown_ms": SLOW_COOLDOWN_MS,
        "cooldown_remaining_ms": cooldown_remaining_ms,
        "max_upload_bytes": MAX_UPLOAD_BYTES,
        "max_source_side": MAX_SOURCE_SIDE,
        "max_source_pixels": MAX_SOURCE_PIXELS,
        "allow_downscale": ALLOW_DOWNSCALE,
        "ort_intra_threads": ORT_INTRA_THREADS,
        "ort_inter_threads": ORT_INTER_THREADS,
        "last_inference_ms": last_inference_ms,
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
    if len(data) > MAX_UPLOAD_BYTES:
        raise HTTPException(status_code=413, detail="image upload exceeds YOLO_MAX_UPLOAD_BYTES")

    try:
        raw_image = Image.open(io.BytesIO(data)).convert("RGB")
    except Exception as exc:
        raise HTTPException(status_code=400, detail=f"invalid image: {exc}") from exc

    image_width, image_height = raw_image.size
    limited_image, source_scale = _apply_source_limits(raw_image)

    logger.info(
        "detect start camera_id=%s size=%dx%d scale=%.3f",
        request_camera,
        image_width,
        image_height,
        source_scale,
    )

    detections: list[dict] = []
    inference_ms: Optional[int] = None
    fallback_reason: Optional[str] = None

    reason = _skip_reason(time.monotonic())
    if reason is not None:
        if FALLBACK_MODE == "error":
            raise HTTPException(status_code=_status_code_for_skip(reason), detail=f"inference unavailable: {reason}")
        fallback_reason = reason
    else:
        acquired = _inference_lock.acquire(blocking=BUSY_POLICY == "wait")
        if not acquired:
            if FALLBACK_MODE == "error":
                raise HTTPException(status_code=429, detail="inference unavailable: busy")
            fallback_reason = "busy"
        else:
            try:
                reason = _skip_reason(time.monotonic())
                if reason is not None:
                    if FALLBACK_MODE == "error":
                        raise HTTPException(
                            status_code=_status_code_for_skip(reason),
                            detail=f"inference unavailable: {reason}",
                        )
                    fallback_reason = reason
                else:
                    infer_started = time.perf_counter()
                    detections = _run_detection(
                        limited_image,
                        conf_threshold=CONFIDENCE,
                        iou_threshold=IOU_THRESHOLD,
                        max_det=MAX_DETECTIONS,
                    )
                    inference_ms = int((time.perf_counter() - infer_started) * 1000)

                    now = time.monotonic()
                    with _state_lock:
                        global _last_inference_monotonic, _last_inference_ms, _cooldown_until_monotonic
                        _last_inference_monotonic = now
                        _last_inference_ms = inference_ms
                        if (
                            SLOW_INFER_MS > 0
                            and SLOW_COOLDOWN_MS > 0
                            and inference_ms >= SLOW_INFER_MS
                        ):
                            _cooldown_until_monotonic = max(
                                _cooldown_until_monotonic,
                                now + (SLOW_COOLDOWN_MS / 1000.0),
                            )
            except HTTPException:
                raise
            except Exception as exc:
                logger.exception("detect error camera_id=%s err=%s", request_camera, exc)
                if FALLBACK_MODE == "error":
                    raise HTTPException(status_code=500, detail=f"inference error: {exc}") from exc
                fallback_reason = "inference_error"
                detections = []
            finally:
                _inference_lock.release()

    if source_scale < 1.0 and detections:
        detections = _rescale_detections(
            detections,
            source_scale=source_scale,
            target_width=image_width,
            target_height=image_height,
        )

    response_ts = timestamp or datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    elapsed_ms = int((time.perf_counter() - started) * 1000)

    labels = ", ".join(d["label"] for d in detections[:8]) or "none"
    if len(detections) > 8:
        labels += ", ..."

    logger.info(
        "detect done camera_id=%s detections=%d labels=%s elapsed_ms=%d fallback=%s",
        request_camera,
        len(detections),
        labels,
        elapsed_ms,
        fallback_reason or "none",
    )

    return {
        "camera_id": camera_id,
        "timestamp": response_ts,
        "image_width": image_width,
        "image_height": image_height,
        "detections": detections,
        "runtime": {
            "mode": "fallback" if fallback_reason else "inference",
            "fallback_reason": fallback_reason,
            "inference_ms": inference_ms,
            "source_scale": round(source_scale, 6),
        },
    }
