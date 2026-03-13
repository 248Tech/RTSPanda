import io
import os
from datetime import datetime, timezone
from typing import Optional

import numpy as np
from fastapi import FastAPI, File, Form, HTTPException, UploadFile
from PIL import Image
from ultralytics import YOLO

MODEL_NAME = os.getenv("YOLO_MODEL", "yolov8n.pt")
CONFIDENCE = float(os.getenv("YOLO_CONFIDENCE", "0.25"))

app = FastAPI(title="RTSPanda AI Worker", version="0.1.0")
model = YOLO(MODEL_NAME)


@app.get("/health")
def health() -> dict:
    return {
        "status": "ok",
        "model": MODEL_NAME,
        "confidence_threshold": CONFIDENCE,
    }


@app.post("/detect")
async def detect(
    image: UploadFile = File(...),
    camera_id: Optional[str] = Form(default=None),
    timestamp: Optional[str] = Form(default=None),
) -> dict:
    if image.content_type and not image.content_type.startswith("image/"):
        raise HTTPException(status_code=400, detail="image must be an image/* upload")

    data = await image.read()
    if not data:
        raise HTTPException(status_code=400, detail="empty image upload")

    try:
        pil_image = Image.open(io.BytesIO(data)).convert("RGB")
    except Exception as exc:
        raise HTTPException(status_code=400, detail=f"invalid image: {exc}") from exc

    frame = np.array(pil_image)
    image_width, image_height = pil_image.size
    results = model.predict(source=frame, conf=CONFIDENCE, verbose=False)

    detections = []
    for result in results:
        for box in result.boxes:
            x1, y1, x2, y2 = box.xyxy[0].tolist()
            cls_idx = int(box.cls[0].item())
            label = str(model.names.get(cls_idx, cls_idx))
            confidence = float(box.conf[0].item())
            detections.append(
                {
                    "label": label,
                    "confidence": round(confidence, 6),
                    "bbox": {
                        "x": int(round(x1)),
                        "y": int(round(y1)),
                        "width": max(0, int(round(x2 - x1))),
                        "height": max(0, int(round(y2 - y1))),
                    },
                }
            )

    response_ts = timestamp or datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    return {
        "camera_id": camera_id,
        "timestamp": response_ts,
        "image_width": image_width,
        "image_height": image_height,
        "detections": detections,
    }
