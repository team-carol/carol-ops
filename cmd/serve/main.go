package main

import (
	"io/fs"
	"log"
	"net/http"
	"os"

	"carol-ops/internal/api"
	"carol-ops/internal/auth"
	"carol-ops/internal/config"
	"carol-ops/internal/dockerctl"
	"carol-ops/internal/status"
	"carol-ops/web"
)

func main() {
	configPath := os.Getenv("CAROL_OPS_CONFIG")
	if configPath == "" {
		configPath = "config.json"
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	if cfg.GeneratedCredentials {
		log.Printf("==============================================================")
		log.Printf(" generated initial admin login (saved to %s):", configPath)
		log.Printf("   username: %s", cfg.AdminUsername)
		log.Printf("   password: %s", cfg.AdminPassword)
		log.Printf(" change adminPassword in config.json once you've logged in.")
		log.Printf("==============================================================")
	}

	staticFS, err := fs.Sub(web.Dist, "dist")
	if err != nil {
		log.Fatalf("mount embedded web build: %v", err)
	}

	router := api.NewRouter(api.Deps{
		Docker:         dockerctl.New(cfg.DockerProxyURL, cfg.ComposeProject),
		Status:         status.New(cfg.CarolStatusURL, cfg.CarolStatusSecret),
		Auth:           auth.New(cfg.AdminUsername, cfg.AdminPassword, cfg.SessionSecret),
		StaticFS:       http.FS(staticFS),
		ConfigJSONPath: cfg.CarolConfigPath,
		EnvPath:        cfg.CarolEnvPath,
	})

	log.Printf("carol-ops listening on %s", cfg.ListenAddr)
	if err := http.ListenAndServe(cfg.ListenAddr, router); err != nil {
		log.Fatal(err)
	}
}
