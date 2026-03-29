package send

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/mailbus/mailbus/pkg/adapter"
	"github.com/mailbus/mailbus/pkg/config"
	"github.com/mailbus/mailbus/pkg/core"
	"github.com/spf13/cobra"
)

var (
	from        string
	to          []string
	subject     string
	body        string
	file        string
	attach      []string
	headers     []string
	contentType string
	priority    string
)

var rootCmd *cobra.Command

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send",
		Short: "Send a message via email",
		Long: `Send a message to one or more recipients via SMTP.

The message body should be in JSON format by default, but can be set to plain text.`,
		Example: `  # Send a simple JSON message
  mailbus send --to agent@example.com --subject "[task] Process data" --body '{"type":"process","id":123}'

  # Send with custom headers
  mailbus send --to agent@example.com --subject "[alert] Error" --body '{"error":"timeout"}' --header "X-Priority=1"

  # Read body from file
  mailbus send --to agent@example.com --subject "[query] Search" --file message.json`,
		RunE: runSend,
	}

	cmd.Flags().StringVar(&from, "from", "", "sender email address (default: from account config)")
	cmd.Flags().StringSliceVar(&to, "to", nil, "recipient email addresses (required)")
	cmd.Flags().StringVarP(&subject, "subject", "s", "", "message subject (required)")
	cmd.Flags().StringVarP(&body, "body", "b", "", "message body")
	cmd.Flags().StringVar(&file, "file", "", "read body from file")
	cmd.Flags().StringSliceVar(&attach, "attach", nil, "file attachments")
	cmd.Flags().StringSliceVar(&headers, "header", nil, "custom headers (key=value)")
	cmd.Flags().StringVar(&contentType, "format", "json", "body format (json or text)")
	cmd.Flags().StringVar(&priority, "priority", "normal", "message priority (high, normal, low)")

	cmd.MarkFlagRequired("to")
	cmd.MarkFlagRequired("subject")

	rootCmd = cmd
	return cmd
}

func runSend(cmd *cobra.Command, args []string) error {
	// Load config
	cfg, err := loadConfigFromCmd(cmd)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get account
	accountName, _ := cmd.Flags().GetString("account")
	account, err := getAccount(cfg, accountName)
	if err != nil {
		return err
	}

	// Get password
	password, err := account.GetPassword()
	if err != nil {
		return err
	}

	// Create message
	msg := core.NewMessage()

	// Set from
	if from == "" {
		from = account.From
		if from == "" {
			from = account.Username
		}
	}
	msg.From = from

	// Set recipients
	msg.To = to

	// Set subject
	msg.Subject = subject

	// Set body
	if file != "" {
		data, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
		msg.Body = string(data)
	} else {
		msg.Body = body
	}

	// Set content type
	if contentType == "json" {
		msg.ContentType = "application/json"
	} else {
		msg.ContentType = "text/plain"
	}

	// Add custom headers
	for _, h := range headers {
		parts := splitHeader(h)
		if len(parts) == 2 {
			msg.AddHeader(parts[0], parts[1])
		}
	}

	// Add priority header
	if priority != "normal" {
		msg.AddHeader("X-Priority", priorityMap(priority))
		msg.AddHeader("Priority", priorityMap(priority))
	}

	// Add attachments (TODO: implement attachment handling)
	if len(attach) > 0 {
		return fmt.Errorf("attachments not yet implemented")
	}

	// Validate message
	if err := msg.Validate(); err != nil {
		return fmt.Errorf("invalid message: %w", err)
	}

	// Connect to SMTP server
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	smtpAdapter := adapter.NewSMTPAdapter()
	connCfg := &adapter.ConnectionConfig{
		Host:     account.SMTP.Host,
		Port:     account.SMTP.Port,
		Username: account.Username,
		Password: password,
		UseTLS:   account.SMTP.UseTLS,
	}

	if err := smtpAdapter.Connect(ctx, connCfg); err != nil {
		return fmt.Errorf("failed to connect to SMTP server: %w", err)
	}
	defer smtpAdapter.Close()

	// Send message
	if err := smtpAdapter.Send(ctx, msg); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	fmt.Printf("Message sent successfully: %s\n", msg.ID)
	return nil
}

func loadConfigFromCmd(cmd *cobra.Command) (*config.Config, error) {
	configPath, _ := cmd.Flags().GetString("config")
	if configPath == "" {
		var err error
		configPath, err = config.ConfigPath()
		if err != nil {
			return nil, err
		}
	}

	return config.LoadConfig(configPath)
}

func getAccount(cfg *config.Config, name string) (*config.AccountConfig, error) {
	if name == "" {
		name = cfg.DefaultAccount
	}
	return cfg.GetAccount(name)
}

func splitHeader(h string) []string {
	for i := 0; i < len(h); i++ {
		if h[i] == '=' {
			return []string{h[:i], h[i+1:]}
		}
	}
	return []string{h}
}

func priorityMap(p string) string {
	switch p {
	case "high":
		return "1"
	case "low":
		return "5"
	default:
		return "3"
	}
}
