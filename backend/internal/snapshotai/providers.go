package snapshotai

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// VisionResult is the structured output from a snapshot vision API call.
type VisionResult struct {
	Detected        bool
	Label           string  // "person", "vehicle", "package delivery", etc.
	Confidence      string  // "low", "medium", or "high"
	ConfidenceFloat float64 // mapped numeric value: low=0.35, medium=0.65, high=0.90
	Summary         string  // one-sentence description from the model
	RawJSON         string  // raw model response for storage
}

// VisionClient submits a JPEG image to an external vision AI and returns
// a structured interpretation.
type VisionClient interface {
	Describe(ctx context.Context, imageBytes []byte, prompt string) (VisionResult, error)
}

// newVisionClient returns the appropriate VisionClient for the given provider.
// Supported providers: "openai", "claude" (alias: "anthropic").
func newVisionClient(provider, apiKey string) (VisionClient, error) {
	switch strings.ToLower(provider) {
	case "openai":
		return &openAIClient{
			apiKey:     apiKey,
			httpClient: &http.Client{Timeout: 45 * time.Second},
		}, nil
	case "claude", "anthropic":
		return &claudeClient{
			apiKey:     apiKey,
			httpClient: &http.Client{Timeout: 45 * time.Second},
		}, nil
	default:
		return nil, fmt.Errorf("unknown vision provider %q (supported: openai, claude)", provider)
	}
}

// visionResponseSuffix is appended to every user prompt to enforce JSON output.
const visionResponseSuffix = `

Respond ONLY with a single JSON object — no markdown fences, no explanation:
{"detected": true, "label": "brief label", "confidence": "low|medium|high", "summary": "one sentence"}`

// ── OpenAI ────────────────────────────────────────────────────────────────────

type openAIClient struct {
	apiKey     string
	httpClient *http.Client
}

func (c *openAIClient) Describe(ctx context.Context, imageBytes []byte, prompt string) (VisionResult, error) {
	b64 := base64.StdEncoding.EncodeToString(imageBytes)

	payload := map[string]any{
		"model": "gpt-4o-mini",
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{"type": "text", "text": prompt + visionResponseSuffix},
					{"type": "image_url", "image_url": map[string]string{
						"url":    "data:image/jpeg;base64," + b64,
						"detail": "low",
					}},
				},
			},
		},
		"max_tokens": 200,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return VisionResult{}, fmt.Errorf("marshal openai request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.openai.com/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return VisionResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return VisionResult{}, fmt.Errorf("openai request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return VisionResult{}, fmt.Errorf("read openai response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return VisionResult{}, fmt.Errorf("openai API status %d: %s", resp.StatusCode, truncate(string(respBody), 200))
	}

	var apiResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return VisionResult{}, fmt.Errorf("decode openai response: %w", err)
	}
	if len(apiResp.Choices) == 0 {
		return VisionResult{}, fmt.Errorf("openai returned no choices")
	}
	return parseVisionJSON(apiResp.Choices[0].Message.Content)
}

// ── Claude ────────────────────────────────────────────────────────────────────

type claudeClient struct {
	apiKey     string
	httpClient *http.Client
}

func (c *claudeClient) Describe(ctx context.Context, imageBytes []byte, prompt string) (VisionResult, error) {
	b64 := base64.StdEncoding.EncodeToString(imageBytes)

	payload := map[string]any{
		"model":      "claude-haiku-4-5-20251001",
		"max_tokens": 200,
		"messages": []map[string]any{
			{
				"role": "user",
				"content": []map[string]any{
					{
						"type": "image",
						"source": map[string]string{
							"type":       "base64",
							"media_type": "image/jpeg",
							"data":       b64,
						},
					},
					{"type": "text", "text": prompt + visionResponseSuffix},
				},
			},
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return VisionResult{}, fmt.Errorf("marshal claude request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	if err != nil {
		return VisionResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return VisionResult{}, fmt.Errorf("claude request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return VisionResult{}, fmt.Errorf("read claude response: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return VisionResult{}, fmt.Errorf("claude API status %d: %s", resp.StatusCode, truncate(string(respBody), 200))
	}

	var apiResp struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return VisionResult{}, fmt.Errorf("decode claude response: %w", err)
	}
	for _, block := range apiResp.Content {
		if block.Type == "text" {
			return parseVisionJSON(block.Text)
		}
	}
	return VisionResult{}, fmt.Errorf("claude returned no text content")
}

// ── Shared helpers ─────────────────────────────────────────────────────────────

// parseVisionJSON parses the structured JSON response expected from every provider.
func parseVisionJSON(raw string) (VisionResult, error) {
	raw = strings.TrimSpace(raw)
	// Strip any accidental markdown fences
	raw = strings.TrimPrefix(raw, "```json")
	raw = strings.TrimPrefix(raw, "```")
	raw = strings.TrimSuffix(raw, "```")
	raw = strings.TrimSpace(raw)

	var parsed struct {
		Detected   bool   `json:"detected"`
		Label      string `json:"label"`
		Confidence string `json:"confidence"`
		Summary    string `json:"summary"`
	}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return VisionResult{}, fmt.Errorf("parse vision response %q: %w", truncate(raw, 120), err)
	}
	return VisionResult{
		Detected:        parsed.Detected,
		Label:           parsed.Label,
		Confidence:      strings.ToLower(parsed.Confidence),
		ConfidenceFloat: confidenceToFloat(parsed.Confidence),
		Summary:         parsed.Summary,
		RawJSON:         raw,
	}, nil
}

func confidenceToFloat(c string) float64 {
	switch strings.ToLower(c) {
	case "low":
		return 0.35
	case "medium":
		return 0.65
	case "high":
		return 0.90
	default:
		return 0.50
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
