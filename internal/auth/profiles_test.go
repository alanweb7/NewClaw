package auth

import "testing"

func TestResolveBearerPrefersProfile(t *testing.T) {
	root := t.TempDir()
	if err := SetProfile(root, "openai-api-key:default", AuthProfile{Type: "api_key", Provider: "openai-compatible", Key: "k_test"}); err != nil {
		t.Fatal(err)
	}
	tok, err := ResolveBearer(root, "openai-compatible", "")
	if err != nil {
		t.Fatal(err)
	}
	if tok != "k_test" {
		t.Fatalf("unexpected token %q", tok)
	}
}
