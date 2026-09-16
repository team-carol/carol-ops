// Package dockerctl talks to a docker-socket-proxy sidecar (Tecnativa-style),
// never to /var/run/docker.sock directly. The proxy only whitelists API
// *categories* (e.g. POST allowed, EXEC denied) — it has no concept of "only
// carol's containers", so any container on the host is reachable through it
// if the category is on. This client is the layer that actually enforces
// "only carol's compose project": every state-changing call inspects the
// target container first and refuses to act if its
// com.docker.compose.project label doesn't match.
package dockerctl

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const projectLabel = "com.docker.compose.project"

// requestTimeout bounds calls to the docker-socket-proxy sidecar so a
// network problem surfaces as a fast error instead of hanging until
// Cloudflare's own tunnel timeout returns an opaque 502 to the browser.
const requestTimeout = 5 * time.Second

type Client struct {
	baseURL string
	project string
	http    *http.Client
}

func New(baseURL, composeProject string) *Client {
	return &Client{baseURL: baseURL, project: composeProject, http: &http.Client{Timeout: requestTimeout}}
}

type Container struct {
	ID     string            `json:"Id"`
	Names  []string          `json:"Names"`
	State  string            `json:"State"`
	Labels map[string]string `json:"Labels"`
}

// List returns containers belonging to the configured compose project only.
func (c *Client) List(ctx context.Context) ([]Container, error) {
	filters := fmt.Sprintf(`{"label":["%s=%s"]}`, projectLabel, c.project)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.baseURL+"/containers/json?all=true&filters="+filters, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("docker proxy list: unexpected status %d", resp.StatusCode)
	}
	var containers []Container
	if err := json.NewDecoder(resp.Body).Decode(&containers); err != nil {
		return nil, err
	}
	return containers, nil
}

var ErrNotOwned = fmt.Errorf("container does not belong to compose project")

func (c *Client) Restart(ctx context.Context, containerID string) error {
	if err := c.ensureOwned(ctx, containerID); err != nil {
		return err
	}
	return c.post(ctx, "/containers/"+containerID+"/restart")
}

func (c *Client) Stop(ctx context.Context, containerID string) error {
	if err := c.ensureOwned(ctx, containerID); err != nil {
		return err
	}
	return c.post(ctx, "/containers/"+containerID+"/stop")
}

func (c *Client) Start(ctx context.Context, containerID string) error {
	if err := c.ensureOwned(ctx, containerID); err != nil {
		return err
	}
	return c.post(ctx, "/containers/"+containerID+"/start")
}

// ensureOwned inspects containerID and rejects it unless its compose-project
// label matches c.project. The docker-socket-proxy itself won't do this
// scoping, so it has to happen here, on every call that changes state —
// never trust a container ID from a request as already being "ours".
func (c *Client) ensureOwned(ctx context.Context, containerID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.baseURL+"/containers/"+containerID+"/json", nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("docker proxy inspect: unexpected status %d", resp.StatusCode)
	}
	var inspected struct {
		Config struct {
			Labels map[string]string `json:"Labels"`
		} `json:"Config"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&inspected); err != nil {
		return err
	}
	if inspected.Config.Labels[projectLabel] != c.project {
		return ErrNotOwned
	}
	return nil
}

func (c *Client) post(ctx context.Context, path string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("docker proxy %s: unexpected status %d", path, resp.StatusCode)
	}
	return nil
}
