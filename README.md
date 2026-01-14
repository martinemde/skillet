# Skillet

A Go CLI tool that runs SKILL.md files by invoking the Claude CLI with the appropriate configuration.

## Overview

Skillet parses SKILL.md files (following the [Agent Skills specification](https://agentskills.io)), extracts frontmatter configuration, interpolates variables, and executes the skill using Claude CLI in headless mode with stream-json output.

## Features

- ✅ **Full SKILL.md parsing** - Supports all frontmatter fields (name, description, allowed-tools, model, etc.)
- ✅ **Variable interpolation** - Automatically expands `{baseDir}` to the skill directory path
- ✅ **Claude CLI integration** - Invokes Claude in print mode with stream-json output
- ✅ **Stream parsing** - Parses and formats JSON stream output from Claude
- ✅ **Comprehensive validation** - Validates skill files against the Agent Skills specification
- ✅ **Well-tested** - Extensive test coverage for all components

## Installation

### Build from source

```bash
go build -o skillet ./cmd/skillet
```

### Install globally

```bash
go install ./cmd/skillet
```

## Usage

Skillet supports multiple ways to specify a skill:

```bash
# Direct file path
skillet path/to/SKILL.md

# Directory (automatically finds SKILL.md inside)
skillet path/to/skill-directory

# Skill name shortcut (looks in .claude/skills/<name>/SKILL.md)
skillet write-skill

# Remote URL
skillet https://raw.githubusercontent.com/user/repo/main/skill.md
```

> [!WARNING]
> **Skills can be a security risk.** Skills can execute commands, exfiltrate data, modify files, and compromise your system. Only use skills from sources you completely trust.

**Additional options:**
- `--prompt "text"` - Custom prompt instead of skill description
- `--dry-run` - Show command without executing
- `--verbose` - Show raw JSON stream
- `--usage` - Show token usage statistics

## Skill Resolution

Skillet resolves skill paths in this order:
1. **URL** - Valid HTTP/HTTPS URL → download and validate
2. **Exact file path** - File exists → use directly
3. **Directory** - Directory with SKILL.md → use that file
4. **Skill name** - Bare word → look in `.claude/skills/<name>/SKILL.md`

## Command-line Options

| Flag | Description |
|------|-------------|
| `--help` | Show help message |
| `--version` | Show version information |
| `--verbose` | Show verbose output including raw JSON stream |
| `--usage` | Show token usage statistics after execution |
| `--dry-run` | Show the command that would be executed without running it |
| `--prompt` | Optional prompt to pass to Claude (default: uses skill description) |

## SKILL.md Format

A SKILL.md file must contain YAML frontmatter followed by markdown content:

```yaml
---
name: skill-name
description: What this skill does and when to use it
allowed-tools: Read,Write,Bash
model: claude-opus-4-5-20251101
---

# Skill Instructions

Your skill instructions go here...
```

### Frontmatter Fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Skill name (lowercase, hyphens only, max 64 chars) |
| `description` | Yes | Description of what the skill does (max 1024 chars) |
| `allowed-tools` | No | Space-delimited list of pre-approved tools |
| `model` | No | Claude model to use (defaults to current model) |
| `license` | No | License information |
| `compatibility` | No | Environment requirements (max 500 chars) |
| `metadata` | No | Additional key-value metadata |
| `version` | No | Skill version |
| `disable-model-invocation` | No | Prevent automatic invocation |
| `mode` | No | Mark as a mode command |

### Variable Interpolation

Skillet automatically expands the following variables:

- `{baseDir}` - Absolute path to the directory containing SKILL.md

Example:

```markdown
Read configuration from {baseDir}/config.json
```

Becomes:

```markdown
Read configuration from /absolute/path/to/skill/config.json
```

## Project Structure

```
skillet/
├── cmd/
│   └── skillet/          # CLI entry point
│       ├── main.go
│       └── main_test.go
├── internal/
│   ├── parser/           # SKILL.md parsing and validation
│   │   ├── parser.go
│   │   └── parser_test.go
│   ├── executor/         # Claude command execution
│   │   ├── executor.go
│   │   └── executor_test.go
│   └── formatter/        # JSON stream formatting
│       ├── formatter.go
│       └── formatter_test.go
├── testdata/             # Test SKILL.md files
│   ├── simple-skill/
│   ├── comprehensive-skill/
│   ├── interpolation-skill/
│   ├── invalid-skill/
│   └── no-frontmatter/
├── docs/                 # Documentation
│   ├── specification.md
│   ├── deep_dive.md
│   └── claude_cli_reference.md
├── go.mod
├── go.sum
└── README.md
```

## Testing

Run all tests:

```bash
go test -v ./...
```

Run tests for a specific package:

```bash
go test -v ./internal/parser
go test -v ./internal/executor
go test -v ./internal/formatter
```

## Examples

### Simple Skill

```yaml
---
name: simple-skill
description: A simple skill for testing. Use when you need basic functionality.
---

# Simple Skill

This skill performs basic tasks.
```

### Comprehensive Skill

```yaml
---
name: code-reviewer
description: Expert code reviewer. Use after making code changes.
allowed-tools: Read Grep Glob Bash(git:*)
model: claude-opus-4-5-20251101
license: Apache-2.0
metadata:
  author: example-org
  version: "1.0.0"
---

# Code Reviewer

Reviews code for quality, security, and best practices.

## Instructions

1. Read the code changes using Read and Grep
2. Analyze for potential issues
3. Provide detailed feedback
```

### With Variable Interpolation

```yaml
---
name: data-processor
description: Processes data files. Use for data analysis tasks.
allowed-tools: Read Write Bash
---

# Data Processor

Process data files from the skill directory.

## Setup

1. Read configuration from {baseDir}/config.json
2. Load data from {baseDir}/data/input.csv
3. Write output to {baseDir}/data/output.csv
```

## How It Works

1. **Parse** - Reads SKILL.md file and extracts YAML frontmatter and markdown content
2. **Validate** - Ensures the skill follows the Agent Skills specification
3. **Interpolate** - Expands variables like `{baseDir}` to absolute paths
4. **Execute** - Builds and runs the Claude CLI command with appropriate flags
5. **Format** - Parses stream-json output and formats it for display

## Claude CLI Integration

Skillet invokes Claude CLI with the following flags:

- `-p` - Print mode (headless, no interactive session)
- `--output-format stream-json` - JSON streaming output
- `--model <model>` - If specified in frontmatter
- `--allowed-tools <tools>` - If specified in frontmatter
- `--system-prompt <content>` - Skill name, description, and content

## Requirements

- Go 1.21 or later
- Claude CLI installed and configured

## Contributing

Contributions are welcome! Please ensure:

1. All tests pass: `go test ./...`
2. Code follows Go conventions: `go fmt ./...`
3. New features include tests
4. Documentation is updated

## License

See LICENSE file for details.

## References

- [Agent Skills Specification](https://agentskills.io)
- [Claude CLI Documentation](https://code.claude.com/docs)
- [SKILL.md Deep Dive](docs/deep_dive.md)

## Version

Current version: 0.1.0
