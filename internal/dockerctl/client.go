// Package dockerctl talks to a docker-socket-proxy sidecar (Tecnativa-style),
// never to /var/run/docker.sock directly. The proxy only whitelists API
// *categories* (e.g. POST allowed, EXEC denied) — it has no concept of "only
// carol's containers", so any container on the host is reachable through it
// if the category is on. This client is the layer that actually enforces
// "only carol's compose project": every call inspects the target container
// first and refuses to act (or return data) if its
// com.docker.compose.project label doesn't match.
package dockerctl

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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
	if _, err := c.rawInspect(ctx, containerID, true); err != nil {
		return err
	}
	return c.post(ctx, "/containers/"+containerID+"/restart")
}

func (c *Client) Stop(ctx context.Context, containerID string) error {
	if _, err := c.rawInspect(ctx, containerID, true); err != nil {
		return err
	}
	return c.post(ctx, "/containers/"+containerID+"/stop")
}

func (c *Client) Start(ctx context.Context, containerID string) error {
	if _, err := c.rawInspect(ctx, containerID, true); err != nil {
		return err
	}
	return c.post(ctx, "/containers/"+containerID+"/start")
}

// Inspect returns the docker-proxy's full inspect JSON for containerID,
// unmodified, once ownership is confirmed — the detail view in the web UI
// picks out whatever fields it wants from this itself.
func (c *Client) Inspect(ctx context.Context, containerID string) (json.RawMessage, error) {
	return c.rawInspect(ctx, containerID, true)
}

// Logs fetches recent stdout+stderr log output for containerID, demultiplexed
// into plain text. tail bounds how many lines from the end are returned.
func (c *Client) Logs(ctx context.Context, containerID string, tail int) (string, error) {
	if _, err := c.rawInspect(ctx, containerID, true); err != nil {
		return "", err
	}
	url := fmt.Sprintf("%s/containers/%s/logs?stdout=1&stderr=1&timestamps=1&tail=%d", c.baseURL, containerID, tail)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("docker proxy logs: unexpected status %d: %s", resp.StatusCode, body)
	}
	return demuxLogs(resp.Body), nil
}

// demuxLogs strips Docker's log-stream framing. None of these containers run
// with a TTY, so the daemon always multiplexes stdout/stderr as a sequence
// of frames — each an 8-byte header (1-byte stream type, 3 bytes unused,
// 4-byte big-endian payload length) followed by that many bytes of log
// data — rather than sending raw bytes.
func demuxLogs(r io.Reader) string {
	var out strings.Builder
	header := make([]byte, 8)
	for {
		if _, err := io.ReadFull(r, header); err != nil {
			break
		}
		size := binary.BigEndian.Uint32(header[4:8])
		chunk := make([]byte, size)
		if _, err := io.ReadFull(r, chunk); err != nil {
			break
		}
		out.Write(chunk)
	}
	return out.String()
}

// rawInspect fetches containerID's docker inspect JSON and, if checkOwned,
// rejects it unless its compose-project label matches c.project. The
// docker-socket-proxy itself won't do this scoping, so it has to happen
// here, on every call that reads or changes a specific container — never
// trust a container ID from a request as already being "ours".
func (c *Client) rawInspect(ctx context.Context, containerID string, checkOwned bool) (json.RawMessage, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		c.baseURL+"/containers/"+containerID+"/json", nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("docker proxy inspect: unexpected status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if checkOwned {
		var probe struct {
			Config struct {
				Labels map[string]string `json:"Labels"`
			} `json:"Config"`
		}
		if err := json.Unmarshal(body, &probe); err != nil {
			return nil, err
		}
		if probe.Config.Labels[projectLabel] != c.project {
			return nil, ErrNotOwned
		}
	}
	return json.RawMessage(body), nil
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
