package main

import (
	"encoding/json"
	"os"
)

type configFile struct {
	VertexProject string `json:"vertex_project"`
	MlxHost       string `json:"mlx_host"`
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
