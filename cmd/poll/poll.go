package poll

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
	subjectFilter     string
	fromFilter        string
	unreadOnly        bool
	once              bool
	continuous        bool
	interval          int
	handler           string
	handlerTimeout    int
	onError           string
	replyWith         bool
	markAfter         string
	folder            string
	outputFormat      string
	downloadAttachments bool
	attachDir          string
)

var rootCmd *cobra.Command

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "poll",
		Short: "Poll for and process incoming messages",
		Long: `Check for new messages matching criteria and optionally process them with a handler.

The poll command can run once or continuously monitor for new messages.`,
		Example: `  # List unread messages with [task] tag
  mailbus poll --unread --subject "\[task\]"

  # Execute handler for matching messages once
  mailbus poll --subject "\[task\]" --handler "./process.sh" --once

  # Continuous monitoring with handler
  mailbus poll --subject "\[alert\]" --handler "./alert.sh" --continuous --interval 30`,
		RunE: runPoll,
	}

	cmd.Flags().StringVar(&subjectFilter, "subject", "", "subject filter (supports regex)")
	cmd.Flags().StringVar(&fromFilter, "from", "", "sender filter")
	cmd.Flags().BoolVar(&unreadOnly, "unread", false, "only process unread messages")
	cmd.Flags().BoolVar(&once, "once", false, "process messages once and exit")
	cmd.Flags().BoolVarP(&continuous, "continuous", "c", false, "continuously poll for messages")
	cmd.Flags().IntVarP(&interval, "interval", "i", 30, "polling interval in seconds")
	cmd.Flags().StringVarP(&handler, "handler", "H", "", "handler command to execute")
	cmd.Flags().IntVar(&handlerTimeout, "handler-timeout", 60, "handler timeout in seconds")
	cmd.Flags().StringVar(&onError, "on-error", "continue", "error handling strategy (continue/stop/retry)")
	cmd.Flags().BoolVar(&replyWith, "reply-with-result", false, "send handler result as reply")
	cmd.Flags().StringVar(&markAfter, "mark-after", "read", "mark action after processing (read/delete/none)")
	cmd.Flags().StringVarP(&folder, "folder", "F", "INBOX", "IMAP folder to check")
	cmd.Flags().StringVar(&outputFormat, "format", "table", "output format (table/json/compact)")

	// Attachment flags
	cmd.Flags().BoolVar(&downloadAttachments, "download-attachments", false, "download message attachments")
	cmd.Flags().StringVar(&attachDir, "attach-dir", "", "directory to save attachments (default: current dir)")

	rootCmd = cmd
	return cmd
}

func runPoll(cmd *cobra.Command, args []string) error {
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

	// Build filter
	filter := buildFilter()

	// Run polling
	if once {
		return pollOnce(cfg, account, password, filter)
	}

	if continuous {
		return pollContinuous(cfg, account, password, filter)
	}

	// Default: list messages
	return listMessages(cfg, account, password, filter)
}

func pollOnce(cfg *config.Config, account *config.AccountConfig, password string, filter *core.Filter) error {
	if handler == "" {
		return fmt.Errorf("handler is required when using --once")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Connect and receive messages
	messages, err := receiveMessages(ctx, account, password, filter)
	if err != nil {
		return err
	}

	if len(messages) == 0 {
		fmt.Println("No matching messages found")
		return nil
	}

	// Process messages
	fmt.Printf("Processing %d message(s)\n", len(messages))
	for _, msg := range messages {
		if err := processMessage(ctx, cfg, account, password, msg); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
	}

	return nil
}

func pollContinuous(cfg *config.Config, account *config.AccountConfig, password string, filter *core.Filter) error {
	if handler == "" {
		return fmt.Errorf("handler is required when using --continuous")
	}

	fmt.Printf("Starting continuous polling (interval: %ds)\n", interval)
	fmt.Println("Press Ctrl+C to stop")

	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)

			messages, err := receiveMessages(ctx, account, password, filter)
			cancel()

			if err != nil {
				fmt.Fprintf(os.Stderr, "Error receiving messages: %v\n", err)
				if onError == "stop" {
					return err
				}
				continue
			}

			if len(messages) > 0 {
				fmt.Printf("Processing %d message(s)\n", len(messages))
				for _, msg := range messages {
					ctx, cancel = context.WithTimeout(context.Background(), time.Duration(handlerTimeout)*time.Second)
					err := processMessage(ctx, cfg, account, password, msg)
					cancel()

					if err != nil {
						fmt.Fprintf(os.Stderr, "Error: %v\n", err)
						if onError == "stop" {
							return err
						}
					}
				}
			}

		case <-interruptChan():
			fmt.Println("\nStopping poll...")
			return nil
		}
	}
}

func listMessages(cfg *config.Config, account *config.AccountConfig, password string, filter *core.Filter) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	messages, err := receiveMessages(ctx, account, password, filter)
	if err != nil {
		return err
	}

	if len(messages) == 0 {
		fmt.Println("No matching messages found")
		return nil
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

func receiveMessages(ctx context.Context, account *config.AccountConfig, password string, filter *core.Filter) ([]*core.Message, error) {
	imapAdapter := adapter.NewIMAPAdapter()
	connCfg := &adapter.ConnectionConfig{
		Host:     account.IMAP.Host,
		Port:     account.IMAP.Port,
		Username: account.Username,
		Password: password,
		UseTLS:   account.IMAP.UseTLS,
	}

	if err := imapAdapter.Connect(ctx, connCfg); err != nil {
		return nil, fmt.Errorf("failed to connect to IMAP server: %w", err)
	}
	defer imapAdapter.Close()

	return imapAdapter.Receive(ctx, filter)
}

func processMessage(ctx context.Context, cfg *config.Config, account *config.AccountConfig, password string, msg *core.Message) error {
	fmt.Printf("Processing: %s\n", msg.Subject)

	// Download attachments if requested
	if downloadAttachments {
		if err := downloadAttachmentsFromMessage(ctx, account, password, msg); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to download attachments: %v\n", err)
		}
	}

	// Execute handler
	result, err := executeHandler(ctx, msg)
	if err != nil {
		return err
	}

	// Print result
	if result.Message != "" {
		fmt.Printf("Result: %s\n", result.Message)
	}

	// Send reply if requested
	if replyWith && result.ReplyBody != "" {
		if err := sendReply(ctx, account, password, msg, result); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to send reply: %v\n", err)
		}
	}

	// Mark message
	if markAfter != "none" {
		if err := markMessage(ctx, account, password, msg, markAfter); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to mark message: %v\n", err)
		}
	}

	return nil
}

func executeHandler(ctx context.Context, msg *core.Message) (*core.HandlerResult, error) {
	if handler == "" {
		return nil, fmt.Errorf("no handler specified")
	}

	// Parse handler command
	parts := splitCommand(handler)
	execHandler := core.NewExecHandler(parts[0], parts[1:]...)
	execHandler.Timeout = time.Duration(handlerTimeout) * time.Second

	return execHandler.Handle(ctx, msg)
}

func sendReply(ctx context.Context, account *config.AccountConfig, password string, originalMsg *core.Message, result *core.HandlerResult) error {
	reply := core.NewMessage()
	reply.From = account.Username
	reply.To = []string{originalMsg.From}

	if result.ReplySubj != "" {
		reply.Subject = result.ReplySubj
	} else {
		reply.Subject = "Re: " + originalMsg.Subject
	}

	reply.Body = result.ReplyBody
	reply.ContentType = "text/plain"

	smtpAdapter := adapter.NewSMTPAdapter()
	connCfg := &adapter.ConnectionConfig{
		Host:     account.SMTP.Host,
		Port:     account.SMTP.Port,
		Username: account.Username,
		Password: password,
		UseTLS:   account.SMTP.UseTLS,
	}

	if err := smtpAdapter.Connect(ctx, connCfg); err != nil {
		return err
	}
	defer smtpAdapter.Close()

	return smtpAdapter.Send(ctx, reply)
}

func markMessage(ctx context.Context, account *config.AccountConfig, password string, msg *core.Message, action string) error {
	imapAdapter := adapter.NewIMAPAdapter()
	connCfg := &adapter.ConnectionConfig{
		Host:     account.IMAP.Host,
		Port:     account.IMAP.Port,
		Username: account.Username,
		Password: password,
		UseTLS:   account.IMAP.UseTLS,
	}

	if err := imapAdapter.Connect(ctx, connCfg); err != nil {
		return err
	}
	defer imapAdapter.Close()

	var markAction adapter.MarkAction
	switch action {
	case "read":
		markAction = adapter.MarkActionSeen
	case "delete":
		markAction = adapter.MarkActionDelete
	default:
		return nil
	}

	return imapAdapter.Mark(ctx, msg.ID, markAction)
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

func splitCommand(cmd string) []string {
	// Simple command splitting
	var parts []string
	current := ""

	for _, c := range cmd {
		switch c {
		case ' ', '\t':
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		default:
			current += string(c)
		}
	}

	if current != "" {
		parts = append(parts, current)
	}

	return parts
}

func interruptChan() chan struct{} {
	ch := make(chan struct{})
	go func() {
		// Simple interrupt handling
		<-make(chan struct{})
		close(ch)
	}()
	return ch
}

func printMessagesTable(messages []*core.Message) error {
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

// downloadAttachmentsFromMessage downloads attachments from a message
// Note: This is a placeholder implementation. Full MIME parsing is needed
// to extract actual attachment files from the email. This requires
// modifying the IMAP adapter to fetch and parse the full MIME structure.
func downloadAttachmentsFromMessage(ctx context.Context, account *config.AccountConfig, password string, msg *core.Message) error {
	// Check if message has attachment metadata in headers
	attachmentsHeader, hasAttachments := msg.GetHeader("X-MailBus-Attachments")
	if !hasAttachments {
		// Check if message has attachments info from front matter
		attachmentsHeader, hasAttachments = msg.GetHeader("X-MailBus-Attachments")
	}

	if !hasAttachments {
		return nil // No attachments to download
	}

	// Determine target directory
	targetDir := attachDir
	if targetDir == "" {
		targetDir = "."
	}

	// Create directory if needed
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create attachment directory: %w", err)
	}

	fmt.Printf("  Attachments: %s\n", attachmentsHeader)

	// Note: Actual attachment download requires:
	// 1. Fetching the full MIME message structure from IMAP
	// 2. Parsing multipart/* content using go-message
	// 3. Extracting each attachment part
	// 4. Writing files to targetDir with checksum verification
	//
	// For now, this is a placeholder that logs the attachment info
	// The full implementation requires significant changes to the IMAP adapter

	return fmt.Errorf("attachment download requires full MIME parsing (not yet implemented)")
}
