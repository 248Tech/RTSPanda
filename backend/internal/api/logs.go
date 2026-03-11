package api

import (
	"net/http"
)

// LogBuffer provides recent log lines for the logs API.
type LogBuffer interface {
	Lines() []string
}

func (s *server) handleLogs(w http.ResponseWriter, r *http.Request) {
	if s.logBuf == nil {
		writeJSON(w, http.StatusOK, map[string]any{"lines": []string{}})
		return
	}
	lines := s.logBuf.Lines()
	writeJSON(w, http.StatusOK, map[string]any{"lines": lines})
}
