package adapter

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/textproto"
	"strings"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/mailbus/mailbus/pkg/core"
)

// IMAPAdapter implements IMAP receiving functionality
type IMAPAdapter struct {
	client *client.Client
	config *ConnectionConfig
}

// NewIMAPAdapter creates a new IMAP adapter
func NewIMAPAdapter() *IMAPAdapter {
	return &IMAPAdapter{}
}

// Connect establishes a connection to the IMAP server
func (a *IMAPAdapter) Connect(ctx context.Context, cfg *ConnectionConfig) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	a.config = cfg

	var imapClient *client.Client
	var err error

	if cfg.UseTLS {
		imapClient, err = client.DialTLS(cfg.Address(), &tls.Config{
			ServerName: cfg.Host,
		})
		if err != nil {
			return fmt.Errorf("failed to connect with TLS: %w", err)
		}
	} else {
		imapClient, err = client.Dial(cfg.Address())
		if err != nil {
			return fmt.Errorf("failed to connect: %w", err)
		}
		// Try STARTTLS
		_ = imapClient.StartTLS(&tls.Config{ServerName: cfg.Host})
	}

	a.client = imapClient

	// Authenticate
	if err := a.client.Login(cfg.Username, cfg.Password); err != nil {
		a.client.Close()
		return fmt.Errorf("authentication failed: %w", err)
	}

	return nil
}

// Close closes the connection
func (a *IMAPAdapter) Close() error {
	if a.client != nil {
		_ = a.client.Logout()
		a.client = nil
	}
	return nil
}

// IsConnected returns true if connected
func (a *IMAPAdapter) IsConnected() bool {
	return a.client != nil
}

// Send is not implemented
func (a *IMAPAdapter) Send(ctx context.Context, msg *core.Message) error {
	return fmt.Errorf("IMAP adapter cannot send messages")
}

// Receive receives messages
func (a *IMAPAdapter) Receive(ctx context.Context, filter *core.Filter) ([]*core.Message, error) {
	if a.client == nil {
		return nil, fmt.Errorf("not connected")
	}

	// Select INBOX
	_, err := a.client.Select("INBOX", false)
	if err != nil {
		return nil, fmt.Errorf("failed to select INBOX: %w", err)
	}

	// Build criteria
	criteria := imap.NewSearchCriteria()
	if filter != nil {
		if filter.UnreadOnly {
			criteria.WithoutFlags = []string{imap.SeenFlag}
		}
		if filter.SubjectPattern != "" {
			if criteria.Header == nil {
				criteria.Header = make(textproto.MIMEHeader)
			}
			criteria.Header.Add("Subject", filter.SubjectPattern)
		}
		if filter.FromPattern != "" {
			if criteria.Header == nil {
				criteria.Header = make(textproto.MIMEHeader)
			}
			criteria.Header.Add("From", filter.FromPattern)
		}
		if filter.ToPattern != "" {
			if criteria.Header == nil {
				criteria.Header = make(textproto.MIMEHeader)
			}
			criteria.Header.Add("To", filter.ToPattern)
		}
		if filter.MinDate != nil {
			criteria.Since = *filter.MinDate
		}
		if filter.MaxDate != nil {
			criteria.Before = *filter.MaxDate
		}
	}

	// Search
	ids, err := a.client.Search(criteria)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	if len(ids) == 0 {
		return []*core.Message{}, nil
	}

	// Fetch messages
	var messages []*core.Message
	seqset := new(imap.SeqSet)
	seqset.AddNum(ids...)

	// Fetch items
	items := []imap.FetchItem{
		imap.FetchEnvelope,
		imap.FetchRFC822,
	}

	messagesChan := make(chan *imap.Message, 1)
	done := make(chan error, 1)

	go func() {
		done <- a.client.Fetch(seqset, items, messagesChan)
	}()

	// Parse messages
	for imapMsg := range messagesChan {
		msg := a.parseMessageSimple(imapMsg)
		if msg != nil {
			if filter == nil || filter.Match(msg) {
				messages = append(messages, msg)
			}
		}
	}

	<-done
	return messages, nil
}

// Mark marks a message
func (a *IMAPAdapter) Mark(ctx context.Context, msgID string, action MarkAction) error {
	if a.client == nil {
		return fmt.Errorf("not connected")
	}

	// Search for message
	criteria := imap.NewSearchCriteria()
	criteria.Header = make(textproto.MIMEHeader)
	criteria.Header.Add("Message-ID", "<"+msgID+">")

	ids, err := a.client.Search(criteria)
	if err != nil || len(ids) == 0 {
		return fmt.Errorf("message not found")
	}

	seqset := new(imap.SeqSet)
	seqset.AddNum(ids...)

	var item imap.StoreItem
	var flags []interface{}

	switch action {
	case MarkActionSeen:
		item = imap.AddFlags
		flags = []interface{}{imap.SeenFlag}
	case MarkActionUnseen:
		item = imap.RemoveFlags
		flags = []interface{}{imap.SeenFlag}
	case MarkActionDelete:
		item = imap.AddFlags
		flags = []interface{}{imap.DeletedFlag}
	case MarkActionUndelete:
		item = imap.RemoveFlags
		flags = []interface{}{imap.DeletedFlag}
	case MarkActionFlag:
		item = imap.AddFlags
		flags = []interface{}{imap.FlaggedFlag}
	case MarkActionUnflag:
		item = imap.RemoveFlags
		flags = []interface{}{imap.FlaggedFlag}
	default:
		return fmt.Errorf("unknown action: %s", action)
	}

	return a.client.Store(seqset, item, flags, nil)
}

// parseMessageSimple parses a message simply
func (a *IMAPAdapter) parseMessageSimple(imapMsg *imap.Message) *core.Message {
	msg := core.NewMessage()

	if imapMsg.Envelope != nil {
		msg.ID = imapMsg.Envelope.MessageId
		msg.Subject = imapMsg.Envelope.Subject
		msg.Timestamp = imapMsg.Envelope.Date

		if len(imapMsg.Envelope.From) > 0 {
			msg.From = imapMsg.Envelope.From[0].Address()
		}

		for _, addr := range imapMsg.Envelope.To {
			msg.To = append(msg.To, addr.Address())
		}

		for _, addr := range imapMsg.Envelope.Cc {
			msg.Cc = append(msg.Cc, addr.Address())
		}
	}

	// Parse flags
	for _, flag := range imapMsg.Flags {
		msg.Flags = append(msg.Flags, core.MessageFlag(flag))
	}

	// Get body from Items
	for item, value := range imapMsg.Items {
		if item == imap.FetchRFC822 {
			if literal, ok := value.(imap.Literal); ok {
				data, _ := io.ReadAll(literal)
				bodyStr := string(data)

				// Parse message body
				a.parseMessageBody(msg, bodyStr, imapMsg.Envelope)
			}
		}
	}

	return msg
}

// parseMessageBody parses the message body with Front Matter support
func (a *IMAPAdapter) parseMessageBody(msg *core.Message, bodyStr string, envelope *imap.Envelope) {
	// Extract custom headers from the raw message
	// Split headers and body
	parts := strings.SplitN(bodyStr, "\r\n\r\n", 2)
	if len(parts) < 2 {
		parts = strings.SplitN(bodyStr, "\n\n", 2)
	}

	headers := make(map[string]string)
	if len(parts) >= 1 {
		headerSection := parts[0]
		for _, line := range strings.Split(headerSection, "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "From:") || strings.HasPrefix(line, "To:") ||
				strings.HasPrefix(line, "Subject:") || strings.HasPrefix(line, "Date:") ||
				strings.HasPrefix(line, "Cc:") || strings.HasPrefix(line, "Message-ID:") {
				continue
			}
			if strings.Contains(line, ":") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					key := strings.TrimSpace(strings.TrimPrefix(parts[0], "X-"))
					value := strings.TrimSpace(parts[1])
					headers[key] = value
				}
			}
		}
	}

	// Check if this is a MailBus format message
	isMailBusFormat := headers["MailBus-Format"] == core.FormatFrontMatter ||
		strings.Contains(bodyStr, core.HeaderMailBusFormat)

	// Get the actual body content (after headers)
	var contentBody string
	if len(parts) >= 2 {
		contentBody = parts[1]
	} else {
		contentBody = bodyStr
	}

	if isMailBusFormat && core.HasFrontMatter(contentBody) {
		// Parse Front Matter
		fm, markdown, err := core.ParseMessage(contentBody)
		if err == nil && fm != nil {
			// Successfully parsed Front Matter
			// Store front matter in headers for later use
			if fm.Task != nil {
				if fm.Task.Type != "" {
					msg.AddHeader("X-MailBus-Task-Type", fm.Task.Type)
				}
				if fm.Task.Priority != "" {
					msg.AddHeader("X-MailBus-Task-Priority", fm.Task.Priority)
				}
				if fm.Task.Language != "" {
					msg.AddHeader("X-MailBus-Language", fm.Task.Language)
				}
			}
			if fm.Priority != "" {
				msg.AddHeader("X-MailBus-Priority", fm.Priority)
			}
			if fm.Type != "" {
				msg.AddHeader("X-MailBus-Type", fm.Type)
			}
			if len(fm.Tags) > 0 {
				msg.AddHeader("X-MailBus-Tags", strings.Join(fm.Tags, ","))
			}

			// Store attachments info
			if len(fm.Attachments) > 0 {
				attachments := make([]string, len(fm.Attachments))
				for i, att := range fm.Attachments {
					attachments[i] = fmt.Sprintf("%s:%d", att.Name, att.Size)
				}
				msg.AddHeader("X-MailBus-Attachments", strings.Join(attachments, ","))
			}

			msg.ContentType = core.ContentTypeMarkdown
			msg.Body = markdown
			return
		}
	}

	// Fallback: detect content type
	if strings.HasPrefix(contentBody, "{") || strings.HasPrefix(contentBody, "[") {
		msg.Body = contentBody
		msg.ContentType = "application/json"
	} else if core.HasFrontMatter(contentBody) {
		// Has front matter but not marked as MailBus format
		// Try to parse anyway
		_, markdown, _ := core.ParseMessage(contentBody)
		msg.ContentType = core.ContentTypeMarkdown
		msg.Body = markdown
	} else {
		msg.Body = contentBody
		msg.ContentType = "text/plain"
	}
}

// ListFolders lists folders
func (a *IMAPAdapter) ListFolders() ([]string, error) {
	if a.client == nil {
		return nil, fmt.Errorf("not connected")
	}

	mailboxes := make(chan *imap.MailboxInfo, 10)
	done := make(chan error, 1)

	go func() {
		done <- a.client.List("", "*", mailboxes)
	}()

	var folders []string
	for m := range mailboxes {
		folders = append(folders, m.Name)
	}

	<-done
	return folders, nil
}

// CreateFolder creates a folder
func (a *IMAPAdapter) CreateFolder(name string) error {
	if a.client == nil {
		return fmt.Errorf("not connected")
	}
	return a.client.Create(name)
}

// DeleteFolder deletes a folder
func (a *IMAPAdapter) DeleteFolder(name string) error {
	if a.client == nil {
		return fmt.Errorf("not connected")
	}
	return a.client.Delete(name)
}

// MoveMessage moves a message
func (a *IMAPAdapter) MoveMessage(msgID, targetFolder string) error {
	if a.client == nil {
		return fmt.Errorf("not connected")
	}

	criteria := imap.NewSearchCriteria()
	criteria.Header = make(textproto.MIMEHeader)
	criteria.Header.Add("Message-ID", "<"+msgID+">")

	ids, err := a.client.Search(criteria)
	if err != nil || len(ids) == 0 {
		return fmt.Errorf("message not found")
	}

	seqset := new(imap.SeqSet)
	seqset.AddNum(ids...)

	return a.client.Move(seqset, targetFolder)
}

// Watch is not implemented
func (a *IMAPAdapter) Watch(ctx context.Context, filter *core.Filter, ch chan<- *core.Message) error {
	return fmt.Errorf("watch not implemented")
}
