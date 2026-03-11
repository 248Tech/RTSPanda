package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/rtspanda/rtspanda/internal/alerts"
)

// AlertService is the interface the API requires.
type AlertService interface {
	ListRules(cameraID string) ([]alerts.AlertRule, error)
	GetRule(id string) (alerts.AlertRule, error)
	CreateRule(input alerts.CreateRuleInput) (alerts.AlertRule, error)
	UpdateRule(id string, input alerts.UpdateRuleInput) (alerts.AlertRule, error)
	DeleteRule(id string) error
	ListEvents(ruleID string) ([]alerts.AlertEvent, error)
	ListEventsByCamera(cameraID string) ([]alerts.AlertEvent, error)
	TriggerEvent(input alerts.CreateEventInput) (alerts.AlertEvent, error)
}

// handleListAlertRules: GET /api/v1/cameras/{id}/alerts
func (s *server) handleListAlertRules(w http.ResponseWriter, r *http.Request) {
	cameraID := r.PathValue("id")
	rules, err := s.alertSvc.ListRules(cameraID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rules)
}

// handleCreateAlertRule: POST /api/v1/cameras/{id}/alerts
func (s *server) handleCreateAlertRule(w http.ResponseWriter, r *http.Request) {
	cameraID := r.PathValue("id")

	var input alerts.CreateRuleInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	input.CameraID = cameraID

	rule, err := s.alertSvc.CreateRule(input)
	if err != nil {
		if errors.Is(err, alerts.ErrInvalid) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, rule)
}

// handleUpdateAlertRule: PUT /api/v1/alerts/{id}
func (s *server) handleUpdateAlertRule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var input alerts.UpdateRuleInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	rule, err := s.alertSvc.UpdateRule(id, input)
	if err != nil {
		if errors.Is(err, alerts.ErrNotFound) {
			writeError(w, http.StatusNotFound, "alert rule not found")
			return
		}
		if errors.Is(err, alerts.ErrInvalid) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, rule)
}

// handleDeleteAlertRule: DELETE /api/v1/alerts/{id}
func (s *server) handleDeleteAlertRule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.alertSvc.DeleteRule(id); err != nil {
		if errors.Is(err, alerts.ErrNotFound) {
			writeError(w, http.StatusNotFound, "alert rule not found")
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// handleListAlertEvents: GET /api/v1/alerts/{id}/events
func (s *server) handleListAlertEvents(w http.ResponseWriter, r *http.Request) {
	ruleID := r.PathValue("id")
	events, err := s.alertSvc.ListEvents(ruleID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, events)
}

// handleTriggerAlertEvent: POST /api/v1/alerts/{id}/events
// Intended for webhooks / external AI systems to report a detection.
func (s *server) handleTriggerAlertEvent(w http.ResponseWriter, r *http.Request) {
	ruleID := r.PathValue("id")

	var input alerts.CreateEventInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	input.RuleID = ruleID

	event, err := s.alertSvc.TriggerEvent(input)
	if err != nil {
		if errors.Is(err, alerts.ErrNotFound) {
			writeError(w, http.StatusNotFound, "alert rule not found")
			return
		}
		if errors.Is(err, alerts.ErrInvalid) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, event)
}

// handleListCameraAlertEvents: GET /api/v1/cameras/{id}/alert-events
func (s *server) handleListCameraAlertEvents(w http.ResponseWriter, r *http.Request) {
	cameraID := r.PathValue("id")
	events, err := s.alertSvc.ListEventsByCamera(cameraID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, events)
}
