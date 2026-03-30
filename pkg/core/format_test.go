package core

import (
	"strings"
	"testing"
)

func TestParseMessage_WithFrontMatter(t *testing.T) {
	content := `---
task:
  type: code_review
  priority: high
language: python
timeout: 300
---
# 代码审查请求

请审查以下代码。`

	fm, markdown, err := ParseMessage(content)
	if err != nil {
		t.Fatalf("ParseMessage failed: %v", err)
	}

	if fm == nil {
		t.Fatal("Expected front matter to be parsed, got nil")
	}

	if fm.Task == nil || fm.Task.Type != "code_review" {
		t.Errorf("Expected task.type = code_review, got %v", fm.Task)
	}

	if fm.Task.Priority != "high" {
		t.Errorf("Expected priority = high, got %s", fm.Task.Priority)
	}

	if fm.Language != "python" {
		t.Errorf("Expected language = python, got %s", fm.Language)
	}

	if fm.Timeout != 300 {
		t.Errorf("Expected timeout = 300, got %d", fm.Timeout)
	}

	expectedMarkdown := "# 代码审查请求\n\n请审查以下代码。"
	if markdown != expectedMarkdown {
		t.Errorf("Expected markdown = %q, got %q", expectedMarkdown, markdown)
	}
}

func TestParseMessage_WithoutFrontMatter(t *testing.T) {
	content := "# Plain markdown\n\nNo front matter here."

	fm, markdown, err := ParseMessage(content)
	if err != nil {
		t.Fatalf("ParseMessage failed: %v", err)
	}

	if fm != nil {
		t.Error("Expected no front matter, got non-nil")
	}

	if markdown != content {
		t.Errorf("Expected markdown = %q, got %q", content, markdown)
	}
}

func TestParseMessage_EmptyFrontMatter(t *testing.T) {
	content := `---
---
# Just markdown`

	fm, markdown, err := ParseMessage(content)
	if err != nil {
		t.Fatalf("ParseMessage failed: %v", err)
	}

	if fm != nil {
		t.Error("Expected no front matter (empty), got non-nil")
	}

	if !strings.Contains(markdown, "# Just markdown") {
		t.Errorf("Expected markdown to contain heading, got %q", markdown)
	}
}

func TestParseMessage_InvalidYAML(t *testing.T) {
	content := `---
task: [unclosed array
---
# Content`

	fm, markdown, err := ParseMessage(content)
	// Should not error, but return as plain text
	if err != nil {
		t.Fatalf("ParseMessage should not error on invalid YAML, got: %v", err)
	}

	if fm != nil {
		t.Error("Expected nil front matter for invalid YAML")
	}

	// Content should be returned trimmed (as it has front matter delimiters)
	expectedContent := strings.TrimSpace(content)
	if markdown != expectedContent {
		t.Errorf("Expected trimmed content, got %q", markdown)
	}
}

func TestGenerateMessage_WithFrontMatter(t *testing.T) {
	fm := &FrontMatter{
		Task: &TaskInfo{
			Type:     "data_analysis",
			Priority: "high",
		},
		Language: "python",
		Timeout:  300,
	}
	markdown := "# 分析任务\n\n请分析数据。"

	content, err := GenerateMessage(fm, markdown)
	if err != nil {
		t.Fatalf("GenerateMessage failed: %v", err)
	}

	// Check structure
	if !strings.HasPrefix(content, "---\n") {
		t.Error("Expected content to start with front matter delimiter")
	}

	if !strings.Contains(content, "type: data_analysis") {
		t.Error("Expected task.type in generated content")
	}

	if !strings.Contains(content, "# 分析任务") {
		t.Error("Expected markdown content in generated message")
	}

	// Should have three parts: opening delimiter, YAML, closing delimiter + content
	parts := strings.Split(content, "---\n")
	if len(parts) != 3 {
		t.Errorf("Expected 3 parts when split by delimiter, got %d", len(parts))
	}
}

func TestGenerateMessage_WithoutFrontMatter(t *testing.T) {
	markdown := "# Plain markdown\n\nNo metadata."

	content, err := GenerateMessage(nil, markdown)
	if err != nil {
		t.Fatalf("GenerateMessage failed: %v", err)
	}

	if content != markdown {
		t.Errorf("Expected markdown unchanged, got %q", content)
	}
}

func TestRoundTrip(t *testing.T) {
	originalFM := &FrontMatter{
		Task: &TaskInfo{
			Type:     "code_review",
			Priority: "high",
		},
		Language: "go",
		Tags:     []string{"security", "performance"},
		Data: map[string]any{
			"repo": "github.com/user/repo",
		},
	}
	originalMarkdown := "# Review this code\n\nPlease check for bugs."

	// Generate
	content, err := GenerateMessage(originalFM, originalMarkdown)
	if err != nil {
		t.Fatalf("GenerateMessage failed: %v", err)
	}

	// Parse
	parsedFM, parsedMarkdown, err := ParseMessage(content)
	if err != nil {
		t.Fatalf("ParseMessage failed: %v", err)
	}

	// Verify front matter
	if parsedFM == nil {
		t.Fatal("Expected front matter to be parsed")
	}

	if parsedFM.Task.Type != originalFM.Task.Type {
		t.Errorf("Task.Type mismatch: got %s, want %s", parsedFM.Task.Type, originalFM.Task.Type)
	}

	if parsedFM.Language != originalFM.Language {
		t.Errorf("Language mismatch: got %s, want %s", parsedFM.Language, originalFM.Language)
	}

	if len(parsedFM.Tags) != len(originalFM.Tags) {
		t.Errorf("Tags length mismatch: got %d, want %d", len(parsedFM.Tags), len(originalFM.Tags))
	}

	if parsedMarkdown != originalMarkdown {
		t.Errorf("Markdown mismatch: got %q, want %q", parsedMarkdown, originalMarkdown)
	}
}

func TestParseFrontMatterFile(t *testing.T) {
	yamlContent := `
task:
  type: translation
  language: en
priority: normal
timeout: 120
`

	fm, err := ParseFrontMatterFile(yamlContent)
	if err != nil {
		t.Fatalf("ParseFrontMatterFile failed: %v", err)
	}

	if fm.Task.Type != "translation" {
		t.Errorf("Expected task.type = translation, got %s", fm.Task.Type)
	}

	if fm.Task.Language != "en" {
		t.Errorf("Expected language = en, got %s", fm.Task.Language)
	}

	if fm.Priority != "normal" {
		t.Errorf("Expected priority = normal, got %s", fm.Priority)
	}
}

func TestParseField(t *testing.T) {
	tests := []struct {
		name  string
		field string
		want  map[string]any
	}{
		{
			name:  "simple key-value",
			field: "priority=high",
			want:  map[string]any{"priority": "high"},
		},
		{
			name:  "nested key",
			field: "task.type=analysis",
			want:  map[string]any{"task": map[string]any{"type": "analysis"}},
		},
		{
			name:  "deeply nested",
			field: "data.repo.url=https://github.com/user/repo",
			want:  map[string]any{"data": map[string]any{"repo": map[string]any{"url": "https://github.com/user/repo"}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseField(tt.field)
			if err != nil {
				t.Fatalf("ParseField failed: %v", err)
			}
			// Compare maps (simplified)
			if len(got) != len(tt.want) {
				t.Errorf("ParseField() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseField_InvalidFormat(t *testing.T) {
	_, err := ParseField("invalid")
	if err == nil {
		t.Error("Expected error for invalid field format, got nil")
	}
}

func TestMergeFrontMatter(t *testing.T) {
	fm1 := &FrontMatter{
		Priority: "high",
		Language: "python",
		Tags:     []string{"tag1"},
		Data:     map[string]any{"key1": "value1"},
	}

	fm2 := &FrontMatter{
		Priority: "normal", // Should override
		Timeout:  300,
		Tags:     []string{"tag2"},
		Data:     map[string]any{"key2": "value2"},
	}

	merged := MergeFrontMatter(fm1, fm2)

	if merged.Priority != "normal" {
		t.Errorf("Expected Priority = normal (from fm2), got %s", merged.Priority)
	}

	if merged.Language != "python" {
		t.Errorf("Expected Language = python (from fm1), got %s", merged.Language)
	}

	if merged.Timeout != 300 {
		t.Errorf("Expected Timeout = 300 (from fm2), got %d", merged.Timeout)
	}

	// Tags should be concatenated
	if len(merged.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(merged.Tags))
	}

	// Data should be merged
	if len(merged.Data) != 2 {
		t.Errorf("Expected 2 data entries, got %d", len(merged.Data))
	}
}

func TestHasFrontMatter(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{
			name:    "with front matter (LF)",
			content: "---\ntask: test\n---\ncontent",
			want:    true,
		},
		{
			name:    "with front matter (CRLF)",
			content: "---\r\ntask: test\r\n---\r\ncontent",
			want:    true,
		},
		{
			name:    "without front matter",
			content: "# Just markdown",
			want:    false,
		},
		{
			name:    "empty content",
			content: "",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasFrontMatter(tt.content); got != tt.want {
				t.Errorf("HasFrontMatter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsMailBusFormat(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string]string
		want    bool
	}{
		{
			name:    "MailBus format",
			headers: map[string]string{"X-MailBus-Format": "frontmatter"},
			want:    true,
		},
		{
			name:    "different format",
			headers: map[string]string{"X-MailBus-Format": "json"},
			want:    false,
		},
		{
			name:    "no format header",
			headers: map[string]string{},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsMailBusFormat(tt.headers); got != tt.want {
				t.Errorf("IsMailBusFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAddAttachment(t *testing.T) {
	fm := &FrontMatter{}

	fm.AddAttachment("data.csv", 1024000, "sha256:abc123", "Sales data")
	fm.AddAttachment("model.pkl", 5120000, "", "Trained model")

	if len(fm.Attachments) != 2 {
		t.Fatalf("Expected 2 attachments, got %d", len(fm.Attachments))
	}

	if fm.Attachments[0].Name != "data.csv" {
		t.Errorf("Expected first attachment name = data.csv, got %s", fm.Attachments[0].Name)
	}

	if fm.Attachments[0].Size != 1024000 {
		t.Errorf("Expected first attachment size = 1024000, got %d", fm.Attachments[0].Size)
	}

	if fm.Attachments[0].Checksum != "sha256:abc123" {
		t.Errorf("Expected first attachment checksum = sha256:abc123, got %s", fm.Attachments[0].Checksum)
	}

	if fm.Attachments[1].Checksum != "" {
		t.Errorf("Expected second attachment checksum to be empty, got %s", fm.Attachments[1].Checksum)
	}
}
