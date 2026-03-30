package send

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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
	meta        string
	markdown    string
	fields      []string
	attach      []string
	attachDesc  []string
	headers     []string
	priority    string
	attachDir   string
)

var rootCmd *cobra.Command

func NewCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "send",
		Short: "Send a message via email",
		Long: `Send a message to one or more recipients via SMTP.

Messages use Front Matter + Markdown format by default:
  - Front Matter (YAML): structured metadata
  - Markdown: human-readable content

The email transport layer (From, To, Subject) is handled separately.
Front Matter only contains business metadata.`,
		Example: `  # Send from a single file with front matter + markdown
  mailbus send --to agent@example.com --subject "[task] Analysis" --file request.md

  # Separate metadata and content files
  mailbus send --to agent@example.com --subject "[task] Analysis" \
    --meta meta.yaml --body content.md

  # Inline message with fields
  mailbus send --to agent@example.com --subject "[task] Analysis" \
    --field "task.type=analysis" --field "priority=high" \
    --markdown "# Analyze Q1 data\n\nPlease analyze..."

  # With attachments
  mailbus send --to agent@example.com --file request.md \
    --attach data.csv --attach-desc "data.csv:Q1 sales data"`,
		RunE: runSend,
	}

	cmd.Flags().StringVar(&from, "from", "", "sender email address (default: from account config)")
	cmd.Flags().StringSliceVar(&to, "to", nil, "recipient email addresses (required)")
	cmd.Flags().StringVarP(&subject, "subject", "s", "", "message subject (required)")

	// Content source flags
	cmd.Flags().StringVar(&file, "file", "", "read complete message (front matter + markdown) from file")
	cmd.Flags().StringVar(&meta, "meta", "", "read front matter metadata from YAML file")
	cmd.Flags().StringVar(&body, "body", "", "read markdown content from file")
	cmd.Flags().StringVar(&markdown, "markdown", "", "inline markdown content")
	cmd.Flags().StringSliceVar(&fields, "field", nil, "add metadata field (key=value, supports dot notation)")

	// Attachment flags
	cmd.Flags().StringSliceVar(&attach, "attach", nil, "file attachments")
	cmd.Flags().StringSliceVar(&attachDesc, "attach-desc", nil, "attachment description (name:description)")
	cmd.Flags().StringVar(&attachDir, "attach-dir", "", "directory for attachment files (default: current dir)")

	// Other flags
	cmd.Flags().StringSliceVar(&headers, "header", nil, "custom headers (key=value)")
	cmd.Flags().StringVar(&priority, "priority", "normal", "message priority (high, normal, low)")

	cmd.MarkFlagRequired("to")
	cmd.MarkFlagRequired("subject")

	rootCmd = cmd
	return cmd
}

func runSend(cmd *cobra.Command, args []string) error {
	// Validate input combinations
	if err := validateFlags(); err != nil {
		return err
	}

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

	// Build message
	msg, err := buildMessage()
	if err != nil {
		return fmt.Errorf("failed to build message: %w", err)
	}

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

// validateFlags checks that flag combinations are valid
func validateFlags() error {
	// Check if we have a content source
	hasFile := file != ""
	hasMeta := meta != ""
	hasBody := body != ""
	hasMarkdown := markdown != ""

	contentCount := 0
	if hasFile {
		contentCount++
	}
	if hasMeta {
		contentCount++
	}
	if hasMarkdown {
		contentCount++
	}

	if contentCount == 0 && hasBody {
		// Only --body without metadata is allowed (plain markdown)
		return nil
	}

	if contentCount == 0 {
		return fmt.Errorf("must specify one of: --file, --meta (with --body/--markdown), or --markdown")
	}

	if hasFile && (hasMeta || hasBody || hasMarkdown) {
		return fmt.Errorf("--file cannot be combined with --meta, --body, or --markdown")
	}

	if hasMeta && !hasBody && !hasMarkdown {
		return fmt.Errorf("--meta requires --body or --markdown")
	}

	return nil
}

// buildMessage constructs the message with front matter and content
func buildMessage() (*core.Message, error) {
	msg := core.NewMessage()
	msg.ContentType = core.ContentTypeMarkdown
	msg.AddHeader(core.HeaderMailBusFormat, core.FormatFrontMatter)

	var fm *core.FrontMatter
	var markdownContent string
	var err error

	// Determine content source
	if file != "" {
		// Single file mode: read file and parse front matter
		content, err := os.ReadFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read file: %w", err)
		}

		fm, markdownContent, err = core.ParseMessage(string(content))
		if err != nil {
			return nil, fmt.Errorf("failed to parse message: %w", err)
		}
	} else if meta != "" {
		// Split mode: separate metadata and content
		metaContent, err := os.ReadFile(meta)
		if err != nil {
			return nil, fmt.Errorf("failed to read metadata file: %w", err)
		}

		fm, err = core.ParseFrontMatterFile(string(metaContent))
		if err != nil {
			return nil, fmt.Errorf("failed to parse metadata: %w", err)
		}

		// Get markdown content
		if body != "" {
			content, err := os.ReadFile(body)
			if err != nil {
				return nil, fmt.Errorf("failed to read body file: %w", err)
			}
			markdownContent = string(content)
		} else {
			markdownContent = markdown
		}
	} else {
		// Inline mode: build from fields and markdown
		if len(fields) > 0 {
			fm = &core.FrontMatter{}
			for _, field := range fields {
				fieldMap, err := core.ParseField(field)
				if err != nil {
					return nil, fmt.Errorf("invalid field '%s': %w", field, err)
				}
				// Merge field into front matter
				fm = mergeFieldMap(fm, fieldMap)
			}
		}
		markdownContent = markdown
	}

	// Handle attachments
	if len(attach) > 0 {
		if fm == nil {
			fm = &core.FrontMatter{}
		}

		// Build attachment descriptions map
		descMap := make(map[string]string)
		for _, desc := range attachDesc {
			parts := strings.SplitN(desc, ":", 2)
			if len(parts) == 2 {
				descMap[parts[0]] = parts[1]
			}
		}

		// Process attachments
		for _, attachPath := range attach {
			info, err := processAttachment(attachPath, descMap[attachPath])
			if err != nil {
				return nil, fmt.Errorf("failed to process attachment '%s': %w", attachPath, err)
			}
			fm.AddAttachment(info.Name, info.Size, info.Checksum, info.Desc)
		}

		// Note: Actual attachment files will be handled by the SMTP adapter
		// For now, we just store the metadata in front matter
		msg.AddHeader("X-MailBus-Attachments", strings.Join(attach, ","))
	}

	// Generate message body
	messageBody, err := core.GenerateMessage(fm, markdownContent)
	if err != nil {
		return nil, fmt.Errorf("failed to generate message: %w", err)
	}

	msg.Body = messageBody
	return msg, nil
}

// processAttachment processes an attachment file and returns its info
func processAttachment(path string, desc string) (*core.AttachmentInfo, error) {
	// Resolve path
	if attachDir != "" {
		path = filepath.Join(attachDir, path)
	}

	// Get file info
	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	// Calculate checksum
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file for checksum: %w", err)
	}

	hash := sha256.Sum256(data)
	checksum := "sha256:" + hex.EncodeToString(hash[:])

	return &core.AttachmentInfo{
		Name:     filepath.Base(path),
		Size:     fileInfo.Size(),
		Checksum: checksum,
		Desc:     desc,
		Type:     guessContentType(path),
	}, nil
}

// guessContentType guesses the content type based on file extension
func guessContentType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".pdf":
		return "application/pdf"
	case ".csv":
		return "text/csv"
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	case ".txt":
		return "text/plain"
	case ".md":
		return "text/markdown"
	case ".html":
		return "text/html"
	case ".zip":
		return "application/zip"
	case ".tar":
		return "application/x-tar"
	case ".gz":
		return "application/gzip"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	default:
		return "application/octet-stream"
	}
}

// mergeFieldMap merges a field map into front matter
func mergeFieldMap(fm *core.FrontMatter, fieldMap map[string]any) *core.FrontMatter {
	if fm == nil {
		fm = &core.FrontMatter{}
	}

	// Handle task fields
	if task, ok := fieldMap["task"]; ok {
		if taskMap, ok := task.(map[string]any); ok {
			if fm.Task == nil {
				fm.Task = &core.TaskInfo{}
			}
			if typ, ok := taskMap["type"].(string); ok {
				fm.Task.Type = typ
			}
			if priority, ok := taskMap["priority"].(string); ok {
				fm.Task.Priority = priority
			}
			if language, ok := taskMap["language"].(string); ok {
				fm.Task.Language = language
			}
			if timeout, ok := taskMap["timeout"].(int); ok {
				fm.Task.Timeout = timeout
			}
		}
	}

	// Handle simple fields
	if priority, ok := fieldMap["priority"].(string); ok {
		fm.Priority = priority
	}
	if language, ok := fieldMap["language"].(string); ok {
		fm.Language = language
	}
	if typ, ok := fieldMap["type"].(string); ok {
		fm.Type = typ
	}

	// Handle timeout (could be int or string depending on YAML parsing)
	if timeout, ok := fieldMap["timeout"]; ok {
		switch v := timeout.(type) {
		case int:
			fm.Timeout = v
		case string:
			// Try to parse as int (for field input)
			var i int
			if _, err := fmt.Sscanf(v, "%d", &i); err == nil {
				fm.Timeout = i
			}
		}
	}

	// Handle tags
	if tags, ok := fieldMap["tags"].([]string); ok {
		fm.Tags = append(fm.Tags, tags...)
	} else if tagStr, ok := fieldMap["tags"].(string); ok {
		fm.Tags = append(fm.Tags, strings.Split(tagStr, ",")...)
	}

	// Store unknown fields in Data map
	if fm.Data == nil {
		fm.Data = make(map[string]any)
	}
	for key, value := range fieldMap {
		switch key {
		case "task", "priority", "language", "type", "timeout", "tags":
			// Already handled
		default:
			fm.Data[key] = value
		}
	}

	return fm
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
