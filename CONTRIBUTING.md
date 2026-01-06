# Contributing to ch

Thank you for your interest in contributing to ch!

## Development Setup

```bash
# Clone the repository
git clone https://github.com/dmora/ch.git
cd ch

# Build
go build -o ch ./cmd/ch

# Run tests
go test ./...

# Run with coverage
go test ./... -cover
```

## Code Style

- Run `gofmt` before committing
- Run `golangci-lint run` to check for issues
- Keep cyclomatic complexity under 15 (we use `gocyclo -over 15`)

## Pull Request Process

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Make your changes
4. Run tests (`go test ./...`)
5. Commit with a descriptive message
6. Push to your fork
7. Open a Pull Request

## Commit Messages

Use conventional commit format:
- `feat:` new feature
- `fix:` bug fix
- `docs:` documentation
- `test:` tests
- `chore:` maintenance

## Questions?

Open an issue for discussion before starting large changes.
