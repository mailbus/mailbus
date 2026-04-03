# MailBus Agent Installation Guide

**Purpose**: This document provides structured installation and usage instructions for AI agents (Claude Code, Codex, OpenClaw, etc.) to install and use MailBus.

---

## Quick Reference

- **Repository**: https://github.com/mailbus/mailbus
- **Version**: v0.3.0
- **Language**: Go 1.21+
- **License**: Apache 2.0 with cloud service restrictions

---

## Installation Commands

### Option 1: Install from Source (Recommended)

```bash
# Prerequisites: Go 1.21 or later must be installed
go version  # Verify Go version

# Install MailBus CLI
go install github.com/mailbus/mailbus/cmd/mailbus@latest

# Verify installation
mailbus version
```

### Option 2: Build from Source

```bash
# Clone repository
git clone https://github.com/mailbus/mailbus.git
cd mailbus

# Build binary
go build -o mailbus ./cmd/mailbus

# Install to system (optional)
sudo mv mailbus /usr/local/bin/  # Linux/macOS
# OR copy to PATH directory on Windows
```

### Option 3: Download Prebuilt Binary

**Automatic Installation (Recommended):**

```bash
# Linux/macOS - Auto-detect platform and install
curl -sSL https://raw.githubusercontent.com/mailbus/mailbus/main/scripts/install.sh | bash

# Or download and run manually
wget https://raw.githubusercontent.com/mailbus/mailbus/main/scripts/install.sh
chmod +x install.sh
./install.sh

# Windows PowerShell - Auto-detect and install
Invoke-WebRequest -Uri "https://raw.githubusercontent.com/mailbus/mailbus/main/scripts/install.ps1" -OutFile "install.ps1"
./install.ps1
```

**Manual Download by Platform:**

```bash
# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# Map architecture
case $ARCH in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  armv7l) ARCH="armv7" ;;
  i386|i686) ARCH="386" ;;
esac

# Download appropriate binary
wget https://github.com/mailbus/mailbus/releases/latest/download/mailbus-${OS}-${ARCH} -O mailbus
chmod +x mailbus
```

**Platform-Specific Downloads:**

```bash
# Linux AMD64
wget https://github.com/mailbus/mailbus/releases/latest/download/mailbus-linux-amd64 -O mailbus

# Linux ARM64 (for ARM servers like AWS Graviton)
wget https://github.com/mailbus/mailbus/releases/latest/download/mailbus-linux-arm64 -O mailbus

# macOS Intel
wget https://github.com/mailbus/mailbus/releases/latest/download/mailbus-darwin-amd64 -O mailbus

# macOS Apple Silicon (M1/M2)
wget https://github.com/mailbus/mailbus/releases/latest/download/mailbus-darwin-arm64 -O mailbus

# Windows (PowerShell)
Invoke-WebRequest -Uri "https://github.com/mailbus/mailbus/releases/latest/download/mailbus-windows-amd64.exe" -OutFile "mailbus.exe"
```

**Verify Downloaded Binary:**

```bash
# Download checksum file
wget https://github.com/mailbus/mailbus/releases/latest/download/mailbus-linux-amd64.sha256 -O mailbus.sha256

# Verify checksum
sha256sum -c mailbus.sha256
```

**Supported Platforms:**

| OS | Architecture | Binary Name |
|----|--------------|--------------|
| Linux | AMD64 (x86_64) | mailbus-linux-amd64 |
| Linux | ARM64 (aarch64) | mailbus-linux-arm64 |
| macOS | AMD64 (Intel) | mailbus-darwin-amd64 |
| macOS | ARM64 (Apple Silicon) | mailbus-darwin-arm64 |
| Windows | AMD64 (x86_64) | mailbus-windows-amd64.exe |

---

## Configuration

### Step 1: Initialize Configuration

```bash
mailbus config init
```

This creates: `~/.mailbus/config.yaml`

### Step 2: Configure Email Account

Edit `~/.mailbus/config.yaml`:

```yaml
default_account: "primary"

accounts:
  primary:
    imap:
      host: imap.gmail.com
      port: 993
      use_tls: true
    smtp:
      host: smtp.gmail.com
      port: 587
      use_tls: true
    username: your-email@gmail.com
    password_env: MAILBUS_PASSWORD
    from: your-email@gmail.com

global:
  poll_interval: 30s
  batch_size: 20
  timeout: 30s
  max_retries: 3
  verbose: false
  log_level: info
  handler_timeout: 60s
```

### Step 3: Set Credentials

**Option 1: Environment Variables (Traditional)**

```bash
# Set password as environment variable
export MAILBUS_PASSWORD=your-app-password

# For persistent configuration, add to ~/.bashrc or ~/.zshrc:
echo 'export MAILBUS_PASSWORD=your-app-password' >> ~/.bashrc
```

**Option 2: Encrypted Password (Recommended)**

```bash
# Set encryption key
export MAILBUS_CRYPTO_KEY="your-encryption-key"

# Generate encrypted password
mailbus crypto encrypt --password "my-password"
# Output: password_encrypted: AGE-ENCRYPTED-BLOB...

# Copy the output to your config.yaml:
```

Update `~/.mailbus/config.yaml`:

```yaml
accounts:
  primary:
    username: your-email@gmail.com
    password_encrypted: "AGE-ENCRYPTED-BLOB..."  # Paste encrypted password here
```

**Why use encrypted passwords?**
- ✅ Passwords not stored in environment variables (which may be logged)
- ✅ Supports git-tracked configurations without exposing secrets
- ✅ Compatible with CI/CD (use secrets for MAILBUS_CRYPTO_KEY)
- ✅ Backward compatible with environment variable method

**Priority:** `password_encrypted` > `password_env`

### Step 4: Validate Configuration

```bash
mailbus config validate
```

---

## Email Provider Setup

### Gmail

1. Enable 2-Factor Authentication
2. Generate App Password:
   - Go to: https://myaccount.google.com/apppasswords
   - Select "Mail" and your device
   - Copy the 16-character password
3. Use App Password in `MAILBUS_PASSWORD`

### Outlook

1. Generate App Password:
   - Go to: https://account.live.com/proofs/AppPassword
   - Create a new app password
   - Copy the password
2. Use App Password in `MAILBUS_PASSWORD`

---

## Message Format

MailBus uses **Front Matter + Markdown** format for messages:

```yaml
---
task:
  type: code_review
  priority: high
language: python
timeout: 300
tags: [security, performance]
---
# Code Review Request

Please review the following code for potential issues...

## Requirements
- [ ] Security vulnerabilities
- [ ] Performance optimization
- [ ] Code style consistency
```

**Email Headers** (transport layer):
- `From`: sender email address
- `To`: recipient email addresses
- `Subject`: human-readable subject with tags
- `Content-Type`: `text/markdown; charset=utf-8`
- `X-MailBus-Format`: `frontmatter`

**Front Matter** (business layer):
- Task metadata (type, priority, language, timeout)
- Tags for categorization
- Data payload
- Attachment metadata
- Custom business fields

---

## Core Commands Reference

### Send Message

**File Mode** (recommended):

```bash
mailbus send \
  --to recipient@example.com \
  --subject "[task] Analysis" \
  --file message.md
```

**Split Mode** (separate metadata and content):

```bash
mailbus send \
  --to recipient@example.com \
  --subject "[task] Analysis" \
  --meta metadata.yaml \
  --body content.md
```

**Inline Mode**:

```bash
mailbus send \
  --to recipient@example.com \
  --subject "[task] Analysis" \
  --field "task.type=analysis" \
  --field "priority=high" \
  --markdown "# Analyze Q1 data\n\nPlease analyze..."
```

**With Attachments**:

```bash
mailbus send \
  --to recipient@example.com \
  --file request.md \
  --attach data.csv \
  --attach-desc "data.csv:Q1 sales data"
```

**Flags:**
- `--to string[]`: Recipient addresses (required)
- `--subject string`: Message subject (required)
- `--file string`: Read complete message (front matter + markdown) from file
- `--meta string`: Read front matter metadata from YAML file
- `--body string`: Read markdown content from file
- `--markdown string`: Inline markdown content
- `--field string[]`: Add metadata field (key=value, supports dot notation)
- `--attach string[]`: File attachments
- `--attach-desc string[]`: Attachment description (name:description)
- `--attach-dir string`: Directory for attachment files (default: current dir)
- `--header stringToString`: Custom headers (key=value)
- `--priority string`: Message priority (high/normal/low)
- `-A, --account string`: Use specified account

### Poll Messages

```bash
mailbus poll \
  --subject "\[task\]" \
  --handler "./handler.sh" \
  --once
```

**Continuous monitoring:**

```bash
mailbus poll \
  --subject "\[alert\]" \
  --handler "./alert.sh" \
  --continuous \
  --interval 30
```

**Flags:**
- `--subject string`: Subject filter (regex supported)
- `--from string`: Sender filter
- `--unread`: Only unread messages
- `--once`: Process once and exit
- `-c, --continuous`: Continuous polling mode
- `-i, --interval int`: Polling interval in seconds (default: 30)
- `-H, --handler string`: Handler command to execute
- `--handler-timeout int`: Handler timeout in seconds
- `--on-error string`: Error handling (continue/stop/retry)
- `--reply-with-result`: Send handler result as reply
- `--mark-after string`: Mark action after processing (read/delete/none)
- `-F, --folder string`: IMAP folder to check (default: INBOX)
- `--format string`: Output format (table/json/compact)
- `--download-attachments`: Download message attachments
- `--attach-dir string`: Directory to save attachments
- `-A, --account string`: Use specified account

### List Messages

```bash
mailbus list --unread --limit 10
```

**Flags:**
- `--subject string`: Subject filter
- `--from string`: Sender filter
- `--unread`: Only unread messages
- `--limit int`: Limit results (default: 20)
- `--format string`: Output format (table/json/compact)
- `-A, --account string`: Use specified account

### Mark Messages

```bash
mailbus mark --id "message-id" --action read
```

**Flags:**
- `--id string`: Message ID (required)
- `--action string`: Action (read/unread/delete/undelete/flag/unflag/move)
- `--folder string`: Target folder (for move action)
- `-A, --account string`: Use specified account

---

## Handler Development

### Handler Input Format

For Front Matter messages, handlers receive:

**Environment Variables:**
```bash
# Email headers (always available)
MAILBUS_FROM="sender@example.com"
MAILBUS_TO="recipient@example.com"
MAILBUS_SUBJECT="[task] Analysis"
MAILBUS_MESSAGE_ID="msg123@domain"

# Front Matter fields (if present)
MAILBUS_TASK_TYPE="analysis"
MAILBUS_TASK_PRIORITY="high"
MAILBUS_LANGUAGE="python"
MAILBUS_TIMEOUT="300"
MAILBUS_TAGS="urgent,q1"

# Stdin contains the markdown content
```

**Stdin Content:**
The markdown body content is passed via stdin.

### Example Handlers

**Bash Handler:**

```bash
#!/bin/bash
# process.sh

# Read markdown content
CONTENT=$(cat)

# Access metadata from environment
TASK_TYPE="$MAILBUS_TASK_TYPE"
PRIORITY="$MAILBUS_TASK_PRIORITY"

echo "Processing $TASK_TYPE task (priority: $PRIORITY)"

# Process the content
RESULT=$(python process.py "$CONTENT")

# Output result (JSON format for reply)
echo "{\"status\":\"success\",\"result\":\"$RESULT\"}"
```

**Python Handler:**

```python
#!/usr/bin/env python3
import os
import sys

# Get metadata from environment
task_type = os.getenv('MAILBUS_TASK_TYPE', 'unknown')
priority = os.getenv('MAILBUS_TASK_PRIORITY', 'normal')

# Read markdown content from stdin
content = sys.stdin.read()

print(f"Processing {task_type} task (priority: {priority})")
print(f"Content length: {len(content)} bytes")

# Output result
print('{"status":"success","result":"processed"}')
```

**Node.js Handler:**

```javascript
#!/usr/bin/env node
const taskType = process.env.MAILBUS_TASK_TYPE || 'unknown';
const priority = process.env.MAILBUS_TASK_PRIORITY || 'normal';

// Read markdown from stdin
let content = '';
process.stdin.on('data', chunk => content += chunk);
process.stdin.on('end', () => {
  console.log(`Processing ${taskType} task (priority: ${priority})`);
  console.log(JSON.stringify({status: "success"}));
});
```

### Handler Response Format

```json
{
  "status": "success",
  "result": "Processing complete",
  "data": {}
}
```

For replies, you can also output:
```json
{
  "status": "success",
  "reply_subject": "Re: [task] Analysis",
  "reply_body": "Analysis complete..."
}
```

---

## Common Workflows

### Send and Wait for Response

```bash
# Send message
mailbus send \
  --to agent@example.com \
  --subject "[query] Get Data" \
  --field "query.type=sql" \
  --markdown "Please run: SELECT * FROM users"

# Poll for response
mailbus poll --subject "\[response\]" --once
```

### Continuous Monitoring

```bash
# Start continuous polling
mailbus poll \
  --subject "\[task\]" \
  --handler "./task_handler.sh" \
  --continuous \
  --interval 60
```

### Filter and Process

```bash
# Process only unread messages from specific sender
mailbus poll \
  --from "trusted@example.com" \
  --unread \
  --handler "./process.sh" \
  --once
```

---

## Troubleshooting

### Issue: "mailbus: command not found"

**Solution:**
```bash
# Add Go bin to PATH
export PATH=$PATH:$(go env GOPATH)/bin
echo 'export PATH=$PATH:$(go env GOPATH)/bin' >> ~/.bashrc
```

### Issue: "Authentication failed"

**Solution:**
```bash
# Verify credentials
echo $MAILBUS_PASSWORD

# For Gmail, ensure App Password is used (not regular password)
# For Outlook, generate App Password from account settings
```

### Issue: "Connection timeout"

**Solution:**
```bash
# Test IMAP connection
telnet imap.gmail.com 993

# Test SMTP connection
telnet smtp.gmail.com 587

# Check firewall settings
```

### Issue: "Handler not found"

**Solution:**
```bash
# Use absolute path
mailbus poll --handler "/full/path/to/handler.sh"

# OR make handler executable
chmod +x handler.sh
```

---

## Testing

### Test Installation

```bash
# Verify mailbus is installed
mailbus version

# Validate configuration
mailbus config validate

# List accounts
mailbus config list
```

### Test Sending

```bash
# Create test message
cat > /tmp/test.md << 'EOF'
---
test:
  run: true
---
# Test Message

This is a test message.
EOF

# Send test message to yourself
mailbus send \
  --to $(mailbus config list | grep username | head -1 | cut -d' ' -f2) \
  --subject "[test] Test Message" \
  --file /tmp/test.md
```

### Test Polling

```bash
# List recent messages
mailbus list --limit 5

# Poll with echo handler
echo '#!/bin/bash
cat' > /tmp/echo.sh
chmod +x /tmp/echo.sh
mailbus poll --subject "\[test\]" --handler "/tmp/echo.sh" --once
```

---

## Integration Examples

### GitHub Actions

```yaml
- name: Check messages
  env:
    MAILBUS_PASSWORD: ${{ secrets.MAILBUS_PASSWORD }}
  run: |
    mailbus poll --subject "\[deploy\]" --handler "./deploy.sh" --once
```

### Systemd Service

```ini
[Unit]
Description=MailBus Polling Service
After=network.target

[Service]
Type=simple
Environment="MAILBUS_PASSWORD=your-password"
ExecStart=/usr/local/bin/mailbus poll --subject "\[alert\]" --continuous
Restart=always

[Install]
WantedBy=multi-user.target
```

---

## License Restrictions

**IMPORTANT**: Commercial cloud hosting is prohibited without explicit permission.

**Allowed:**
- Self-hosting for personal/organizational use
- Single-tenant managed hosting
- Consulting and integration services
- Derivative works under same license

**Prohibited:**
- Multi-tenant cloud SaaS offerings
- Commercial cloud service platforms
- Competing cloud services

See [LICENSE](https://github.com/mailbus/mailbus/blob/main/LICENSE) for details.

---

## Support

- **Issues**: https://github.com/mailbus/mailbus/issues
- **Discussions**: https://github.com/mailbus/mailbus/discussions
- **Documentation**: https://github.com/mailbus/mailbus/tree/main/docs

---

## Version History

- **v0.3.0** (2024-03-30): Front Matter + Markdown format
  - Message format changed from JSON to YAML Front Matter + Markdown
  - New CLI flags: --file, --meta, --body, --field, --markdown, --attach
  - Attachment metadata framework
  - Handler environment variables for Front Matter fields

- **v0.1.0** (2024-03-29): Initial release
  - SMTP/IMAP support
  - Handler system
  - Configuration management
  - Message filtering

---

**End of Agent Installation Guide**
