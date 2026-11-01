# MailBus

> The first Email-based message bus for agent communication

MailBus is a CLI tool that turns email into a message bus for agent communication. It enables any script or program to communicate asynchronously using standard email protocols (SMTP/IMAP).

## 🚀 Quick Start

### Installation

```bash
# Install from source
go install github.com/mailbus/mailbus/cmd/mailbus@latest

# Or download a binary from releases
wget https://github.com/mailbus/mailbus/releases/latest/download/mailbus-linux-amd64
chmod +x mailbus-linux-amd64
sudo mv mailbus-linux-amd64 /usr/local/bin/mailbus
```

### Initial Setup

```bash
# Initialize configuration
mailbus config init

# Edit the configuration file with your email credentials
nano ~/.mailbus/config.yaml

# Set your password as environment variable
export MAILBUS_PASSWORD=your-app-password

# Validate configuration
mailbus config validate
```

### First Message

```bash
# Send a message with Front Matter + Markdown
mailbus send \
  --to agent@example.com \
  --subject "[task] Hello World" \
  --file task.md

# List messages
mailbus list --unread

# Poll and process with a handler
mailbus poll --subject "\[task\]" --handler "./process.sh" --once
```

## 📖 Concepts

### Email as Message Bus

MailBus leverages the ubiquity and reliability of email protocols to create a simple yet powerful message bus:

- **Any email provider works**: Gmail, Outlook, self-hosted servers
- **Language agnostic**: Any program that can send/receive email can use MailBus
- **Zero infrastructure**: No need to deploy message queue servers
- **Asynchronous by nature**: Email's store-and-forward model fits perfectly

### Message Format

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

### Tags in Subject

Tags in square brackets enable message routing:

- `[task]` - Task assignments
- `[data]` - Data transfers
- `[alert]` - Alerts and notifications
- `[query]` - Query requests
- `[response]` - Response messages

## 🛠️ Commands

### send

Send messages via SMTP:

```bash
mailbus send [flags]

# Single file mode (Front Matter + Markdown)
mailbus send --to agent@example.com --subject "[task] Analysis" --file request.md

# Split mode (separate metadata and content)
mailbus send --to agent@example.com --subject "[task] Analysis" \
  --meta meta.yaml --body content.md

# Inline mode
mailbus send --to agent@example.com --subject "[task] Analysis" \
  --field "task.type=analysis" \
  --field "priority=high" \
  --markdown "# Analyze Q1 data\n\nPlease analyze..."

# With attachments
mailbus send --to agent@example.com --file request.md \
  --attach data.csv \
  --attach-desc "data.csv:Q1 sales data"

Flags:
  --to string[]           recipient addresses (required)
  --subject string        message subject (required)
  --file string           read complete message (front matter + markdown) from file
  --meta string           read front matter metadata from YAML file
  --body string           read markdown content from file
  --markdown string       inline markdown content
  --field string[]        add metadata field (key=value, supports dot notation)
  --attach string[]       file attachments
  --attach-desc string[]  attachment description (name:description)
  --attach-dir string     directory for attachment files (default: current dir)
  --header stringToString custom headers (key=value)
  --priority string       message priority (high/normal/low)
  -A, --account string     use specified account
```

### poll

Poll for and process incoming messages:

```bash
mailbus poll [flags]

# List unread messages
mailbus poll --unread

# Execute handler once
mailbus poll --subject "\[task\]" --handler "./process.sh" --once

# Continuous monitoring with attachments
mailbus poll --subject "\[task\]" --handler "./process.sh" \
  --continuous --interval 30 --download-attachments --attach-dir ./workspace

Flags:
  --subject string            subject filter (supports regex)
  --from string               sender filter
  --unread                    only process unread messages
  --once                      process once and exit
  -c, --continuous            continuously poll for messages
  -i, --interval int          polling interval in seconds (default: 30)
  -H, --handler string        handler command to execute
  --handler-timeout int       handler timeout in seconds
  --on-error string           error handling (continue/stop/retry)
  --reply-with-result         send handler result as reply
  --mark-after string         mark action after processing (read/delete/none)
  -F, --folder string         IMAP folder to check (default: INBOX)
  --format string             output format (table/json/compact)
  --download-attachments      download message attachments
  --attach-dir string         directory to save attachments
  -A, --account string        use specified account
```

### list

List messages matching criteria:

```bash
mailbus list [flags]

Flags:
  --subject string   subject filter (supports regex)
  --from string      sender filter
  --unread           only show unread messages
  --limit int        limit number of messages (default: 20)
  --format string    output format (table/json/compact)
  -A, --account string use specified account
```

### mark

Mark messages with actions:

```bash
mailbus mark [flags]

Flags:
  --id string       message ID (required)
  --action string   action: read/unread/delete/undelete/flag/unflag/move
  --folder string   target folder (required for move)
  -A, --account string use specified account
```

### config

Manage configuration:

```bash
mailbus config [subcommand]

Subcommands:
  init      Initialize configuration file
  validate  Validate configuration
  list      List all accounts
  add       Add a new account
  remove    Remove an account
  set       Set global options
```

## 📝 Configuration

Configuration file location: `~/.mailbus/config.yaml`

```yaml
# Default account to use
default_account: "gmail"

# Account configurations
accounts:
  gmail:
    imap:
      host: imap.gmail.com
      port: 993
      use_tls: true
    smtp:
      host: smtp.gmail.com
      port: 587
      use_tls: true
    username: your-email@gmail.com
    password_env: GMAIL_APP_PASSWORD  # Environment variable name
    from: your-email@gmail.com

  outlook:
    imap:
      host: outlook.office365.com
      port: 993
      use_tls: true
    smtp:
      host: smtp-mail.outlook.com
      port: 587
      use_tls: true
    username: your-email@outlook.com
    password_env: OUTLOOK_PASSWORD
    from: your-email@outlook.com

# Global settings
global:
  poll_interval: 30s    # Default polling interval
  batch_size: 20        # Default batch size
  timeout: 30s          # Default operation timeout
  max_retries: 3        # Default max retries
  verbose: false        # Verbose output
  log_level: info       # Log level
  handler_timeout: 60s  # Default handler timeout
```

## 🔐 Password Security

MailBus supports encrypted password storage for improved security:

### Option 1: Environment Variables (Traditional)

```yaml
accounts:
  myaccount:
    password_env: MY_PASSWORD  # Reference environment variable
```

```bash
export MY_PASSWORD="my-plaintext-password"
```

### Option 2: Encrypted Password (Recommended)

```yaml
accounts:
  myaccount:
    password_encrypted: "BASE64-ENCRYPTED-BLOB..."  # Encrypted password
```

```bash
# Set encryption key
export MAILBUS_CRYPTO_KEY="your-encryption-key"

# Generate encrypted password
./mailbus crypto encrypt --password "my-password"
# Output: password_encrypted: BASE64-ENCRYPTED-BLOB...
```

**Why use encrypted passwords?**
- ✅ Passwords not stored in environment variables (which may be logged)
- ✅ Supports git-tracked configurations without exposing secrets
- ✅ Compatible with CI/CD (use secrets for MAILBUS_CRYPTO_KEY)
- ✅ Backward compatible with environment variable method

**Priority:** `password_encrypted` > `password_env`

## 🔌 Handlers

Handlers are scripts or commands that process messages.

### Message Handler Input

The handler receives the message via stdin. For Front Matter messages, the handler gets:
- Email headers as environment variables
- Front Matter fields as environment variables
- Markdown content via stdin

### Environment Variables Available to Handlers

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

# Message content
# Stdin contains the markdown content
```

### Bash Handler Example

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

# Output result
echo "{\"status\":\"success\",\"result\":\"$RESULT\"}"
```

### Python Handler Example

```python
#!/usr/bin/env python3
# handler.py

import os
import sys

# Get metadata from environment
task_type = os.getenv('MAILBUS_TASK_TYPE', 'unknown')
priority = os.getenv('MAILBUS_TASK_PRIORITY', 'normal')

# Read markdown content from stdin
content = sys.stdin.read()

print(f"Processing {task_type} task (priority: {priority})")
print(f"Content length: {len(content)} bytes")
```

### Using Handlers

```bash
# Execute handler once
mailbus poll --subject "\[task\]" --handler "./handler.sh" --once

# Continuous monitoring
mailbus poll --subject "\[alert\]" --handler "./alert.sh" --continuous

# With reply
mailbus poll --subject "\[query\]" --handler "python query.py" \
  --reply-with-result --mark-after read

# With attachment downloads
mailbus poll --subject "\[data\]" --handler "./process.sh" \
  --download-attachments --attach-dir ./workspace
```

## 🌐 Email Provider Setup

### Gmail

1. Enable 2-Factor Authentication
2. Generate an App Password: Google Account → Security → App Passwords
3. Use the App Password in your configuration:

```yaml
accounts:
  gmail:
    username: your-email@gmail.com
    password_env: GMAIL_APP_PASSWORD
```

### Outlook

1. Generate an App Password: Microsoft Account → Security → App Passwords
2. Use the App Password in your configuration

### Self-Hosted (Postfix/Dovecot)

Configure your mail server settings:

```yaml
accounts:
  selfhosted:
    imap:
      host: mail.yourdomain.com
      port: 993
      use_tls: true
    smtp:
      host: mail.yourdomain.com
      port: 587
      use_tls: true
    username: user@yourdomain.com
    password_env: MAIL_PASSWORD
```

## 🔄 CI/CD Integration

### GitHub Actions

```yaml
name: Process MailBus Messages

on:
  schedule:
    - cron: '*/5 * * * *'  # Every 5 minutes

jobs:
  process:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Install MailBus
        run: go install github.com/mailbus/mailbus/cmd/mailbus@latest

      - name: Check and process messages
        env:
          MAILBUS_PASSWORD: ${{ secrets.MAILBUS_PASSWORD }}
        run: |
          mailbus poll --subject "\[deploy\]" --handler "./deploy.sh" --once
```

## 🧪 Testing

```bash
# Run unit tests
go test ./...

# Run integration tests
go test -tags=integration ./...

# Run with coverage
go test -cover ./...
```

## 🤖 For AI Agents

AI agents (Claude Code, Codex, OpenClaw, etc.) can refer to [AGENT_INSTALLATION.md](AGENT_INSTALLATION.md) for structured installation and usage instructions.

## 🤝 Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details.

## 📄 License

Apache License 2.0 with additional terms - see [LICENSE](LICENSE) for details.

**Important Restrictions:**
- **Cloud Services**: Commercial cloud hosting of MailBus is prohibited without explicit permission
- **Trademark**: The "MailBus" name and trademark are reserved
- **Enterprise Features**: Commercial rights to webhook, rule engine, and advanced features are reserved

**Allowed Uses:**
- Self-hosting for personal or organizational use
- Single-tenant managed hosting for individual clients
- Consulting and integration services
- Creating derivative works under the same license

## 🙏 Acknowledgments

- Built with [Cobra](https://github.com/spf13/cobra) for CLI
- Uses [go-imap](https://github.com/emersion/go-imap) for IMAP
- Uses [go-smtp](https://github.com/emersion/go-smtp) for SMTP
- Front Matter format inspired by Jekyll and Hugo static site generators

## 📮 Contact

- GitHub Issues: [github.com/mailbus/mailbus/issues](https://github.com/mailbus/mailbus/issues)
- Discussions: [github.com/mailbus/mailbus/discussions](https://github.com/mailbus/mailbus/discussions)
