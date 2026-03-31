package modelconfig

import (
	"bufio"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"newclaw/internal/auth"
	"newclaw/internal/config"
	"newclaw/internal/store"
	"newclaw/pkg/types"
)

const (
	defaultCodexClientID   = "app_EMoamEEZ73f0CkXaXp7hrann"
	defaultAuthorizeURL    = "https://auth.openai.com/oauth/authorize"
	defaultTokenURL        = "https://auth.openai.com/oauth/token"
	defaultRedirectURI     = "http://localhost:1455/auth/callback"
	defaultCodexOriginator = "pi"
)

func Run(root string) error {
	cfg, err := config.LoadOrCreate(root)
	if err != nil {
		return err
	}
	if err := auth.Ensure(root); err != nil {
		return err
	}

	r := bufio.NewReader(os.Stdin)
	fmt.Println("Escolha o provedor de autenticacao OpenAI:")
	fmt.Println("  1) OAuth")
	fmt.Println("  2) API KEY")
	choice := strings.TrimSpace(readLine(r, "Opcao [1/2]: "))

	switch choice {
	case "1":
		if err := setupOAuth(root, &cfg, r); err != nil {
			return err
		}
	case "2":
		if err := setupAPIKey(root, &cfg, r); err != nil {
			return err
		}
	default:
		return fmt.Errorf("opcao invalida")
	}

	if err := store.WriteJSON(config.ConfigPath(root), cfg); err != nil {
		return err
	}
	fmt.Println("Configuracao de modelo salva com sucesso.")
	return nil
}

func setupAPIKey(root string, cfg *types.RuntimeConfig, r *bufio.Reader) error {
	model := strings.TrimSpace(readLine(r, "Modelo (default gpt-5.3-codex): "))
	if model == "" {
		model = "gpt-5.3-codex"
	}

	fmt.Println("Como deseja fornecer a chave?")
	fmt.Println("  1) Salvar no perfil do NewClaw")
	fmt.Println("  2) Usar variavel de ambiente OPENAI_API_KEY")
	keyMode := strings.TrimSpace(readLine(r, "Opcao [1/2]: "))

	cfg.Model.Provider = "openai-compatible"
	cfg.Model.BaseURL = "https://api.openai.com/v1"
	cfg.Model.DefaultModel = model
	cfg.Model.APIKeyEnv = "OPENAI_API_KEY"

	if keyMode == "1" {
		key := strings.TrimSpace(readLine(r, "Cole sua API key: "))
		if key == "" {
			return fmt.Errorf("api key vazia")
		}
		id := "openai-api-key:default"
		return auth.SetProfile(root, id, auth.AuthProfile{
			Type:     "api_key",
			Provider: "openai-compatible",
			Key:      key,
		})
	}

	fmt.Println("OK. Defina no PowerShell: $env:OPENAI_API_KEY=\"sua_chave\"")
	return nil
}

func setupOAuth(root string, cfg *types.RuntimeConfig, r *bufio.Reader) error {
	email := strings.TrimSpace(readLine(r, "Email da conta OpenAI/ChatGPT: "))
	if email == "" {
		email = "default"
	}

	clientID := strings.TrimSpace(readLine(r, "OAuth Client ID [default app_EMoamEEZ73f0CkXaXp7hrann]: "))
	if clientID == "" {
		clientID = defaultCodexClientID
	}
	authorizeURL := strings.TrimSpace(readLine(r, "Authorize URL [default https://auth.openai.com/oauth/authorize]: "))
	if authorizeURL == "" {
		authorizeURL = defaultAuthorizeURL
	}
	tokenURL := strings.TrimSpace(readLine(r, "Token URL [default https://auth.openai.com/oauth/token]: "))
	if tokenURL == "" {
		tokenURL = defaultTokenURL
	}
	redirectURI := strings.TrimSpace(readLine(r, "Redirect URI [default http://localhost:1455/auth/callback]: "))
	if redirectURI == "" {
		redirectURI = defaultRedirectURI
	}

	codeVerifier, codeChallenge, state := pkceState()
	scopes := "openid profile email offline_access"
	authLink := buildAuthURL(authorizeURL, oauthAuthorizeParams{
		ClientID:             clientID,
		RedirectURI:          redirectURI,
		Scopes:               scopes,
		State:                state,
		CodeChallenge:        codeChallenge,
		IDTokenOrganizations: true,
		SimplifiedFlow:       true,
		Originator:           defaultCodexOriginator,
	})

	fmt.Println("Abra o link abaixo no navegador e conclua o login OAuth:")
	fmt.Println(authLink)
	fmt.Println("Depois cole a URL final de callback (ou somente o parametro code).")

	callback := strings.TrimSpace(readLine(r, "Callback URL/code: "))
	code, gotState := parseCallback(callback)
	if code == "" {
		return fmt.Errorf("nao foi possivel extrair o code do callback")
	}
	if gotState != "" && gotState != state {
		return fmt.Errorf("state invalido")
	}

	access := ""
	refresh := ""
	expiresAt := int64(0)
	if clientID != "" {
		tok, err := exchangeToken(tokenURL, clientID, code, redirectURI, codeVerifier)
		if err == nil {
			access = tok.AccessToken
			refresh = tok.RefreshToken
			if tok.ExpiresIn > 0 {
				expiresAt = time.Now().UnixMilli() + int64(tok.ExpiresIn*1000)
			}
		}
	}

	if access == "" {
		fmt.Println("Nao foi possivel trocar code automaticamente. Cole os tokens manualmente:")
		access = strings.TrimSpace(readLine(r, "Access token: "))
		refresh = strings.TrimSpace(readLine(r, "Refresh token (opcional): "))
		expiresRaw := strings.TrimSpace(readLine(r, "Expires (unix ms, opcional): "))
		if expiresRaw != "" {
			if n, err := strconv.ParseInt(expiresRaw, 10, 64); err == nil {
				expiresAt = n
			}
		}
	}
	if access == "" {
		return fmt.Errorf("access token vazio")
	}

	cfg.Model.Provider = "openai-codex"
	cfg.Model.BaseURL = "https://chatgpt.com/backend-api"
	cfg.Model.DefaultModel = "gpt-5.3-codex"
	cfg.Model.APIKeyEnv = ""

	id := "openai-codex:" + email
	return auth.SetProfile(root, id, auth.AuthProfile{
		Type:     "oauth",
		Provider: "openai-codex",
		Email:    email,
		Access:   access,
		Refresh:  refresh,
		Expires:  expiresAt,
	})
}

func readLine(r *bufio.Reader, prompt string) string {
	fmt.Print(prompt)
	line, _ := r.ReadString('\n')
	return strings.TrimSpace(line)
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

func exchangeToken(tokenURL, clientID, code, redirectURI, verifier string) (tokenResponse, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", clientID)
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("code_verifier", verifier)

	resp, err := http.PostForm(tokenURL, form)
	if err != nil {
		return tokenResponse{}, err
	}
	defer resp.Body.Close()
	var out tokenResponse
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return out, fmt.Errorf("token endpoint status %d", resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return tokenResponse{}, err
	}
	return out, nil
}

func pkceState() (verifier, challenge, state string) {
	vb := make([]byte, 32)
	sb := make([]byte, 32)
	_, _ = rand.Read(vb)
	_, _ = rand.Read(sb)
	verifier = base64.RawURLEncoding.EncodeToString(vb)
	h := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(h[:])
	state = base64.RawURLEncoding.EncodeToString(sb)
	return verifier, challenge, state
}

type oauthAuthorizeParams struct {
	ClientID             string
	RedirectURI          string
	Scopes               string
	State                string
	CodeChallenge        string
	IDTokenOrganizations bool
	SimplifiedFlow       bool
	Originator           string
}

func buildAuthURL(authorizeURL string, p oauthAuthorizeParams) string {
	u, err := url.Parse(authorizeURL)
	if err != nil {
		return authorizeURL
	}
	q := u.Query()
	q.Set("response_type", "code")
	q.Set("client_id", p.ClientID)
	q.Set("redirect_uri", p.RedirectURI)
	q.Set("scope", p.Scopes)
	q.Set("state", p.State)
	q.Set("code_challenge", p.CodeChallenge)
	q.Set("code_challenge_method", "S256")
	if p.IDTokenOrganizations {
		q.Set("id_token_add_organizations", "true")
	}
	if p.SimplifiedFlow {
		q.Set("codex_cli_simplified_flow", "true")
	}
	if strings.TrimSpace(p.Originator) != "" {
		q.Set("originator", p.Originator)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func parseCallback(input string) (code, state string) {
	if strings.Contains(input, "code=") {
		u, err := url.Parse(input)
		if err == nil {
			return u.Query().Get("code"), u.Query().Get("state")
		}
		if i := strings.Index(input, "code="); i >= 0 {
			raw := input[i+5:]
			if j := strings.Index(raw, "&"); j >= 0 {
				raw = raw[:j]
			}
			return raw, ""
		}
	}
	return strings.TrimSpace(input), ""
}
