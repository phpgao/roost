package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCodexScanner_ScanProjects(t *testing.T) {
	// 构造临时目录模拟 ~/.codex/sessions/YYYY/MM/DD/*.jsonl
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "sessions", "2026", "05", "15")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// 写入一个模拟 session 文件
	sessionFile := filepath.Join(sessionsDir, "rollout-2026-05-15T10-00-00-abc123.jsonl")
	lines := []map[string]any{
		{
			"timestamp": "2026-05-15T10:00:00Z",
			"type":      "session_meta",
			"payload": map[string]any{
				"id":             "abc123-def456",
				"cwd":            "/Users/test/myproject",
				"timestamp":      "2026-05-15T10:00:00Z",
				"model_provider": "openai",
				"cli_version":    "0.120.0",
			},
		},
		{
			"timestamp": "2026-05-15T10:00:05Z",
			"type":      "turn_context",
			"payload": map[string]any{
				"model": "gpt-5.4",
			},
		},
		{
			"timestamp": "2026-05-15T10:00:10Z",
			"type":      "event_msg",
			"payload": map[string]any{
				"type":    "user_message",
				"message": "帮我修复这个 bug\n详细描述在这里",
			},
		},
		{
			"timestamp": "2026-05-15T10:01:00Z",
			"type":      "event_msg",
			"payload": map[string]any{
				"type":    "agent_message",
				"message": "好的，我来看一下",
			},
		},
		{
			"timestamp": "2026-05-15T10:02:00Z",
			"type":      "event_msg",
			"payload": map[string]any{
				"type":    "user_message",
				"message": "谢谢",
			},
		},
	}

	f, err := os.Create(sessionFile)
	if err != nil {
		t.Fatal(err)
	}
	enc := json.NewEncoder(f)
	for _, line := range lines {
		if encErr := enc.Encode(line); encErr != nil {
			t.Fatal(encErr)
		}
	}
	f.Close()

	// 创建 scanner，注入测试目录
	scanner := &CodexScanner{dataDir: tmpDir}

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
	if p.Name != "test/myproject" {
		t.Errorf("Name = %q, want %q", p.Name, "test/myproject")
	}
	if len(p.Sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(p.Sessions))
	}

	sess := p.Sessions[0]
	if sess.ID != "abc123-def456" {
		t.Errorf("ID = %q, want %q", sess.ID, "abc123-def456")
	}
	if sess.Platform != PlatformCodex {
		t.Errorf("Platform = %v, want PlatformCodex", sess.Platform)
	}
	if sess.Title != "帮我修复这个 bug" {
		t.Errorf("Title = %q, want %q", sess.Title, "帮我修复这个 bug")
	}
	if sess.Model != "gpt-5.4" {
		t.Errorf("Model = %q, want %q", sess.Model, "gpt-5.4")
	}
	if sess.MsgCount != 3 {
		t.Errorf("MsgCount = %d, want 3", sess.MsgCount)
	}
	expectedTime, _ := time.Parse(time.RFC3339, "2026-05-15T10:02:00Z")
	if !sess.LastActive.Equal(expectedTime) {
		t.Errorf("LastActive = %v, want %v", sess.LastActive, expectedTime)
	}
	if sess.ResumeArg != "abc123-def456" {
		t.Errorf("ResumeArg = %q, want %q", sess.ResumeArg, "abc123-def456")
	}
}

func TestCodexScanner_MultipleProjectGrouping(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "sessions", "2026", "05", "15")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// 两个 session 来自不同 cwd
	writeCodexSession(t, sessionsDir, "sess1.jsonl", "id-1", "/project/a", "hello from a")
	writeCodexSession(t, sessionsDir, "sess2.jsonl", "id-2", "/project/b", "hello from b")
	// 第三个 session 同属 project/a
	writeCodexSession(t, sessionsDir, "sess3.jsonl", "id-3", "/project/a", "another in a")

	scanner := &CodexScanner{dataDir: tmpDir}
	projects, err := scanner.ScanProjects()
	if err != nil {
		t.Fatalf("ScanProjects() error: %v", err)
	}
	if len(projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(projects))
	}

	// 找到 project/a
	var projA *Project
	for i := range projects {
		if projects[i].FullPath == "/project/a" {
			projA = &projects[i]
			break
		}
	}
	if projA == nil {
		t.Fatal("project /project/a not found")
	}
	if len(projA.Sessions) != 2 {
		t.Errorf("/project/a sessions = %d, want 2", len(projA.Sessions))
	}
}

func TestCodexScanner_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	scanner := &CodexScanner{dataDir: tmpDir}

	projects, err := scanner.ScanProjects()
	if err != nil {
		t.Fatalf("ScanProjects() error: %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("expected 0 projects, got %d", len(projects))
	}
}

func TestCodexScanner_ResumeCmd(t *testing.T) {
	scanner := &CodexScanner{dataDir: "/tmp", bin: "codex"}
	sess := Session{ResumeArg: "abc-123"}
	cmd := scanner.ResumeCmd(sess)
	expected := []string{"codex", codexCmdResume, "abc-123"}
	if len(cmd) != len(expected) {
		t.Fatalf("ResumeCmd len = %d, want %d", len(cmd), len(expected))
	}
	for i := range expected {
		if cmd[i] != expected[i] {
			t.Errorf("ResumeCmd[%d] = %q, want %q", i, cmd[i], expected[i])
		}
	}
}

func TestCodexScanner_DeleteSession(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "sessions", "2026", "05", "15")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	filePath := filepath.Join(sessionsDir, "test.jsonl")
	if err := os.WriteFile(filePath, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	scanner := &CodexScanner{dataDir: tmpDir}
	sess := Session{FilePath: filePath}
	if err := scanner.DeleteSession(sess); err != nil {
		t.Fatalf("DeleteSession error: %v", err)
	}
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("file should be deleted")
	}
}

func TestCodexScanner_DeleteProject(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "sessions", "2026", "05", "15")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	file1 := filepath.Join(sessionsDir, "proj_a.jsonl")
	file2 := filepath.Join(sessionsDir, "proj_b.jsonl")
	if err := os.WriteFile(file1, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file2, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	scanner := &CodexScanner{dataDir: tmpDir}
	proj := Project{
		Name: "test/proj", FullPath: "/test/proj",
		Sessions: []Session{
			{ID: "s1", FilePath: file1},
			{ID: "s2", FilePath: file2},
		},
	}
	if err := scanner.DeleteProject(proj); err != nil {
		t.Fatalf("DeleteProject error: %v", err)
	}
	if _, err := os.Stat(file1); !os.IsNotExist(err) {
		t.Error("file1 should be deleted")
	}
	if _, err := os.Stat(file2); !os.IsNotExist(err) {
		t.Error("file2 should be deleted")
	}
}

func TestCodexScanner_PlatformAndDataDir(t *testing.T) {
	s := &CodexScanner{dataDir: "/home/user/.codex"}
	if s.Platform() != PlatformCodex {
		t.Errorf("Platform() = %v, want PlatformCodex", s.Platform())
	}
	if s.DataDir() != "/home/user/.codex" {
		t.Errorf("DataDir() = %q, want %q", s.DataDir(), "/home/user/.codex")
	}
}

func TestCodexScanner_SkipInvalidJSONL(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "sessions", "2026", "05", "15")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write a file without session_meta — should be skipped
	f, err := os.Create(filepath.Join(sessionsDir, "invalid.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(`{"type":"event_msg","payload":{"type":"user_message","message":"no meta"}}` + "\n")
	f.Close()

	scanner := &CodexScanner{dataDir: tmpDir}
	projects, err := scanner.ScanProjects()
	if err != nil {
		t.Fatalf("ScanProjects() error: %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("expected 0 projects for invalid session, got %d", len(projects))
	}
}

func TestCodexScanner_SkipEmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	sessionsDir := filepath.Join(tmpDir, "sessions", "2026", "05", "15")
	if err := os.MkdirAll(sessionsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write empty file
	if err := os.WriteFile(filepath.Join(sessionsDir, "empty.jsonl"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}

	scanner := &CodexScanner{dataDir: tmpDir}
	projects, err := scanner.ScanProjects()
	if err != nil {
		t.Fatalf("ScanProjects() error: %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("expected 0 projects for empty file, got %d", len(projects))
	}
}

func writeCodexSession(t *testing.T, dir, filename, id, cwd, msg string) {
	t.Helper()
	lines := []map[string]any{
		{
			"timestamp": "2026-05-15T10:00:00Z",
			"type":      "session_meta",
			"payload": map[string]any{
				"id":        id,
				"cwd":       cwd,
				"timestamp": "2026-05-15T10:00:00Z",
			},
		},
		{
			"timestamp": "2026-05-15T10:00:10Z",
			"type":      "event_msg",
			"payload": map[string]any{
				"type":    "user_message",
				"message": msg,
			},
		},
	}
	f, err := os.Create(filepath.Join(dir, filename))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	for _, line := range lines {
		if encErr := enc.Encode(line); encErr != nil {
			t.Fatal(encErr)
		}
	}
}
