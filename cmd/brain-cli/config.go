package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server string `yaml:"server"`
	APIKey string `yaml:"api_key"`
}

const defaultServer = "https://manageaibrain.com"

func configDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".brain")
}

func configPath() string {
	return filepath.Join(configDir(), "config.yaml")
}

func loadConfig() (*Config, error) {
	cfg := &Config{Server: defaultServer}

	// Priority 1: environment variable
	if key := os.Getenv("MANAGEMENT_BRAIN_API_KEY"); key != "" {
		cfg.APIKey = key
		if srv := os.Getenv("MANAGEMENT_BRAIN_SERVER"); srv != "" {
			cfg.Server = srv
		}
		return cfg, nil
	}

	// Priority 2: config file
	data, err := os.ReadFile(configPath())
	if err != nil {
		return cfg, nil // no config file, return defaults (no key)
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}
	if cfg.Server == "" {
		cfg.Server = defaultServer
	}
	return cfg, nil
}

func saveConfig(cfg *Config) error {
	if err := os.MkdirAll(configDir(), 0700); err != nil {
		return err
	}
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(configPath(), data, 0600)
}

func mustLoadConfig() (*Config, *BrainClient) {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}
	if cfg.APIKey == "" {
		fmt.Fprintln(os.Stderr, "No API key configured. Run `brain login` to set up.")
		os.Exit(1)
	}
	return cfg, NewBrainClient(cfg.Server, cfg.APIKey)
}
