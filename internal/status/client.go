// Package status polls carol-bot's in-process admin status route — the one
// piece of state (guild count, gateway latency, last /sync processed at)
// that isn't visible from outside the Node process via docker alone.
package status

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// requestTimeout bounds calls to carol-bot so a network hiccup (wrong
// docker network, DNS not resolving carol-bot) surfaces as a fast error
// instead of hanging until Cloudflare's own tunnel timeout returns an
// opaque 502 to the browser.
const requestTimeout = 5 * time.Second

type Client struct {
	baseURL string
	secret  string
	http    *http.Client
}

func New(baseURL, secret string) *Client {
	return &Client{baseURL: baseURL, secret: secret, http: &http.Client{Timeout: requestTimeout}}
}

// BotStatus mirrors the JSON shape carol-bot's GET /admin/status returns.
// Keep this in sync by hand with src/web/index.ts on the carol-bot side —
// there is no shared type across the two repos.
type BotStatus struct {
	GuildCount       int    `json:"guildCount"`
	GatewayPingMs    int    `json:"gatewayPingMs"`
	LastSyncAt       string `json:"lastSyncAt"`
	UptimeSeconds    int    `json:"uptimeSeconds"`
}

func (c *Client) Fetch(ctx context.Context) (*BotStatus, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/admin/status", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Carol-Ops-Secret", c.secret)
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bot status: unexpected status %d", resp.StatusCode)
	}
	var s BotStatus
	if err := json.NewDecoder(resp.Body).Decode(&s); err != nil {
		return nil, err
	}
	return &s, nil
}
