package agents

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"newclaw/internal/llm"
	"newclaw/internal/skills"
	"newclaw/internal/store"
	"newclaw/internal/tools"
	"newclaw/internal/workspace"
	"newclaw/pkg/types"
)

type Service struct {
	root      string
	cfg       types.RuntimeConfig
	llmClient *llm.Client
	tools     *tools.Executor
}

func NewService(root string, cfg types.RuntimeConfig) *Service {
	return &Service{
		root:      root,
		cfg:       cfg,
		llmClient: llm.New(root, cfg.Model),
		tools:     tools.NewExecutor(root, cfg.Tools),
	}
}

func EnsureAgent(root string, cfg types.RuntimeConfig, agentID string) error {
	if agentID == "" {
		agentID = "main"
	}
	agentDir := filepath.Join(root, ".newclaw", "agents", agentID)
	sessionsDir := filepath.Join(agentDir, "sessions")
	if err := store.EnsureDir(sessionsDir); err != nil {
		return err
	}
	agentFile := filepath.Join(agentDir, "agent.json")
	if _, err := os.Stat(agentFile); os.IsNotExist(err) {
		agent := types.AgentConfig{ID: agentID, Name: agentID, Model: cfg.Model.DefaultModel}
		if err := store.WriteJSON(agentFile, agent); err != nil {
			return err
		}
	}
	indexFile := filepath.Join(sessionsDir, "sessions.json")
	if _, err := os.Stat(indexFile); os.IsNotExist(err) {
		index := types.SessionIndex{Version: 1, Items: map[string]types.SessionState{}}
		if err := store.WriteJSON(indexFile, index); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) CreateSession(agentID string) (types.SessionState, error) {
	if agentID == "" {
		agentID = "main"
	}
	id := genID()
	file := filepath.Join(s.root, ".newclaw", "agents", agentID, "sessions", id+".jsonl")
	state := types.SessionState{
		SessionID:   id,
		AgentID:     agentID,
		UpdatedAt:   time.Now().UTC(),
		EventCount:  0,
		SessionFile: file,
	}
	if err := s.upsertSession(state); err != nil {
		return types.SessionState{}, err
	}
	return state, nil
}

func (s *Service) ListSessions(agentID string) ([]types.SessionState, error) {
	idx, err := s.loadIndex(agentID)
	if err != nil {
		return nil, err
	}
	out := make([]types.SessionState, 0, len(idx.Items))
	for _, v := range idx.Items {
		out = append(out, v)
	}
	return out, nil
}

func (s *Service) History(agentID, sessionID string) ([]types.SessionEvent, error) {
	path := filepath.Join(s.root, ".newclaw", "agents", agentID, "sessions", sessionID+".jsonl")
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	out := []types.SessionEvent{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var ev types.SessionEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			continue
		}
		out = append(out, ev)
	}
	return out, scanner.Err()
}

func (s *Service) SendMessage(ctx context.Context, agentID, sessionID, message string) (types.SessionEvent, error) {
	if agentID == "" {
		agentID = "main"
	}
	if sessionID == "" {
		st, err := s.CreateSession(agentID)
		if err != nil {
			return types.SessionEvent{}, err
		}
		sessionID = st.SessionID
	}
	userEvent := types.SessionEvent{
		ID:        genID(),
		SessionID: sessionID,
		Role:      "user",
		Content:   message,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.appendEvent(agentID, sessionID, userEvent); err != nil {
		return types.SessionEvent{}, err
	}

	systemPrompt, err := s.buildSystemPrompt()
	if err != nil {
		return types.SessionEvent{}, err
	}
	answer, err := s.llmClient.Complete(ctx, systemPrompt, message)
	if err != nil {
		answer = "error generating response: " + err.Error()
	}
	assistantEvent := types.SessionEvent{
		ID:        genID(),
		SessionID: sessionID,
		Role:      "assistant",
		Content:   answer,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.appendEvent(agentID, sessionID, assistantEvent); err != nil {
		return types.SessionEvent{}, err
	}
	return assistantEvent, nil
}

func (s *Service) ExecuteTool(ctx context.Context, req types.ToolRequest) types.ToolResult {
	return s.tools.Run(ctx, req)
}

func (s *Service) buildSystemPrompt() (string, error) {
	files, _ := workspace.LoadCoreFiles(s.root)
	skillsList, _ := skills.List(s.root)
	var b strings.Builder
	b.WriteString("You are NewClaw.\n")
	for name, content := range files {
		b.WriteString("\n## ")
		b.WriteString(name)
		b.WriteString("\n")
		if len(content) > 2000 {
			content = content[:2000]
		}
		b.WriteString(content)
	}
	b.WriteString("\n\nAvailable skills:\n")
	for _, sk := range skillsList {
		b.WriteString("- ")
		b.WriteString(sk.Name)
		if sk.Description != "" {
			b.WriteString(": ")
			b.WriteString(sk.Description)
		}
		b.WriteString("\n")
	}
	return b.String(), nil
}

func (s *Service) appendEvent(agentID, sessionID string, ev types.SessionEvent) error {
	path := filepath.Join(s.root, ".newclaw", "agents", agentID, "sessions", sessionID+".jsonl")
	if err := store.AppendJSONL(path, ev); err != nil {
		return err
	}
	idx, err := s.loadIndex(agentID)
	if err != nil {
		return err
	}
	state := idx.Items[sessionID]
	state.SessionID = sessionID
	state.AgentID = agentID
	state.UpdatedAt = time.Now().UTC()
	state.EventCount = state.EventCount + 1
	state.SessionFile = path
	idx.Items[sessionID] = state
	return s.saveIndex(agentID, idx)
}

func (s *Service) upsertSession(st types.SessionState) error {
	idx, err := s.loadIndex(st.AgentID)
	if err != nil {
		return err
	}
	idx.Items[st.SessionID] = st
	return s.saveIndex(st.AgentID, idx)
}

func (s *Service) loadIndex(agentID string) (types.SessionIndex, error) {
	path := filepath.Join(s.root, ".newclaw", "agents", agentID, "sessions", "sessions.json")
	var idx types.SessionIndex
	if err := store.ReadJSON(path, &idx); err != nil {
		return types.SessionIndex{}, err
	}
	if idx.Items == nil {
		idx.Items = map[string]types.SessionState{}
	}
	if idx.Version == 0 {
		idx.Version = 1
	}
	return idx, nil
}

func (s *Service) saveIndex(agentID string, idx types.SessionIndex) error {
	path := filepath.Join(s.root, ".newclaw", "agents", agentID, "sessions", "sessions.json")
	return store.WriteJSON(path, idx)
}

func genID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
