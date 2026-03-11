package alerts

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

// --- Rules ---

func (s *Service) ListRules(cameraID string) ([]AlertRule, error) {
	return s.repo.ListRules(cameraID)
}

func (s *Service) GetRule(id string) (AlertRule, error) {
	return s.repo.GetRule(id)
}

func (s *Service) CreateRule(input CreateRuleInput) (AlertRule, error) {
	if input.CameraID == "" {
		return AlertRule{}, fmt.Errorf("%w: camera_id is required", ErrInvalid)
	}
	if input.Name == "" {
		return AlertRule{}, fmt.Errorf("%w: name is required", ErrInvalid)
	}
	if !validType(input.Type) {
		return AlertRule{}, fmt.Errorf("%w: type must be motion, connectivity, or object_detection", ErrInvalid)
	}

	enabled := true
	if input.Enabled != nil {
		enabled = *input.Enabled
	}
	config := input.Config
	if config == "" {
		config = "{}"
	}

	now := time.Now()
	rule := AlertRule{
		ID:        uuid.New().String(),
		CameraID:  input.CameraID,
		Name:      input.Name,
		Type:      input.Type,
		Enabled:   enabled,
		Config:    config,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.CreateRule(rule); err != nil {
		return AlertRule{}, fmt.Errorf("create alert rule: %w", err)
	}
	return rule, nil
}

func (s *Service) UpdateRule(id string, input UpdateRuleInput) (AlertRule, error) {
	rule, err := s.repo.GetRule(id)
	if err != nil {
		return AlertRule{}, err
	}

	if input.Name != nil {
		if *input.Name == "" {
			return AlertRule{}, fmt.Errorf("%w: name cannot be empty", ErrInvalid)
		}
		rule.Name = *input.Name
	}
	if input.Type != nil {
		if !validType(*input.Type) {
			return AlertRule{}, fmt.Errorf("%w: invalid type", ErrInvalid)
		}
		rule.Type = *input.Type
	}
	if input.Enabled != nil {
		rule.Enabled = *input.Enabled
	}
	if input.Config != nil {
		rule.Config = *input.Config
	}

	if err := s.repo.UpdateRule(rule); err != nil {
		return AlertRule{}, fmt.Errorf("update alert rule: %w", err)
	}
	return rule, nil
}

func (s *Service) DeleteRule(id string) error {
	return s.repo.DeleteRule(id)
}

// --- Events ---

func (s *Service) ListEvents(ruleID string) ([]AlertEvent, error) {
	return s.repo.ListEvents(ruleID, 50)
}

func (s *Service) ListEventsByCamera(cameraID string) ([]AlertEvent, error) {
	return s.repo.ListEventsByCamera(cameraID, 100)
}

// TriggerEvent records an alert firing. Called by detection integrations or webhooks.
func (s *Service) TriggerEvent(input CreateEventInput) (AlertEvent, error) {
	if input.RuleID == "" {
		return AlertEvent{}, fmt.Errorf("%w: rule_id is required", ErrInvalid)
	}

	// Validate rule exists
	rule, err := s.repo.GetRule(input.RuleID)
	if err != nil {
		return AlertEvent{}, err
	}

	metadata := input.Metadata
	if metadata == "" {
		metadata = "{}"
	}

	e := AlertEvent{
		ID:           uuid.New().String(),
		RuleID:       input.RuleID,
		CameraID:     rule.CameraID,
		TriggeredAt:  time.Now(),
		SnapshotPath: input.SnapshotPath,
		Metadata:     metadata,
	}
	if err := s.repo.CreateEvent(e); err != nil {
		return AlertEvent{}, fmt.Errorf("create alert event: %w", err)
	}
	return e, nil
}

func validType(t AlertType) bool {
	return t == AlertTypeMotion || t == AlertTypeConnectivity || t == AlertTypeObjectDetection
}
