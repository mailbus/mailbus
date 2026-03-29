package core

import (
	"encoding/json"
	"fmt"
	"mime"
	"net/mail"
	"strings"
	"time"
)

// Message represents an email message that can be sent or received
type Message struct {
	ID          string            // Message unique ID (Message-ID header)
	From        string            // Sender email address
	To          []string          // Recipient email addresses
	Cc          []string          // CC recipients
	Bcc         []string          // BCC recipients
	Subject     string            // Subject line
	Body        string            // Body content (JSON or text)
	ContentType string            // Content type (e.g., "application/json", "text/plain")
	Headers     map[string]string // Extended headers
	Attachments []Attachment      // File attachments
	Timestamp   time.Time         // Message timestamp
	Flags       []MessageFlag     // Message flags (seen, answered, etc.)
	Raw         []byte            // Raw message bytes (optional)
}

// Attachment represents a file attachment
type Attachment struct {
	Filename    string
	ContentType string
	Size        int64
	ContentID   string
	Data        []byte // Content data (nil if not loaded)
}

// MessageFlag represents the status of a message
type MessageFlag string

const (
	FlagSeen     MessageFlag = "\\Seen"
	FlagAnswered MessageFlag = "\\Answered"
	FlagFlagged  MessageFlag = "\\Flagged"
	FlagDeleted  MessageFlag = "\\Deleted"
	FlagDraft    MessageFlag = "\\Draft"
	FlagRecent   MessageFlag = "\\Recent"
)

// IsSeen returns true if the message has been read
func (m *Message) IsSeen() bool {
	return m.hasFlag(FlagSeen)
}

// IsAnswered returns true if the message has been answered
func (m *Message) IsAnswered() bool {
	return m.hasFlag(FlagAnswered)
}

// IsFlagged returns true if the message is flagged
func (m *Message) IsFlagged() bool {
	return m.hasFlag(FlagFlagged)
}

// IsDeleted returns true if the message is marked for deletion
func (m *Message) IsDeleted() bool {
	return m.hasFlag(FlagDeleted)
}

// hasFlag checks if a message has a specific flag
func (m *Message) hasFlag(flag MessageFlag) bool {
	for _, f := range m.Flags {
		if f == flag {
			return true
		}
	}
	return false
}

// ParseJSONBody parses the message body as JSON
func (m *Message) ParseJSONBody(v interface{}) error {
	if m.ContentType != "application/json" && m.ContentType != "text/json" {
		return fmt.Errorf("message content type is %s, not JSON", m.ContentType)
	}
	return json.Unmarshal([]byte(m.Body), v)
}

// GetTags extracts tags from the subject line (e.g., "[task.research]" -> ["task", "research"])
func (m *Message) GetTags() []string {
	return ParseSubjectTags(m.Subject)
}

// ParseSubjectTags extracts tags from a subject line
// Tags are in the format [tag] or [category.subcategory]
func ParseSubjectTags(subject string) []string {
	var tags []string
	start := strings.Index(subject, "[")
	for start != -1 {
		end := strings.Index(subject[start:], "]")
		if end == -1 {
			break
		}
		end += start
		tag := strings.Trim(subject[start+1:end], "[]")
		tag = strings.TrimSpace(tag)
		if tag != "" {
			// Split by dot for hierarchical tags
			parts := strings.Split(tag, ".")
			tags = append(tags, parts...)
		}
		subject = subject[end+1:]
		start = strings.Index(subject, "[")
	}
	return tags
}

// Validate checks if the message has required fields
func (m *Message) Validate() error {
	if m.From == "" {
		return fmt.Errorf("from address is required")
	}
	if _, err := mail.ParseAddress(m.From); err != nil {
		return fmt.Errorf("invalid from address: %w", err)
	}
	if len(m.To) == 0 {
		return fmt.Errorf("at least one recipient is required")
	}
	for _, to := range m.To {
		if _, err := mail.ParseAddress(to); err != nil {
			return fmt.Errorf("invalid to address %s: %w", to, err)
		}
	}
	if m.Subject == "" {
		return fmt.Errorf("subject is required")
	}
	if m.Body == "" && len(m.Attachments) == 0 {
		return fmt.Errorf("message must have either a body or attachments")
	}
	return nil
}

// AddHeader adds a header to the message
func (m *Message) AddHeader(key, value string) {
	if m.Headers == nil {
		m.Headers = make(map[string]string)
	}
	m.Headers[key] = value
}

// GetHeader gets a header value
func (m *Message) GetHeader(key string) (string, bool) {
	if m.Headers == nil {
		return "", false
	}
	v, ok := m.Headers[key]
	return v, ok
}

// String returns a string representation of the message
func (m *Message) String() string {
	return fmt.Sprintf("Message{ID: %s, From: %s, To: %v, Subject: %s}",
		m.ID, m.From, m.To, m.Subject)
}

// NewMessage creates a new message with default values
func NewMessage() *Message {
	return &Message{
		Headers:     make(map[string]string),
		ContentType: "application/json",
		Timestamp:   time.Now(),
		Flags:       []MessageFlag{},
	}
}

// BuildMIME builds the MIME message for sending
func (m *Message) BuildMIME() (string, error) {
	var builder strings.Builder

	// Headers
	builder.WriteString(fmt.Sprintf("From: %s\r\n", m.From))
	builder.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(m.To, ", ")))
	if len(m.Cc) > 0 {
		builder.WriteString(fmt.Sprintf("Cc: %s\r\n", strings.Join(m.Cc, ", ")))
	}
	builder.WriteString(fmt.Sprintf("Subject: %s\r\n", m.Subject))
	builder.WriteString(fmt.Sprintf("Date: %s\r\n", m.Timestamp.Format(time.RFC1123Z)))
	builder.WriteString(fmt.Sprintf("Content-Type: %s; charset=utf-8\r\n", m.ContentType))
	builder.WriteString("MIME-Version: 1.0\r\n")

	// Custom headers
	for k, v := range m.Headers {
		builder.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}

	// Generate Message-ID if not set
	if m.ID == "" {
		m.ID = generateMessageID(m.From)
	}
	builder.WriteString(fmt.Sprintf("Message-ID: <%s>\r\n", m.ID))

	builder.WriteString("\r\n")

	// Body
	builder.WriteString(m.Body)

	return builder.String(), nil
}

// generateMessageID generates a unique message ID
func generateMessageID(domain string) string {
	// Extract domain from email if needed
	if strings.Contains(domain, "@") {
		parts := strings.Split(domain, "@")
		if len(parts) > 1 {
			domain = parts[1]
		}
	}
	return fmt.Sprintf("%d@mailbus.%s", time.Now().UnixNano(), domain)
}

// ParseContentType parses the content type and returns the main type and charset
func ParseContentType(contentType string) (mimeType, charset string, err error) {
	if contentType == "" {
		return "text/plain", "utf-8", nil
	}

	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return "", "", err
	}

	charset = params["charset"]
	if charset == "" {
		charset = "utf-8"
	}

	return mediaType, charset, nil
}
