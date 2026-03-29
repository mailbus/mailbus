#!/usr/bin/env python3
"""
Data processing handler for MailBus
Usage: mailbus poll --subject "[data]" --handler "python examples/handlers/process_data.py" --once
"""

import sys
import json
import datetime

def main():
    # Read message from stdin
    message = json.load(sys.stdin)

    print(f"Processing message: {message.get('subject', 'unknown')}", file=sys.stderr)

    # Extract data from message body
    try:
        body = json.loads(message.get('body', '{}'))
        data_type = body.get('type', 'unknown')

        # Process different data types
        if data_type == 'report':
            result = process_report(body)
        elif data_type == 'query':
            result = process_query(body)
        else:
            result = {"status": "unknown_type", "type": data_type}

    except json.JSONDecodeError:
        result = {"status": "error", "message": "Invalid JSON in body"}

    # Output result as JSON
    print(json.dumps(result, indent=2))

    return 0 if result.get('status') == 'success' else 1

def process_report(data):
    """Process report data"""
    report_id = data.get('id', 'unknown')
    print(f"Processing report: {report_id}", file=sys.stderr)

    # Simulate processing
    return {
        "status": "success",
        "report_id": report_id,
        "processed_at": datetime.datetime.now().isoformat(),
        "result": "Report processed successfully"
    }

def process_query(data):
    """Process query data"""
    query = data.get('query', '')
    print(f"Processing query: {query}", file=sys.stderr)

    # Simulate query processing
    return {
        "status": "success",
        "query": query,
        "results": ["result1", "result2", "result3"],
        "count": 3
    }

if __name__ == '__main__':
    sys.exit(main())
