// Package status polls carol-bot's in-process admin status route — the one
// piece of state (guild count, gateway latency, last /sync processed at)
// that isn't visible from outside the Node process via docker alone.
package status

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type Client struct {
	baseURL string
	secret  string
	http    *http.Client
}

func New(baseURL, secret string) *Client {
	return &Client{baseURL: baseURL, secret: secret, http: &http.Client{}}
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
