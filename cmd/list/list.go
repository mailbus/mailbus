package list

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/mailbus/mailbus/pkg/adapter"
	"github.com/mailbus/mailbus/pkg/config"
	"github.com/mailbus/mailbus/pkg/core"
	"github.com/spf13/cobra"
)

var (
	subjectFilter string
	fromFilter    string
	unreadOnly    bool
	limit         int
	offset        int
	outputFormat  string
	accountName   string
)

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List messages matching criteria",
		Long: `List messages from the inbox that match the specified criteria.

Messages can be filtered by subject, sender, read status, and more.`,
		Example: `  # List all unread messages
  mailbus list --unread

  # List messages with [task] tag
  mailbus list --subject "\[task\]"

  # List messages in JSON format
  mailbus list --unread --format json

  # List last 10 messages
  mailbus list --limit 10`,
		RunE: runList,
	}

	cmd.Flags().StringVar(&subjectFilter, "subject", "", "subject filter (supports regex)")
	cmd.Flags().StringVar(&fromFilter, "from", "", "sender filter")
	cmd.Flags().BoolVar(&unreadOnly, "unread", false, "only show unread messages")
	cmd.Flags().IntVarP(&limit, "limit", "n", 20, "limit number of messages")
	cmd.Flags().IntVar(&offset, "offset", 0, "offset for pagination")
	cmd.Flags().StringVarP(&outputFormat, "format", "F", "table", "output format (table/json/compact)")
	cmd.Flags().StringVarP(&accountName, "account", "A", "", "account to use")

	return cmd
}

func runList(cmd *cobra.Command, args []string) error {
	// Load config
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get account
	account, err := getAccount(cfg, accountName)
	if err != nil {
		return err
	}

	// Get password
	password, err := account.GetPassword()
	if err != nil {
		return err
	}

	// Build filter
	filter := buildFilter()

	// Connect to IMAP server
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	imapAdapter := adapter.NewIMAPAdapter()
	connCfg := &adapter.ConnectionConfig{
		Host:     account.IMAP.Host,
		Port:     account.IMAP.Port,
		Username: account.Username,
		Password: password,
		UseTLS:   account.IMAP.UseTLS,
	}

	if err := imapAdapter.Connect(ctx, connCfg); err != nil {
		return fmt.Errorf("failed to connect to IMAP server: %w", err)
	}
	defer imapAdapter.Close()

	// Receive messages
	messages, err := imapAdapter.Receive(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to receive messages: %w", err)
	}

	// Apply limit and offset
	if offset >= len(messages) {
		messages = []*core.Message{}
	} else {
		end := offset + limit
		if end > len(messages) {
			end = len(messages)
		}
		messages = messages[offset:end]
	}

	// Print messages
	switch outputFormat {
	case "json":
		return printMessagesJSON(messages)
	case "compact":
		return printMessagesCompact(messages)
	default:
		return printMessagesTable(messages)
	}
}

func buildFilter() *core.Filter {
	filter := core.NewFilter()

	if subjectFilter != "" {
		filter.SubjectPattern = subjectFilter
	}

	if fromFilter != "" {
		filter.FromPattern = fromFilter
	}

	if unreadOnly {
		filter.UnreadOnly = true
	}

	return filter
}

func printMessagesTable(messages []*core.Message) error {
	if len(messages) == 0 {
		fmt.Println("No messages found")
		return nil
	}

	fmt.Printf("Found %d message(s):\n\n", len(messages))
	fmt.Println("─────────────────────────────────────────────────────────────────")

	for i, msg := range messages {
		fmt.Printf("#%d\n", i+1)
		fmt.Printf("From:    %s\n", msg.From)
		fmt.Printf("Subject: %s\n", msg.Subject)
		fmt.Printf("Date:    %s\n", msg.Timestamp.Format("2006-01-02 15:04:05"))
		if msg.IsSeen() {
			fmt.Printf("Status:  Read\n")
		} else {
			fmt.Printf("Status:  Unread\n")
		}
		fmt.Printf("ID:      %s\n", msg.ID)
		fmt.Println("─────────────────────────────────────────────────────────────────")
	}

	return nil
}

func printMessagesJSON(messages []*core.Message) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(messages)
}

func printMessagesCompact(messages []*core.Message) error {
	if len(messages) == 0 {
		fmt.Println("No messages found")
		return nil
	}

	for _, msg := range messages {
		status := "R"
		if !msg.IsSeen() {
			status = "U"
		}
		fmt.Printf("[%s] %s %s: %s\n",
			msg.Timestamp.Format("2006-01-02 15:04"),
			status,
			msg.From,
			msg.Subject)
	}

	return nil
}

// Helper functions
func loadConfig() (*config.Config, error) {
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

// cmd variable for flag access
var cmd = &cobra.Command{}
