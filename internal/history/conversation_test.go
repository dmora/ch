package history

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanConversationMeta(t *testing.T) {
	// Find a real conversation file to test with
	projectsDir := DefaultProjectsDir()
	projects, err := ListProjects(projectsDir)
	if err != nil {
		t.Skipf("No projects found: %v", err)
	}

	if len(projects) == 0 {
		t.Skip("No projects available")
	}

	// Find a conversation file
	var testFile string
	for _, p := range projects {
		entries, err := os.ReadDir(p.Dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() && !IsAgentFile(e.Name()) && IsConversationFile(e.Name()) {
				testFile = filepath.Join(p.Dir, e.Name())
				break
			}
		}
		if testFile != "" {
			break
		}
	}

	if testFile == "" {
		t.Skip("No conversation file found")
	}

	t.Logf("Testing with file: %s", testFile)

	meta, err := ScanConversationMeta(testFile)
	if err != nil {
		t.Fatalf("ScanConversationMeta failed: %v", err)
	}

	t.Logf("ID: %s", meta.ID)
	t.Logf("SessionID: %s", meta.SessionID)
	t.Logf("MessageCount: %d", meta.MessageCount)
	t.Logf("Preview: '%s'", meta.Preview)
	t.Logf("Model: %s", meta.Model)
	t.Logf("Timestamp: %s", meta.Timestamp)

	if meta.MessageCount == 0 {
		t.Error("Expected MessageCount > 0")
	}
}
