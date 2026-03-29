#!/bin/bash
# Simple handler that echoes received messages
# Usage: mailbus poll --handler "./examples/handlers/echo.sh" --once

# Read message from stdin
MESSAGE=$(cat)

# Extract relevant fields
echo "=== Received Message ==="
echo "$MESSAGE" | jq -r 'if .from then "From: \(.from)" else empty end'
echo "$MESSAGE" | jq -r 'if .subject then "Subject: \(.subject)" else empty end'
echo "$MESSAGE" | jq -r 'if .timestamp then "Timestamp: \(.timestamp)" else empty end'
echo "$MESSAGE" | jq -r 'if .body then "Body: \(.body)" else empty end'
echo "=== End Message ==="

# Return success
echo '{"status":"success","message":"Message received"}'
