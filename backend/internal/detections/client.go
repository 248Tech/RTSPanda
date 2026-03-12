package detections

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	http    *http.Client
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	u := strings.TrimRight(baseURL, "/")
	return &Client{
		baseURL: u,
		http: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) BaseURL() string {
	return c.baseURL
}

func (c *Client) DetectFile(cameraID string, timestamp time.Time, path string) (DetectResponse, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return DetectResponse{}, "", fmt.Errorf("open snapshot: %w", err)
	}
	defer file.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := writer.CreateFormFile("image", filepath.Base(path))
	if err != nil {
		return DetectResponse{}, "", fmt.Errorf("create form file: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return DetectResponse{}, "", fmt.Errorf("copy image: %w", err)
	}

	if cameraID != "" {
		_ = writer.WriteField("camera_id", cameraID)
	}
	_ = writer.WriteField("timestamp", timestamp.UTC().Format(time.RFC3339))

	if err := writer.Close(); err != nil {
		return DetectResponse{}, "", fmt.Errorf("close multipart writer: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+"/detect", &body)
	if err != nil {
		return DetectResponse{}, "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.http.Do(req)
	if err != nil {
		return DetectResponse{}, "", fmt.Errorf("request detector: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return DetectResponse{}, "", fmt.Errorf("read detector response: %w", err)
	}

	if resp.StatusCode >= 300 {
		return DetectResponse{}, string(raw), fmt.Errorf("detector status %d", resp.StatusCode)
	}

	var out DetectResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return DetectResponse{}, string(raw), fmt.Errorf("decode detector response: %w", err)
	}
	return out, string(raw), nil
}

func (c *Client) Healthy() bool {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return false
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode >= 200 && resp.StatusCode < 300
}
