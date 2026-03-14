package detections

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Client struct {
	baseURLs []string
	http     *http.Client
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	u := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if u == "" {
		u = "http://127.0.0.1:8090"
	}
	return &Client{
		baseURLs: buildDetectorURLs(u),
		http: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) BaseURL() string {
	if len(c.baseURLs) == 0 {
		return ""
	}
	return c.baseURLs[0]
}

func (c *Client) BaseURLs() []string {
	out := make([]string, len(c.baseURLs))
	copy(out, c.baseURLs)
	return out
}

func (c *Client) DetectFile(cameraID string, timestamp time.Time, path string) (DetectResponse, string, error) {
	file, err := os.Open(path)
	if err != nil {
		return DetectResponse{}, "", fmt.Errorf("open snapshot: %w", err)
	}
	defer file.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	part, err := createImageFormPart(writer, filepath.Base(path))
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

	attemptErrors := make([]string, 0, len(c.baseURLs))
	lastRaw := ""
	contentType := writer.FormDataContentType()
	bodyBytes := body.Bytes()

	for _, baseURL := range c.baseURLs {
		req, err := http.NewRequest(http.MethodPost, baseURL+"/detect", bytes.NewReader(bodyBytes))
		if err != nil {
			return DetectResponse{}, "", fmt.Errorf("create request: %w", err)
		}
		req.Header.Set("Content-Type", contentType)

		resp, err := c.http.Do(req)
		if err != nil {
			attemptErrors = append(attemptErrors, fmt.Sprintf("%s (%v)", baseURL, err))
			continue
		}

		raw, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			attemptErrors = append(attemptErrors, fmt.Sprintf("%s (read response: %v)", baseURL, readErr))
			continue
		}
		lastRaw = string(raw)

		if resp.StatusCode >= 500 {
			attemptErrors = append(attemptErrors, fmt.Sprintf("%s (status %d)", baseURL, resp.StatusCode))
			continue
		}
		if resp.StatusCode >= 300 {
			return DetectResponse{}, string(raw), fmt.Errorf("detector status %d from %s", resp.StatusCode, baseURL)
		}

		var out DetectResponse
		if err := json.Unmarshal(raw, &out); err != nil {
			return DetectResponse{}, string(raw), fmt.Errorf("decode detector response from %s: %w", baseURL, err)
		}
		return out, string(raw), nil
	}

	if len(attemptErrors) == 0 {
		return DetectResponse{}, lastRaw, fmt.Errorf("request detector: no detector URLs configured")
	}
	return DetectResponse{}, lastRaw, fmt.Errorf("request detector failed across %d URL(s): %s", len(c.baseURLs), strings.Join(attemptErrors, " | "))
}

func createImageFormPart(writer *multipart.Writer, fileName string) (io.Writer, error) {
	contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(fileName)))
	if contentType == "" {
		contentType = "image/jpeg"
	}
	header := make(textproto.MIMEHeader)
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, "image", escapeQuotes(fileName)))
	header.Set("Content-Type", contentType)
	return writer.CreatePart(header)
}

func escapeQuotes(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, `"`, `\"`)
	return value
}

func (c *Client) Healthy() bool {
	for _, baseURL := range c.baseURLs {
		req, err := http.NewRequest(http.MethodGet, baseURL+"/health", nil)
		if err != nil {
			continue
		}
		resp, err := c.http.Do(req)
		if err != nil {
			continue
		}
		resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return true
		}
	}
	return false
}

func buildDetectorURLs(primary string) []string {
	candidates := []string{strings.TrimRight(primary, "/")}

	u, err := url.Parse(primary)
	if err == nil {
		host := strings.ToLower(u.Hostname())
		switch host {
		case "ai-worker":
			candidates = append(candidates,
				"http://rtspanda-ai-worker:8090",
				"http://host.docker.internal:8090",
				"http://127.0.0.1:8090",
				"http://localhost:8090",
			)
		case "127.0.0.1", "localhost":
			candidates = append(candidates, "http://ai-worker:8090", "http://rtspanda-ai-worker:8090")
		default:
			// No host-specific fallback.
		}
	}

	seen := make(map[string]struct{}, len(candidates))
	out := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		candidate = strings.TrimRight(strings.TrimSpace(candidate), "/")
		if candidate == "" {
			continue
		}
		if _, exists := seen[candidate]; exists {
			continue
		}
		seen[candidate] = struct{}{}
		out = append(out, candidate)
	}
	if len(out) == 0 {
		return []string{"http://127.0.0.1:8090"}
	}
	return out
}
