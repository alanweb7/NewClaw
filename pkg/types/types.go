package types

import "time"

type RuntimeConfig struct {
	Version int         `json:"version"`
	HTTP    HTTPConfig  `json:"http"`
	Model   ModelConfig `json:"model"`
	Tools   ToolPolicy  `json:"tools"`
}

type HTTPConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

type ModelConfig struct {
	Provider        string `json:"provider"`
	BaseURL         string `json:"base_url"`
	Transport       string `json:"transport"`
	APIKeyEnv       string `json:"api_key_env"`
	DefaultModel    string `json:"default_model"`
	MaxOutputTokens int    `json:"max_output_tokens"`
	StopOnFirstLine bool   `json:"stop_on_first_line"`
	RequestTimeout  int    `json:"request_timeout_seconds"`
	MaxRetries      int    `json:"max_retries"`
}

type ToolPolicy struct {
	ExecAllow []string `json:"exec_allow"`
	ExecDeny  []string `json:"exec_deny"`
}

type AgentConfig struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Model string `json:"model"`
}

type Identity struct {
	Version       int    `json:"version"`
	DeviceID      string `json:"device_id"`
	PublicKeyPEM  string `json:"public_key_pem"`
	PrivateKeyPEM string `json:"private_key_pem"`
	CreatedAtMs   int64  `json:"created_at_ms"`
}

type OperatorToken struct {
	Token     string   `json:"token"`
	Role      string   `json:"role"`
	Scopes    []string `json:"scopes"`
	UpdatedAt int64    `json:"updated_at_ms"`
}

type DeviceAuth struct {
	Version  int                      `json:"version"`
	DeviceID string                   `json:"device_id"`
	Tokens   map[string]OperatorToken `json:"tokens"`
}

type SkillSourceInfo struct {
	Path   string `json:"path"`
	Source string `json:"source"`
}

type SkillDescriptor struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	FilePath    string          `json:"file_path"`
	BaseDir     string          `json:"base_dir"`
	SourceInfo  SkillSourceInfo `json:"source_info"`
}

type SessionEvent struct {
	ID        string                 `json:"id"`
	SessionID string                 `json:"session_id"`
	Role      string                 `json:"role"`
	Content   string                 `json:"content"`
	CreatedAt time.Time              `json:"created_at"`
	Meta      map[string]interface{} `json:"meta,omitempty"`
}

type SessionState struct {
	SessionID   string    `json:"session_id"`
	AgentID     string    `json:"agent_id"`
	UpdatedAt   time.Time `json:"updated_at"`
	EventCount  int       `json:"event_count"`
	SessionFile string    `json:"session_file"`
}

type SessionIndex struct {
	Version int                     `json:"version"`
	Items   map[string]SessionState `json:"items"`
}

type ToolSpec struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type ToolRequest struct {
	Name   string                 `json:"name"`
	Params map[string]interface{} `json:"params"`
}

type ToolResult struct {
	OK     bool                   `json:"ok"`
	Output string                 `json:"output,omitempty"`
	Data   map[string]interface{} `json:"data,omitempty"`
	Error  string                 `json:"error,omitempty"`
}
