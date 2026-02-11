package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	CurrentVersion = 1
	DefaultPath    = "~/.reloquent/reloquent.yaml"
)

// Config is the top-level configuration.
type Config struct {
	Version int          `yaml:"version"`
	Source  SourceConfig `yaml:"source"`
	Target  TargetConfig `yaml:"target"`
	AWS     AWSConfig    `yaml:"aws,omitempty"`
	Logging LogConfig    `yaml:"logging,omitempty"`
}

// SourceConfig defines the source database connection.
type SourceConfig struct {
	Type           string `yaml:"type"` // postgresql or oracle
	Host           string `yaml:"host"`
	Port           int    `yaml:"port"`
	Database       string `yaml:"database"`
	Schema         string `yaml:"schema,omitempty"`
	Username       string `yaml:"username"`
	Password       string `yaml:"password"`
	SSL            bool   `yaml:"ssl,omitempty"`
	ReadOnly       bool   `yaml:"read_only,omitempty"`
	MaxConnections int    `yaml:"max_connections,omitempty"` // default 20, max 50
}

// TargetConfig defines the MongoDB target connection.
type TargetConfig struct {
	Type             string `yaml:"type"` // mongodb
	ConnectionString string `yaml:"connection_string"`
	Database         string `yaml:"database"`
}

// AWSConfig defines AWS infrastructure settings.
type AWSConfig struct {
	Region   string            `yaml:"region,omitempty"`
	Profile  string            `yaml:"profile,omitempty"`
	Platform string            `yaml:"platform,omitempty"` // emr or glue
	S3Bucket string            `yaml:"s3_bucket,omitempty"`
	Tags     map[string]string `yaml:"tags,omitempty"`
}

// LogConfig defines logging settings.
type LogConfig struct {
	Level         string `yaml:"level,omitempty"`     // debug, info, warn, error
	Directory     string `yaml:"directory,omitempty"`  // default ~/.reloquent/logs/
	RetentionDays int    `yaml:"retention_days,omitempty"` // default 30
}

// Load reads and parses the config file from the given path.
func Load(path string) (*Config, error) {
	if path == "" {
		path = ExpandHome(DefaultPath)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	if cfg.Version != CurrentVersion {
		return nil, fmt.Errorf("unsupported config version %d (expected %d)", cfg.Version, CurrentVersion)
	}

	if err := cfg.resolveSecrets(); err != nil {
		return nil, fmt.Errorf("resolving secrets: %w", err)
	}

	cfg.applyDefaults()
	return cfg, nil
}

// Save writes the config to the given path.
func (c *Config) Save(path string) error {
	if path == "" {
		path = ExpandHome(DefaultPath)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	return os.WriteFile(path, data, 0o600)
}

func (c *Config) applyDefaults() {
	if c.Source.MaxConnections == 0 {
		c.Source.MaxConnections = 20
	}
	if c.Source.MaxConnections > 50 {
		c.Source.MaxConnections = 50
	}
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	if c.Logging.Directory == "" {
		c.Logging.Directory = ExpandHome("~/.reloquent/logs/")
	}
	if c.Logging.RetentionDays == 0 {
		c.Logging.RetentionDays = 30
	}
}

var secretPattern = regexp.MustCompile(`\$\{(ENV|VAULT|AWS_SM):([^}]+)\}`)

func (c *Config) resolveSecrets() error {
	var err error
	c.Source.Password, err = ResolveValue(c.Source.Password)
	if err != nil {
		return fmt.Errorf("source password: %w", err)
	}
	c.Target.ConnectionString, err = ResolveValue(c.Target.ConnectionString)
	if err != nil {
		return fmt.Errorf("target connection string: %w", err)
	}
	return nil
}

// ResolveValue resolves secret references in a string value.
func ResolveValue(val string) (string, error) {
	matches := secretPattern.FindStringSubmatch(val)
	if matches == nil {
		return val, nil
	}

	provider := matches[1]
	ref := matches[2]

	switch provider {
	case "ENV":
		v := os.Getenv(ref)
		if v == "" {
			return "", fmt.Errorf("environment variable %s not set", ref)
		}
		return v, nil
	case "VAULT":
		return resolveVault(ref)
	case "AWS_SM":
		return resolveAWSSecretsManager(ref)
	default:
		return "", fmt.Errorf("unknown secrets provider: %s", provider)
	}
}

// ExpandHome expands ~ to the user's home directory.
func ExpandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}
