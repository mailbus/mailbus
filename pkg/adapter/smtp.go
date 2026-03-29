package adapter

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/smtp"

	"github.com/mailbus/mailbus/pkg/core"
)

// SMTPAdapter implements SMTP sending functionality
type SMTPAdapter struct {
	client *smtp.Client
	config *ConnectionConfig
}

// NewSMTPAdapter creates a new SMTP adapter
func NewSMTPAdapter() *SMTPAdapter {
	return &SMTPAdapter{}
}

// Connect establishes a connection to the SMTP server
func (a *SMTPAdapter) Connect(ctx context.Context, cfg *ConnectionConfig) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	a.config = cfg

	// Connect to server
	var client *smtp.Client
	var err error

	if cfg.UseTLS {
		// Direct TLS connection
		tlsConfig := &tls.Config{
			ServerName:         cfg.Host,
			InsecureSkipVerify: false,
		}

		conn, err := tls.Dial("tcp", cfg.Address(), tlsConfig)
		if err != nil {
			return fmt.Errorf("failed to connect with TLS: %w", err)
		}

		client, err = smtp.NewClient(conn, cfg.Host)
		if err != nil {
			conn.Close()
			return fmt.Errorf("failed to create SMTP client: %w", err)
		}
	} else {
		// Plain connection, will try STARTTLS
		client, err = smtp.Dial(cfg.Address())
		if err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}

		// Try to upgrade to TLS if available
		if ok, _ := client.Extension("STARTTLS"); ok {
			if err := client.StartTLS(&tls.Config{
				ServerName:         cfg.Host,
				InsecureSkipVerify: false,
			}); err != nil {
				client.Close()
				return fmt.Errorf("failed to start TLS: %w", err)
			}
		}
	}

	a.client = client

	// Authenticate if password is provided
	if cfg.Password != "" {
		auth := smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
		if err := client.Auth(auth); err != nil {
			client.Close()
			return fmt.Errorf("authentication failed: %w", err)
		}
	}

	return nil
}

// Close closes the connection to the SMTP server
func (a *SMTPAdapter) Close() error {
	if a.client != nil {
		err := a.client.Close()
		a.client = nil
		return err
	}
	return nil
}

// IsConnected returns true if connected to the server
func (a *SMTPAdapter) IsConnected() bool {
	return a.client != nil
}

// Send sends a message
func (a *SMTPAdapter) Send(ctx context.Context, msg *core.Message) error {
	if a.client == nil {
		return fmt.Errorf("not connected to SMTP server")
	}

	if err := msg.Validate(); err != nil {
		return fmt.Errorf("invalid message: %w", err)
	}

	// Build MIME message
	mimeMsg, err := msg.BuildMIME()
	if err != nil {
		return fmt.Errorf("failed to build MIME message: %w", err)
	}

	// Set sender
	if err := a.client.Mail(msg.From); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Add recipients (To)
	for _, to := range msg.To {
		if err := a.client.Rcpt(to); err != nil {
			return fmt.Errorf("failed to add recipient %s: %w", to, err)
		}
	}

	// Add recipients (Cc)
	for _, cc := range msg.Cc {
		if err := a.client.Rcpt(cc); err != nil {
			return fmt.Errorf("failed to add CC recipient %s: %w", cc, err)
		}
	}

	// Note: BCC recipients are added via RCPT but not in headers
	// We don't store BCC in the message headers

	// Get data writer
	writer, err := a.client.Data()
	if err != nil {
		return fmt.Errorf("failed to get data writer: %w", err)
	}
	defer writer.Close()

	// Write message
	_, err = fmt.Fprint(writer, mimeMsg)
	if err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

// Receive is not implemented for SMTP adapter
func (a *SMTPAdapter) Receive(ctx context.Context, filter *core.Filter) ([]*core.Message, error) {
	return nil, fmt.Errorf("SMTP adapter cannot receive messages")
}

// Mark is not implemented for SMTP adapter
func (a *SMTPAdapter) Mark(ctx context.Context, msgID string, action MarkAction) error {
	return fmt.Errorf("SMTP adapter cannot mark messages")
}

// SendBatch sends multiple messages efficiently
func (a *SMTPAdapter) SendBatch(ctx context.Context, messages []*core.Message) error {
	for _, msg := range messages {
		if err := a.Send(ctx, msg); err != nil {
			return fmt.Errorf("failed to send message %s: %w", msg.ID, err)
		}
	}
	return nil
}

// SendQuick sends a message quickly with minimal setup
func (a *SMTPAdapter) SendQuick(ctx context.Context, from string, to []string, subject, body string) error {
	msg := core.NewMessage()
	msg.From = from
	msg.To = to
	msg.Subject = subject
	msg.Body = body
	msg.ContentType = "text/plain"

	return a.Send(ctx, msg)
}

// VerifyConnection verifies the connection is still alive
func (a *SMTPAdapter) VerifyConnection() error {
	if a.client == nil {
		return fmt.Errorf("not connected")
	}

	// Try to send a NOOP command
	return a.client.Noop()
}

// SupportsExtension checks if the server supports a specific extension
func (a *SMTPAdapter) SupportsExtension(extension string) bool {
	if a.client == nil {
		return false
	}

	ok, _ := a.client.Extension(extension)
	return ok
}

// GetServerInfo returns information about the SMTP server
func (a *SMTPAdapter) GetServerInfo() map[string]interface{} {
	info := make(map[string]interface{})

	if a.client == nil {
		return info
	}

	if a.client != nil {
		// Check for common extensions
		info["starttls"] = a.SupportsExtension("STARTTLS")
		info["8bitmime"] = a.SupportsExtension("8BITMIME")
		info["size"] = a.SupportsExtension("SIZE")
		info["pipelining"] = a.SupportsExtension("PIPELINING")
	}

	return info
}

// TestSend sends a test message to verify the connection
func (a *SMTPAdapter) TestSend(ctx context.Context, testAddress string) error {
	if a.client == nil {
		return fmt.Errorf("not connected")
	}

	msg := core.NewMessage()
	msg.From = a.config.Username
	msg.To = []string{testAddress}
	msg.Subject = "MailBus Test"
	msg.Body = `{"type":"test","message":"This is a test message from MailBus"}`
	msg.ContentType = "application/json"

	return a.Send(ctx, msg)
}

// MessageBuilder helps build messages incrementally
type MessageBuilder struct {
	msg *core.Message
}

// NewMessageBuilder creates a new message builder
func NewMessageBuilder() *MessageBuilder {
	return &MessageBuilder{
		msg: core.NewMessage(),
	}
}

// From sets the sender
func (b *MessageBuilder) From(from string) *MessageBuilder {
	b.msg.From = from
	return b
}

// To adds recipients
func (b *MessageBuilder) To(to ...string) *MessageBuilder {
	b.msg.To = append(b.msg.To, to...)
	return b
}

// Subject sets the subject
func (b *MessageBuilder) Subject(subject string) *MessageBuilder {
	b.msg.Subject = subject
	return b
}

// Body sets the body
func (b *MessageBuilder) Body(body string) *MessageBuilder {
	b.msg.Body = body
	return b
}

// ContentType sets the content type
func (b *MessageBuilder) ContentType(contentType string) *MessageBuilder {
	b.msg.ContentType = contentType
	return b
}

// AddHeader adds a custom header
func (b *MessageBuilder) AddHeader(key, value string) *MessageBuilder {
	b.msg.AddHeader(key, value)
	return b
}

// Build returns the built message
func (b *MessageBuilder) Build() *core.Message {
	return b.msg
}
