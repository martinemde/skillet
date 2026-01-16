# Claude Development Notes

This file contains reminders and best practices for AI-assisted development on this project.

## Pre-Commit Checklist

Before committing any code changes, **always** run:

1. **Format code**: `go fmt ./...`
2. **Run linting**: `golangci-lint run` (if available) or `go vet ./...`
3. **Run tests**: `go test ./...`
4. **Build check**: `go build ./...`

These steps ensure code quality and prevent common issues from being committed.

## Testing

- Run all tests: `go test -v ./...`
- Run tests for specific package: `go test -v ./cmd/skillet/`
- Run with coverage: `go test -cover ./...`

## Project Structure

- `cmd/skillet/` - Main CLI application
- `internal/formatter/` - Output formatting and styling
- `internal/executor/` - Claude CLI execution
- `internal/parser/` - SKILL.md parsing
- `internal/resolver/` - Skill path resolution

## Code Style

- Follow standard Go conventions
- Use `gofmt` for consistent formatting
- Keep functions focused and single-purpose
- Add comments for exported functions and types
