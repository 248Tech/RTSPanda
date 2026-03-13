package detections

import "time"

type BBox struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

type Detection struct {
	Label      string  `json:"label"`
	Confidence float64 `json:"confidence"`
	BBox       BBox    `json:"bbox"`
}

type DetectResponse struct {
	CameraID    string      `json:"camera_id,omitempty"`
	Timestamp   string      `json:"timestamp"`
	ImageWidth  int         `json:"image_width,omitempty"`
	ImageHeight int         `json:"image_height,omitempty"`
	Detections  []Detection `json:"detections"`
}

type Event struct {
	ID           string    `json:"id"`
	CameraID     string    `json:"camera_id"`
	ObjectLabel  string    `json:"object_label"`
	Confidence   float64   `json:"confidence"`
	BBox         BBox      `json:"bbox"`
	SnapshotPath string    `json:"snapshot_path"`
	FrameWidth   int       `json:"frame_width,omitempty"`
	FrameHeight  int       `json:"frame_height,omitempty"`
	RawPayload   *string   `json:"raw_payload,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type Snapshot struct {
	CameraID  string    `json:"camera_id"`
	Timestamp time.Time `json:"timestamp"`
	Path      string    `json:"path"`
}

type Health struct {
	Status            string `json:"status"`
	DetectorURL       string `json:"detector_url"`
	DetectorHealthy   bool   `json:"detector_healthy"`
	QueueDepth        int    `json:"queue_depth"`
	QueueCapacity     int    `json:"queue_capacity"`
	SamplerEnabled    bool   `json:"sampler_enabled"`
	WorkerConcurrency int    `json:"worker_concurrency"`
}
