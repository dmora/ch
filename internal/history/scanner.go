package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/dmora/ch/internal/jsonl"
)

// ScannerOptions configures the conversation scanner.
type ScannerOptions struct {
	ProjectsDir    string // Base projects directory (default: ~/.claude/projects)
	ProjectPath    string // Filter to specific project path (empty = all)
	IncludeAgents  bool   // Include agent conversations
	Limit          int    // Maximum number of results (0 = no limit)
	Workers        int    // Number of parallel workers (default: 4)
	SortByTime     bool   // Sort by timestamp (newest first)
}

// DefaultScannerOptions returns default scanner options.
func DefaultScannerOptions() ScannerOptions {
	return ScannerOptions{
		ProjectsDir: DefaultProjectsDir(),
		Workers:     4,
		SortByTime:  true,
	}
}

// Scanner scans conversation files efficiently.
type Scanner struct {
	opts ScannerOptions
}

// NewScanner creates a new conversation scanner.
func NewScanner(opts ScannerOptions) *Scanner {
	if opts.ProjectsDir == "" {
		opts.ProjectsDir = DefaultProjectsDir()
	}
	if opts.Workers <= 0 {
		opts.Workers = 4
	}
	return &Scanner{opts: opts}
}

// ScanAll scans all conversations matching the options.
func (s *Scanner) ScanAll() ([]*ConversationMeta, error) {
	files, err := s.findFiles()
	if err != nil {
		return nil, err
	}

	results := s.scanFiles(files)

	// Sort by timestamp if requested
	if s.opts.SortByTime {
		sort.Slice(results, func(i, j int) bool {
			return results[i].Timestamp.After(results[j].Timestamp)
		})
	}

	// Apply limit
	if s.opts.Limit > 0 && len(results) > s.opts.Limit {
		results = results[:s.opts.Limit]
	}

	return results, nil
}

// ScanProject scans conversations for a specific project path.
func (s *Scanner) ScanProject(projectPath string) ([]*ConversationMeta, error) {
	s.opts.ProjectPath = projectPath
	return s.ScanAll()
}

// findFiles finds all conversation files matching the options.
func (s *Scanner) findFiles() ([]string, error) {
	var files []string

	if s.opts.ProjectPath != "" {
		// Scan specific project
		projectDir := GetProjectDir(s.opts.ProjectsDir, s.opts.ProjectPath)
		return s.scanDir(projectDir)
	}

	// Scan all projects
	entries, err := os.ReadDir(s.opts.ProjectsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		projectDir := filepath.Join(s.opts.ProjectsDir, entry.Name())
		projectFiles, err := s.scanDir(projectDir)
		if err != nil {
			continue // Skip directories we can't read
		}
		files = append(files, projectFiles...)
	}

	return files, nil
}

// scanDir scans a single directory for conversation files.
func (s *Scanner) scanDir(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var files []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if !IsConversationFile(entry.Name()) {
			continue
		}
		if !s.opts.IncludeAgents && IsAgentFile(entry.Name()) {
			continue
		}
		files = append(files, filepath.Join(dir, entry.Name()))
	}

	return files, nil
}

// scanFiles scans multiple files in parallel.
func (s *Scanner) scanFiles(files []string) []*ConversationMeta {
	if len(files) == 0 {
		return nil
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	results := make([]*ConversationMeta, 0, len(files))

	// Create worker pool
	fileChan := make(chan string, len(files))
	for _, f := range files {
		fileChan <- f
	}
	close(fileChan)

	for i := 0; i < s.opts.Workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range fileChan {
				meta, err := ScanConversationMeta(path)
				if err != nil {
					continue // Skip files we can't parse
				}
				mu.Lock()
				results = append(results, meta)
				mu.Unlock()
			}
		}()
	}

	wg.Wait()
	return results
}

// CountAgents counts the number of agent files for a given session ID.
func (s *Scanner) CountAgents(projectDir, sessionID string) int {
	entries, err := os.ReadDir(projectDir)
	if err != nil {
		return 0
	}

	count := 0
	for _, entry := range entries {
		if entry.IsDir() || !IsAgentFile(entry.Name()) {
			continue
		}
		// Check if this agent belongs to the session
		path := filepath.Join(projectDir, entry.Name())
		meta, err := ScanConversationMeta(path)
		if err != nil {
			continue
		}
		if meta.ParentSessionID == sessionID {
			count++
		}
	}
	return count
}

// FindAgents finds all agent conversations for a given session ID.
func (s *Scanner) FindAgents(projectDir, sessionID string) ([]*ConversationMeta, error) {
	entries, err := os.ReadDir(projectDir)
	if err != nil {
		return nil, err
	}

	var agents []*ConversationMeta
	for _, entry := range entries {
		if entry.IsDir() || !IsAgentFile(entry.Name()) {
			continue
		}
		path := filepath.Join(projectDir, entry.Name())
		meta, err := ScanConversationMeta(path)
		if err != nil {
			continue
		}
		if meta.ParentSessionID == sessionID {
			agents = append(agents, meta)
		}
	}

	// Sort by timestamp
	sort.Slice(agents, func(i, j int) bool {
		return agents[i].Timestamp.Before(agents[j].Timestamp)
	})

	return agents, nil
}

// AgentInfo contains detailed information about an agent extracted from parent conversation.
type AgentInfo struct {
	AgentID     string // Agent ID (from agent file)
	SubagentType string // Type of agent (e.g., "Explore", "Plan")
	Prompt      string // The prompt passed to the Task tool
	Description string // Short description from Task tool
}

// ExtractAgentInfo extracts agent type and prompt from a parent conversation.
// It finds the Task tool_use block that spawned the given agent.
func ExtractAgentInfo(parentPath, agentID string) (*AgentInfo, error) {
	conv, err := LoadConversation(parentPath)
	if err != nil {
		return nil, err
	}

	// Normalize agent ID (remove "agent-" prefix if present)
	normalizedID := strings.TrimPrefix(agentID, "agent-")

	// Search through all entries for Task tool calls
	for _, entry := range conv.Entries {
		if entry.Type != jsonl.EntryTypeAssistant || entry.Message == nil {
			continue
		}

		msg, err := jsonl.ParseMessage(entry)
		if err != nil || msg == nil {
			continue
		}

		// Look for Task tool calls
		for _, block := range msg.Content {
			if block.Type != jsonl.BlockTypeToolUse || block.Name != "Task" {
				continue
			}

			// Parse the input to check if this is our agent
			if block.Input == nil {
				continue
			}

			var input map[string]interface{}
			if err := json.Unmarshal(block.Input, &input); err != nil {
				continue
			}

			// Check if this Task spawned our agent by looking at the tool ID
			// The tool ID often matches or contains the agent ID
			if block.ID != "" && strings.Contains(block.ID, normalizedID) {
				info := &AgentInfo{AgentID: agentID}
				if st, ok := input["subagent_type"].(string); ok {
					info.SubagentType = st
				}
				if p, ok := input["prompt"].(string); ok {
					info.Prompt = p
				}
				if d, ok := input["description"].(string); ok {
					info.Description = d
				}
				return info, nil
			}
		}
	}

	// If we didn't find a direct ID match, try matching by subagent_type pattern
	// Sometimes the agent ID and tool ID don't match directly
	return nil, nil
}

// FindAgentsWithType finds agents matching a specific subagent_type.
func (s *Scanner) FindAgentsWithType(projectDir, sessionID, agentType string) ([]*ConversationMeta, error) {
	agents, err := s.FindAgents(projectDir, sessionID)
	if err != nil {
		return nil, err
	}

	if agentType == "" {
		return agents, nil
	}

	// Find parent conversation path
	parentPath := filepath.Join(projectDir, sessionID+".jsonl")

	// Filter agents by type
	var filtered []*ConversationMeta
	for _, agent := range agents {
		info, err := ExtractAgentInfo(parentPath, agent.ID)
		if err != nil {
			continue
		}
		if info != nil && info.SubagentType == agentType {
			filtered = append(filtered, agent)
		}
	}

	return filtered, nil
}

// GetAgentTypes returns all unique agent types for a conversation's agents.
func (s *Scanner) GetAgentTypes(projectDir, sessionID string) ([]string, error) {
	agents, err := s.FindAgents(projectDir, sessionID)
	if err != nil {
		return nil, err
	}

	parentPath := filepath.Join(projectDir, sessionID+".jsonl")
	typeSet := make(map[string]bool)

	for _, agent := range agents {
		info, err := ExtractAgentInfo(parentPath, agent.ID)
		if err != nil || info == nil {
			continue
		}
		if info.SubagentType != "" {
			typeSet[info.SubagentType] = true
		}
	}

	var types []string
	for t := range typeSet {
		types = append(types, t)
	}
	sort.Strings(types)
	return types, nil
}
