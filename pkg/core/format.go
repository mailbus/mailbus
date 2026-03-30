package core

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	// FrontMatterDelimiter is the delimiter used to separate front matter from content
	FrontMatterDelimiter = "---"

	// ContentTypeMarkdown is the content type for markdown messages
	ContentTypeMarkdown = "text/markdown; charset=utf-8"

	// HeaderMailBusFormat is the header name for the MailBus format identifier
	HeaderMailBusFormat = "X-MailBus-Format"

	// FormatFrontMatter is the format identifier for front matter messages
	FormatFrontMatter = "frontmatter"
)

// FrontMatter represents the structured metadata in a message
// Only business metadata should be stored here, not email transport fields
type FrontMatter struct {
	// Task metadata
	Task       *TaskInfo       `yaml:"task,omitempty"`
	Priority   string          `yaml:"priority,omitempty"`   // high, normal, low
	Timeout    int             `yaml:"timeout,omitempty"`    // timeout in seconds
	Tags       []string        `yaml:"tags,omitempty"`
	Language   string          `yaml:"language,omitempty"`
	Type       string          `yaml:"type,omitempty"`       // message type

	// Data payload
	Data       map[string]any  `yaml:"data,omitempty"`

	// Attachments metadata (auto-generated from actual attachments)
	Attachments []AttachmentInfo `yaml:"attachments,omitempty"`

	// Response expectations
	ExpectedResponse *ResponseInfo `yaml:"expected_response,omitempty"`

	// Custom business metadata
	Metadata    map[string]any  `yaml:"metadata,omitempty"`

	// Workflow tracking (optional)
	WorkflowID  string          `yaml:"workflow_id,omitempty"`
	Step        int             `yaml:"step,omitempty"`
	Chain       []ChainEntry    `yaml:"chain,omitempty"`

	// Timestamps (optional, can be derived from email headers)
	CreatedAt   time.Time       `yaml:"created_at,omitempty"`
	ExpiresAt   time.Time       `yaml:"expires_at,omitempty"`
}

// TaskInfo describes a task to be performed
type TaskInfo struct {
	Type      string `yaml:"type,omitempty"`
	Priority  string `yaml:"priority,omitempty"`
	Language  string `yaml:"language,omitempty"`
	Timeout   int    `yaml:"timeout,omitempty"`
}

// AttachmentInfo describes an attachment
type AttachmentInfo struct {
	Name     string `yaml:"name"`
	Size     int64  `yaml:"size"`
	Checksum string `yaml:"checksum,omitempty"`
	Desc     string `yaml:"desc,omitempty"`
	Type     string `yaml:"type,omitempty"`
}

// ResponseInfo describes expected response format
type ResponseInfo struct {
	Format   string   `yaml:"format,omitempty"`
	Timeout  int      `yaml:"timeout,omitempty"`
	Notify   string   `yaml:"notify,omitempty"`
	Include  []string `yaml:"include,omitempty"`
	Deadline string   `yaml:"deadline,omitempty"`
}

// ChainEntry represents an entry in the processing chain
type ChainEntry struct {
	Agent     string    `yaml:"agent"`
	Action    string    `yaml:"action"`
	Timestamp time.Time `yaml:"timestamp"`
	MessageID string    `yaml:"message_id,omitempty"`
}

// ParseMessage parses a message body that may contain front matter
// Returns the front matter (if any) and the markdown content
func ParseMessage(content string) (*FrontMatter, string, error) {
	// Check if content starts with front matter delimiter
	if !strings.HasPrefix(content, FrontMatterDelimiter+"\n") && !strings.HasPrefix(content, FrontMatterDelimiter+"\r\n") {
		// No front matter, return entire content as markdown
		return nil, strings.TrimSpace(content), nil
	}

	// Find the end of the front matter
	// Split by the delimiter
	parts := strings.SplitN(content, FrontMatterDelimiter, 3)
	if len(parts) < 3 {
		// Malformed front matter, treat as plain text
		return nil, strings.TrimSpace(content), nil
	}

	// parts[0] is empty (before first delimiter)
	// parts[1] is the YAML front matter
	// parts[2] is the markdown content

	yamlContent := strings.TrimSpace(parts[1])
	markdownContent := strings.TrimSpace(parts[2])

	if yamlContent == "" {
		// Empty front matter, treat as plain markdown
		return nil, markdownContent, nil
	}

	// Parse YAML
	var fm FrontMatter
	if err := yaml.Unmarshal([]byte(yamlContent), &fm); err != nil {
		// Invalid YAML - treat the entire content as plain markdown
		// This makes the system more resilient to malformed input
		return nil, strings.TrimSpace(content), nil
	}

	return &fm, markdownContent, nil
}

// GenerateMessage generates a message body with front matter and markdown content
// If front matter is nil, only the markdown content is returned
func GenerateMessage(fm *FrontMatter, markdown string) (string, error) {
	// If no front matter, return just the markdown
	if fm == nil {
		return markdown, nil
	}

	// Marshal front matter to YAML
	yamlBytes, err := yaml.Marshal(fm)
	if err != nil {
		return "", fmt.Errorf("failed to marshal front matter: %w", err)
	}

	// Build the message
	var buf bytes.Buffer
	buf.WriteString(FrontMatterDelimiter)
	buf.WriteString("\n")
	buf.Write(yamlBytes)
	buf.WriteString(FrontMatterDelimiter)
	buf.WriteString("\n")

	if markdown != "" {
		buf.WriteString(markdown)
		if !strings.HasSuffix(markdown, "\n") {
			buf.WriteString("\n")
		}
	}

	return buf.String(), nil
}

// ParseFrontMatterFile parses a file that contains only front matter (no markdown)
// Useful for --meta flag where metadata is in a separate YAML file
func ParseFrontMatterFile(yamlContent string) (*FrontMatter, error) {
	var fm FrontMatter
	if err := yaml.Unmarshal([]byte(yamlContent), &fm); err != nil {
		return nil, fmt.Errorf("failed to parse front matter file: %w", err)
	}
	return &fm, nil
}

// MergeFrontMatter merges multiple front matter objects
// Later values override earlier ones
func MergeFrontMatter(fms ...*FrontMatter) *FrontMatter {
	result := &FrontMatter{}
	for _, fm := range fms {
		if fm == nil {
			continue
		}
		if fm.Task != nil {
			if result.Task == nil {
				result.Task = &TaskInfo{}
			}
			*result.Task = *fm.Task
		}
		if fm.Priority != "" {
			result.Priority = fm.Priority
		}
		if fm.Timeout > 0 {
			result.Timeout = fm.Timeout
		}
		if len(fm.Tags) > 0 {
			result.Tags = append(result.Tags, fm.Tags...)
		}
		if fm.Language != "" {
			result.Language = fm.Language
		}
		if fm.Type != "" {
			result.Type = fm.Type
		}
		if len(fm.Data) > 0 {
			if result.Data == nil {
				result.Data = make(map[string]any)
			}
			for k, v := range fm.Data {
				result.Data[k] = v
			}
		}
		if len(fm.Attachments) > 0 {
			result.Attachments = append(result.Attachments, fm.Attachments...)
		}
		if fm.ExpectedResponse != nil {
			if result.ExpectedResponse == nil {
				result.ExpectedResponse = &ResponseInfo{}
			}
			*result.ExpectedResponse = *fm.ExpectedResponse
		}
		if len(fm.Metadata) > 0 {
			if result.Metadata == nil {
				result.Metadata = make(map[string]any)
			}
			for k, v := range fm.Metadata {
				result.Metadata[k] = v
			}
		}
		if fm.WorkflowID != "" {
			result.WorkflowID = fm.WorkflowID
		}
		if fm.Step > 0 {
			result.Step = fm.Step
		}
		if len(fm.Chain) > 0 {
			result.Chain = append(result.Chain, fm.Chain...)
		}
		if !fm.CreatedAt.IsZero() {
			result.CreatedAt = fm.CreatedAt
		}
		if !fm.ExpiresAt.IsZero() {
			result.ExpiresAt = fm.ExpiresAt
		}
	}
	return result
}

// ParseField parses a key=value string into a map
// Supports nested keys with dot notation (e.g., "task.type=analysis")
func ParseField(field string) (map[string]any, error) {
	parts := strings.SplitN(field, "=", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid field format: %s (expected key=value)", field)
	}

	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	// Parse nested keys
	keys := strings.Split(key, ".")

	return buildNestedMap(keys, value), nil
}

// buildNestedMap builds a nested map from a key path and value
func buildNestedMap(keys []string, value string) map[string]any {
	result := make(map[string]any)
	current := result

	for i, key := range keys {
		if i == len(keys)-1 {
			// Last key, set the value
			current[key] = value
		} else {
			// Intermediate key, create nested map if needed
			if _, ok := current[key]; !ok {
				current[key] = make(map[string]any)
			}
			current = current[key].(map[string]any)
		}
	}

	return result
}

// AddAttachment adds an attachment to the front matter
func (fm *FrontMatter) AddAttachment(name string, size int64, checksum string, desc string) {
	if fm.Attachments == nil {
		fm.Attachments = make([]AttachmentInfo, 0)
	}
	fm.Attachments = append(fm.Attachments, AttachmentInfo{
		Name:     name,
		Size:     size,
		Checksum: checksum,
		Desc:     desc,
	})
}

// HasFrontMatter checks if a message body contains front matter
func HasFrontMatter(content string) bool {
	return strings.HasPrefix(content, FrontMatterDelimiter+"\n") ||
		strings.HasPrefix(content, FrontMatterDelimiter+"\r\n")
}

// IsMailBusFormat checks if the message is in MailBus front matter format
func IsMailBusFormat(headers map[string]string) bool {
	format, ok := headers[HeaderMailBusFormat]
	return ok && format == FormatFrontMatter
}
