package notifications

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rtspanda/rtspanda/internal/cameras"
	"github.com/rtspanda/rtspanda/internal/detections"
)

const defaultMotionClipDuration = 4 * time.Second

type DiscordNotifierConfig struct {
	FFmpegBin          string
	MotionClipDuration time.Duration
}

type DiscordNotifier struct {
	httpClient *http.Client

	ffmpegBin          string
	motionClipDuration time.Duration
	motionClipEnabled  bool

	mu       sync.Mutex
	lastSent map[string]time.Time
}

type webhookAttachment struct {
	Path string
	Name string
}

type clipCaptureOptions struct {
	PreferredFormat string
	Duration        time.Duration
}

func NewDiscordNotifier(timeout time.Duration, cfg DiscordNotifierConfig) *DiscordNotifier {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	ffmpegBin := strings.TrimSpace(cfg.FFmpegBin)
	if ffmpegBin == "" {
		ffmpegBin = "ffmpeg"
	}
	resolvedFFmpegBin := ffmpegBin
	motionClipEnabled := false
	if p, err := exec.LookPath(ffmpegBin); err == nil {
		resolvedFFmpegBin = p
		motionClipEnabled = true
	}

	clipDuration := cfg.MotionClipDuration
	if clipDuration <= 0 {
		clipDuration = defaultMotionClipDuration
	}

	return &DiscordNotifier{
		httpClient:         &http.Client{Timeout: timeout},
		ffmpegBin:          resolvedFFmpegBin,
		motionClipDuration: clipDuration,
		motionClipEnabled:  motionClipEnabled,
		lastSent:           make(map[string]time.Time),
	}
}

func (n *DiscordNotifier) NotifyDetectionEvents(
	ctx context.Context,
	camera cameras.Camera,
	snapshot detections.Snapshot,
	events []detections.Event,
) error {
	if len(events) == 0 {
		return nil
	}
	if !camera.DiscordAlertsEnabled {
		return nil
	}
	if !camera.DiscordTriggerOnDetection {
		return nil
	}

	webhookURL := strings.TrimSpace(camera.DiscordWebhookURL)
	if webhookURL == "" {
		return nil
	}

	if !n.allowSend(camera.ID, camera.DiscordCooldownSeconds) {
		return nil
	}

	if err := n.sendDetectionWebhook(ctx, webhookURL, camera, snapshot, events); err != nil {
		return err
	}
	n.markSent(camera.ID)
	return nil
}

func (n *DiscordNotifier) SendCameraSnapshot(
	ctx context.Context,
	camera cameras.Camera,
	snapshot detections.Snapshot,
	includeMotionClip bool,
) error {
	webhookURL := strings.TrimSpace(camera.DiscordWebhookURL)
	if webhookURL == "" {
		return fmt.Errorf("discord webhook URL is not configured for this camera")
	}

	mention := strings.TrimSpace(camera.DiscordMention)
	snapshotAttachment := buildAttachment(snapshot.Path, "snapshot.jpg")
	attachments := make([]webhookAttachment, 0, 2)
	if snapshotAttachment != nil {
		attachments = append(attachments, *snapshotAttachment)
	}

	if includeMotionClip {
		clip, err := n.captureMotionClip(ctx, camera, clipCaptureOptions{
			PreferredFormat: "webm",
			Duration:        time.Duration(camera.DiscordMotionClipSeconds) * time.Second,
		})
		if err != nil {
			log.Printf("notifications: motion clip capture failed camera=%s err=%v", camera.ID, err)
		} else if clip != nil {
			attachments = append(attachments, *clip)
			defer os.Remove(clip.Path)
		}
	}

	description := "Manual screenshot requested from the camera stream."
	if includeMotionClip {
		description = "Manual screenshot requested from the camera stream. Motion clip attached when available."
	}

	embed := discordEmbed{
		Title:       "Manual Camera Snapshot",
		Description: description,
		Color:       5793266,
		Timestamp:   snapshot.Timestamp.UTC().Format(time.RFC3339),
		Footer:      &discordFooter{Text: "RTSPanda"},
	}
	if snapshotAttachment != nil {
		embed.Image = &discordImage{URL: "attachment://" + snapshotAttachment.Name}
	}

	payload := discordWebhookPayload{
		Content: strings.TrimSpace(strings.Join([]string{
			mention,
			fmt.Sprintf(
				"Manual snapshot from **%s** at %s",
				camera.Name,
				snapshot.Timestamp.UTC().Format(time.RFC3339),
			),
		}, "\n")),
		Embeds:          []discordEmbed{embed},
		AllowedMentions: &discordAllowedMentions{Parse: allowedMentionParse(mention)},
	}

	return n.sendWebhookPayload(ctx, webhookURL, payload, attachments)
}

func (n *DiscordNotifier) SendCameraRecording(
	ctx context.Context,
	camera cameras.Camera,
	durationSeconds int,
	format string,
) error {
	webhookURL := strings.TrimSpace(camera.DiscordWebhookURL)
	if webhookURL == "" {
		return fmt.Errorf("discord webhook URL is not configured for this camera")
	}

	if durationSeconds <= 0 {
		durationSeconds = camera.DiscordRecordDurationSeconds
	}
	if durationSeconds <= 0 {
		durationSeconds = 60
	}
	if strings.TrimSpace(format) == "" {
		format = camera.DiscordRecordFormat
	}
	if strings.TrimSpace(format) == "" {
		format = "webp"
	}

	clip, err := n.captureMotionClip(ctx, camera, clipCaptureOptions{
		PreferredFormat: format,
		Duration:        time.Duration(durationSeconds) * time.Second,
	})
	if err != nil {
		return err
	}
	if clip == nil {
		return fmt.Errorf("recording clip is unavailable")
	}
	defer os.Remove(clip.Path)

	mention := strings.TrimSpace(camera.DiscordMention)
	payload := discordWebhookPayload{
		Content: strings.TrimSpace(strings.Join([]string{
			mention,
			fmt.Sprintf(
				"Manual recording from **%s** (%ds, %s)",
				camera.Name,
				durationSeconds,
				strings.TrimPrefix(strings.ToLower(filepath.Ext(clip.Name)), "."),
			),
		}, "\n")),
		Embeds: []discordEmbed{
			{
				Title:       "Manual Camera Recording",
				Description: "Requested from RTSPanda camera view.",
				Color:       5793266,
				Timestamp:   time.Now().UTC().Format(time.RFC3339),
				Footer:      &discordFooter{Text: "RTSPanda"},
			},
		},
		AllowedMentions: &discordAllowedMentions{Parse: allowedMentionParse(mention)},
	}

	return n.sendWebhookPayload(ctx, webhookURL, payload, []webhookAttachment{*clip})
}

func (n *DiscordNotifier) allowSend(cameraID string, cooldownSeconds int) bool {
	if cooldownSeconds <= 0 {
		return true
	}

	cooldown := time.Duration(cooldownSeconds) * time.Second
	now := time.Now()

	n.mu.Lock()
	defer n.mu.Unlock()

	last, ok := n.lastSent[cameraID]
	if !ok {
		return true
	}
	return now.Sub(last) >= cooldown
}

func (n *DiscordNotifier) markSent(cameraID string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.lastSent[cameraID] = time.Now()
}

func (n *DiscordNotifier) sendDetectionWebhook(
	ctx context.Context,
	webhookURL string,
	camera cameras.Camera,
	snapshot detections.Snapshot,
	events []detections.Event,
) error {
	mention := strings.TrimSpace(camera.DiscordMention)

	snapshotAttachment := buildAttachment(snapshot.Path, "snapshot.jpg")
	attachments := make([]webhookAttachment, 0, 2)
	if snapshotAttachment != nil {
		attachments = append(attachments, *snapshotAttachment)
	}

	embed := discordEmbed{
		Title:       "YOLOv8 Detection Event",
		Description: "Per-camera AI tracking reported one or more detections.",
		Color:       5793266,
		Timestamp:   events[0].CreatedAt.UTC().Format(time.RFC3339),
		Fields:      buildFields(events),
		Footer:      &discordFooter{Text: "RTSPanda"},
	}
	if snapshotAttachment != nil {
		embed.Image = &discordImage{URL: "attachment://" + snapshotAttachment.Name}
	}

	if camera.DiscordIncludeMotionClip {
		if clip, err := n.captureMotionClip(ctx, camera, clipCaptureOptions{
			PreferredFormat: "webm",
			Duration:        time.Duration(camera.DiscordMotionClipSeconds) * time.Second,
		}); err != nil {
			log.Printf("notifications: motion clip capture failed camera=%s err=%v", camera.ID, err)
		} else if clip != nil {
			attachments = append(attachments, *clip)
			defer os.Remove(clip.Path)
			embed.Fields = append(embed.Fields, discordField{
				Name:   "Motion Clip",
				Value:  "Attached",
				Inline: true,
			})
		}
	}

	payload := discordWebhookPayload{
		Content: strings.TrimSpace(strings.Join([]string{
			mention,
			fmt.Sprintf(
				"Camera **%s** detected %d object(s): %s",
				camera.Name,
				len(events),
				summarizeDetections(events),
			),
		}, "\n")),
		Embeds:          []discordEmbed{embed},
		AllowedMentions: &discordAllowedMentions{Parse: allowedMentionParse(mention)},
	}

	return n.sendWebhookPayload(ctx, webhookURL, payload, attachments)
}

func (n *DiscordNotifier) sendWebhookPayload(
	ctx context.Context,
	webhookURL string,
	payload discordWebhookPayload,
	attachments []webhookAttachment,
) error {
	if len(attachments) == 0 {
		return n.sendJSONWebhook(ctx, webhookURL, payload)
	}

	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode discord payload: %w", err)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("payload_json", string(payloadJSON)); err != nil {
		return fmt.Errorf("write payload_json: %w", err)
	}

	uploaded := 0
	for _, attachment := range attachments {
		file, err := os.Open(attachment.Path)
		if err != nil {
			continue
		}

		part, err := writer.CreateFormFile(fmt.Sprintf("files[%d]", uploaded), attachment.Name)
		if err != nil {
			file.Close()
			return fmt.Errorf("create multipart file: %w", err)
		}
		if _, err := io.Copy(part, file); err != nil {
			file.Close()
			return fmt.Errorf("copy attachment: %w", err)
		}
		file.Close()
		uploaded++
	}

	if uploaded == 0 {
		return n.sendJSONWebhook(ctx, webhookURL, payload)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, &body)
	if err != nil {
		return fmt.Errorf("create discord request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("post discord webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("discord webhook returned %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	return nil
}

func (n *DiscordNotifier) sendJSONWebhook(
	ctx context.Context,
	webhookURL string,
	payload discordWebhookPayload,
) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("encode discord payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("create discord request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("post discord webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("discord webhook returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func (n *DiscordNotifier) captureMotionClip(ctx context.Context, camera cameras.Camera, opts clipCaptureOptions) (*webhookAttachment, error) {
	if !n.motionClipEnabled {
		return nil, nil
	}

	rtspURL := strings.TrimSpace(camera.RTSPURL)
	if rtspURL == "" {
		return nil, nil
	}

	cameraToken := sanitizeFileToken(camera.Name)
	if cameraToken == "" {
		cameraToken = sanitizeFileToken(camera.ID)
	}
	if cameraToken == "" {
		cameraToken = "camera"
	}
	ts := time.Now().UTC().Format("20060102T150405Z")
	duration := opts.Duration
	if duration <= 0 {
		duration = time.Duration(camera.DiscordMotionClipSeconds) * time.Second
	}
	if duration <= 0 {
		duration = n.motionClipDuration
	}
	if duration <= 0 {
		duration = defaultMotionClipDuration
	}
	durationArg := fmt.Sprintf("%.2f", duration.Seconds())

	formatOrder := clipFormatOrder(opts.PreferredFormat)
	var lastErr error
	for _, format := range formatOrder {
		attachment, err := n.captureClipFormat(ctx, rtspURL, cameraToken, ts, durationArg, format)
		if err == nil {
			return attachment, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("failed to capture motion clip")
}

func (n *DiscordNotifier) captureClipFormat(
	ctx context.Context,
	rtspURL string,
	cameraToken string,
	ts string,
	durationArg string,
	format string,
) (*webhookAttachment, error) {
	ext := strings.ToLower(strings.TrimSpace(format))
	if ext == "" {
		ext = "webm"
	}

	name := fmt.Sprintf("%s_%s.%s", cameraToken, ts, ext)
	path := filepath.Join(os.TempDir(), "rtspanda_"+name)

	args := []string{
		"-hide_banner",
		"-loglevel", "error",
		"-i", rtspURL,
		"-an",
		"-t", durationArg,
	}
	switch ext {
	case "webm":
		args = append(args,
			"-vf", "fps=10,scale=960:-2:flags=lanczos",
			"-c:v", "libvpx-vp9",
			"-deadline", "realtime",
			"-cpu-used", "5",
			"-crf", "35",
			"-b:v", "0",
			"-pix_fmt", "yuv420p",
		)
	case "webp":
		args = append(args,
			"-vf", "fps=8,scale=960:-2:flags=lanczos",
			"-loop", "0",
			"-c:v", "libwebp",
			"-q:v", "70",
		)
	case "gif":
		args = append(args,
			"-vf", "fps=8,scale=640:-2:flags=lanczos",
			"-loop", "0",
		)
	default:
		return nil, fmt.Errorf("unsupported clip format %q", format)
	}
	args = append(args, "-y", path)

	output, err := runRTSPFFmpegCommand(ctx, n.ffmpegBin, args)
	if err != nil {
		_ = os.Remove(path)
		if !isCodecOrMuxerError(output) {
			return nil, fmt.Errorf("capture %s motion clip: %w (%s)", ext, err, strings.TrimSpace(string(output)))
		}
		return nil, fmt.Errorf("capture %s motion clip failed: %s", ext, strings.TrimSpace(string(output)))
	}

	return &webhookAttachment{Path: path, Name: name}, nil
}

func clipFormatOrder(preferred string) []string {
	switch strings.ToLower(strings.TrimSpace(preferred)) {
	case "webp":
		return []string{"webp", "webm", "gif"}
	case "gif":
		return []string{"gif", "webm", "webp"}
	default:
		return []string{"webm", "webp", "gif"}
	}
}

func buildAttachment(path string, defaultName string) *webhookAttachment {
	p := strings.TrimSpace(path)
	if p == "" {
		return nil
	}
	if _, err := os.Stat(p); err != nil {
		return nil
	}
	name := filepath.Base(p)
	if name == "" || name == "." || name == string(filepath.Separator) {
		name = defaultName
	}
	return &webhookAttachment{
		Path: p,
		Name: name,
	}
}

func runRTSPFFmpegCommand(ctx context.Context, ffmpegBin string, args []string) ([]byte, error) {
	withRWTimeout := append(
		[]string{"-rtsp_transport", "tcp", "-rw_timeout", "5000000"},
		args...,
	)
	output, err := runFFmpegCommand(ctx, ffmpegBin, withRWTimeout)
	if err == nil {
		return output, nil
	}
	if !isMissingFFmpegOption(output, "rw_timeout") {
		return output, err
	}

	withTimeout := append(
		[]string{"-rtsp_transport", "tcp", "-timeout", "5000000"},
		args...,
	)
	output, err = runFFmpegCommand(ctx, ffmpegBin, withTimeout)
	if err == nil {
		return output, nil
	}
	if !isMissingFFmpegOption(output, "timeout") {
		return output, err
	}

	withoutTimeout := append([]string{"-rtsp_transport", "tcp"}, args...)
	return runFFmpegCommand(ctx, ffmpegBin, withoutTimeout)
}

func runFFmpegCommand(ctx context.Context, ffmpegBin string, args []string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, ffmpegBin, args...)
	return cmd.CombinedOutput()
}

func isMissingFFmpegOption(output []byte, option string) bool {
	msg := strings.ToLower(string(output))
	return strings.Contains(msg, "option "+strings.ToLower(option)+" not found")
}

func isCodecOrMuxerError(output []byte) bool {
	msg := strings.ToLower(string(output))
	return strings.Contains(msg, "unknown encoder") ||
		strings.Contains(msg, "encoder not found") ||
		strings.Contains(msg, "error selecting an encoder") ||
		strings.Contains(msg, "unknown format") ||
		strings.Contains(msg, "could not write header")
}

func sanitizeFileToken(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return ""
	}

	var b strings.Builder
	lastDash := false
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}

	out := strings.Trim(b.String(), "-")
	if out == "" {
		return ""
	}
	return out
}

func summarizeDetections(events []detections.Event) string {
	if len(events) == 0 {
		return "none"
	}

	parts := make([]string, 0, len(events))
	for _, event := range events {
		parts = append(parts, fmt.Sprintf("%s (%.0f%%)", event.ObjectLabel, event.Confidence*100))
	}
	return strings.Join(parts, ", ")
}

func buildFields(events []detections.Event) []discordField {
	max := len(events)
	if max > 10 {
		max = 10
	}

	fields := make([]discordField, 0, max)
	for i := 0; i < max; i++ {
		event := events[i]
		fields = append(fields, discordField{
			Name:   fmt.Sprintf("Detection %d", i+1),
			Value:  fmt.Sprintf("%s\nConfidence: %.1f%%\nBBox: x=%d y=%d w=%d h=%d", event.ObjectLabel, event.Confidence*100, event.BBox.X, event.BBox.Y, event.BBox.Width, event.BBox.Height),
			Inline: true,
		})
	}
	return fields
}

func allowedMentionParse(mention string) []string {
	if strings.TrimSpace(mention) == "" {
		return []string{}
	}
	return []string{"users", "roles", "everyone"}
}

type discordWebhookPayload struct {
	Content         string                  `json:"content,omitempty"`
	Embeds          []discordEmbed          `json:"embeds,omitempty"`
	AllowedMentions *discordAllowedMentions `json:"allowed_mentions,omitempty"`
}

type discordAllowedMentions struct {
	Parse []string `json:"parse,omitempty"`
}

type discordEmbed struct {
	Title       string         `json:"title,omitempty"`
	Description string         `json:"description,omitempty"`
	Color       int            `json:"color,omitempty"`
	Timestamp   string         `json:"timestamp,omitempty"`
	Fields      []discordField `json:"fields,omitempty"`
	Image       *discordImage  `json:"image,omitempty"`
	Footer      *discordFooter `json:"footer,omitempty"`
}

type discordField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

type discordImage struct {
	URL string `json:"url"`
}

type discordFooter struct {
	Text string `json:"text"`
}
