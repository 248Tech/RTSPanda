from __future__ import annotations

import importlib
import sys
import time
import types
from pathlib import Path

import numpy as np
from PIL import Image
import pytest


def import_main_with_fake_onnx(
    monkeypatch: pytest.MonkeyPatch,
    *,
    env: dict[str, str] | None = None,
    input_shape: list[object] | None = None,
    session_error: str | None = None,
    session_output: np.ndarray | None = None,
):
    """Import app.main with a fake onnxruntime module and optional env overrides."""

    for key in [
        "YOLO_PI_MODE",
        "YOLO_MODEL_PATH",
        "YOLO_CONFIDENCE",
        "YOLO_IOU",
        "YOLO_MAX_DETECTIONS",
        "YOLO_ORT_INTRA_THREADS",
        "YOLO_ORT_INTER_THREADS",
        "YOLO_INFER_SIZE",
        "YOLO_MODEL_REQUIRED",
        "YOLO_ENABLE_MODEL",
        "YOLO_MAX_MODEL_MB",
        "YOLO_FALLBACK_MODE",
        "YOLO_BUSY_POLICY",
        "YOLO_MIN_REQUEST_INTERVAL_MS",
        "YOLO_MAX_UPLOAD_BYTES",
        "YOLO_MAX_SOURCE_SIDE",
        "YOLO_MAX_SOURCE_PIXELS",
        "YOLO_ALLOW_DOWNSCALE",
        "YOLO_SLOW_INFER_MS",
        "YOLO_SLOW_COOLDOWN_MS",
        "YOLO_LOG_LEVEL",
    ]:
        monkeypatch.delenv(key, raising=False)

    if env:
        for key, value in env.items():
            monkeypatch.setenv(key, str(value))

    class FakeSessionOptions:
        def __init__(self):
            self.intra_op_num_threads = None
            self.inter_op_num_threads = None
            self.graph_optimization_level = None

    class FakeGraphOptimizationLevel:
        ORT_ENABLE_ALL = "ORT_ENABLE_ALL"

    class FakeInput:
        name = "images"
        shape = input_shape if input_shape is not None else [1, 3, 640, 640]

    class FakeSession:
        def __init__(self, *args, **kwargs):
            if session_error is not None:
                raise RuntimeError(session_error)
            self._inputs = [FakeInput()]

        def get_inputs(self):
            return self._inputs

        def get_providers(self):
            return ["CPUExecutionProvider"]

        def run(self, *_args, **_kwargs):
            if session_output is not None:
                return [session_output]
            # [1, 84, N] with one low-confidence prediction.
            return [np.zeros((1, 84, 1), dtype=np.float32)]

    fake_ort = types.ModuleType("onnxruntime")
    fake_ort.__version__ = "1.21.0"
    fake_ort.SessionOptions = FakeSessionOptions
    fake_ort.GraphOptimizationLevel = FakeGraphOptimizationLevel
    fake_ort.InferenceSession = FakeSession

    monkeypatch.setitem(sys.modules, "onnxruntime", fake_ort)
    sys.modules.pop("app.main", None)
    return importlib.import_module("app.main")


def test_letterbox_preserves_aspect_ratio(monkeypatch: pytest.MonkeyPatch):
    main = import_main_with_fake_onnx(monkeypatch)

    image = Image.new("RGB", (200, 100), color=(255, 255, 255))
    canvas, scale, pad_x, pad_y = main._letterbox(image, 640)

    assert canvas.size == (640, 640)
    assert scale == pytest.approx(3.2)
    assert pad_x == 0
    assert pad_y == 160


def test_nms_filters_overlapping_boxes(monkeypatch: pytest.MonkeyPatch):
    main = import_main_with_fake_onnx(monkeypatch)

    boxes = np.array(
        [
            [0.0, 0.0, 10.0, 10.0],
            [1.0, 1.0, 9.0, 9.0],
            [20.0, 20.0, 30.0, 30.0],
        ],
        dtype=np.float32,
    )
    scores = np.array([0.95, 0.90, 0.80], dtype=np.float32)

    keep = main._nms(boxes, scores, iou_threshold=0.5)
    assert keep == [0, 2]


def test_run_detection_returns_empty_for_low_confidence_predictions(monkeypatch: pytest.MonkeyPatch):
    main = import_main_with_fake_onnx(monkeypatch)

    image = Image.new("RGB", (64, 64), color=(0, 0, 0))
    detections = main._run_detection(image, conf_threshold=0.5, iou_threshold=0.45, max_det=10)

    assert detections == []


def test_make_session_options_uses_configured_threads(monkeypatch: pytest.MonkeyPatch):
    main = import_main_with_fake_onnx(
        monkeypatch,
        env={"YOLO_ORT_INTRA_THREADS": "3", "YOLO_ORT_INTER_THREADS": "2"},
    )

    opts = main._make_session_options()

    assert opts.intra_op_num_threads == 3
    assert opts.inter_op_num_threads == 2


def test_health_reports_degraded_when_model_disabled(monkeypatch: pytest.MonkeyPatch):
    main = import_main_with_fake_onnx(
        monkeypatch,
        env={"YOLO_ENABLE_MODEL": "false", "YOLO_MODEL_REQUIRED": "false"},
    )

    payload = main.health()

    assert payload["status"] == "degraded"
    assert payload["model_loaded"] is False
    assert "YOLO_ENABLE_MODEL" in payload["model_error"]


def test_apply_source_limits_downscales_large_image(monkeypatch: pytest.MonkeyPatch):
    main = import_main_with_fake_onnx(monkeypatch, env={"YOLO_MAX_SOURCE_SIDE": "640"})

    image = Image.new("RGB", (1920, 1080), color=(255, 255, 255))
    resized, scale = main._apply_source_limits(image)

    assert resized.size == (640, 360)
    assert scale == pytest.approx(640 / 1920)


def test_apply_source_limits_rejects_when_downscale_disabled(monkeypatch: pytest.MonkeyPatch):
    main = import_main_with_fake_onnx(
        monkeypatch,
        env={"YOLO_MAX_SOURCE_SIDE": "640", "YOLO_ALLOW_DOWNSCALE": "false"},
    )

    image = Image.new("RGB", (1920, 1080), color=(255, 255, 255))
    with pytest.raises(main.HTTPException) as exc_info:
        main._apply_source_limits(image)

    assert exc_info.value.status_code == 413


def test_pi_mode_uses_pi_defaults(monkeypatch: pytest.MonkeyPatch):
    main = import_main_with_fake_onnx(monkeypatch, env={"YOLO_PI_MODE": "on"})

    assert main.PI_MODE is True
    assert main.MAX_DETECTIONS == 25
    assert main.BUSY_POLICY == "drop"


def test_skip_reason_is_throttled_after_recent_inference(monkeypatch: pytest.MonkeyPatch):
    main = import_main_with_fake_onnx(monkeypatch, env={"YOLO_MIN_REQUEST_INTERVAL_MS": "1000"})

    with main._state_lock:
        main._last_inference_monotonic = time.monotonic()

    assert main._skip_reason(time.monotonic()) == "throttled"


def test_model_size_limit_can_force_degraded_mode(monkeypatch: pytest.MonkeyPatch, tmp_path: Path):
    model_path = tmp_path / "model.onnx"
    model_path.write_bytes(b"x" * 1024)

    main = import_main_with_fake_onnx(
        monkeypatch,
        env={
            "YOLO_MODEL_PATH": str(model_path),
            "YOLO_MAX_MODEL_MB": "0.0001",
            "YOLO_MODEL_REQUIRED": "false",
        },
    )

    payload = main.health()

    assert payload["status"] == "degraded"
    assert payload["model_loaded"] is False
    assert "YOLO_MAX_MODEL_MB" in payload["model_error"]
