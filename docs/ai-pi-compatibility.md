# AI Worker Raspberry Pi Compatibility

This document summarizes practical AI worker behavior on Raspberry Pi and the runtime controls added for Pi viability.

## 1) ONNX Runtime on ARM (Pi)

### Compatibility conclusion

- **Raspberry Pi OS 64-bit (ARM64 / `aarch64`)**: viable target for `onnxruntime` Python wheels.
- **Raspberry Pi OS 32-bit (`armv7`)**: not a practical ONNX Runtime target for this worker path; treat as degraded/no-model mode.

### Why

- ONNX Runtime's official platform compatibility docs list Linux ARM64 support, while ARM32/armv7 is not a primary wheel target.
- PyPI wheel availability for `onnxruntime` tracks this in practice (ARM64 wheels are available; ARMv7 wheels are typically absent).

References:

- [ONNX Runtime compatibility](https://onnxruntime.ai/docs/reference/compatibility.html)
- [onnxruntime on PyPI](https://pypi.org/project/onnxruntime/)

## 2) AI Worker Runtime Changes for Pi Viability

The worker now includes explicit low-power controls:

- **Pi mode autodetect** (`YOLO_PI_MODE=auto|on|off`)
- **Thread capping** for ONNX Runtime (`YOLO_ORT_INTRA_THREADS`, `YOLO_ORT_INTER_THREADS`)
- **Model-optional startup** (`YOLO_MODEL_REQUIRED=false`) for degraded operation instead of crash-looping
- **Fallback behavior** (`YOLO_FALLBACK_MODE=empty|error`)
- **Backpressure policy** (`YOLO_BUSY_POLICY=drop|wait`)
- **Request pacing** (`YOLO_MIN_REQUEST_INTERVAL_MS`)
- **Frame size controls** (`YOLO_MAX_SOURCE_SIDE`, `YOLO_MAX_SOURCE_PIXELS`, `YOLO_ALLOW_DOWNSCALE`)
- **Upload cap** (`YOLO_MAX_UPLOAD_BYTES`)
- **Heavy-model guardrail** (`YOLO_MAX_MODEL_MB`)
- **Slow-inference cooldown** (`YOLO_SLOW_INFER_MS`, `YOLO_SLOW_COOLDOWN_MS`)
- **Expanded `/health` telemetry** for model/degraded state and runtime limits

## 3) CPU-Only Inference Expectations (Realistic)

These are practical planning ranges for a **nano-class ONNX detector such as `yolo11n`** on CPU-only Pi, assuming one worker and no GPU/NPU acceleration:

- **Pi 5 (8 GB)**, 640 input: typically **~250-450 ms/inference** (about 2-4 FPS max sustained detector throughput).
- **Pi 4 (4-8 GB)**, 640 input: typically **~600-1200 ms/inference** (about 0.8-1.6 FPS).
- **Pi 4 (4-8 GB)**, reduced effective source size before letterboxing: often **~400-900 ms/inference**.

Operational guidance:

- For backend sampling, plan around **1 frame every 1-3 seconds per actively tracked camera** on Pi 4.
- Use **single detector concurrency** and keep queue depth low.
- Prefer `yolo11n` for Pi; larger variants (`s/m/l/x`) are usually too slow for practical multi-camera tracking.

## 4) Frame Processing Limits and Tunables

Recommended defaults for Pi:

```bash
YOLO_PI_MODE=on
YOLO_ORT_INTRA_THREADS=1
YOLO_ORT_INTER_THREADS=1
YOLO_MAX_DETECTIONS=25
YOLO_MIN_REQUEST_INTERVAL_MS=750
YOLO_BUSY_POLICY=drop
YOLO_MAX_SOURCE_SIDE=1280
YOLO_MAX_SOURCE_PIXELS=0
YOLO_ALLOW_DOWNSCALE=true
YOLO_MAX_UPLOAD_BYTES=4194304
```

Conservative profile (Pi 4 with multiple active cameras):

```bash
YOLO_MIN_REQUEST_INTERVAL_MS=1200
YOLO_MAX_SOURCE_SIDE=960
YOLO_SLOW_INFER_MS=1200
YOLO_SLOW_COOLDOWN_MS=3000
```

## 5) Practical Fallback Strategy (When Models Are Too Heavy)

Use this if startup fails, inference is unstable, or latency is too high.

### Toggle set (recommended)

```bash
YOLO_MODEL_REQUIRED=false
YOLO_FALLBACK_MODE=empty
YOLO_BUSY_POLICY=drop
YOLO_MAX_MODEL_MB=25
```

Behavior:

- If model load fails (or model exceeds `YOLO_MAX_MODEL_MB`), worker stays up in degraded mode.
- `/detect` returns an empty detections array instead of hard-failing (when `YOLO_FALLBACK_MODE=empty`).
- Worker health stays queryable and reports degraded/model state explicitly.

### Hard-disable AI on constrained hosts

```bash
YOLO_ENABLE_MODEL=false
YOLO_MODEL_REQUIRED=false
YOLO_FALLBACK_MODE=empty
```

This keeps API compatibility while disabling heavy inference entirely.

## 6) Recommended Model/Runtime Adjustments

- Prefer **YOLO11n ONNX**.
- Keep backend detection sampling conservative (seconds, not sub-second) on Pi.
- Keep detector concurrency at 1 where possible.
- Avoid large source frames; downscale before inference.
- Use `YOLO_MAX_MODEL_MB` as a hard stop to prevent accidentally deploying heavy models.

## 7) requirements.txt Notes

`ai_worker/requirements.txt` now installs `onnxruntime` only on architectures where wheel support is expected (`x86_64`, `AMD64`, `aarch64`, `arm64`). On unsupported ARM32 targets, install can still succeed and the worker can run in degraded mode.
