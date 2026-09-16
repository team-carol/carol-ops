package config

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
)

// Config is carol-ops' own settings, kept separate from carol-bot's config.json.
type Config struct {
	// ListenAddr is the address the admin web server binds to.
	ListenAddr string `json:"listenAddr"`

	// DockerProxyURL points at the docker-socket-proxy sidecar, not the raw
	// docker.sock, so carol-ops never holds host-root-equivalent access directly.
	DockerProxyURL string `json:"dockerProxyUrl"`
	// ComposeProject filters container operations to carol-bot's compose project
	// so carol-ops can't accidentally touch unrelated containers on the host.
	ComposeProject string `json:"composeProject"`

	// CarolConfigPath is the on-disk path to carol-bot's config.json, and
	// CarolEnvPath to its .env (CF_TUNNEL_TOKEN only) — both mounted
	// read/write into the carol-ops container for the key-value editor.
	CarolConfigPath string `json:"carolConfigPath"`
	CarolEnvPath    string `json:"carolEnvPath"`

	// CarolStatusURL + CarolStatusSecret authenticate against carol-bot's
	// in-process /admin/status route (shared-secret header, same pattern as
	// carolIssueBaseUrl/carolSharedSecret in the bot repo).
	CarolStatusURL    string `json:"carolStatusUrl"`
	CarolStatusSecret string `json:"carolStatusSecret"`

	// Single admin login for the web UI. AdminPassword is a plain shared
	// secret (same posture as carol-bot's carolSharedSecret) — this tool has
	// exactly one account, not a multi-user system, so hashing buys little.
	AdminUsername string `json:"adminUsername"`
	AdminPassword string `json:"adminPassword"`

	// SessionSecret signs session cookies. Left empty in config.json.example;
	// generated on first start and written back, same as carol-bot's
	// encryptionKey — do not overwrite the file after first run.
	SessionSecret string `json:"sessionSecret"`

	// GeneratedCredentials is true when Load just minted AdminUsername/
	// AdminPassword because config.json had none — unlike SessionSecret, the
	// operator actually needs to see this value once, so it's not persisted
	// silently; main.go prints it. Never itself written to config.json.
	GeneratedCredentials bool `json:"-"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if cfg.ListenAddr == "" {
		cfg.ListenAddr = ":8090"
	}
	if cfg.SessionSecret == "" {
		secret, err := randomSecret(32)
		if err != nil {
			return nil, fmt.Errorf("generate session secret: %w", err)
		}
		cfg.SessionSecret = secret
		if err := writeBack(path, "sessionSecret", secret); err != nil {
			return nil, fmt.Errorf("persist session secret: %w", err)
		}
	}
	if cfg.AdminUsername == "" {
		cfg.AdminUsername = "admin"
		if err := writeBack(path, "adminUsername", cfg.AdminUsername); err != nil {
			return nil, fmt.Errorf("persist admin username: %w", err)
		}
	}
	if cfg.AdminPassword == "" {
		password, err := randomSecret(12)
		if err != nil {
			return nil, fmt.Errorf("generate admin password: %w", err)
		}
		cfg.AdminPassword = password
		if err := writeBack(path, "adminPassword", password); err != nil {
			return nil, fmt.Errorf("persist admin password: %w", err)
		}
		cfg.GeneratedCredentials = true
	}
	return &cfg, nil
}

func randomSecret(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// writeBack sets a single top-level key in the JSON file at path without
// disturbing any other keys or their formatting-adjacent values.
func writeBack(path, key, value string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	raw[key] = encoded
	out, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, out, 0o644)
}
