// Package api wires the HTTP surface: JSON endpoints under /api/, and the
// embedded React build (web/dist, via go:embed in cmd/serve) served for
// everything else so carol-ops ships as a single binary/image.
//
// No auth middleware lives here — this app has no ingress path other than a
// Cloudflare Access-protected tunnel hostname (see docker-compose.yml), and
// Access's per-email policy is the only auth boundary. Do not add an
// app-level login back in without also reconsidering that boundary.
package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"carol-ops/internal/dockerctl"
	"carol-ops/internal/kvfile"
	"carol-ops/internal/status"
)

type Deps struct {
	Docker   *dockerctl.Client
	Status   *status.Client
	StaticFS http.FileSystem

	// Paths the key-value editor reads/writes. Kept as plain paths (not a
	// carol-specific abstraction) so a third file surface can be added later
	// without touching the handler shape.
	ConfigJSONPath string
	EnvPath        string
}

func NewRouter(deps Deps) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/containers", func(w http.ResponseWriter, r *http.Request) {
		containers, err := deps.Docker.List(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(containers)
	})

	mux.HandleFunc("POST /api/containers/{id}/{action}", handleContainerAction(deps.Docker))

	mux.HandleFunc("GET /api/status", func(w http.ResponseWriter, r *http.Request) {
		botStatus, err := deps.Status.Fetch(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(botStatus)
	})

	mux.HandleFunc("GET /api/config", handleReadKV(deps.ConfigJSONPath, kvfile.ReadJSON))
	mux.HandleFunc("PUT /api/config", handleWriteKV(deps.ConfigJSONPath, kvfile.WriteJSON))
	mux.HandleFunc("GET /api/env", handleReadKV(deps.EnvPath, kvfile.ReadEnv))
	mux.HandleFunc("PUT /api/env", handleWriteKV(deps.EnvPath, kvfile.WriteEnv))

	mux.Handle("GET /", http.FileServer(deps.StaticFS))
	return mux
}

func handleContainerAction(docker *dockerctl.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		var err error
		switch r.PathValue("action") {
		case "start":
			err = docker.Start(r.Context(), id)
		case "stop":
			err = docker.Stop(r.Context(), id)
		case "restart":
			err = docker.Restart(r.Context(), id)
		default:
			http.Error(w, "unknown action", http.StatusBadRequest)
			return
		}
		if errors.Is(err, dockerctl.ErrNotOwned) {
			http.Error(w, "container is not managed by this carol-ops instance", http.StatusForbidden)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleReadKV(path string, read func(string) ([]kvfile.Entry, error)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		entries, err := read(path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(entries)
	}
}

func handleWriteKV(path string, write func(string, []kvfile.Entry) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var entries []kvfile.Entry
		if err := json.NewDecoder(r.Body).Decode(&entries); err != nil {
			http.Error(w, "invalid body: "+err.Error(), http.StatusBadRequest)
			return
		}
		if err := write(path, entries); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
