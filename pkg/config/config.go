package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mailbus/mailbus/pkg/crypto"
	"gopkg.in/yaml.v3"
)

// Config represents the mailbus configuration
type Config struct {
	DefaultAccount string                    `yaml:"default_account"`
	Accounts       map[string]*AccountConfig `yaml:"accounts"`
	Global         *GlobalConfig             `yaml:"global"`
}

// AccountConfig represents a single mail account configuration
type AccountConfig struct {
	Name    string `yaml:"name"` // Account identifier
	IMAP    struct {
		Host   string `yaml:"host"`
		Port   int    `yaml:"port"`
		UseTLS bool   `yaml:"use_tls"`
	} `yaml:"imap"`
	SMTP struct {
		Host   string `yaml:"host"`
		Port   int    `yaml:"port"`
		UseTLS bool   `yaml:"use_tls"`
	} `yaml:"smtp"`
	Username         string `yaml:"username"`
	PasswordEnv      string `yaml:"password_env"`       // Environment variable name (optional)
	PasswordEncrypted string `yaml:"password_encrypted"` // Encrypted password (optional)
	From             string `yaml:"from"`                // Default from address
}

// GlobalConfig represents global mailbus settings
type GlobalConfig struct {
	PollInterval   time.Duration `yaml:"poll_interval"`   // Default polling interval
	BatchSize      int            `yaml:"batch_size"`      // Default batch size
	Timeout        time.Duration `yaml:"timeout"`         // Default operation timeout
	MaxRetries     int            `yaml:"max_retries"`     // Default max retries
	Verbose        bool           `yaml:"verbose"`         // Verbose output
	LogLevel       string         `yaml:"log_level"`       // Log level
	LogFile        string         `yaml:"log_file"`        // Log file path
	HandlerTimeout time.Duration `yaml:"handler_timeout"` // Default handler timeout
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		DefaultAccount: "default",
		Accounts: make(map[string]*AccountConfig),
		Global: &GlobalConfig{
			PollInterval:   30 * time.Second,
			BatchSize:      20,
			Timeout:        30 * time.Second,
			MaxRetries:     3,
			Verbose:        false,
			LogLevel:       "info",
			HandlerTimeout: 60 * time.Second,
		},
	}
}

// GetAccount returns an account configuration by name
func (c *Config) GetAccount(name string) (*AccountConfig, error) {
	if name == "" {
		name = c.DefaultAccount
	}

	account, ok := c.Accounts[name]
	if !ok {
		return nil, fmt.Errorf("account '%s' not found in configuration", name)
	}

	return account, nil
}

// GetPassword returns the password for an account
// Priority: password_encrypted > password_env
func (a *AccountConfig) GetPassword() (string, error) {
	// Priority 1: Use encrypted password (requires MAILBUS_CRYPTO_KEY)
	if a.PasswordEncrypted != "" {
		cryptoKey := os.Getenv("MAILBUS_CRYPTO_KEY")
		if cryptoKey == "" {
			return "", fmt.Errorf("password_encrypted is set but MAILBUS_CRYPTO_KEY environment variable is not set")
		}
		password, err := crypto.DecryptPassword(a.PasswordEncrypted, cryptoKey)
		if err != nil {
			return "", fmt.Errorf("failed to decrypt password: %w", err)
		}
		return password, nil
	}

	// Priority 2: Use environment variable (existing method)
	if a.PasswordEnv != "" {
		password := os.Getenv(a.PasswordEnv)
		if password == "" {
			return "", fmt.Errorf("environment variable '%s' is not set", a.PasswordEnv)
		}
		return password, nil
	}

	return "", fmt.Errorf("no password configured for account '%s' (set password_encrypted or password_env)", a.Name)
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.DefaultAccount == "" {
		return fmt.Errorf("default_account is required")
	}

	if len(c.Accounts) == 0 {
		return fmt.Errorf("at least one account must be configured")
	}

	if _, ok := c.Accounts[c.DefaultAccount]; !ok {
		return fmt.Errorf("default_account '%s' not found in accounts", c.DefaultAccount)
	}

	for name, account := range c.Accounts {
		if err := account.Validate(); err != nil {
			return fmt.Errorf("account '%s': %w", name, err)
		}
	}

	return nil
}

// Validate validates the account configuration
func (a *AccountConfig) Validate() error {
	if a.IMAP.Host == "" {
		return fmt.Errorf("IMAP host is required")
	}
	if a.IMAP.Port <= 0 || a.IMAP.Port > 65535 {
		return fmt.Errorf("IMAP port must be between 1 and 65535")
	}
	if a.SMTP.Host == "" {
		return fmt.Errorf("SMTP host is required")
	}
	if a.SMTP.Port <= 0 || a.SMTP.Port > 65535 {
		return fmt.Errorf("SMTP port must be between 1 and 65535")
	}
	if a.Username == "" {
		return fmt.Errorf("username is required")
	}
	if a.PasswordEnv == "" && a.PasswordEncrypted == "" {
		return fmt.Errorf("either password_env or password_encrypted is required")
	}
	return nil
}

// LoadConfig loads configuration from a file
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config := DefaultConfig()
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set account names
	for name, account := range config.Accounts {
		account.Name = name
	}

	if err := config.Validate(); err != nil {
		return nil, err
	}

	return config, nil
}

// SaveConfig saves configuration to a file
func SaveConfig(path string, config *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// ConfigPath returns the default configuration file path
func ConfigPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".mailbus")
	return filepath.Join(configDir, "config.yaml"), nil
}
