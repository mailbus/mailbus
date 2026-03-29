# Contributing to MailBus

Thank you for your interest in contributing to MailBus! This document provides guidelines for contributing.

## Code of Conduct

- Be respectful and inclusive
- Constructive feedback only
- Focus on what is best for the community

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/your-username/mailbus.git`
3. Create a branch: `git checkout -b feature/your-feature-name`

## Development

### Prerequisites

- Go 1.21 or later
- Make (optional, for using makefiles)

### Building

```bash
go build ./cmd/mailbus
```

### Running Tests

```bash
go test ./...
go test -v ./pkg/...
```

### Code Style

- Follow Go conventions (gofmt, golint)
- Write tests for new features
- Document exported functions

## Submitting Changes

1. Ensure tests pass
2. Update documentation if needed
3. Commit your changes with a clear message
4. Push to your fork
5. Create a pull request

## Pull Request Guidelines

- Describe what your PR does
- Link to related issues
- Include tests for new features
- Update documentation

## Reporting Issues

When reporting issues, please include:

- MailBus version
- Go version
- Operating system
- Steps to reproduce
- Expected vs actual behavior

## License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.
