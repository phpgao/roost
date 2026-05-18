package main

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

// createTestOpenCodeDB creates a minimal OpenCode SQLite database for testing.
func createTestOpenCodeDB(t *testing.T, dbPath string) {
	t.Helper()
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// Create tables
	_, err = db.Exec(`
		CREATE TABLE project (
			id TEXT PRIMARY KEY,
			worktree TEXT NOT NULL,
			vcs TEXT,
			name TEXT,
			icon_url TEXT,
			icon_color TEXT,
			time_created INTEGER NOT NULL,
			time_updated INTEGER NOT NULL,
			time_initialized INTEGER,
			sandboxes TEXT NOT NULL DEFAULT ''
		);
		CREATE TABLE session (
			id TEXT PRIMARY KEY,
			project_id TEXT NOT NULL,
			parent_id TEXT,
			slug TEXT NOT NULL,
			directory TEXT NOT NULL,
			title TEXT NOT NULL,
			version TEXT NOT NULL,
			share_url TEXT,
			summary_additions INTEGER,
			summary_deletions INTEGER,
			summary_files INTEGER,
			summary_diffs TEXT,
			revert TEXT,
			permission TEXT,
			time_created INTEGER NOT NULL,
			time_updated INTEGER NOT NULL,
			time_compacting INTEGER,
			time_archived INTEGER,
			workspace_id TEXT,
			path TEXT,
			agent TEXT,
			model TEXT,
			cost REAL DEFAULT 0 NOT NULL,
			tokens_input INTEGER DEFAULT 0 NOT NULL,
			tokens_output INTEGER DEFAULT 0 NOT NULL,
			tokens_reasoning INTEGER DEFAULT 0 NOT NULL,
			tokens_cache_read INTEGER DEFAULT 0 NOT NULL,
			tokens_cache_write INTEGER DEFAULT 0 NOT NULL,
			FOREIGN KEY (project_id) REFERENCES project(id) ON DELETE CASCADE
		);
		CREATE TABLE session_message (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			type TEXT NOT NULL,
			time_created INTEGER NOT NULL,
			time_updated INTEGER NOT NULL,
			data TEXT NOT NULL,
			FOREIGN KEY (session_id) REFERENCES session(id) ON DELETE CASCADE
		);
	`)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now().Unix()

	// Insert test project
	_, err = db.Exec(`
		INSERT INTO project (id, worktree, vcs, name, time_created, time_updated, sandboxes)
		VALUES (?, ?, 'git', ?, ?, ?, '')
	`, "proj-1", "/Users/test/myproject", "myproject", now, now)
	if err != nil {
		t.Fatal(err)
	}

	// Insert test sessions
	_, err = db.Exec(`
		INSERT INTO session (id, project_id, slug, directory, title, version, model, agent, time_created, time_updated)
		VALUES (?, ?, 'test-slug', ?, ?, '1.15.3', ?, 'build', ?, ?)
	`,
		"ses-001", "proj-1", "/Users/test/myproject", "Hello World",
		`{"id":"big-pickle","providerID":"opencode"}`,
		now-100, now)
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Exec(`
		INSERT INTO session (id, project_id, slug, directory, title, version, model, agent, time_created, time_updated)
		VALUES (?, ?, 'test-slug-2', ?, ?, '1.15.3', ?, 'build', ?, ?)
	`,
		"ses-002", "proj-1", "/Users/test/myproject", "Second Session",
		`{"id":"small-pickle","providerID":"opencode"}`,
		now-200, now-50)
	if err != nil {
		t.Fatal(err)
	}

	// Insert messages for ses-001
	for i := 0; i < 3; i++ {
		_, err = db.Exec(`
			INSERT INTO session_message (id, session_id, type, time_created, time_updated, data)
			VALUES (?, ?, 'user', ?, ?, '{}')
		`, fmt.Sprintf("msg-%d", i), "ses-001", now-int64(i), now-int64(i))
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestOpenCodeScanner_ScanProjects(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "opencode.db")
	createTestOpenCodeDB(t, dbPath)

	scanner := &OpenCodeScanner{bin: "opencode", dbPath: dbPath}
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
	if len(p.Sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(p.Sessions))
	}

	// First session should be the most recently updated
	sess := p.Sessions[0]
	if sess.ID != "ses-001" {
		t.Errorf("first session ID = %q, want %q", sess.ID, "ses-001")
	}
	if sess.Platform != PlatformOpenCode {
		t.Errorf("Platform = %v, want PlatformOpenCode", sess.Platform)
	}
	if sess.Title != "Hello World" {
		t.Errorf("Title = %q, want %q", sess.Title, "Hello World")
	}
	if sess.Model != "big-pickle" {
		t.Errorf("Model = %q, want %q", sess.Model, "big-pickle")
	}
	if sess.MsgCount != 3 {
		t.Errorf("MsgCount = %d, want 3", sess.MsgCount)
	}
	if sess.ResumeArg != "ses-001" {
		t.Errorf("ResumeArg = %q, want %q", sess.ResumeArg, "ses-001")
	}
	if sess.ProjectPath != "/Users/test/myproject" {
		t.Errorf("ProjectPath = %q, want %q", sess.ProjectPath, "/Users/test/myproject")
	}
}

func TestOpenCodeScanner_EmptyDBPath(t *testing.T) {
	scanner := &OpenCodeScanner{bin: "opencode", dbPath: ""}
	projects, err := scanner.ScanProjects()
	if err != nil {
		t.Fatalf("ScanProjects() error: %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("expected 0 projects for empty dbPath, got %d", len(projects))
	}
}

func TestOpenCodeScanner_DBNotExist(t *testing.T) {
	scanner := &OpenCodeScanner{bin: "opencode", dbPath: "/nonexistent/opencode.db"}
	projects, err := scanner.ScanProjects()
	if err != nil {
		t.Fatalf("ScanProjects() error: %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("expected 0 projects for nonexistent DB, got %d", len(projects))
	}
}

func TestOpenCodeScanner_DeleteSession(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "opencode.db")
	createTestOpenCodeDB(t, dbPath)

	scanner := &OpenCodeScanner{bin: "opencode", dbPath: dbPath}

	sess := Session{ID: "ses-001"}
	if err := scanner.DeleteSession(sess); err != nil {
		t.Fatalf("DeleteSession error: %v", err)
	}

	// Verify session and messages are deleted
	projects, _ := scanner.ScanProjects()
	if len(projects) != 1 {
		t.Fatalf("expected 1 project after delete, got %d", len(projects))
	}
	for _, s := range projects[0].Sessions {
		if s.ID == "ses-001" {
			t.Error("session ses-001 should be deleted")
		}
	}
}

func TestOpenCodeScanner_DeleteProject(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "opencode.db")
	createTestOpenCodeDB(t, dbPath)

	scanner := &OpenCodeScanner{bin: "opencode", dbPath: dbPath}

	proj := Project{
		Name:     "myproject",
		FullPath: "/Users/test/myproject",
		Sessions: []Session{
			{ID: "ses-001"},
			{ID: "ses-002"},
		},
	}
	if err := scanner.DeleteProject(proj); err != nil {
		t.Fatalf("DeleteProject error: %v", err)
	}

	// Verify all sessions are deleted
	projects, _ := scanner.ScanProjects()
	if len(projects) != 0 {
		t.Errorf("expected 0 projects after deleting all sessions, got %d", len(projects))
	}
}

func TestOpenCodeScanner_ResumeCmd(t *testing.T) {
	scanner := &OpenCodeScanner{bin: "opencode", dbPath: "/tmp/test.db"}
	sess := Session{ResumeArg: "ses-abc123"}
	cmd := scanner.ResumeCmd(sess)
	expected := []string{"opencode", "-s", "ses-abc123"}
	if len(cmd) != len(expected) {
		t.Fatalf("ResumeCmd len = %d, want %d", len(cmd), len(expected))
	}
	for i := range expected {
		if cmd[i] != expected[i] {
			t.Errorf("ResumeCmd[%d] = %q, want %q", i, cmd[i], expected[i])
		}
	}
}

func TestOpenCodeScanner_DataDir(t *testing.T) {
	scanner := &OpenCodeScanner{bin: "opencode", dbPath: "/home/user/.local/share/opencode/opencode.db"}
	if scanner.DataDir() != "/home/user/.local/share/opencode" {
		t.Errorf("DataDir = %q, want %q", scanner.DataDir(), "/home/user/.local/share/opencode")
	}
}

func TestOpenCodeScanner_DataDirEmpty(t *testing.T) {
	scanner := &OpenCodeScanner{bin: "opencode", dbPath: ""}
	if scanner.DataDir() != "" {
		t.Errorf("DataDir = %q, want empty string", scanner.DataDir())
	}
}

func TestParseOpenCodeModel(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`{"id":"big-pickle","providerID":"opencode"}`, "big-pickle"},
		{`{"id":"gpt-4o","providerID":"openai"}`, "gpt-4o"},
		{`invalid json`, "invalid json"},
		{`{}`, `{}`},
	}
	for _, tt := range tests {
		got := parseOpenCodeModel(tt.input)
		if got != tt.want {
			t.Errorf("parseOpenCodeModel(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
