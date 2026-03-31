package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"newclaw/internal/agents"
	"newclaw/internal/skills"
)

type Server struct {
	agents *agents.Service
	root   string
}

func New(root string, agentSvc *agents.Service) *Server {
	return &Server{agents: agentSvc, root: root}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.healthz)
	mux.HandleFunc("/v1/sessions", s.sessions)
	mux.HandleFunc("/v1/sessions/", s.sessionRoutes)
	mux.HandleFunc("/v1/skills", s.skills)
	return mux
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true})
}

func (s *Server) sessions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		st, err := s.agents.CreateSession("main")
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusCreated, st)
	case http.MethodGet:
		list, err := s.agents.ListSessions("main")
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, list)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (s *Server) sessionRoutes(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.TrimPrefix(r.URL.Path, "/v1/sessions/")
	parts := strings.Split(trimmed, "/")
	if len(parts) < 2 {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	sessionID := parts[0]
	resource := parts[1]

	switch resource {
	case "history":
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		h, err := s.agents.History("main", sessionID)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, h)
	case "messages":
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var req struct {
			Message string `json:"message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErr(w, err)
			return
		}
		resp, err := s.agents.SendMessage(r.Context(), "main", sessionID, req.Message)
		if err != nil {
			writeErr(w, err)
			return
		}
		writeJSON(w, http.StatusOK, resp)
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (s *Server) skills(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	list, err := skills.List(s.root)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, err error) {
	writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
}
