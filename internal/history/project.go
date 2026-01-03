package history

import (
	"os"
	"path/filepath"
	"sort"
)

// Project represents a Claude Code project.
type Project struct {
	Name              string // Encoded directory name
	Path              string // Decoded filesystem path
	Dir               string // Full path to the project directory
	ConversationCount int    // Number of conversation files
	AgentCount        int    // Number of agent files
	TotalSize         int64  // Total size of all files
}

// ListProjects lists all Claude Code projects.
func ListProjects(projectsDir string) ([]*Project, error) {
	if projectsDir == "" {
		projectsDir = DefaultProjectsDir()
	}

	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var projects []*Project
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		projectDir := filepath.Join(projectsDir, entry.Name())
		project := &Project{
			Name: entry.Name(),
			Path: DecodeProjectPath(entry.Name()),
			Dir:  projectDir,
		}

		// Count files and calculate size
		files, err := os.ReadDir(projectDir)
		if err != nil {
			continue
		}

		for _, f := range files {
			if f.IsDir() || !IsConversationFile(f.Name()) {
				continue
			}
			if IsAgentFile(f.Name()) {
				project.AgentCount++
			} else {
				project.ConversationCount++
			}
			if info, err := f.Info(); err == nil {
				project.TotalSize += info.Size()
			}
		}

		if project.ConversationCount > 0 || project.AgentCount > 0 {
			projects = append(projects, project)
		}
	}

	// Sort by path
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Path < projects[j].Path
	})

	return projects, nil
}

// FindProject finds a project by its path.
func FindProject(projectsDir, path string) (*Project, error) {
	projects, err := ListProjects(projectsDir)
	if err != nil {
		return nil, err
	}

	for _, p := range projects {
		if p.Path == path || p.Name == EncodeProjectPath(path) {
			return p, nil
		}
	}

	return nil, nil
}

// GetCurrentProject returns the project for the current working directory.
func GetCurrentProject() (*Project, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	return FindProject(DefaultProjectsDir(), cwd)
}

// ProjectStats contains aggregate statistics for a project.
type ProjectStats struct {
	Project           *Project
	ConversationCount int
	AgentCount        int
	MessageCount      int
	TotalSize         int64
	OldestTimestamp   string
	NewestTimestamp   string
}

// GetProjectStats calculates statistics for a project.
func GetProjectStats(project *Project) (*ProjectStats, error) {
	stats := &ProjectStats{
		Project:           project,
		ConversationCount: project.ConversationCount,
		AgentCount:        project.AgentCount,
		TotalSize:         project.TotalSize,
	}

	scanner := NewScanner(ScannerOptions{
		ProjectsDir:   filepath.Dir(project.Dir),
		ProjectPath:   project.Path,
		IncludeAgents: true,
	})

	metas, err := scanner.ScanAll()
	if err != nil {
		return stats, nil
	}

	for _, meta := range metas {
		stats.MessageCount += meta.MessageCount
		if stats.OldestTimestamp == "" || meta.Timestamp.Format("2006-01-02") < stats.OldestTimestamp {
			stats.OldestTimestamp = meta.Timestamp.Format("2006-01-02")
		}
		if stats.NewestTimestamp == "" || meta.Timestamp.Format("2006-01-02") > stats.NewestTimestamp {
			stats.NewestTimestamp = meta.Timestamp.Format("2006-01-02")
		}
	}

	return stats, nil
}
