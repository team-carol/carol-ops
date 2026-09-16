package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config is carol-ops' own settings, kept separate from carol-bot's config.json.
//
// There is no app-level login: this tool sits entirely behind a Cloudflare
// Access-protected tunnel hostname (no port is ever published to the host),
// so Access's per-email policy is the only auth boundary. Don't reintroduce
// a shared password here — it'd be redundant with Access and worse (shared
// credential vs. per-person identity).
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
	return &cfg, nil
}
