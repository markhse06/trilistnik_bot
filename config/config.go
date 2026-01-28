package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Telegram TelegramConfig `yaml:"telegram"`
	Database DatabaseConfig `yaml:"database"`
}

type TelegramConfig struct {
	Token    string  `yaml:"token"`
	AdminIDs []int64 `yaml:"admin_ids"`
	GroupIDs []int64 `yaml:"group_ids"`
}

type DatabaseConfig struct {
	DSN string `yaml:"dsn"`
}

func Load(configPath string) (*Config, error) {
	// Проверяем .env переменные
	if token := os.Getenv("TELEGRAM_TOKEN"); token != "" {
		return loadFromEnv()
	}

	// Иначе из config.yml
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}

func loadFromEnv() (*Config, error) {
	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
		return nil, fmt.Errorf("TELEGRAM_TOKEN not set")
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return nil, fmt.Errorf("DATABASE_URL not set")
	}

	adminIDsStr := os.Getenv("ADMIN_IDS")
	var adminIDs []int64
	if adminIDsStr != "" {
		for _, id := range strings.Split(adminIDsStr, ",") {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			parsed, _ := strconv.ParseInt(id, 10, 64)
			adminIDs = append(adminIDs, parsed)
		}
	}

	groupIDsStr := os.Getenv("GROUP_IDS")
	var groupIDs []int64
	if groupIDsStr != "" {
		for _, id := range strings.Split(groupIDsStr, ",") {
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			parsed, _ := strconv.ParseInt(id, 10, 64)
			groupIDs = append(groupIDs, parsed)
		}
	}

	return &Config{
		Telegram: TelegramConfig{
			Token:    token,
			AdminIDs: adminIDs,
			GroupIDs: groupIDs,
		},
		Database: DatabaseConfig{
			DSN: dsn,
		},
	}, nil
}

func (cfg *Config) IsAdmin(id int64) bool {
	for _, adminID := range cfg.Telegram.AdminIDs {
		if adminID == id {
			return true
		}
	}
	return false
}
