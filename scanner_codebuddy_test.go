package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCodeBuddyDecodeDirName(t *testing.T) {
	trustedDirs := []string{
		"/home/user/projects/middle-back",
		"/home/user/projects/middle-back-configs",
		"/home/user/projects/roost",
		"/tmp/test",
		"/opt/homebrew/etc/nginx",
	}
	s := &CodeBuddyScanner{dataDir: "/fake", trustedDirs: trustedDirs}

	tests := []struct {
		name    string
		encoded string
		want    string
	}{
		{
			name:    "path with hyphen in directory name",
			encoded: "home-user-projects-middle-back",
			want:    "/home/user/projects/middle-back",
		},
		{
			name:    "longer path with hyphen takes longest match",
			encoded: "home-user-projects-middle-back-configs",
			want:    "/home/user/projects/middle-back-configs",
		},
		{
			name:    "simple path without ambiguity",
			encoded: "home-user-projects-roost",
			want:    "/home/user/projects/roost",
		},
		{
			name:    "short path",
			encoded: "tmp-test",
			want:    "/tmp/test",
		},
		{
			name:    "path with multiple hyphens",
			encoded: "opt-homebrew-etc-nginx",
			want:    "/opt/homebrew/etc/nginx",
		},
		{
			name:    "unknown path fallback to simple replace",
			encoded: "home-user-unknown-project",
			want:    "/home/user/unknown/project",
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

func TestCodeBuddyParseSession(t *testing.T) {
	// 准备 testdata：一个最小的 CodeBuddy JSONL session 文件
	dir := t.TempDir()
	sessionFile := filepath.Join(dir, "abc123.jsonl")

	content := `{"type":"message","role":"user","timestamp":1715760000000,"content":"hello"}
{"type":"ai-title","aiTitle":"Test Session Title","timestamp":1715760001000}
{"type":"message","role":"assistant","timestamp":1715760002000,"providerData":{"model":"claude-sonnet-4-20250514","agent":"cli"},"content":"hi there"}
{"type":"message","role":"user","timestamp":1715760003000,"content":"bye"}
{"type":"message","role":"assistant","timestamp":1715760004000,"providerData":{"model":"claude-sonnet-4-20250514","agent":"Explore"},"content":"goodbye"}
`
	if err := os.WriteFile(sessionFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	s := &CodeBuddyScanner{dataDir: "/fake"}
	sess, err := s.parseSession(sessionFile, "abc123", "home_user_roost", "/home/user/roost")
	if err != nil {
		t.Fatalf("parseSession error: %v", err)
	}

	t.Run("extracts session ID", func(t *testing.T) {
		if sess.ID != "abc123" {
			t.Errorf("ID = %q, want %q", sess.ID, "abc123")
		}
	})

	t.Run("extracts title from ai-title line", func(t *testing.T) {
		if sess.Title != "Test Session Title" {
			t.Errorf("Title = %q, want %q", sess.Title, "Test Session Title")
		}
	})

	t.Run("extracts model from last assistant message", func(t *testing.T) {
		if sess.Model != "claude-sonnet-4-20250514" {
			t.Errorf("Model = %q, want %q", sess.Model, "claude-sonnet-4-20250514")
		}
	})

	t.Run("extracts agent type from last assistant message", func(t *testing.T) {
		if sess.AgentType != "Explore" {
			t.Errorf("AgentType = %q, want %q", sess.AgentType, "Explore")
		}
	})

	t.Run("counts user and assistant messages", func(t *testing.T) {
		if sess.MsgCount != 4 {
			t.Errorf("MsgCount = %d, want %d", sess.MsgCount, 4)
		}
	})

	t.Run("extracts last active time from timestamp", func(t *testing.T) {
		want := time.UnixMilli(1715760004000)
		if !sess.LastActive.Equal(want) {
			t.Errorf("LastActive = %v, want %v", sess.LastActive, want)
		}
	})

	t.Run("sets platform to CodeBuddy", func(t *testing.T) {
		if sess.Platform != PlatformCodeBuddy {
			t.Errorf("Platform = %d, want %d", sess.Platform, PlatformCodeBuddy)
		}
	})
}

func TestCodeBuddyParseSessionUntitled(t *testing.T) {
	dir := t.TempDir()
	sessionFile := filepath.Join(dir, "no-title.jsonl")

	// 没有 ai-title 行的 session
	content := `{"type":"message","role":"user","timestamp":1715760000000,"content":"hello"}
{"type":"message","role":"assistant","timestamp":1715760001000,"providerData":{"model":"gpt-4o","agent":"cli"},"content":"hi"}
`
	if err := os.WriteFile(sessionFile, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	s := &CodeBuddyScanner{dataDir: "/fake"}
	sess, err := s.parseSession(sessionFile, "no-title", "proj", "/proj")
	if err != nil {
		t.Fatalf("parseSession error: %v", err)
	}

	if sess.Title != "(untitled)" {
		t.Errorf("Title = %q, want %q", sess.Title, "(untitled)")
	}
}

func TestCodeBuddyScanProjects(t *testing.T) {
	// 创建模拟 CodeBuddy 目录结构
	dir := t.TempDir()
	projDir := filepath.Join(dir, "projects", "home-user-roost")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}

	sessionContent := `{"type":"message","role":"user","timestamp":1715760000000,"content":"test"}
{"type":"ai-title","aiTitle":"My Session","timestamp":1715760001000}
{"type":"message","role":"assistant","timestamp":1715760002000,"providerData":{"model":"sonnet","agent":"cli"},"content":"ok"}
`
	if err := os.WriteFile(filepath.Join(projDir, "sess1.jsonl"), []byte(sessionContent), 0o644); err != nil {
		t.Fatal(err)
	}

	s := &CodeBuddyScanner{dataDir: dir}
	projects, err := s.ScanProjects()
	if err != nil {
		t.Fatalf("ScanProjects error: %v", err)
	}

	if len(projects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(projects))
	}

	p := projects[0]
	if p.FullPath != "/home/user/roost" {
		t.Errorf("FullPath = %q, want %q", p.FullPath, "/home/user/roost")
	}
	if p.Name != "user/roost" {
		t.Errorf("Name = %q, want %q", p.Name, "user/roost")
	}
	if len(p.Sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(p.Sessions))
	}
	if p.Sessions[0].Title != "My Session" {
		t.Errorf("session Title = %q, want %q", p.Sessions[0].Title, "My Session")
	}
}

func TestCodeBuddyScanProjectsNotExist(t *testing.T) {
	s := &CodeBuddyScanner{dataDir: "/nonexistent/path"}
	projects, err := s.ScanProjects()
	if err != nil {
		t.Fatalf("expected nil error for non-existent dir, got: %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("expected 0 projects, got %d", len(projects))
	}
}

func TestCodeBuddyScanner_DeleteSession(t *testing.T) {
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
	taskDir := filepath.Join(dir, "tasks", "sess-del")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatal(err)
	}
	fileHistDir := filepath.Join(dir, "file-history", "sess-del")
	if err := os.MkdirAll(fileHistDir, 0o755); err != nil {
		t.Fatal(err)
	}

	scanner := &CodeBuddyScanner{dataDir: dir}
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
	if _, err := os.Stat(taskDir); !os.IsNotExist(err) {
		t.Error("task directory should be deleted")
	}
	if _, err := os.Stat(fileHistDir); !os.IsNotExist(err) {
		t.Error("file-history directory should be deleted")
	}
}

func TestCodeBuddyScanner_DeleteProject(t *testing.T) {
	dir := t.TempDir()
	projDir := filepath.Join(dir, "projects", "encoded-proj")
	if err := os.MkdirAll(projDir, 0o755); err != nil {
		t.Fatal(err)
	}
	taskDir := filepath.Join(dir, "tasks", "sess-1")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatal(err)
	}

	scanner := &CodeBuddyScanner{dataDir: dir}
	proj := Project{
		Sessions: []Session{{ID: "sess-1", ProjectDir: "encoded-proj"}},
	}
	if err := scanner.DeleteProject(proj); err != nil {
		t.Fatalf("DeleteProject error: %v", err)
	}
	if _, err := os.Stat(projDir); !os.IsNotExist(err) {
		t.Error("project directory should be deleted")
	}
	if _, err := os.Stat(taskDir); !os.IsNotExist(err) {
		t.Error("task directory should be deleted")
	}
}

func TestCodeBuddyScanner_DeleteProjectEmpty(t *testing.T) {
	scanner := &CodeBuddyScanner{dataDir: "/tmp"}
	proj := Project{Sessions: []Session{}}
	if err := scanner.DeleteProject(proj); err != nil {
		t.Fatalf("DeleteProject with empty sessions should not error, got: %v", err)
	}
}

func TestCodeBuddyScanner_ResumeCmd(t *testing.T) {
	scanner := &CodeBuddyScanner{bin: "codebuddy"}
	sess := Session{ResumeArg: "sess-abc"}
	cmd := scanner.ResumeCmd(sess)
	expected := []string{"codebuddy", "--resume", "sess-abc"}
	if len(cmd) != len(expected) {
		t.Fatalf("ResumeCmd len = %d, want %d", len(cmd), len(expected))
	}
	for i := range expected {
		if cmd[i] != expected[i] {
			t.Errorf("ResumeCmd[%d] = %q, want %q", i, cmd[i], expected[i])
		}
	}
}

func TestCodeBuddyScanner_PlatformAndDataDir(t *testing.T) {
	s := &CodeBuddyScanner{dataDir: "/home/user/.codebuddy"}
	if s.Platform() != PlatformCodeBuddy {
		t.Errorf("Platform() = %v, want PlatformCodeBuddy", s.Platform())
	}
	if s.DataDir() != "/home/user/.codebuddy" {
		t.Errorf("DataDir() = %q, want %q", s.DataDir(), "/home/user/.codebuddy")
	}
}
