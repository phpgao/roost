package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGeminiScanner_ScanProjects(t *testing.T) {
	tmpDir := t.TempDir()

	// Create projects.json mapping
	projects := map[string]string{
		"/Users/test/myproject": "myproject",
	}
	projData, _ := json.Marshal(geminiProjects{Projects: projects})
	if err := os.WriteFile(filepath.Join(tmpDir, "projects.json"), projData, 0o644); err != nil {
		t.Fatal(err)
	}

	// Create session file
	chatsDir := filepath.Join(tmpDir, "tmp", "myproject", "chats")
	if err := os.MkdirAll(chatsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	sess := geminiSession{
		SessionID:   "gem-sess-001",
		StartTime:   "2026-05-15T10:00:00Z",
		LastUpdated: "2026-05-15T10:05:00Z",
		Messages: []geminiMessage{
			{Type: "user", Content: json.RawMessage(`"Hello Gemini"`)},
			{Type: "gemini", Model: "gemini-2.5-pro", Content: json.RawMessage(`"Hi there"`)},
			{Type: "user", Content: json.RawMessage(`"Help me code"`)},
		},
	}
	sessData, _ := json.Marshal(sess)
	if err := os.WriteFile(filepath.Join(chatsDir, "gem-sess-001.json"), sessData, 0o644); err != nil {
		t.Fatal(err)
	}

	scanner := &GeminiScanner{dataDir: tmpDir, bin: "gemini"}
	gotProjects, err := scanner.ScanProjects()
	if err != nil {
		t.Fatalf("ScanProjects() error: %v", err)
	}
	if len(gotProjects) != 1 {
		t.Fatalf("expected 1 project, got %d", len(gotProjects))
	}

	p := gotProjects[0]
	if p.FullPath != "/Users/test/myproject" {
		t.Errorf("FullPath = %q, want %q", p.FullPath, "/Users/test/myproject")
	}
	if len(p.Sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(p.Sessions))
	}

	s := p.Sessions[0]
	if s.ID != "gem-sess-001" {
		t.Errorf("ID = %q, want %q", s.ID, "gem-sess-001")
	}
	if s.Platform != PlatformGemini {
		t.Errorf("Platform = %v, want PlatformGemini", s.Platform)
	}
	if s.Title != "Hello Gemini" {
		t.Errorf("Title = %q, want %q", s.Title, "Hello Gemini")
	}
	if s.Model != "gemini-2.5-pro" {
		t.Errorf("Model = %q, want %q", s.Model, "gemini-2.5-pro")
	}
	if s.MsgCount != 3 {
		t.Errorf("MsgCount = %d, want 3", s.MsgCount)
	}
	expectedTime, _ := time.Parse(time.RFC3339, "2026-05-15T10:05:00Z")
	if !s.LastActive.Equal(expectedTime) {
		t.Errorf("LastActive = %v, want %v", s.LastActive, expectedTime)
	}
	if s.ResumeArg != "gem-sess-001" {
		t.Errorf("ResumeArg = %q, want %q", s.ResumeArg, "gem-sess-001")
	}
}

func TestGeminiScanner_MultipleProjects(t *testing.T) {
	tmpDir := t.TempDir()

	projects := map[string]string{
		"/Users/test/projA": "projA",
		"/Users/test/projB": "projB",
	}
	projData, _ := json.Marshal(geminiProjects{Projects: projects})
	if err := os.WriteFile(filepath.Join(tmpDir, "projects.json"), projData, 0o644); err != nil {
		t.Fatal(err)
	}

	for _, name := range []string{"projA", "projB"} {
		chatsDir := filepath.Join(tmpDir, "tmp", name, "chats")
		if err := os.MkdirAll(chatsDir, 0o755); err != nil {
			t.Fatal(err)
		}
		sess := geminiSession{
			SessionID: name + "-sess",
			StartTime: "2026-05-15T10:00:00Z",
			Messages: []geminiMessage{
				{Type: "user", Content: json.RawMessage(`"hello"`)},
			},
		}
		sessData, _ := json.Marshal(sess)
		if err := os.WriteFile(filepath.Join(chatsDir, name+"-sess.json"), sessData, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	scanner := &GeminiScanner{dataDir: tmpDir, bin: "gemini"}
	gotProjects, err := scanner.ScanProjects()
	if err != nil {
		t.Fatalf("ScanProjects() error: %v", err)
	}
	if len(gotProjects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(gotProjects))
	}
}

func TestGeminiScanner_NoProjectsFile(t *testing.T) {
	tmpDir := t.TempDir()
	scanner := &GeminiScanner{dataDir: tmpDir, bin: "gemini"}

	projects, err := scanner.ScanProjects()
	if err != nil {
		t.Fatalf("ScanProjects() error: %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("expected 0 projects, got %d", len(projects))
	}
}

func TestGeminiScanner_EmptyProjectsMap(t *testing.T) {
	tmpDir := t.TempDir()
	projData, _ := json.Marshal(geminiProjects{Projects: map[string]string{}})
	if err := os.WriteFile(filepath.Join(tmpDir, "projects.json"), projData, 0o644); err != nil {
		t.Fatal(err)
	}

	scanner := &GeminiScanner{dataDir: tmpDir, bin: "gemini"}
	projects, err := scanner.ScanProjects()
	if err != nil {
		t.Fatalf("ScanProjects() error: %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("expected 0 projects for empty map, got %d", len(projects))
	}
}

func TestGeminiScanner_NoChatsDir(t *testing.T) {
	tmpDir := t.TempDir()
	projects := map[string]string{
		"/Users/test/myproject": "myproject",
	}
	projData, _ := json.Marshal(geminiProjects{Projects: projects})
	if err := os.WriteFile(filepath.Join(tmpDir, "projects.json"), projData, 0o644); err != nil {
		t.Fatal(err)
	}
	// No tmp/myproject/chats directory

	scanner := &GeminiScanner{dataDir: tmpDir, bin: "gemini"}
	gotProjects, err := scanner.ScanProjects()
	if err != nil {
		t.Fatalf("ScanProjects() error: %v", err)
	}
	if len(gotProjects) != 0 {
		t.Errorf("expected 0 projects when no chats dir, got %d", len(gotProjects))
	}
}

func TestGeminiScanner_UntitledWhenNoUserMessage(t *testing.T) {
	tmpDir := t.TempDir()

	projects := map[string]string{"/Users/test/proj": "proj"}
	projData, _ := json.Marshal(geminiProjects{Projects: projects})
	if err := os.WriteFile(filepath.Join(tmpDir, "projects.json"), projData, 0o644); err != nil {
		t.Fatal(err)
	}

	chatsDir := filepath.Join(tmpDir, "tmp", "proj", "chats")
	if err := os.MkdirAll(chatsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	sess := geminiSession{
		SessionID: "no-user-msg",
		StartTime: "2026-05-15T10:00:00Z",
		Messages: []geminiMessage{
			{Type: "gemini", Model: "gemini-2.5-pro", Content: json.RawMessage(`"response only"`)},
		},
	}
	sessData, _ := json.Marshal(sess)
	if err := os.WriteFile(filepath.Join(chatsDir, "no-user-msg.json"), sessData, 0o644); err != nil {
		t.Fatal(err)
	}

	scanner := &GeminiScanner{dataDir: tmpDir, bin: "gemini"}
	gotProjects, _ := scanner.ScanProjects()
	if len(gotProjects) != 1 {
		t.Fatal("expected 1 project")
	}
	if gotProjects[0].Sessions[0].Title != "(untitled)" {
		t.Errorf("Title = %q, want %q", gotProjects[0].Sessions[0].Title, "(untitled)")
	}
}

func TestGeminiScanner_FallbackToStartTime(t *testing.T) {
	tmpDir := t.TempDir()

	projects := map[string]string{"/Users/test/proj": "proj"}
	projData, _ := json.Marshal(geminiProjects{Projects: projects})
	if err := os.WriteFile(filepath.Join(tmpDir, "projects.json"), projData, 0o644); err != nil {
		t.Fatal(err)
	}

	chatsDir := filepath.Join(tmpDir, "tmp", "proj", "chats")
	if err := os.MkdirAll(chatsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	sess := geminiSession{
		SessionID: "no-lastupdated",
		StartTime: "2026-05-15T10:00:00Z",
		Messages: []geminiMessage{
			{Type: "user", Content: json.RawMessage(`"hi"`)},
		},
	}
	sessData, _ := json.Marshal(sess)
	if err := os.WriteFile(filepath.Join(chatsDir, "no-lastupdated.json"), sessData, 0o644); err != nil {
		t.Fatal(err)
	}

	scanner := &GeminiScanner{dataDir: tmpDir, bin: "gemini"}
	gotProjects, _ := scanner.ScanProjects()
	if len(gotProjects) != 1 {
		t.Fatal("expected 1 project")
	}
	expectedTime, _ := time.Parse(time.RFC3339, "2026-05-15T10:00:00Z")
	if !gotProjects[0].Sessions[0].LastActive.Equal(expectedTime) {
		t.Errorf("LastActive = %v, want %v (fallback to StartTime)", gotProjects[0].Sessions[0].LastActive, expectedTime)
	}
}

func TestGeminiScanner_DeleteSession(t *testing.T) {
	tmpDir := t.TempDir()
	chatsDir := filepath.Join(tmpDir, "tmp", "proj", "chats")
	if err := os.MkdirAll(chatsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	filePath := filepath.Join(chatsDir, "test.json")
	if err := os.WriteFile(filePath, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	scanner := &GeminiScanner{dataDir: tmpDir, bin: "gemini"}
	sess := Session{FilePath: filePath}
	if err := scanner.DeleteSession(sess); err != nil {
		t.Fatalf("DeleteSession error: %v", err)
	}
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		t.Error("file should be deleted")
	}
}

func TestGeminiScanner_DeleteProject(t *testing.T) {
	tmpDir := t.TempDir()
	chatsDir := filepath.Join(tmpDir, "tmp", "proj", "chats")
	if err := os.MkdirAll(chatsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	filePath := filepath.Join(chatsDir, "test.json")
	if err := os.WriteFile(filePath, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	scanner := &GeminiScanner{dataDir: tmpDir, bin: "gemini"}
	proj := Project{
		Sessions: []Session{{FilePath: filePath}},
	}
	if err := scanner.DeleteProject(proj); err != nil {
		t.Fatalf("DeleteProject error: %v", err)
	}
	// The entire project dir (tmp/proj/) should be removed
	projectDir := filepath.Join(tmpDir, "tmp", "proj")
	if _, err := os.Stat(projectDir); !os.IsNotExist(err) {
		t.Error("project directory should be deleted")
	}
}

func TestGeminiScanner_ResumeCmd(t *testing.T) {
	scanner := &GeminiScanner{bin: "gemini"}
	sess := Session{ResumeArg: "gem-123"}
	cmd := scanner.ResumeCmd(sess)
	expected := []string{"gemini", "--resume", "gem-123"}
	if len(cmd) != len(expected) {
		t.Fatalf("ResumeCmd len = %d, want %d", len(cmd), len(expected))
	}
	for i := range expected {
		if cmd[i] != expected[i] {
			t.Errorf("ResumeCmd[%d] = %q, want %q", i, cmd[i], expected[i])
		}
	}
}

func TestGeminiScanner_PlatformAndDataDir(t *testing.T) {
	s := &GeminiScanner{dataDir: "/home/user/.gemini", bin: "gemini"}
	if s.Platform() != PlatformGemini {
		t.Errorf("Platform() = %v, want PlatformGemini", s.Platform())
	}
	if s.DataDir() != "/home/user/.gemini" {
		t.Errorf("DataDir() = %q, want %q", s.DataDir(), "/home/user/.gemini")
	}
}

func TestGeminiScanner_DeleteProjectEmpty(t *testing.T) {
	scanner := &GeminiScanner{dataDir: "/tmp", bin: "gemini"}
	proj := Project{Sessions: []Session{}}
	if err := scanner.DeleteProject(proj); err != nil {
		t.Fatalf("DeleteProject with empty sessions should not error, got: %v", err)
	}
}

func TestExtractGeminiText(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain string", `"hello world"`, "hello world"},
		{"text blocks", `[{"text":"block text"}]`, "block text"},
		{"empty string", `""`, ""},
		{"invalid json", `{bad`, ""},
		{"nil input", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var raw json.RawMessage
			if tt.input != "" {
				raw = json.RawMessage(tt.input)
			}
			got := extractGeminiText(raw)
			if got != tt.want {
				t.Errorf("extractGeminiText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestGeminiScanner_SkipNonJSONFiles(t *testing.T) {
	tmpDir := t.TempDir()

	projects := map[string]string{"/Users/test/proj": "proj"}
	projData, _ := json.Marshal(geminiProjects{Projects: projects})
	if err := os.WriteFile(filepath.Join(tmpDir, "projects.json"), projData, 0o644); err != nil {
		t.Fatal(err)
	}

	chatsDir := filepath.Join(tmpDir, "tmp", "proj", "chats")
	if err := os.MkdirAll(chatsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write a non-JSON file (should be skipped)
	if err := os.WriteFile(filepath.Join(chatsDir, "readme.txt"), []byte("not a session"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Write a subdirectory (should be skipped)
	if err := os.MkdirAll(filepath.Join(chatsDir, "subdir"), 0o755); err != nil {
		t.Fatal(err)
	}

	scanner := &GeminiScanner{dataDir: tmpDir, bin: "gemini"}
	gotProjects, err := scanner.ScanProjects()
	if err != nil {
		t.Fatalf("ScanProjects() error: %v", err)
	}
	if len(gotProjects) != 0 {
		t.Errorf("expected 0 projects (no valid session files), got %d", len(gotProjects))
	}
}

func TestGeminiScanner_SkipInvalidSessionJSON(t *testing.T) {
	tmpDir := t.TempDir()

	projects := map[string]string{"/Users/test/proj": "proj"}
	projData, _ := json.Marshal(geminiProjects{Projects: projects})
	if err := os.WriteFile(filepath.Join(tmpDir, "projects.json"), projData, 0o644); err != nil {
		t.Fatal(err)
	}

	chatsDir := filepath.Join(tmpDir, "tmp", "proj", "chats")
	if err := os.MkdirAll(chatsDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Write invalid JSON session file
	if err := os.WriteFile(filepath.Join(chatsDir, "bad.json"), []byte("not valid json"), 0o644); err != nil {
		t.Fatal(err)
	}

	scanner := &GeminiScanner{dataDir: tmpDir, bin: "gemini"}
	gotProjects, err := scanner.ScanProjects()
	if err != nil {
		t.Fatalf("ScanProjects() error: %v", err)
	}
	if len(gotProjects) != 0 {
		t.Errorf("expected 0 projects (invalid JSON), got %d", len(gotProjects))
	}
}
