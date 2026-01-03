# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test Commands

```bash
# Build
go build -o ch ./cmd/ch

# Run all tests
go test ./...

# Run with coverage
go test ./... -cover

# Run a single test
go test -v -run TestFunctionName ./internal/package/

# Run E2E tests only
go test -v -run TestE2E
```

## Architecture

This is a memory-efficient Go CLI for viewing Claude Code conversation history stored in `~/.claude/projects/`.

### Package Structure

- **cmd/ch** - Entry point, calls `cli.Execute()`
- **internal/cli** - Cobra commands (list, show, search, resume, agents, projects, stats)
- **internal/jsonl** - JSONL types and streaming parser
- **internal/history** - Conversation scanning, project management, search
- **internal/display** - Table rendering, colored output, TTY detection
- **internal/config** - Configuration loading

### Key Design Patterns

**Two-Phase Parsing**: The `RawEntry` struct uses `json.RawMessage` for the `Message` field to defer full parsing. This allows scanning thousands of files while only parsing message content when actually needed (e.g., for display or search).

**Worker Pool**: Scanner and Search use bounded goroutine pools (default 4 workers) with channels to process files in parallel without memory explosion.

**Polymorphic Content**: Message content can be a string OR an array of ContentBlocks. Custom `UnmarshalJSON` in `internal/jsonl/types.go` handles both formats.

### Claude Code Data Model

Conversations are stored as JSONL files in `~/.claude/projects/{encoded-path}/`:
- Main conversations: `{sessionId}.jsonl`
- Agent conversations: `agent-{agentId}.jsonl`

Path encoding: `/Users/foo/bar` becomes `-Users-foo-bar` (slashes and dots replaced with dashes).

Agent-to-parent linkage: Agents have `sessionId` field pointing to parent's filename (without .jsonl).

### Entry Types

```
user, assistant, system     - Conversation messages
summary                     - Context summaries
file-history-snapshot       - File state snapshots
queue-operation            - Internal queue ops
```

### Content Block Types

```
text, thinking, tool_use, tool_result, image
```
