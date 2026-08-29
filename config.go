package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type configFile struct {
	VertexProject string            `json:"vertex_project"`
	MlxHost       string            `json:"mlx_host"`  // legacy single-host fallback
	MlxHosts      map[string]string `json:"mlx_hosts"` // name -> "host:port"
}

// mlxEndpoint resolves the v1 base URL for an MLX model. Precedence:
//  1. a named entry in mlx_hosts (port is part of the value),
//  2. the legacy mlx_host combined with the model's per-entry port.
//
// Returns ("", false) when no host is configured.
func (c configFile) mlxEndpoint(key string, port int) (string, bool) {
	if c.MlxHosts != nil {
		if ep, ok := c.MlxHosts[key]; ok {
			return fmt.Sprintf("http://%s/v1", ep), true
		}
	}
	if c.MlxHost != "" {
		return fmt.Sprintf("http://%s:%d/v1", c.MlxHost, port), true
	}
	return "", false
}

func loadConfig() (configFile, error) {
	var cfg configFile
	data, err := os.ReadFile("config.json")
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}
