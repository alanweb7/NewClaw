# NewClaw

Assistente local em Go com:

- CLI para chat e sessões
- API HTTP simples
- Configuração de modelo OpenAI por API Key ou OAuth (Codex)

## Requisitos

- Go `1.25.7` (ou compatível com o `go.mod`)
- Sistema operacional com terminal (PowerShell, bash, etc.)

## Estrutura básica

Após inicializar, o projeto cria dados de runtime em:

- `.newclaw/newclaw.json` (configuração principal)
- `.newclaw/auth-profiles.json` (perfis de autenticação)
- `.newclaw/agents/main/sessions/` (sessões JSONL)
- `.newclaw/workspace/` (arquivos de contexto)

## Como iniciar o projeto

No diretório raiz do projeto:

```bash
go run ./cmd/newclaw init
```

Isso cria a estrutura inicial em `.newclaw`.

### Rodar API HTTP

```bash
go run ./cmd/newclaw run
```

Por padrão, sobe em `127.0.0.1:7840`.

Endpoints principais:

- `GET /healthz`
- `POST /v1/sessions`
- `GET /v1/sessions`
- `GET /v1/sessions/{id}/history`
- `POST /v1/sessions/{id}/messages`
- `GET /v1/skills`

Exemplo de envio de mensagem via HTTP:

```bash
export NEWCLAW_BASE_URL="http://127.0.0.1:7840"
curl -X POST "$NEWCLAW_BASE_URL/v1/sessions"
```

```bash
curl -X POST "$NEWCLAW_BASE_URL/v1/sessions/SESSAO/messages" \
  -H "Content-Type: application/json" \
  -d "{\"message\":\"Olá\"}"
```

Para disparar de fora da VPS, use o IP/domínio público:

```bash
export NEWCLAW_BASE_URL="http://SEU_IP_OU_DOMINIO:7840"
curl -X POST "$NEWCLAW_BASE_URL/v1/sessions"
```

### Rodar chat via CLI

Mensagem única:

```bash
go run ./cmd/newclaw chat --message "Olá"
```

Com sessão específica:

```bash
go run ./cmd/newclaw chat --session 123 --message "Continuar conversa"
```

Listar sessões:

```bash
go run ./cmd/newclaw session list
```

Histórico de sessão:

```bash
go run ./cmd/newclaw session history --id 123
```

## Como autenticar OAuth (OpenAI/Codex)

O caminho recomendado já está no comando interativo:

```bash
go run ./cmd/newclaw model config
```

Depois selecione:

1. `1) OAuth`
2. Preencha email (ou deixe default)
3. Mantenha defaults quando aplicável:
   - Client ID: `app_EMoamEEZ73f0CkXaXp7hrann`
   - Authorize URL: `https://auth.openai.com/oauth/authorize`
   - Token URL: `https://auth.openai.com/oauth/token`
   - Redirect URI: `http://localhost:1455/auth/callback`
4. Escolha transporte:
   - `1) OpenClaw-compatible (recomendado)`
5. Abra a URL de autorização gerada no navegador
6. Faça login e cole a callback URL final (ou só o `code`) no terminal

Se a troca automática de token falhar, o fluxo pede `access token` manualmente.

### Onde o OAuth fica salvo

O perfil OAuth é salvo em:

- `.newclaw/auth-profiles.json`

Formato esperado (resumo):

- `type: "oauth"`
- `provider: "openai-codex"`
- `access`, `refresh`, `expires`

Quando OAuth está ativo, a configuração do modelo fica com:

- `provider: "openai-codex"`
- `base_url: "https://chatgpt.com/backend-api"`
- `transport: "openclaw-codex-responses"` (ou legado)
- `default_model: "gpt-5.3-codex"`

## Alternativa: API Key

Também via:

```bash
go run ./cmd/newclaw model config
```

Escolha:

1. `2) API KEY`
2. Salvar no perfil local **ou** usar `OPENAI_API_KEY`

Para variável de ambiente:

```powershell
$env:OPENAI_API_KEY="sua_chave"
```

ou:

```bash
export OPENAI_API_KEY="sua_chave"
```

## Observação de funcionamento

Se não houver token/API key resolvido, o cliente LLM retorna resposta mock:

- `"[mock-response] <sua mensagem>"`

Isso ajuda a validar fluxo sem credenciais, mas não chama modelo real.
