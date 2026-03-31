package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"newclaw/internal/auth"
	"newclaw/pkg/types"
)

type Client struct {
	root       string
	cfg        types.ModelConfig
	httpClient *http.Client
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
}

func New(root string, cfg types.ModelConfig) *Client {
	return &Client{
		root: root,
		cfg:  cfg,
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.RequestTimeout) * time.Second,
		},
	}
}

func (c *Client) Complete(ctx context.Context, systemPrompt, userPrompt string) (string, error) {
	bearer, _ := auth.ResolveBearer(c.root, c.cfg.Provider, c.cfg.APIKeyEnv)
	if strings.TrimSpace(bearer) == "" {
		return "[mock-response] " + userPrompt, nil
	}

	endpoint := strings.TrimRight(c.cfg.BaseURL, "/") + "/chat/completions"
	body := chatRequest{
		Model: c.cfg.DefaultModel,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	}
	b, _ := json.Marshal(body)

	var lastErr error
	for i := 0; i <= c.cfg.MaxRetries; i++ {
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+bearer)
		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		respBody, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			lastErr = fmt.Errorf("llm error: %s", string(respBody))
			continue
		}
		var parsed chatResponse
		if err := json.Unmarshal(respBody, &parsed); err != nil {
			lastErr = err
			continue
		}
		if len(parsed.Choices) == 0 {
			lastErr = fmt.Errorf("empty choices")
			continue
		}
		return parsed.Choices[0].Message.Content, nil
	}
	return "", lastErr
}
