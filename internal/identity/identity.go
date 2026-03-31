package identity

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"path/filepath"
	"time"

	"newclaw/internal/store"
	"newclaw/pkg/types"
)

func Ensure(root string) (types.Identity, types.DeviceAuth, error) {
	idPath := filepath.Join(root, ".newclaw", "identity", "device.json")
	authPath := filepath.Join(root, ".newclaw", "identity", "device-auth.json")

	var id types.Identity
	if err := store.ReadJSON(idPath, &id); err != nil {
		newID, err := generateIdentity()
		if err != nil {
			return types.Identity{}, types.DeviceAuth{}, err
		}
		id = newID
		if err := store.WriteJSON(idPath, id); err != nil {
			return types.Identity{}, types.DeviceAuth{}, err
		}
	}

	var auth types.DeviceAuth
	if err := store.ReadJSON(authPath, &auth); err != nil {
		auth = defaultAuth(id.DeviceID)
		if err := store.WriteJSON(authPath, auth); err != nil {
			return types.Identity{}, types.DeviceAuth{}, err
		}
	}

	return id, auth, nil
}

func generateIdentity() (types.Identity, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return types.Identity{}, err
	}
	h := sha256.Sum256(pub)
	deviceID := hex.EncodeToString(h[:])

	pubPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pub})
	privPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: priv})

	return types.Identity{
		Version:       1,
		DeviceID:      deviceID,
		PublicKeyPEM:  string(pubPEM),
		PrivateKeyPEM: string(privPEM),
		CreatedAtMs:   time.Now().UnixMilli(),
	}, nil
}

func defaultAuth(deviceID string) types.DeviceAuth {
	buf := make([]byte, 32)
	_, _ = rand.Read(buf)
	token := base64.RawURLEncoding.EncodeToString(buf)
	return types.DeviceAuth{
		Version:  1,
		DeviceID: deviceID,
		Tokens: map[string]types.OperatorToken{
			"operator": {
				Token:     token,
				Role:      "operator",
				Scopes:    []string{"operator.read", "operator.write", "operator.admin"},
				UpdatedAt: time.Now().UnixMilli(),
			},
		},
	}
}
