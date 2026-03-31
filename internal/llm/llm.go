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

type responsesRequest struct {
	Model        string `json:"model"`
	Instructions string `json:"instructions,omitempty"`
	Input        string `json:"input"`
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

	transport := strings.TrimSpace(c.cfg.Transport)
	if transport == "" {
		transport = "openai-chat-completions"
	}

	switch transport {
	case "openclaw-codex-responses":
		return c.completeCodexResponses(ctx, bearer, systemPrompt, userPrompt)
	default:
		return c.completeChatCompletions(ctx, bearer, systemPrompt, userPrompt)
	}
}

func (c *Client) completeChatCompletions(ctx context.Context, bearer, systemPrompt, userPrompt string) (string, error) {
	endpoint := strings.TrimRight(c.cfg.BaseURL, "/") + "/chat/completions"
	body := chatRequest{
		Model: c.cfg.DefaultModel,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	}
	b, _ := json.Marshal(body)
	return c.postWithRetries(ctx, endpoint, b, bearer, func(respBody []byte) (string, error) {
		var parsed chatResponse
		if err := json.Unmarshal(respBody, &parsed); err != nil {
			return "", err
		}
		if len(parsed.Choices) == 0 {
			return "", fmt.Errorf("empty choices")
		}
		return parsed.Choices[0].Message.Content, nil
	})
}

func (c *Client) completeCodexResponses(ctx context.Context, bearer, systemPrompt, userPrompt string) (string, error) {
	paths := []string{"/openai-codex-responses", "/codex/responses", "/responses", "/v1/responses"}
	body := responsesRequest{
		Model:        c.cfg.DefaultModel,
		Instructions: systemPrompt,
		Input:        userPrompt,
	}
	b, _ := json.Marshal(body)

	var lastErr error
	for _, p := range paths {
		endpoint := strings.TrimRight(c.cfg.BaseURL, "/") + p
		answer, err := c.postWithRetries(ctx, endpoint, b, bearer, parseResponsesText)
		if err == nil {
			return answer, nil
		}
		lastErr = err
	}
	return "", lastErr
}

func (c *Client) postWithRetries(
	ctx context.Context,
	endpoint string,
	body []byte,
	bearer string,
	parse func([]byte) (string, error),
) (string, error) {
	var lastErr error
	for i := 0; i <= c.cfg.MaxRetries; i++ {
		req, _ := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("Authorization", "Bearer "+bearer)
		req.Header.Set("User-Agent", "NewClaw/1.0")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		respBody, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode > 299 {
			lastErr = formatHTTPError(respBody)
			continue
		}

		text, err := parse(respBody)
		if err != nil {
			lastErr = err
			continue
		}
		if strings.TrimSpace(text) == "" {
			lastErr = fmt.Errorf("empty model response")
			continue
		}
		return text, nil
	}
	return "", lastErr
}

func parseResponsesText(respBody []byte) (string, error) {
	// Try a few common wire shapes used by responses-style APIs.
	var obj map[string]interface{}
	if err := json.Unmarshal(respBody, &obj); err != nil {
		return "", err
	}
	if v := asString(obj["output_text"]); v != "" {
		return v, nil
	}
	if out, ok := obj["output"].([]interface{}); ok {
		var parts []string
		for _, it := range out {
			m, ok := it.(map[string]interface{})
			if !ok {
				continue
			}
			if content, ok := m["content"].([]interface{}); ok {
				for _, c := range content {
					cm, ok := c.(map[string]interface{})
					if !ok {
						continue
					}
					if t := asString(cm["text"]); t != "" {
						parts = append(parts, t)
					}
				}
			}
		}
		if len(parts) > 0 {
			return strings.Join(parts, "\n"), nil
		}
	}
	if choices, ok := obj["choices"].([]interface{}); ok && len(choices) > 0 {
		first, _ := choices[0].(map[string]interface{})
		msg, _ := first["message"].(map[string]interface{})
		if t := asString(msg["content"]); t != "" {
			return t, nil
		}
	}
	return "", fmt.Errorf("could not parse responses payload")
}

func formatHTTPError(respBody []byte) error {
	body := string(respBody)
	if looksLikeCloudflareChallenge(body) {
		return fmt.Errorf("llm blocked by Cloudflare challenge on this endpoint")
	}
	if len(body) > 2000 {
		body = body[:2000] + "..."
	}
	return fmt.Errorf("llm error: %s", body)
}

func looksLikeCloudflareChallenge(body string) bool {
	s := strings.ToLower(body)
	return strings.Contains(s, "__cf_chl") || strings.Contains(s, "cloudflare") || strings.Contains(s, "cf-challenge")
}

func asString(v interface{}) string {
	s, _ := v.(string)
	return s
}
