# MailBus v0.1.0 - Initial Release

MailBus is a CLI tool that transforms email into a message bus for agent communication. It enables email-driven automation, polling, and command execution through a simple command-line interface.

## Features

### Core Commands
- **send** - Send messages via SMTP
- **poll** - Poll for messages via IMAP with optional handler execution
- **list** - List and filter messages by subject, sender, read status
- **mark** - Mark messages as read/unread/delete
- **config** - Manage configuration file

### Key Capabilities
- Message filtering by subject, sender, and read status
- Handler system for processing received messages
- Support for multiple email providers (Gmail, Outlook, any IMAP/SMTP server)
- Configuration file management for credentials
- Extensible through custom handlers (Bash, Python, Node.js)

## Installation

### Via Go
```bash
go install github.com/mailbus/mailbus/cmd/mailbus@v0.1.0
```

### From Binary
Download the appropriate binary for your platform from the [Releases](https://github.com/mailbus/mailbus/releases) page.

## Quick Start

1. Configure your email:
```bash
mailbus config init
# Edit ~/.mailbus/config.json with your credentials
```

2. Send a message:
```bash
mailbus send --to recipient@example.com --subject "[deploy] Deploy production" --body '{"env":"prod"}'
```

3. Poll for messages:
```bash
mailbus poll --subject "[deploy]" --once
```

4. Use with a handler:
```bash
mailbus poll --subject "[data]" --handler "./process.sh" --once
```

## Example Handlers

### Bash Handler
```bash
#!/bin/bash
MESSAGE=$(cat)
SUBJECT=$(echo "$MESSAGE" | jq -r '.subject')
echo "{\"status\":\"success\",\"subject\":\"$SUBJECT\"}"
```

### Python Handler
```python
#!/usr/bin/env python3
import sys, json
msg = json.load(sys.stdin)
print(json.dumps({"status": "success"}))
```

## Configuration

Configuration is stored in `~/.mailbus/config.json`:

```json
{
  "imap": {
    "server": "imap.gmail.com:993",
    "username": "your-email@gmail.com",
    "password": "your-app-password"
  },
  "smtp": {
    "server": "smtp.gmail.com:587",
    "username": "your-email@gmail.com",
    "password": "your-app-password"
  }
}
```

## Documentation

- [README.md](https://github.com/mailbus/mailbus/blob/main/README.md)
- [DESIGN.md](https://github.com/mailbus/mailbus/blob/main/DESIGN.md)
- [Handler Examples](https://github.com/mailbus/mailbus/tree/main/examples/handlers)

## Supported Email Providers

- Gmail (requires App Password)
- Outlook/Hotmail
- Any IMAP/SMTP compatible server

## License

Apache License 2.0 with additional terms

**Important Restrictions:**
- **Cloud Services**: Commercial cloud hosting of MailBus is prohibited without explicit permission
- **Trademark**: The "MailBus" name and trademark are reserved
- **Enterprise Features**: Commercial rights to webhook, rule engine, and advanced features are reserved

**Allowed Uses:**
- Self-hosting for personal or organizational use
- Single-tenant managed hosting for individual clients
- Consulting and integration services
- Creating derivative works under the same license

See [LICENSE](https://github.com/mailbus/mailbus/blob/main/LICENSE) for complete details.

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](https://github.com/mailbus/mailbus/blob/main/CONTRIBUTING.md) for guidelines.

## Roadmap

Future releases will include:
- Attachment support
- IDLE/push notification support
- Webhook adapter
- Message encryption/signing
- Rate limiting
- Better error handling and retries
- Comprehensive test suite

---

**Note:** This is an initial release. Please report issues at [GitHub Issues](https://github.com/mailbus/mailbus/issues).
