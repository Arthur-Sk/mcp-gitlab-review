package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
	uconfig "go.uber.org/config"
)

type Config struct {
	GitLab GitLabConfig `yaml:"gitlab"`
	Cache  CacheConfig  `yaml:"cache"`
	MCP    MCPConfig    `yaml:"mcp"`
}

type GitLabConfig struct {
	APIURL string `yaml:"apiURL"`
	Token  string `yaml:"token"`
}

type CacheConfig struct {
	TTL     time.Duration `yaml:"ttl"`
	MaxSize int           `yaml:"maxSize"`
}

type MCPConfig struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

func New() (*Config, error) {
	loadEnvFile()

	configSource, err := resolveConfigSource()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	provider, err := uconfig.NewYAML(
		configSource,
		uconfig.Expand(os.LookupEnv),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	var cfg Config
	if err := provider.Get("").Populate(&cfg); err != nil {
		return nil, fmt.Errorf("failed to populate config: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

func loadEnvFile() {
	_ = godotenv.Load()

	execPath, err := os.Executable()
	if err == nil {
		execDir := filepath.Dir(execPath)
		_ = godotenv.Load(filepath.Join(execDir, ".env"))
	}
}

func resolveConfigSource() (uconfig.YAMLOption, error) {
	candidates := []string{
		"config/base.yaml",
	}

	execPath, err := os.Executable()
	if err == nil {
		execDir := filepath.Dir(execPath)
		candidates = append([]string{
			filepath.Join(execDir, "config", "base.yaml"),
		}, candidates...)
	}

	for _, path := range candidates {
		if data, err := os.ReadFile(path); err == nil {
			return uconfig.Source(bytes.NewReader(data)), nil
		}
	}

	return nil, fmt.Errorf("config/base.yaml not found (checked paths relative to executable and CWD)")
}

func (c *Config) validate() error {
	if c.GitLab.Token == "" {
		return fmt.Errorf("GITLAB_PERSONAL_ACCESS_TOKEN is required")
	}

	if c.GitLab.APIURL == "" {
		return fmt.Errorf("GITLAB_API_URL is required")
	}

	return nil
}
