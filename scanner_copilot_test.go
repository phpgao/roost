package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

func TestCopilotScanner_ScanProjects(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "session-state", "sess-001")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write workspace.yaml
	ws := copilotWorkspace{
		ID:        "sess-001",
		Cwd:       "/Users/test/myproject",
		GitRoot:   "/Users/test/myproject",
		Name:      "Fix bug",
		Branch:    "main",
		HostType:  "local",
		CreatedAt: "2026-05-15T10:00:00Z",
		UpdatedAt: "2026-05-15T10:05:00Z",
	}
	wsData, _ := yaml.Marshal(ws)
	if err := os.WriteFile(filepath.Join(sessionsDir, "workspace.yaml"), wsData, 0o644); err != nil {
		t.Fatal(err)
	}

	// Write events.jsonl with a turn and model change
	events := []map[string]any{
		{
			"type": "turn",
			"data": map[string]any{},
		},
		{
			"type": "session.model_change",
			"data": map[string]any{
				"newModel": "gpt-4o",
			},
		},
		{
			"type": "message",
			"data": map[string]any{},
		},
	}
	f, err := os.Create(filepath.Join(sessionsDir, "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	enc := json.NewEncoder(f)
	for _, evt := range events {
		if e := enc.Encode(evt); e != nil {
			t.Fatal(err)
		}
	}
	f.Close()

	scanner := &CopilotScanner{dataDir: tmpDir, bin: "copilot"}
	projects, err := scanner.ScanProjects()
	if err != nil {
		t.Fatalf("ScanProjects() error: %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(projects))
	}

	p := projects[0]
	if p.FullPath != "/Users/test/myproject" {
		t.Errorf("FullPath = %q, want %q", p.FullPath, "/Users/test/myproject")
	}
	if len(p.Sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(p.Sessions))
	}

	sess := p.Sessions[0]
	if sess.ID != "sess-001" {
		t.Errorf("ID = %q, want %q", sess.ID, "sess-001")
	}
	if sess.Platform != PlatformCopilot {
		t.Errorf("Platform = %v, want PlatformCopilot", sess.Platform)
	}
	if sess.Title != "Fix bug" {
		t.Errorf("Title = %q, want %q", sess.Title, "Fix bug")
	}
	if sess.Model != "gpt-4o" {
		t.Errorf("Model = %q, want %q", sess.Model, "gpt-4o")
	}
	if sess.MsgCount != 2 {
		t.Errorf("MsgCount = %d, want 2", sess.MsgCount)
	}
	expectedTime, _ := time.Parse(time.RFC3339, "2026-05-15T10:05:00Z")
	if !sess.LastActive.Equal(expectedTime) {
		t.Errorf("LastActive = %v, want %v", sess.LastActive, expectedTime)
	}
	if sess.ResumeArg != "sess-001" {
		t.Errorf("ResumeArg = %q, want %q", sess.ResumeArg, "sess-001")
	}
}

func TestCopilotScanner_MultipleProjects(t *testing.T) {
	tmpDir := t.TempDir()

	// Create two sessions in different projects
	for _, id := range []string{"sess-a", "sess-b"} {
		sessionsDir := filepath.Join(tmpDir, "session-state", id)
		if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
			t.Fatal(err)
		}
		cwd := "/Users/test/projA"
		if id == "sess-b" {
			cwd = "/Users/test/projB"
		}
		ws := copilotWorkspace{
			ID:        id,
			Cwd:       cwd,
			Name:      "session " + id,
			CreatedAt: "2026-05-15T10:00:00Z",
			UpdatedAt: "2026-05-15T10:00:00Z",
		}
		wsData, _ := yaml.Marshal(ws)
		if err := os.WriteFile(filepath.Join(sessionsDir, "workspace.yaml"), wsData, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	scanner := &CopilotScanner{dataDir: tmpDir, bin: "copilot"}
	projects, err := scanner.ScanProjects()
	if err != nil {
		t.Fatalf("ScanProjects() error: %v", err)
	}
	if len(projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(projects))
	}
}

func TestCopilotScanner_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	scanner := &CopilotScanner{dataDir: tmpDir, bin: "copilot"}

	projects, err := scanner.ScanProjects()
	if err != nil {
		t.Fatalf("ScanProjects() error: %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("expected 0 projects, got %d", len(projects))
	}
}

func TestCopilotScanner_NoSessionStateDir(t *testing.T) {
	tmpDir := t.TempDir()
	// No session-state directory at all
	scanner := &CopilotScanner{dataDir: tmpDir, bin: "copilot"}

	projects, err := scanner.ScanProjects()
	if err != nil {
		t.Fatalf("ScanProjects() error: %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("expected 0 projects, got %d", len(projects))
	}
}

func TestCopilotScanner_SkipNoWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a session dir without workspace.yaml
	sessionsDir := filepath.Join(tmpDir, "session-state", "orphan")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	scanner := &CopilotScanner{dataDir: tmpDir, bin: "copilot"}
	projects, err := scanner.ScanProjects()
	if err != nil {
		t.Fatalf("ScanProjects() error: %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("expected 0 projects for session without workspace, got %d", len(projects))
	}
}

func TestCopilotScanner_SkipEmptyCwd(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "session-state", "no-cwd")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	ws := copilotWorkspace{ID: "no-cwd", Cwd: ""}
	wsData, _ := yaml.Marshal(ws)
	if err := os.WriteFile(filepath.Join(sessionsDir, "workspace.yaml"), wsData, 0o644); err != nil {
		t.Fatal(err)
	}

	scanner := &CopilotScanner{dataDir: tmpDir, bin: "copilot"}
	projects, err := scanner.ScanProjects()
	if err != nil {
		t.Fatalf("ScanProjects() error: %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("expected 0 projects for empty cwd, got %d", len(projects))
	}
}

func TestCopilotScanner_UntitledWhenNoName(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "session-state", "unnamed")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	ws := copilotWorkspace{
		ID:        "unnamed",
		Cwd:       "/project/test",
		Name:      "", // no name
		CreatedAt: "2026-05-15T10:00:00Z",
	}
	wsData, _ := yaml.Marshal(ws)
	if err := os.WriteFile(filepath.Join(sessionsDir, "workspace.yaml"), wsData, 0o644); err != nil {
		t.Fatal(err)
	}

	scanner := &CopilotScanner{dataDir: tmpDir, bin: "copilot"}
	projects, _ := scanner.ScanProjects()
	if len(projects) != 1 {
		t.Fatal("expected 1 project")
	}
	if projects[0].Sessions[0].Title != "(untitled)" {
		t.Errorf("Title = %q, want %q", projects[0].Sessions[0].Title, "(untitled)")
	}
}

func TestCopilotScanner_DeleteSession(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "session-state", "to-delete")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	scanner := &CopilotScanner{dataDir: tmpDir, bin: "copilot"}
	sess := Session{FilePath: sessionsDir}
	if err := scanner.DeleteSession(sess); err != nil {
		t.Fatalf("DeleteSession error: %v", err)
	}
	if _, err := os.Stat(sessionsDir); !os.IsNotExist(err) {
		t.Error("session directory should be deleted")
	}
}

func TestCopilotScanner_DeleteProject(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "session-state", "sess-del")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	scanner := &CopilotScanner{dataDir: tmpDir, bin: "copilot"}
	proj := Project{
		Sessions: []Session{{FilePath: sessionsDir}},
	}
	if err := scanner.DeleteProject(proj); err != nil {
		t.Fatalf("DeleteProject error: %v", err)
	}
	if _, err := os.Stat(sessionsDir); !os.IsNotExist(err) {
		t.Error("session directory should be deleted")
	}
}

func TestCopilotScanner_ResumeCmd(t *testing.T) {
	scanner := &CopilotScanner{bin: "copilot"}
	sess := Session{ResumeArg: "sess-xyz"}
	cmd := scanner.ResumeCmd(sess)
	expected := []string{"copilot", "--resume=sess-xyz"}
	if len(cmd) != len(expected) {
		t.Fatalf("ResumeCmd len = %d, want %d", len(cmd), len(expected))
	}
	for i := range expected {
		if cmd[i] != expected[i] {
			t.Errorf("ResumeCmd[%d] = %q, want %q", i, cmd[i], expected[i])
		}
	}
}

func TestCopilotScanner_PlatformAndDataDir(t *testing.T) {
	s := &CopilotScanner{dataDir: "/home/user/.copilot", bin: "copilot"}
	if s.Platform() != PlatformCopilot {
		t.Errorf("Platform() = %v, want PlatformCopilot", s.Platform())
	}
	if s.DataDir() != "/home/user/.copilot" {
		t.Errorf("DataDir() = %q, want %q", s.DataDir(), "/home/user/.copilot")
	}
}

func TestCopilotScanner_ParseEventsNoFile(t *testing.T) {
	s := &CopilotScanner{dataDir: "/tmp", bin: "copilot"}
	msgCount, model := s.parseEvents("/nonexistent/events.jsonl")
	if msgCount != 0 {
		t.Errorf("msgCount = %d, want 0 for nonexistent file", msgCount)
	}
	if model != "" {
		t.Errorf("model = %q, want empty for nonexistent file", model)
	}
}
