# MailBus Handler Examples

This directory contains example handlers for MailBus.

## Echo Handler (Bash)

The `echo.sh` handler simply echoes received messages:

```bash
mailbus poll --subject "\[test\]" --handler "./examples/handlers/echo.sh" --once
```

## Data Processing Handler (Python)

The `process_data.py` handler demonstrates data processing:

```bash
mailbus poll --subject "\[data\]" --handler "python examples/handlers/process_data.py" --once
```

### Message Format

Handlers receive messages as JSON on stdin:

```json
{
  "id": "message-id@example.com",
  "from": "sender@example.com",
  "to": ["recipient@example.com"],
  "subject": "[tag] Message Subject",
  "body": "{\"type\":\"request\",\"data\":{...}}",
  "timestamp": "2024-03-29T10:30:00Z",
  "flags": ["\\Seen"]
}
```

### Response Format

Handlers should output results as JSON:

```json
{
  "status": "success",
  "result": "Processing result",
  "data": { ... }
}
```

## Creating Custom Handlers

### Bash Handler

```bash
#!/bin/bash
MESSAGE=$(cat)

# Parse message
SUBJECT=$(echo "$MESSAGE" | jq -r '.subject')
BODY=$(echo "$MESSAGE" | jq -r '.body')

# Process
echo '{"status":"success","result":"Done"}'
```

### Python Handler

```python
#!/usr/bin/env python3
import sys, json

msg = json.load(sys.stdin)
# Process message
print(json.dumps({"status": "success"}))
```

### Node.js Handler

```javascript
#!/usr/bin/env node

const msg = JSON.parse(require('fs').readFileSync(0, 'utf-8'));
// Process message
console.log(JSON.stringify({status: "success"}));
```

## CI/CD Integration Example

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

      - name: Setup Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Install MailBus
        run: go install github.com/mailbus/mailbus/cmd/mailbus@latest

      - name: Check and process messages
        env:
          MAILBUS_PASSWORD: ${{ secrets.MAILBUS_PASSWORD }}
        run: |
          mailbus poll --subject "\[deploy\]" --handler "./deploy.sh" --once
```

### GitLab CI

```yaml
process-messages:
  script:
    - go install github.com/mailbus/mailbus/cmd/mailbus@latest
    - mailbus poll --subject "\[deploy\]" --handler "./deploy.sh" --once
  only:
    - schedules
```

## Best Practices

1. **Error Handling**: Always return proper exit codes
2. **Logging**: Use stderr for logs, stdout for JSON results
3. **Timeout**: Set appropriate timeouts for long-running handlers
4. **Idempotency**: Make handlers idempotent when possible
5. **Validation**: Validate input before processing

## Testing Handlers

```bash
# Test handler manually
echo '{"from":"test@example.com","subject":"[test] Test","body":"{}"}' | ./handler.sh

# Test with mailbus
mailbus send --to $USER_EMAIL --subject "[test] Test" --body '{}'
mailbus poll --subject "\[test\]" --handler "./handler.sh" --once
```
