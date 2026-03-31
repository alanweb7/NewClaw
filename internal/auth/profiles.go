package auth

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"newclaw/internal/store"
)

type ProfilesFile struct {
	Version  int                    `json:"version"`
	Profiles map[string]AuthProfile `json:"profiles"`
	LastGood map[string]string      `json:"last_good"`
}

type AuthProfile struct {
	Type     string `json:"type"`
	Provider string `json:"provider"`
	Email    string `json:"email,omitempty"`
	Access   string `json:"access,omitempty"`
	Refresh  string `json:"refresh,omitempty"`
	Expires  int64  `json:"expires,omitempty"`
	Key      string `json:"key,omitempty"`
}

func Path(root string) string {
	return filepath.Join(root, ".newclaw", "auth-profiles.json")
}

func Ensure(root string) error {
	path := Path(root)
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		pf := ProfilesFile{Version: 1, Profiles: map[string]AuthProfile{}, LastGood: map[string]string{}}
		return store.WriteJSON(path, pf)
	}
	return nil
}

func Load(root string) (ProfilesFile, error) {
	if err := Ensure(root); err != nil {
		return ProfilesFile{}, err
	}
	var pf ProfilesFile
	if err := store.ReadJSON(Path(root), &pf); err != nil {
		return ProfilesFile{}, err
	}
	if pf.Version == 0 {
		pf.Version = 1
	}
	if pf.Profiles == nil {
		pf.Profiles = map[string]AuthProfile{}
	}
	if pf.LastGood == nil {
		pf.LastGood = map[string]string{}
	}
	return pf, nil
}

func Save(root string, pf ProfilesFile) error {
	return store.WriteJSON(Path(root), pf)
}

func SetProfile(root, id string, p AuthProfile) error {
	pf, err := Load(root)
	if err != nil {
		return err
	}
	pf.Profiles[id] = p
	pf.LastGood[p.Provider] = id
	return Save(root, pf)
}

func PreferredProfile(root, provider string) (AuthProfile, bool, error) {
	pf, err := Load(root)
	if err != nil {
		return AuthProfile{}, false, err
	}
	if id := pf.LastGood[provider]; id != "" {
		if p, ok := pf.Profiles[id]; ok {
			return p, true, nil
		}
	}
	for _, p := range pf.Profiles {
		if p.Provider == provider {
			return p, true, nil
		}
	}
	return AuthProfile{}, false, nil
}

func ResolveBearer(root, provider, envVar string) (string, error) {
	p, ok, err := PreferredProfile(root, provider)
	if err != nil {
		return "", err
	}
	if ok {
		switch p.Type {
		case "oauth":
			if strings.TrimSpace(p.Access) != "" {
				return p.Access, nil
			}
		case "api_key":
			if strings.TrimSpace(p.Key) != "" {
				return p.Key, nil
			}
		}
	}
	if strings.TrimSpace(envVar) == "" {
		return "", nil
	}
	return os.Getenv(envVar), nil
}
