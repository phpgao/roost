package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestClaudeDecodeDirName(t *testing.T) {
	knownPaths := []string{
		"/home/user/projects/roost",
		"/home/user/.dotfiles",
		"/home/user/projects/middle-back",
		"/home/user/Library/Mobile Documents/iCloud~com~tw93~miaoyan/Documents",
		"/tmp/test",
	}
	s := &ClaudeScanner{dataDir: "/fake", knownPaths: knownPaths}

	tests := []struct {
		name    string
		encoded string
		want    string
	}{
		{
			name:    "typical path",
			encoded: "-home-user-projects-roost",
			want:    "/home/user/projects/roost",
		},
		{
			name:    "dotfile path (dot encoded as dash)",
			encoded: "-home-user--dotfiles",
			want:    "/home/user/.dotfiles",
		},
		{
			name:    "path with hyphen in dir name",
			encoded: "-home-user-projects-middle-back",
			want:    "/home/user/projects/middle-back",
		},
		{
			name:    "short path",
			encoded: "-tmp-test",
			want:    "/tmp/test",
		},
		{
			name:    "unknown path fallback",
			encoded: "-home-user-unknown",
			want:    "/home/user/unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s.decodeDirName(tt.encoded)
			if got != tt.want {
				t.Errorf("decodeDirName(%q) = %q, want %q", tt.encoded, got, tt.want)
			}
		})
	}
}

func TestClaudeParseSession(t *testing.T) {
	dir := t.TempDir()
	sessionFile := filepath.Join(dir, "uuid-123.jsonl")

	// Claude JSONL: type 在顶层，message 含 role/model/content
	content := `{"type":"user","uuid":"u1","timestamp":"2026-05-15T10:00:00+08:00","message":{"role":"user","content":[{"type":"text","text":"帮我写个函数"}]}}
{"type":"assistant","uuid":"u2","timestamp":"2026-05-15T10:00:05+08:00","message":{"role":"assistant","model":"claude-sonnet-4-20250514","content":[{"type":"text","text":"好的"}]}}
{"type":"user","uuid":"u3","timestamp":"2026-05-15T10:01:00+08:00","message":{"role":"user","content":"再改一下"}}
{"type":"assistant","uuid":"u4","timestamp":"2026-05-15T10:01:05+08:00","message":{"role":"assistant","model":"claude-4-opus","content":[{"type":"text","text":"改好了"}]}}
`
	if err := os.WriteFile(sessionFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	s := &ClaudeScanner{dataDir: "/fake"}
	sess, err := s.parseSession(sessionFile, "uuid-123", "-home-user-roost", "/home/user/roost")
	if err != nil {
		t.Fatalf("parseSession error: %v", err)
	}

	t.Run("extracts session ID", func(t *testing.T) {
		if sess.ID != "uuid-123" {
			t.Errorf("ID = %q, want %q", sess.ID, "uuid-123")
		}
	})

	t.Run("title is first user message truncated", func(t *testing.T) {
		if sess.Title != "帮我写个函数" {
			t.Errorf("Title = %q, want %q", sess.Title, "帮我写个函数")
		}
	})

	t.Run("model is last assistant message model", func(t *testing.T) {
		if sess.Model != "claude-4-opus" {
			t.Errorf("Model = %q, want %q", sess.Model, "claude-4-opus")
		}
	})

	t.Run("counts user and assistant messages", func(t *testing.T) {
		if sess.MsgCount != 4 {
			t.Errorf("MsgCount = %d, want %d", sess.MsgCount, 4)
		}
	})

	t.Run("last active from latest timestamp", func(t *testing.T) {
		want, _ := time.Parse(time.RFC3339, "2026-05-15T10:01:05+08:00")
		if !sess.LastActive.Equal(want) {
			t.Errorf("LastActive = %v, want %v", sess.LastActive, want)
		}
	})

	t.Run("platform is Claude", func(t *testing.T) {
		if sess.Platform != PlatformClaude {
			t.Errorf("Platform = %d, want %d", sess.Platform, PlatformClaude)
		}
	})
}

func TestClaudeParseSessionStringContent(t *testing.T) {
	// content 是纯字符串而非数组
	dir := t.TempDir()
	sessionFile := filepath.Join(dir, "s2.jsonl")
	content := `{"type":"user","uuid":"u1","timestamp":"2026-05-15T10:00:00+08:00","message":{"role":"user","content":"plain string content"}}
{"type":"assistant","uuid":"u2","timestamp":"2026-05-15T10:00:01+08:00","message":{"role":"assistant","model":"sonnet","content":"reply"}}
`
	if err := os.WriteFile(sessionFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	s := &ClaudeScanner{dataDir: "/fake"}
	sess, err := s.parseSession(sessionFile, "s2", "proj", "/proj")
	if err != nil {
		t.Fatalf("parseSession error: %v", err)
	}

	if sess.Title != "plain string content" {
		t.Errorf("Title = %q, want %q", sess.Title, "plain string content")
	}
}

func TestClaudeScanProjects(t *testing.T) {
	dir := t.TempDir()
	projDir := filepath.Join(dir, "projects", "-home-user-roost")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}

	content := `{"type":"user","uuid":"u1","timestamp":"2026-05-15T10:00:00+08:00","message":{"role":"user","content":"hello"}}
{"type":"assistant","uuid":"u2","timestamp":"2026-05-15T10:00:01+08:00","message":{"role":"assistant","model":"sonnet","content":"hi"}}
`
	if err := os.WriteFile(filepath.Join(projDir, "sess1.jsonl"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	s := &ClaudeScanner{dataDir: dir}
	projects, err := s.ScanProjects()
	if err != nil {
		t.Fatalf("ScanProjects error: %v", err)
	}

	if len(projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(projects))
	}
	if projects[0].FullPath != "/home/user/roost" {
		t.Errorf("FullPath = %q, want %q", projects[0].FullPath, "/home/user/roost")
	}
	if projects[0].Name != "user/roost" {
		t.Errorf("Name = %q, want %q", projects[0].Name, "user/roost")
	}
}

func TestClaudeScanProjectsNotExist(t *testing.T) {
	s := &ClaudeScanner{dataDir: "/nonexistent/path"}
	projects, err := s.ScanProjects()
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("expected 0 projects, got %d", len(projects))
	}
}

func TestClaudeScanProjectsSkipsInvalidEntries(t *testing.T) {
	dir := t.TempDir()
	projectsDir := filepath.Join(dir, "projects")
	if err := os.MkdirAll(projectsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(projectsDir, "README.txt"), []byte("ignore me"), 0o644); err != nil {
		t.Fatal(err)
	}

	emptyProjDir := filepath.Join(projectsDir, "-home-user-empty")
	if err := os.MkdirAll(emptyProjDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(emptyProjDir, "notes.txt"), []byte("skip non-jsonl"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(emptyProjDir, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}

	validProjDir := filepath.Join(projectsDir, "-home-user-valid")
	if err := os.MkdirAll(validProjDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := `{"type":"user","uuid":"u1","timestamp":"2026-05-15T10:00:00Z","message":{"role":"user","content":"hello"}}
{"type":"assistant","uuid":"u2","timestamp":"2026-05-15T10:00:01Z","message":{"role":"assistant","model":"sonnet","content":"hi"}}
`
	if err := os.WriteFile(filepath.Join(validProjDir, "sess1.jsonl"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	s := &ClaudeScanner{dataDir: dir}
	projects, err := s.ScanProjects()
	if err != nil {
		t.Fatalf("ScanProjects error: %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("expected 1 valid project, got %d", len(projects))
	}
	if projects[0].FullPath != "/home/user/valid" {
		t.Errorf("FullPath = %q, want %q", projects[0].FullPath, "/home/user/valid")
	}
}

func TestClaudeExtractClaudeText(t *testing.T) {
	tests := []struct {
		name string
		raw  json.RawMessage
		want string
	}{
		{name: "nil", raw: nil, want: ""},
		{name: "string", raw: json.RawMessage(`"plain text"`), want: "plain text"},
		{name: "blocks", raw: json.RawMessage(`[{"type":"text","text":"from block"}]`), want: "from block"},
		{name: "blocks skip empty", raw: json.RawMessage(`[{"type":"text","text":""},{"type":"text","text":"next"}]`), want: "next"},
		{name: "invalid", raw: json.RawMessage(`{"unexpected":true}`), want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractClaudeText(tt.raw); got != tt.want {
				t.Errorf("extractClaudeText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestClaudeScanner_DeleteSession(t *testing.T) {
	dir := t.TempDir()
	projDir := filepath.Join(dir, "projects", "encoded-proj")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}

	sessFile := filepath.Join(projDir, "sess-del.jsonl")
	if err := os.WriteFile(sessFile, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	sessDir := filepath.Join(projDir, "sess-del")
	if err := os.MkdirAll(sessDir, 0o755); err != nil {
		t.Fatal(err)
	}

	scanner := &ClaudeScanner{dataDir: dir}
	sess := Session{ID: "sess-del", ProjectDir: "encoded-proj"}
	if err := scanner.DeleteSession(sess); err != nil {
		t.Fatalf("DeleteSession error: %v", err)
	}
	if _, err := os.Stat(sessFile); !os.IsNotExist(err) {
		t.Error("session file should be deleted")
	}
	if _, err := os.Stat(sessDir); !os.IsNotExist(err) {
		t.Error("session directory should be deleted")
	}
}

func TestClaudeScanner_DeleteProject(t *testing.T) {
	dir := t.TempDir()
	projDir := filepath.Join(dir, "projects", "encoded-proj")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}

	scanner := &ClaudeScanner{dataDir: dir}
	proj := Project{
		Sessions: []Session{{ID: "sess-1", ProjectDir: "encoded-proj"}},
	}
	if err := scanner.DeleteProject(proj); err != nil {
		t.Fatalf("DeleteProject error: %v", err)
	}
	if _, err := os.Stat(projDir); !os.IsNotExist(err) {
		t.Error("project directory should be deleted")
	}
}

func TestClaudeScanner_DeleteProjectEmpty(t *testing.T) {
	scanner := &ClaudeScanner{dataDir: "/tmp"}
	proj := Project{Sessions: []Session{}}
	if err := scanner.DeleteProject(proj); err != nil {
		t.Fatalf("DeleteProject with empty sessions should not error, got: %v", err)
	}
}

func TestClaudeScanner_ResumeCmd(t *testing.T) {
	scanner := &ClaudeScanner{bin: "claude"}
	sess := Session{ResumeArg: "sess-xyz"}
	cmd := scanner.ResumeCmd(sess)
	expected := []string{"claude", "--resume", "sess-xyz"}
	if len(cmd) != len(expected) {
		t.Fatalf("ResumeCmd len = %d, want %d", len(cmd), len(expected))
	}
	for i := range expected {
		if cmd[i] != expected[i] {
			t.Errorf("ResumeCmd[%d] = %q, want %q", i, cmd[i], expected[i])
		}
	}
}

func TestClaudeScanner_PlatformAndDataDir(t *testing.T) {
	s := &ClaudeScanner{dataDir: "/home/user/.claude"}
	if s.Platform() != PlatformClaude {
		t.Errorf("Platform() = %v, want PlatformClaude", s.Platform())
	}
	if s.DataDir() != "/home/user/.claude" {
		t.Errorf("DataDir() = %q, want %q", s.DataDir(), "/home/user/.claude")
	}
}
