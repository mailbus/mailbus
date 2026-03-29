# MailBus Agent Installation Guide

**Purpose**: This document provides structured installation and usage instructions for AI agents (Claude Code, Codex, OpenClaw, etc.) to install and use MailBus.

---

## Quick Reference

- **Repository**: https://github.com/mailbus/mailbus
- **Version**: v0.1.0
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

### Option 3: Download Binary

```bash
# Linux
wget https://github.com/mailbus/mailbus/releases/latest/download/mailbus-linux-amd64 -O mailbus
chmod +x mailbus

# macOS
wget https://github.com/mailbus/mailbus/releases/latest/download/mailbus-darwin-amd64 -O mailbus
chmod +x mailbus

# Windows (PowerShell)
Invoke-WebRequest -Uri "https://github.com/mailbus/mailbus/releases/latest/download/mailbus-windows-amd64.exe" -OutFile "mailbus.exe"
```

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

```bash
# Set password as environment variable
export MAILBUS_PASSWORD=your-app-password

# For persistent configuration, add to ~/.bashrc or ~/.zshrc:
echo 'export MAILBUS_PASSWORD=your-app-password' >> ~/.bashrc
```

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

## Core Commands Reference

### Send Message

```bash
mailbus send \
  --to recipient@example.com \
  --subject "[tag] Message Subject" \
  --body '{"key":"value"}'
```

**Flags:**
- `--to string[]`: Recipient addresses (required)
- `--subject string`: Message subject (required)
- `--body string`: Message body (JSON format recommended)
- `--file string`: Read body from file
- `--format string`: Body format (json/text)
- `--priority string`: Priority (high/normal/low)

### Poll Messages

```bash
mailbus poll \
  --subject "\[tag\]" \
  --handler "./handler.sh" \
  --once
```

**Flags:**
- `--subject string`: Subject filter (regex supported)
- `--from string`: Sender filter
- `--unread`: Only unread messages
- `--once`: Process once and exit
- `--continuous`: Continuous polling mode
- `--interval int`: Polling interval in seconds (default: 30)
- `--handler string`: Handler command to execute
- `--on-error string`: Error handling (continue/stop/retry)

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

### Mark Messages

```bash
mailbus mark --id "message-id" --action read
```

**Flags:**
- `--id string`: Message ID (required)
- `--action string`: Action (read/unread/delete)
- `--folder string`: Target folder (for move action)

---

## Handler Development

### Message Format (Input to Handler)

```json
{
  "id": "message-id@example.com",
  "from": "sender@example.com",
  "to": ["recipient@example.com"],
  "subject": "[tag] Subject",
  "body": "{\"type\":\"request\",\"data\":{}}",
  "timestamp": "2024-03-29T10:30:00Z",
  "flags": ["\\Seen"]
}
```

### Handler Response Format

```json
{
  "status": "success",
  "result": "Processing complete",
  "data": {}
}
```

### Example Handlers

**Bash Handler:**

```bash
#!/bin/bash
# process.sh

MESSAGE=$(cat)
SUBJECT=$(echo "$MESSAGE" | jq -r '.subject')
BODY=$(echo "$MESSAGE" | jq -r '.body')

# Process message
echo '{"status":"success","result":"processed"}'
```

**Python Handler:**

```python
#!/usr/bin/env python3
import sys, json

msg = json.load(sys.stdin)
subject = msg.get('subject', '')
body = msg.get('body', '{}')

# Process message
print(json.dumps({"status": "success"}))
```

**Node.js Handler:**

```javascript
#!/usr/bin/env node
const msg = JSON.parse(require('fs').readFileSync(0, 'utf-8'));
// Process message
console.log(JSON.stringify({status: "success"}));
```

---

## Common Workflows

### Send and Wait for Response

```bash
# Send message
mailbus send \
  --to agent@example.com \
  --subject "[query] Get Data" \
  --body '{"query":"SELECT * FROM users"}'

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
# Send test message to yourself
mailbus send \
  --to $(mailbus config list | grep username | head -1 | cut -d' ' -f2) \
  --subject "[test] Test Message" \
  --body '{"test":true}'
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

- **v0.1.0** (2024-03-29): Initial release
  - SMTP/IMAP support
  - Handler system
  - Configuration management
  - Message filtering

---

**End of Agent Installation Guide**
