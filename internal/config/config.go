package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	Hotkey          string `json:"hotkey"`
	AutoSave        bool   `json:"auto_save"`
	CopyToClipboard bool   `json:"copy_to_clipboard"`
}

func Load() (*Config, error) {
	configPath := getConfigPath()

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		defaultConfig := &Config{
			Hotkey:          "Ctrl+Shift+S",
			AutoSave:        false,
			CopyToClipboard: true,
		}
		return defaultConfig, nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func Save(cfg *Config) error {
	configPath := getConfigPath()

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

func getConfigPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".config", "snaphook", "config.json")
}

func GetAutoSaveDir() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, "Pictures", "SnapHook")
}

func EnsureAutoSaveDir() error {
	dir := GetAutoSaveDir()
	return os.MkdirAll(dir, 0755)
}
