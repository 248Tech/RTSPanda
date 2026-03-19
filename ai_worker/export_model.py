"""
Export a YOLO .pt model to ONNX format for use with the RTSPanda AI worker.

Usage (run once, requires ultralytics):

    pip install ultralytics
    python ai_worker/export_model.py
    # produces ai_worker/yolo11n.onnx

Then point the AI worker at the file:

    YOLO_MODEL_PATH=/path/to/yolo11n.onnx uvicorn app.main:app ...

For Docker users, export the model ahead of time and either:
- place it at ./model.onnx before building the image, or
- mount it to /model/model.onnx at runtime.
"""

import argparse
import shutil
from pathlib import Path


def export(model_name: str, output_dir: Path) -> Path:
    try:
        from ultralytics import YOLO  # type: ignore
    except ImportError as exc:
        raise SystemExit(
            "ultralytics is required for export. Install it with:\n"
            "  pip install ultralytics\n"
            "(It does NOT need to be installed in the runtime container.)"
        ) from exc

    print(f"Exporting {model_name}.pt → {model_name}.onnx ...")
    m = YOLO(f"{model_name}.pt")
    m.export(format="onnx", imgsz=640, simplify=True, dynamic=False)

    src = Path(f"{model_name}.onnx")
    if not src.exists():
        raise FileNotFoundError(f"Export produced no file at {src}")

    output_dir.mkdir(parents=True, exist_ok=True)
    dest = output_dir / src.name
    shutil.move(str(src), dest)
    print(f"Saved to {dest}")
    return dest


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Export YOLO model to ONNX")
    parser.add_argument(
        "--model",
        default="yolo11n",
        help="Model name without extension, e.g. yolo11n, yolo11s (default: yolo11n)",
    )
    parser.add_argument(
        "--output-dir",
        default="ai_worker",
        help="Directory to save the .onnx file (default: ai_worker/)",
    )
    args = parser.parse_args()
    export(args.model, Path(args.output_dir))
