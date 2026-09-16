// Package kvfile reads/writes the two key-value config surfaces carol-bot
// actually has: config.json (flat JSON object) and .env (KEY=VALUE lines,
// only used by docker compose for CF_TUNNEL_TOKEN). The web editor works
// against the same []Entry shape for both so the UI doesn't need to care
// which file a key came from.
package kvfile

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

type Entry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// ReadJSON loads a flat JSON object (carol-bot's config.json) as entries.
// Nested values are rejected rather than silently flattened/stringified —
// the editor is for scalar settings, not for reshaping the config schema.
func ReadJSON(path string) ([]Entry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	entries := make([]Entry, 0, len(raw))
	for k, v := range raw {
		var scalar any
		if err := json.Unmarshal(v, &scalar); err != nil {
			return nil, fmt.Errorf("%s: value for %q is not scalar-decodable: %w", path, k, err)
		}
		switch scalar.(type) {
		case map[string]any, []any:
			return nil, fmt.Errorf("%s: key %q is an object/array; kvfile only edits scalar values", path, k)
		}
		entries = append(entries, Entry{Key: k, Value: fmt.Sprintf("%v", scalar)})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Key < entries[j].Key })
	return entries, nil
}

// WriteJSON replaces the JSON object at path with exactly the given entries
// — a key present in the file but absent from entries is deleted. The web
// editor always PUTs its full draft (see KeyValueEditor.tsx), so this is how
// removing a row actually removes the key rather than merely not updating
// it. Each kept key's original JSON type (string vs number vs bool) is
// preserved rather than overwriting everything as strings; new keys are
// written as JSON strings.
func WriteJSON(path string, entries []Entry) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	var existing map[string]json.RawMessage
	if err := json.Unmarshal(data, &existing); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	next := make(map[string]json.RawMessage, len(entries))
	for _, e := range entries {
		old, ok := existing[e.Key]
		encoded, err := encodeLike(old, ok, e.Value)
		if err != nil {
			return fmt.Errorf("%s: encode %q: %w", path, e.Key, err)
		}
		next[e.Key] = encoded
	}
	out, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}

// encodeLike re-encodes newValue as JSON, matching the JSON type the key
// already had (number/bool/string) so e.g. webPort doesn't turn into "8080".
// New keys (ok == false) are written as JSON strings.
func encodeLike(existing json.RawMessage, ok bool, newValue string) (json.RawMessage, error) {
	if !ok {
		b, err := json.Marshal(newValue)
		return b, err
	}
	var probe any
	if err := json.Unmarshal(existing, &probe); err != nil {
		return nil, err
	}
	switch probe.(type) {
	case bool:
		b, err := json.Marshal(newValue == "true")
		return b, err
	case float64:
		var n json.Number
		if err := json.Unmarshal([]byte(newValue), &n); err != nil {
			return nil, fmt.Errorf("expected a number, got %q", newValue)
		}
		return json.RawMessage(newValue), nil
	default:
		b, err := json.Marshal(newValue)
		return b, err
	}
}

// ReadEnv loads a .env file's KEY=VALUE lines, skipping blanks and comments.
func ReadEnv(path string) ([]Entry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	defer f.Close()

	var entries []Entry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		entries = append(entries, Entry{Key: strings.TrimSpace(k), Value: strings.Trim(strings.TrimSpace(v), `"`)})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

// WriteEnv replaces the .env file at path with exactly the given entries, in
// the given order — same full-replace contract as WriteJSON, so omitting a
// row's key from the PUT is how the editor deletes it.
func WriteEnv(path string, entries []Entry) error {
	var b strings.Builder
	for _, e := range entries {
		fmt.Fprintf(&b, "%s=%s\n", e.Key, e.Value)
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}
