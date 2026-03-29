# MailBus

> Email-based message bus for agent communication

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
# Send a message
mailbus send \
  --to agent@example.com \
  --subject "[task] Hello World" \
  --body '{"type":"greeting","message":"Hello from MailBus"}'

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

Messages are typically JSON-formatted emails with structured subjects:

```
Subject: [tag.category] Human readable title
X-MailBus-Version: 1.0
Body: {"type":"request","data":{...}}
```

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

Flags:
  --to string[]           recipient addresses (required)
  --subject string        message subject (required)
  --body string           message body
  --file string           read body from file
  --attach string[]       file attachments
  --header stringToString custom headers (key=value)
  --format string         body format (json or text)
  --priority string       message priority (high/normal/low)
```

### poll

Poll for and process incoming messages:

```bash
mailbus poll [flags]

Flags:
  --subject string       subject filter (supports regex)
  --from string          sender filter
  --unread               only process unread messages
  --once                 process once and exit
  --continuous           continuously poll for messages
  --interval int         polling interval in seconds (default: 30)
  --handler string       handler command to execute
  --on-error string      error handling (continue/stop/retry)
  --reply-with-result    send handler result as reply
  --mark-after string    mark action after processing
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
```

### mark

Mark messages with actions:

```bash
mailbus mark [flags]

Flags:
  --id string       message ID (required)
  --action string   action: read/unread/delete/undelete/flag/unflag/move
  --folder string   target folder (required for move)
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
    password_env: MAILBUS_PASSWORD  # Environment variable name
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

## 🔌 Handlers

Handlers are scripts or commands that process messages.

### Bash Handler

```bash
#!/bin/bash
# process.sh

# Read message from stdin
MESSAGE=$(cat)

# Extract data using jq
QUERY=$(echo "$MESSAGE" | jq -r '.payload.query')

# Process the data
RESULT=$(python process.py "$QUERY")

# Output result as JSON
echo "{\"status\":\"success\",\"result\":\"$RESULT\"}"
```

### Python Handler

```python
#!/usr/bin/env python3
# handler.py

import sys
import json

# Read message from stdin
message = json.load(sys.stdin)

# Process the message
query = message.get('payload', {}).get('query', '')
result = process(query)

# Output result as JSON
print(json.dumps({"status": "success", "result": result}))
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

## 🤝 Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details.

## 📄 License

Apache License 2.0 - see [LICENSE](LICENSE) for details.

## 🙏 Acknowledgments

- Built with [Cobra](https://github.com/spf13/cobra) for CLI
- Uses [go-imap](https://github.com/emersion/go-imap) for IMAP
- Uses [go-smtp](https://github.com/emersion/go-smtp) for SMTP

## 📮 Contact

- GitHub Issues: [github.com/mailbus/mailbus/issues](https://github.com/mailbus/mailbus/issues)
- Discussions: [github.com/mailbus/mailbus/discussions](https://github.com/mailbus/mailbus/discussions)
