# Changelog

All notable changes to MailBus will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Password encryption** using age encryption with scrypt key derivation
  - `mailbus crypto encrypt` command for generating encrypted passwords
  - `password_encrypted` field in config.yaml for secure password storage
  - `MAILBUS_CRYPTO_KEY` environment variable for decryption key
  - Priority: `password_encrypted` > `password_env` for backward compatibility
- Front Matter + Markdown message format with metadata support
- Attachment support with file descriptions
- Multiple send modes: file, split (meta+body), inline, and field-based

### Changed
- IMAP Subject search now searches headers instead of body (fixes subject filtering)
- Markdown content converted to text/plain for better email client compatibility

### Fixed
- IMAP SubjectPattern search criteria (was searching body, now searches header)
- scrypt work factor consistency for password encryption (fixed at 15)

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
