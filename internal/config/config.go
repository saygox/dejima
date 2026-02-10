package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds persistent application settings.
type Config struct {
	SerialPort       string `json:"serial_port"`
	BaudRate         int    `json:"baud_rate"`
	DeviceIndex      int    `json:"device_index"`
	DevicePath       string `json:"device_path"`
	JpegQuality      int    `json:"jpeg_quality"`
	CaptureWidth     int    `json:"capture_width"`
	CaptureHeight    int    `json:"capture_height"`
	AudioDeviceID    string `json:"audio_device_id"`
	AudioVolume      int    `json:"audio_volume"` // 0-100, default 80
	AudioMuted       bool   `json:"audio_muted"`
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() *Config {
	return &Config{
		SerialPort:    "",
		BaudRate:      115200,
		DeviceIndex:   0,
		JpegQuality:   80,
		CaptureWidth:  0,
		CaptureHeight: 0,
		AudioVolume:   80,
	}
}

// configPath returns the path to the config file.
func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("getting config dir: %w", err)
	}
	return filepath.Join(dir, "kvm-like", "config.json"), nil
}

// Load reads the config from disk. Returns default config if file doesn't exist.
func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return DefaultConfig(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	cfg := DefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return cfg, nil
}

// Save writes the config to disk.
func (c *Config) Save() error {
	path, err := configPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}
