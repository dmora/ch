# ch - Claude History CLI

A memory-efficient Go CLI tool for viewing Claude Code conversation history, including subagent conversations.

## Problem Statement

The existing Rust tool (`claude-history`) gets OOM killed when processing ~2400 agent files because it loads all conversation content into memory simultaneously. With 80MB files and parallel processing, this causes memory exhaustion.

## Requirements

### Commands

| Command | Description |
|---------|-------------|
| `ch list` | List conversations (table format, paginated) |
| `ch show <id>` | Show specific conversation |
| `ch search <query>` | Search across conversations |
| `ch resume <id>` | Resume conversation in Claude Code |
| `ch agents <id>` | List agents spawned by a conversation |
| `ch projects` | List all projects |
| `ch stats` | Show usage statistics |

### Flags

**list:**
- `-a, --agents` - Include agent/subagent conversations
- `-p, --project <name>` - Filter by project
- `-n, --limit <num>` - Limit results (default 50)
- `-g, --global` - All projects (default: current dir's project)
- `--json` - JSON output

**show:**
- `--thinking` - Include thinking blocks
- `--tools` - Include tool calls
- `--json` - JSON output
- `--raw` - Raw JSONL output

## Project Structure

```
/Users/davidmora/Projects/github.com/dmora/ch/
├── cmd/
│   └── ch/
│       └── main.go           # Entry point
├── internal/
│   ├── cli/
│   │   ├── root.go          # Root command setup
│   │   ├── list.go          # ch list command
│   │   ├── show.go          # ch show command
│   │   ├── search.go        # ch search command
│   │   ├── resume.go        # ch resume command
│   │   ├── agents.go        # ch agents command
│   │   ├── projects.go      # ch projects command
│   │   └── stats.go         # ch stats command
│   ├── config/
│   │   └── config.go        # Configuration handling
│   ├── history/
│   │   ├── paths.go         # Project path encoding/decoding
│   │   ├── scanner.go       # Memory-efficient JSONL scanner
│   │   ├── conversation.go  # Conversation metadata extraction
│   │   ├── project.go       # Project listing
│   │   └── search.go        # Search implementation
│   ├── jsonl/
│   │   ├── types.go         # JSONL entry type definitions
│   │   ├── parser.go        # Streaming JSON parser
│   │   └── content.go       # Content extraction utilities
│   └── display/
│       ├── table.go         # Table formatting
│       ├── conversation.go  # Conversation display
│       └── colors.go        # TTY detection and colors
├── go.mod
├── go.sum
├── PLAN.md
└── README.md
```

## Key Data Structures

```go
// internal/jsonl/types.go

type EntryType string

const (
    EntryTypeUser         EntryType = "user"
    EntryTypeAssistant    EntryType = "assistant"
    EntryTypeSummary      EntryType = "summary"
    EntryTypeSystem       EntryType = "system"
    EntryTypeFileSnapshot EntryType = "file-history-snapshot"
    EntryTypeQueueOp      EntryType = "queue-operation"
)

type ContentBlockType string

const (
    BlockTypeText       ContentBlockType = "text"
    BlockTypeThinking   ContentBlockType = "thinking"
    BlockTypeToolUse    ContentBlockType = "tool_use"
    BlockTypeToolResult ContentBlockType = "tool_result"
    BlockTypeImage      ContentBlockType = "image"
)

// Raw JSON entry - minimal parsing for scanning
type RawEntry struct {
    Type        EntryType       `json:"type"`
    Timestamp   string          `json:"timestamp,omitempty"`
    UUID        string          `json:"uuid,omitempty"`
    ParentUUID  string          `json:"parentUuid,omitempty"`
    SessionID   string          `json:"sessionId,omitempty"`
    IsSidechain bool            `json:"isSidechain,omitempty"`
    AgentID     string          `json:"agentId,omitempty"`
    CWD         string          `json:"cwd,omitempty"`
    Message     json.RawMessage `json:"message,omitempty"` // Defer parsing
    Summary     string          `json:"summary,omitempty"`
}

// Full message parsing (only when needed)
type Message struct {
    Role    string         `json:"role"`
    Model   string         `json:"model,omitempty"`
    Content []ContentBlock `json:"content"`
}

type ContentBlock struct {
    Type      ContentBlockType `json:"type"`
    Text      string           `json:"text,omitempty"`
    Thinking  string           `json:"thinking,omitempty"`
    ID        string           `json:"id,omitempty"`
    Name      string           `json:"name,omitempty"`
    Input     json.RawMessage  `json:"input,omitempty"`
    ToolUseID string           `json:"tool_use_id,omitempty"`
    Content   json.RawMessage  `json:"content,omitempty"`
    IsError   bool             `json:"is_error,omitempty"`
}

// internal/history/conversation.go

// ConversationMeta - lightweight metadata for listing
type ConversationMeta struct {
    ID              string    // UUID/AgentID from filename
    SessionID       string    // Session ID (for agents, points to parent)
    Path            string    // Full file path
    Project         string    // Project directory name
    ProjectPath     string    // Decoded project path
    Timestamp       time.Time // From first entry or file mtime
    Preview         string    // First ~100 chars of first user message
    MessageCount    int       // Number of user+assistant messages
    IsAgent         bool      // Is this an agent/sidechain conversation
    AgentCount      int       // Number of agents spawned (for main conversations)
    ParentSessionID string    // Parent session ID (for agents only)
    FileSize        int64     // For stats
}
```

## Memory Optimization Strategy

### Problem
The existing tool loads `full_text` (all message content joined) for every conversation into memory simultaneously. With 80MB files and parallel processing, this causes OOM.

### Solution: Two-Phase Approach

**Phase 1: Metadata Scanning (Fast, Low Memory)**
```go
func ScanConversationMeta(path string) (*ConversationMeta, error) {
    file, _ := os.Open(path)
    defer file.Close()

    scanner := bufio.NewScanner(file)
    var meta ConversationMeta
    var firstUserText string
    messageCount := 0

    for scanner.Scan() {
        line := scanner.Bytes()

        // Quick check without full parse
        if bytes.Contains(line, []byte(`"type":"user"`)) {
            messageCount++
            if firstUserText == "" {
                var entry RawEntry
                json.Unmarshal(line, &entry)
                firstUserText = extractPreview(entry.Message, 200)
            }
        } else if bytes.Contains(line, []byte(`"type":"assistant"`)) {
            messageCount++
        }
    }

    meta.Preview = firstUserText
    meta.MessageCount = messageCount
    return &meta, nil
}
```

**Phase 2: Full Content Loading (On-Demand)**
Only load full content when displaying a specific conversation.

### Optimizations
1. **Lazy Preview**: Only parse first user message for preview
2. **Byte-level Quick Checks**: Use `bytes.Contains` before JSON parsing
3. **Streaming JSON**: Use `json.Decoder` instead of reading entire file
4. **Bounded Buffer**: Scanner with reasonable buffer size
5. **Parallel with Limits**: Use worker pool with bounded concurrency (4 workers)
6. **Early Termination**: For search, stop after finding N matches

## Implementation Phases

### Phase 1: Core Infrastructure
- Project setup (go.mod, directory structure)
- JSONL type definitions
- Streaming scanner implementation
- Path encoding/decoding utilities

### Phase 2: List Command
- Project discovery and filtering
- Conversation metadata extraction
- Table output with pagination
- JSON output mode
- Agent filtering

### Phase 3: Show Command
- Full conversation loading
- Content block rendering
- Thinking/tools toggle
- Raw JSONL mode
- Colored output

### Phase 4: Search Command
- Streaming content search
- Early termination on limit
- Highlight matches
- Search across projects

### Phase 5: Resume/Projects/Stats
- Resume: exec `claude --resume`
- Projects: list and filter projects
- Stats: aggregate conversation/token counts

### Phase 6: Polish
- TTY detection for colors
- Piping support
- Error handling
- Documentation
- Testing

## Dependencies

**Required:**
- `github.com/spf13/cobra` - CLI framework
- `github.com/fatih/color` - Terminal colors
- `github.com/olekukonko/tablewriter` - Table formatting
- `github.com/mattn/go-isatty` - TTY detection

**Optional:**
- `github.com/sahilm/fuzzy` - Fuzzy matching for search

## Performance Targets

| Operation | Target |
|-----------|--------|
| `ch list` (50 items) | < 500ms |
| `ch list -g` (all projects) | < 2s |
| `ch show` (single file) | < 200ms |
| Memory usage | < 100MB even with 1GB of JSONL files |
| Startup time | < 50ms |

## JSONL File Locations

- Main storage: `~/.claude/projects/`
- Project encoding: Path with `/` replaced by `-` (e.g., `-Users-davidmora-Projects-foo`)
- Main conversations: `{sessionId}.jsonl`
- Agent conversations: `agent-{agentId}.jsonl`

## Main ↔ Agent Linkage

**The link between main conversations and their spawned agents is `sessionId`.**

### File Structure
```
~/.claude/projects/-Users-davidmora-Projects-foo/
├── 9dbf1107-d255-4d17-a544-aadb594fc786.jsonl    # Main conversation (filename = sessionId)
├── agent-d0e14239.jsonl                           # Agent spawned by main
├── agent-a32de17.jsonl                            # Another agent spawned by main
└── agent-acc7877.jsonl                            # Another agent spawned by main
```

### Linkage Fields
| Field | Main Conversation | Agent Conversation |
|-------|-------------------|-------------------|
| Filename | `{sessionId}.jsonl` | `agent-{agentId}.jsonl` |
| `sessionId` | (is the filename) | Points to parent's sessionId |
| `agentId` | N/A | Unique ID (matches filename) |
| `isSidechain` | `false` or absent | `true` |

### Key Facts
- **One file per agent spawn**: Each Task tool invocation creates a new `agent-{agentId}.jsonl`
- **No nesting**: Agents cannot spawn other agents
- **Multiple agents per session**: A main conversation can spawn 50+ agents
- **No direct Task→Agent link**: The Task tool's `tool_use_id` doesn't match `agentId`
- **Correlation by timestamp**: To match a Task call to its agent, compare timestamps

### Example Agent Entry (first line)
```json
{
  "sessionId": "9dbf1107-d255-4d17-a544-aadb594fc786",
  "agentId": "d0e14239",
  "isSidechain": true,
  "type": "assistant",
  "message": {...}
}
```

### Commands for Agent Relationships
```bash
# List main conversation with agent count
ch list                    # Shows: "abc123 [+53 agents] ..."

# List agents for a specific conversation
ch agents abc123           # Lists all agents spawned by abc123

# Show agent with parent link
ch show agent-d0e14239     # Header shows: "Parent: 9dbf1107-..."

# Show main with inline agents (future)
ch show abc123 --expand-agents
```

## Command Examples

```bash
# List recent 50 conversations in current project
ch list

# List all conversations including agents
ch list -a

# List from all projects globally
ch list -g -n 100

# Show specific conversation with thinking
ch show abc123 --thinking

# Search for "docker" in all projects
ch search "docker" -g

# Resume a conversation
ch resume abc123

# List projects
ch projects

# Show stats
ch stats
```
