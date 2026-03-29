# Changelog

All notable changes to MailBus will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial CLI implementation with send/poll/list/mark/config commands
- SMTP adapter for sending messages
- IMAP adapter for receiving messages
- Message filtering by subject, sender, read status
- Handler system for processing messages
- Configuration file management

### To Be Added
- Attachment support
- IDLE/push notification support
- Webhook adapter
- Message encryption/signing
- Rate limiting
- Better error handling and retries
- Comprehensive test suite

## [0.1.0] - TBD

### Initial Release

#### Features
- Send messages via SMTP
- Poll for messages via IMAP
- List messages with filtering
- Mark messages (read/unread/delete)
- Configuration file management
- Handler execution system

#### Email Providers
- Gmail
- Outlook
- Any IMAP/SMTP compatible server

#### Documentation
- README with quick start guide
- Example handlers
- Architecture documentation
