package adapter

import (
	"context"
	"fmt"
	"time"

	"github.com/mailbus/mailbus/pkg/core"
)

// ConnectionConfig holds connection configuration for mail servers
type ConnectionConfig struct {
	Host     string `yaml:"host" json:"host"`
	Port     int    `yaml:"port" json:"port"`
	Username string `yaml:"username" json:"username"`
	Password string `yaml:"password" json:"password"` // Should be from env var
	UseTLS   bool   `yaml:"use_tls" json:"use_tls"`
}

// Validate checks if the connection config is valid
func (c *ConnectionConfig) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("host is required")
	}
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}
	if c.Username == "" {
		return fmt.Errorf("username is required")
	}
	return nil
}

// Address returns the full address (host:port)
func (c *ConnectionConfig) Address() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// Adapter defines the interface for mail protocol adapters
type Adapter interface {
	// Connect establishes a connection to the mail server
	Connect(ctx context.Context, cfg *ConnectionConfig) error

	// Close closes the connection to the mail server
	Close() error

	// Send sends a message
	Send(ctx context.Context, msg *core.Message) error

	// Receive receives messages matching the given filter
	Receive(ctx context.Context, filter *core.Filter) ([]*core.Message, error)

	// Mark performs an action on a message (read, delete, move, etc.)
	Mark(ctx context.Context, msgID string, action MarkAction) error

	// IsConnected returns true if connected
	IsConnected() bool
}

// MarkAction represents actions that can be performed on a message
type MarkAction string

const (
	MarkActionSeen     MarkAction = "seen"
	MarkActionUnseen   MarkAction = "unseen"
	MarkActionDelete   MarkAction = "delete"
	MarkActionUndelete MarkAction = "undelete"
	MarkActionFlag     MarkAction = "flag"
	MarkActionUnflag   MarkAction = "unflag"
)

// SendAdapter defines the interface for sending messages
type SendAdapter interface {
	Connect(ctx context.Context, cfg *ConnectionConfig) error
	Close() error
	Send(ctx context.Context, msg *core.Message) error
	IsConnected() bool
}

// ReceiveAdapter defines the interface for receiving messages
type ReceiveAdapter interface {
	Connect(ctx context.Context, cfg *ConnectionConfig) error
	Close() error
	Receive(ctx context.Context, filter *core.Filter) ([]*core.Message, error)
	Mark(ctx context.Context, msgID string, action MarkAction) error
	IsConnected() bool
}

// RetryConfig holds retry configuration
type RetryConfig struct {
	MaxRetries    int           // Maximum number of retry attempts
	InitialDelay  time.Duration // Initial delay before first retry
	MaxDelay      time.Duration // Maximum delay between retries
	BackoffFactor float64       // Multiplier for delay after each retry
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:    3,
		InitialDelay:  time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
	}
}
