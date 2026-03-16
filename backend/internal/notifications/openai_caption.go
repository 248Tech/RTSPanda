package notifications

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rtspanda/rtspanda/internal/cameras"
	"github.com/rtspanda/rtspanda/internal/detections"
)

type openAIChatCompletionsRequest struct {
	Model       string                 `json:"model"`
	Temperature float64                `json:"temperature,omitempty"`
	MaxTokens   int                    `json:"max_tokens,omitempty"`
	Messages    []openAIMessageRequest `json:"messages"`
}

type openAIMessageRequest struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type openAIContentBlock struct {
	Type     string                 `json:"type"`
	Text     string                 `json:"text,omitempty"`
	ImageURL map[string]interface{} `json:"image_url,omitempty"`
}

type openAIChatCompletionsResponse struct {
	Choices []struct {
		Message struct {
			Content json.RawMessage `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error,omitempty"`
}

type sceneCaption struct {
	Short   string `json:"short"`
	Verbose string `json:"verbose"`
}

func (n *DiscordNotifier) describeSnapshot(
	ctx context.Context,
	camera cameras.Camera,
	snapshotPath string,
	events []detections.Event,
) (string, string) {
	if n.openAIConfigProvider == nil {
		return "", ""
	}
	path := strings.TrimSpace(snapshotPath)
	if path == "" {
		return "", ""
	}

	cfg, err := n.openAIConfigProvider()
	if err != nil {
		log.Printf("notifications: openai config load failed camera=%s err=%v", camera.ID, err)
		return "", ""
	}
	if !cfg.Enabled || strings.TrimSpace(cfg.APIKey) == "" {
		return "", ""
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = "gpt-4o-mini"
	}

	imageDataURL, err := imageFileDataURL(path)
	if err != nil {
		log.Printf("notifications: openai snapshot read failed camera=%s path=%s err=%v", camera.ID, path, err)
		return "", ""
	}

	detectionHint := summarizeDetections(events)
	if detectionHint == "none" {
		detectionHint = ""
	}

	systemPrompt := "You write concise home-security camera alert text. Do not invent details. If uncertain, use words like possible/probable."
	userPrompt := strings.TrimSpace(strings.Join([]string{
		fmt.Sprintf("Camera name: %s", camera.Name),
		fmt.Sprintf("Timestamp: %s", time.Now().UTC().Format(time.RFC3339)),
		func() string {
			if detectionHint == "" {
				return "Detection hint: none"
			}
			return "Detection hint: " + detectionHint
		}(),
		"Return strict JSON only with keys:",
		`{"short":"3-8 word label, e.g. person at front door","verbose":"one sentence up to 24 words"}`,
	}, "\n"))

	requestBody := openAIChatCompletionsRequest{
		Model:       model,
		Temperature: 0.2,
		MaxTokens:   120,
		Messages: []openAIMessageRequest{
			{
				Role:    "system",
				Content: systemPrompt,
			},
			{
				Role: "user",
				Content: []openAIContentBlock{
					{
						Type: "text",
						Text: userPrompt,
					},
					{
						Type: "image_url",
						ImageURL: map[string]interface{}{
							"url": imageDataURL,
						},
					},
				},
			},
		},
	}

	rawReq, err := json.Marshal(requestBody)
	if err != nil {
		log.Printf("notifications: openai request encode failed camera=%s err=%v", camera.ID, err)
		return "", ""
	}

	aiCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(aiCtx, http.MethodPost, "https://api.openai.com/v1/chat/completions", bytes.NewReader(rawReq))
	if err != nil {
		log.Printf("notifications: openai request create failed camera=%s err=%v", camera.ID, err)
		return "", ""
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.openAIHTTPClient.Do(req)
	if err != nil {
		log.Printf("notifications: openai request failed camera=%s err=%v", camera.ID, err)
		return "", ""
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("notifications: openai response read failed camera=%s err=%v", camera.ID, err)
		return "", ""
	}
	if resp.StatusCode >= 300 {
		log.Printf("notifications: openai response bad status camera=%s status=%d body=%s", camera.ID, resp.StatusCode, strings.TrimSpace(string(respBody)))
		return "", ""
	}

	var out openAIChatCompletionsResponse
	if err := json.Unmarshal(respBody, &out); err != nil {
		log.Printf("notifications: openai response decode failed camera=%s err=%v", camera.ID, err)
		return "", ""
	}
	if out.Error != nil {
		log.Printf("notifications: openai error camera=%s type=%s msg=%s", camera.ID, out.Error.Type, out.Error.Message)
		return "", ""
	}
	if len(out.Choices) == 0 {
		return "", ""
	}

	content := extractOpenAIContentText(out.Choices[0].Message.Content)
	short, verbose := parseSceneCaption(content)
	if short == "" && verbose == "" {
		return "", ""
	}
	log.Printf("notifications: openai caption camera=%s short=%q verbose=%q", camera.ID, short, verbose)
	return short, verbose
}

func imageFileDataURL(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(path)))
	if contentType == "" {
		contentType = "image/jpeg"
	}
	return "data:" + contentType + ";base64," + base64.StdEncoding.EncodeToString(raw), nil
}

func extractOpenAIContentText(raw json.RawMessage) string {
	var plain string
	if err := json.Unmarshal(raw, &plain); err == nil {
		return strings.TrimSpace(plain)
	}

	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return ""
	}
	parts := make([]string, 0, len(blocks))
	for _, block := range blocks {
		if block.Type == "text" && strings.TrimSpace(block.Text) != "" {
			parts = append(parts, strings.TrimSpace(block.Text))
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}

func parseSceneCaption(raw string) (string, string) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", ""
	}

	var caption sceneCaption
	if err := json.Unmarshal([]byte(value), &caption); err == nil {
		return normalizeCaptionLine(caption.Short, 80), normalizeCaptionLine(caption.Verbose, 240)
	}

	trimmed := strings.Trim(value, "`")
	trimmed = strings.TrimPrefix(trimmed, "json")
	trimmed = strings.TrimSpace(trimmed)
	if err := json.Unmarshal([]byte(trimmed), &caption); err == nil {
		return normalizeCaptionLine(caption.Short, 80), normalizeCaptionLine(caption.Verbose, 240)
	}

	lines := splitNonEmptyLines(value)
	if len(lines) == 0 {
		return "", ""
	}
	short := normalizeCaptionLine(lines[0], 80)
	verbose := short
	if len(lines) > 1 {
		verbose = normalizeCaptionLine(lines[1], 240)
	}
	return short, verbose
}

func splitNonEmptyLines(raw string) []string {
	parts := strings.Split(raw, "\n")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}

func normalizeCaptionLine(value string, maxLen int) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, `"`)
	value = strings.Join(strings.Fields(value), " ")
	if value == "" {
		return ""
	}
	if maxLen > 0 && len(value) > maxLen {
		return strings.TrimSpace(value[:maxLen-1]) + "…"
	}
	return value
}
