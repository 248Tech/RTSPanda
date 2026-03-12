package cameras

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) List() ([]Camera, error) {
	return s.repo.List()
}

func (s *Service) Get(id string) (Camera, error) {
	return s.repo.GetByID(id)
}

func (s *Service) Create(input CreateInput) (Camera, error) {
	if input.Name == "" {
		return Camera{}, fmt.Errorf("%w: name is required", ErrInvalid)
	}
	if input.RTSPURL == "" {
		return Camera{}, fmt.Errorf("%w: rtsp_url is required", ErrInvalid)
	}
	if !strings.HasPrefix(strings.TrimSpace(input.RTSPURL), "rtsp://") {
		return Camera{}, fmt.Errorf("%w: rtsp_url must start with rtsp://", ErrInvalid)
	}

	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	recordEnabled := false
	if input.RecordEnabled != nil {
		recordEnabled = *input.RecordEnabled
	}
	if input.DetectionSampleSeconds != nil && *input.DetectionSampleSeconds <= 0 {
		return Camera{}, fmt.Errorf("%w: detection_sample_seconds must be > 0", ErrInvalid)
	}

	now := time.Now()
	c := Camera{
		ID:                     uuid.New().String(),
		Name:                   input.Name,
		RTSPURL:                input.RTSPURL,
		Enabled:                enabled,
		RecordEnabled:          recordEnabled,
		DetectionSampleSeconds: input.DetectionSampleSeconds,
		Position:               0,
		CreatedAt:              now,
		UpdatedAt:              now,
	}
	if err := s.repo.Create(c); err != nil {
		return Camera{}, fmt.Errorf("create camera: %w", err)
	}
	return c, nil
}

func (s *Service) Update(id string, input UpdateInput) (Camera, error) {
	c, err := s.repo.GetByID(id)
	if err != nil {
		return Camera{}, err
	}

	if input.Name != nil {
		if *input.Name == "" {
			return Camera{}, fmt.Errorf("%w: name cannot be empty", ErrInvalid)
		}
		c.Name = *input.Name
	}
	if input.RTSPURL != nil {
		if *input.RTSPURL == "" {
			return Camera{}, fmt.Errorf("%w: rtsp_url cannot be empty", ErrInvalid)
		}
		if !strings.HasPrefix(strings.TrimSpace(*input.RTSPURL), "rtsp://") {
			return Camera{}, fmt.Errorf("%w: rtsp_url must start with rtsp://", ErrInvalid)
		}
		c.RTSPURL = *input.RTSPURL
	}
	if input.Enabled != nil {
		c.Enabled = *input.Enabled
	}
	if input.RecordEnabled != nil {
		c.RecordEnabled = *input.RecordEnabled
	}
	if input.DetectionSampleSeconds != nil {
		if *input.DetectionSampleSeconds <= 0 {
			return Camera{}, fmt.Errorf("%w: detection_sample_seconds must be > 0", ErrInvalid)
		}
		c.DetectionSampleSeconds = input.DetectionSampleSeconds
	}
	if input.Position != nil {
		c.Position = *input.Position
	}

	if err := s.repo.Update(c); err != nil {
		return Camera{}, fmt.Errorf("update camera: %w", err)
	}
	return c, nil
}

func (s *Service) Delete(id string) error {
	return s.repo.Delete(id)
}
