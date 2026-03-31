package types

import (
	"encoding/json"
	"testing"
	"time"
)

func TestSessionAndIdentitySerialization(t *testing.T) {
	id := Identity{Version: 1, DeviceID: "abc", PublicKeyPEM: "pub", PrivateKeyPEM: "priv", CreatedAtMs: 1}
	b, err := json.Marshal(id)
	if err != nil {
		t.Fatal(err)
	}
	var id2 Identity
	if err := json.Unmarshal(b, &id2); err != nil {
		t.Fatal(err)
	}
	if id2.DeviceID != id.DeviceID {
		t.Fatal("identity serialization mismatch")
	}

	ev := SessionEvent{ID: "1", SessionID: "s", Role: "user", Content: "hi", CreatedAt: time.Now().UTC()}
	b, err = json.Marshal(ev)
	if err != nil {
		t.Fatal(err)
	}
	var ev2 SessionEvent
	if err := json.Unmarshal(b, &ev2); err != nil {
		t.Fatal(err)
	}
	if ev2.Role != ev.Role {
		t.Fatal("session event serialization mismatch")
	}
}
