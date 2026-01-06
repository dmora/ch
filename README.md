# ch - Claude History CLI

A memory-efficient Go CLI tool for viewing Claude Code conversation history, including subagent conversations.

## Why ch?

Claude Code's UI shows history for humans. `ch` exposes it for **agentic systems**:

- **JSON output** for programmatic access (`--json` flag on all commands)
- **Memory-efficient streaming** handles thousands of conversations without OOM
- **Foundation for MCP server wrappers** - designed for AI-to-AI context retrieval
- **Cross-project search** and filtering across all your Claude Code sessions

## Disclaimer

This tool accesses Claude Code's local conversation cache (`~/.claude/projects/`).
This is **undocumented, unofficial, and not supported by Anthropic**. The data format
may change without notice. Use at your own risk.

## Installation

```bash
go install github.com/dmora/ch/cmd/ch@latest
```

Or build from source:

```bash
git clone https://github.com/dmora/ch.git
cd ch
go build -o ch ./cmd/ch
```

## Commands

| Command | Description |
|---------|-------------|
| `ch list` | List conversations (table format) |
| `ch show <id>` | Show specific conversation |
| `ch search <query>` | Search across conversations |
| `ch resume <id>` | Resume conversation in Claude Code |
| `ch agents <id>` | List agents spawned by a conversation |
| `ch projects` | List all projects |
| `ch stats` | Show usage statistics |

## Flags

### list

- `-a, --agents` - Include agent/subagent conversations
- `-p, --project <name>` - Filter by project
- `-n, --limit <num>` - Limit results (default 50)
- `-g, --global` - All projects (default: current dir's project)
- `--json` - JSON output

### show

- `--thinking` - Include thinking blocks
- `--tools` - Include tool calls
- `--json` - JSON output
- `--raw` - Raw JSONL output

### search

- `-a, --agents` - Include agent conversations
- `-p, --project <name>` - Filter by project
- `-n, --limit <num>` - Limit results (default 20)
- `-g, --global` - Search all projects
- `-c, --case-sensitive` - Case-sensitive search
- `--json` - JSON output

## Examples

```bash
# List recent 50 conversations in current project
ch list

# List all conversations including agents
ch list -a

# List from all projects globally
ch list -g -n 100

# Show specific conversation with thinking blocks
ch show abc123 --thinking

# Search for "docker" in all projects
ch search "docker" -g

# Resume a conversation
ch resume abc123

# List agents for a conversation
ch agents abc123

# List projects
ch projects

# Show stats
ch stats
```

## Memory Efficiency

Unlike tools that load entire conversation files into memory, ch uses:

- **Streaming JSONL parsing** - Only parses one line at a time
- **Lazy content loading** - Only loads full content when displaying specific conversations
- **Worker pool with bounded concurrency** - Parallel processing without memory explosion
- **Byte-level quick checks** - Avoids JSON parsing when possible

This allows ch to handle conversation histories with thousands of files totaling gigabytes of data without running out of memory.

## Environment Variables

- `CLAUDE_PROJECTS_DIR` - Override the default projects directory (`~/.claude/projects`)
- `CLAUDE_BIN` - Override the Claude CLI binary path (default: `claude`)

## Testing

```bash
# Run all tests
go test ./...

# Run with coverage
go test ./... -cover

# Run e2e tests
go test -v -run TestE2E
```

## License

MIT
