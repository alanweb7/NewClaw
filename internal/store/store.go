package store

import (
	"bufio"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

func EnsureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}

func WriteJSON(path string, v interface{}) error {
	if err := EnsureDir(filepath.Dir(path)); err != nil {
		return err
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}

func ReadJSON(path string, out interface{}) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, out)
}

func AppendJSONL(path string, v interface{}) error {
	if err := EnsureDir(filepath.Dir(path)); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	return enc.Encode(v)
}

func ReadJSONL(path string, out interface{}) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var lines []json.RawMessage
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		cpy := make([]byte, len(line))
		copy(cpy, line)
		lines = append(lines, json.RawMessage(cpy))
	}
	if err := scanner.Err(); err != nil {
		return err
	}

	b, err := json.Marshal(lines)
	if err != nil {
		return err
	}
	if len(lines) == 0 {
		return errors.New("empty_jsonl")
	}
	return json.Unmarshal(b, out)
}
