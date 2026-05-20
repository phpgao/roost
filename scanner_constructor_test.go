package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewClaudeScanner(t *testing.T) {
	t.Run("loads knownPaths from .claude.json", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, ".claude.json")

		configContent := `{
			"projects": {
				"/home/user/roost": {},
				"/home/user/dotfiles": {},
				"/tmp/test": {}
			}
		}`
		if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
			t.Fatal(err)
		}

		cfg := Config{
			Platforms: PlatformConfigs{
				Claude: PlatformConfig{
					Bin:     "claude",
					DataDir: dir, // absolute path, filepath.Join will use it directly
				},
			},
		}
		s := NewClaudeScanner(cfg)

		if s.bin != "claude" {
			t.Errorf("bin = %q, want %q", s.bin, "claude")
		}
		if s.dataDir != dir {
			t.Errorf("dataDir = %q, want %q", s.dataDir, dir)
		}
		if len(s.knownPaths) != 3 {
			t.Errorf("expected 3 known paths, got %d", len(s.knownPaths))
		}

		expectedPaths := map[string]bool{
			"/home/user/roost":    false,
			"/home/user/dotfiles": false,
			"/tmp/test":           false,
		}
		for _, p := range s.knownPaths {
			if _, ok := expectedPaths[p]; ok {
				expectedPaths[p] = true
			}
		}
		for p, found := range expectedPaths {
			if !found {
				t.Errorf("expected path %q not found in knownPaths", p)
			}
		}
	})

	t.Run("handles missing config file gracefully", func(t *testing.T) {
		dir := t.TempDir()
		cfg := Config{
			Platforms: PlatformConfigs{
				Claude: PlatformConfig{
					Bin:     "claude",
					DataDir: dir,
				},
			},
		}
		s := NewClaudeScanner(cfg)

		if len(s.knownPaths) != 0 {
			t.Errorf("expected 0 known paths, got %d", len(s.knownPaths))
		}
	})

	t.Run("handles invalid JSON gracefully", func(t *testing.T) {
		dir := t.TempDir()
		configPath := filepath.Join(dir, ".claude.json")
		if err := os.WriteFile(configPath, []byte("invalid json"), 0o644); err != nil {
			t.Fatal(err)
		}

		cfg := Config{
			Platforms: PlatformConfigs{
				Claude: PlatformConfig{
					Bin:     "claude",
					DataDir: dir,
				},
			},
		}
		s := NewClaudeScanner(cfg)

		if len(s.knownPaths) != 0 {
			t.Errorf("expected 0 known paths, got %d", len(s.knownPaths))
		}
	})

	t.Run("expands relative data dir from home", func(t *testing.T) {
		cfg := Config{
			Platforms: PlatformConfigs{
				Claude: PlatformConfig{
					Bin:     "claude",
					DataDir: ".claude",
				},
			},
		}

		s := NewClaudeScanner(cfg)

		home, _ := os.UserHomeDir()
		expectedDataDir := filepath.Join(home, ".claude")
		if s.dataDir != expectedDataDir {
			t.Errorf("dataDir = %q, want %q", s.dataDir, expectedDataDir)
		}
	})
}

func TestNewCodeBuddyScanner(t *testing.T) {
	t.Run("loads trustedDirs from settings.json", func(t *testing.T) {
		dir := t.TempDir()
		settingsPath := filepath.Join(dir, "settings.json")

		settingsContent := `{
			"trustedDirectories": [
				"/home/user/roost",
				"/home/user/dotfiles",
				"/tmp/test",
				"/path/with/wildcard/**"
			]
		}`
		if err := os.WriteFile(settingsPath, []byte(settingsContent), 0o644); err != nil {
			t.Fatal(err)
		}

		cfg := Config{
			Platforms: PlatformConfigs{
				CodeBuddy: PlatformConfig{
					Bin:     "codebuddy",
					DataDir: dir,
				},
			},
		}
		s := NewCodeBuddyScanner(cfg)

		if s.bin != "codebuddy" {
			t.Errorf("bin = %q, want %q", s.bin, "codebuddy")
		}
		if s.dataDir != dir {
			t.Errorf("dataDir = %q, want %q", s.dataDir, dir)
		}
		if len(s.trustedDirs) != 3 {
			t.Errorf("expected 3 trusted dirs (wildcard filtered), got %d", len(s.trustedDirs))
		}

		expectedDirs := map[string]bool{
			"/home/user/roost":    false,
			"/home/user/dotfiles": false,
			"/tmp/test":           false,
		}
		for _, d := range s.trustedDirs {
			if _, ok := expectedDirs[d]; ok {
				expectedDirs[d] = true
			}
		}
		for d, found := range expectedDirs {
			if !found {
				t.Errorf("expected dir %q not found in trustedDirs", d)
			}
		}
	})

	t.Run("handles missing settings file gracefully", func(t *testing.T) {
		dir := t.TempDir()
		cfg := Config{
			Platforms: PlatformConfigs{
				CodeBuddy: PlatformConfig{
					Bin:     "codebuddy",
					DataDir: dir,
				},
			},
		}
		s := NewCodeBuddyScanner(cfg)

		if len(s.trustedDirs) != 0 {
			t.Errorf("expected 0 trusted dirs, got %d", len(s.trustedDirs))
		}
	})

	t.Run("handles invalid JSON gracefully", func(t *testing.T) {
		dir := t.TempDir()
		settingsPath := filepath.Join(dir, "settings.json")
		if err := os.WriteFile(settingsPath, []byte("invalid json"), 0o644); err != nil {
			t.Fatal(err)
		}

		cfg := Config{
			Platforms: PlatformConfigs{
				CodeBuddy: PlatformConfig{
					Bin:     "codebuddy",
					DataDir: dir,
				},
			},
		}
		s := NewCodeBuddyScanner(cfg)

		if len(s.trustedDirs) != 0 {
			t.Errorf("expected 0 trusted dirs, got %d", len(s.trustedDirs))
		}
	})

	t.Run("expands relative data dir from home", func(t *testing.T) {
		cfg := Config{
			Platforms: PlatformConfigs{
				CodeBuddy: PlatformConfig{
					Bin:     "codebuddy",
					DataDir: ".codebuddy",
				},
			},
		}

		s := NewCodeBuddyScanner(cfg)

		home, _ := os.UserHomeDir()
		expectedDataDir := filepath.Join(home, ".codebuddy")
		if s.dataDir != expectedDataDir {
			t.Errorf("dataDir = %q, want %q", s.dataDir, expectedDataDir)
		}
	})
}

func TestNewCodexScanner(t *testing.T) {
	t.Run("sets bin and dataDir from config", func(t *testing.T) {
		cfg := Config{
			Platforms: PlatformConfigs{
				Codex: PlatformConfig{
					Bin:     "codex",
					DataDir: ".codex",
				},
			},
		}

		s := NewCodexScanner(cfg)

		if s.bin != "codex" {
			t.Errorf("bin = %q, want %q", s.bin, "codex")
		}

		home, _ := os.UserHomeDir()
		expectedDataDir := filepath.Join(home, ".codex")
		if s.dataDir != expectedDataDir {
			t.Errorf("dataDir = %q, want %q", s.dataDir, expectedDataDir)
		}
	})

	t.Run("uses defaults when config is empty", func(t *testing.T) {
		cfg := Config{}

		s := NewCodexScanner(cfg)

		if s.bin != "codex" {
			t.Errorf("bin = %q, want %q", s.bin, "codex")
		}

		home, _ := os.UserHomeDir()
		expectedDataDir := filepath.Join(home, ".codex")
		if s.dataDir != expectedDataDir {
			t.Errorf("dataDir = %q, want %q", s.dataDir, expectedDataDir)
		}
	})
}

func TestNewCopilotScanner(t *testing.T) {
	t.Run("sets bin and dataDir from config", func(t *testing.T) {
		cfg := Config{
			Platforms: PlatformConfigs{
				Copilot: PlatformConfig{
					Bin:     "copilot",
					DataDir: ".copilot",
				},
			},
		}

		s := NewCopilotScanner(cfg)

		if s.bin != "copilot" {
			t.Errorf("bin = %q, want %q", s.bin, "copilot")
		}

		home, _ := os.UserHomeDir()
		expectedDataDir := filepath.Join(home, ".copilot")
		if s.dataDir != expectedDataDir {
			t.Errorf("dataDir = %q, want %q", s.dataDir, expectedDataDir)
		}
	})

	t.Run("uses defaults when config is empty", func(t *testing.T) {
		cfg := Config{}

		s := NewCopilotScanner(cfg)

		if s.bin != "copilot" {
			t.Errorf("bin = %q, want %q", s.bin, "copilot")
		}

		home, _ := os.UserHomeDir()
		expectedDataDir := filepath.Join(home, ".copilot")
		if s.dataDir != expectedDataDir {
			t.Errorf("dataDir = %q, want %q", s.dataDir, expectedDataDir)
		}
	})
}

func TestNewGeminiScanner(t *testing.T) {
	t.Run("sets bin and dataDir from config", func(t *testing.T) {
		cfg := Config{
			Platforms: PlatformConfigs{
				Gemini: PlatformConfig{
					Bin:     "gemini",
					DataDir: ".gemini",
				},
			},
		}

		s := NewGeminiScanner(cfg)

		if s.bin != "gemini" {
			t.Errorf("bin = %q, want %q", s.bin, "gemini")
		}

		home, _ := os.UserHomeDir()
		expectedDataDir := filepath.Join(home, ".gemini")
		if s.dataDir != expectedDataDir {
			t.Errorf("dataDir = %q, want %q", s.dataDir, expectedDataDir)
		}
	})

	t.Run("uses defaults when config is empty", func(t *testing.T) {
		cfg := Config{}

		s := NewGeminiScanner(cfg)

		if s.bin != "gemini" {
			t.Errorf("bin = %q, want %q", s.bin, "gemini")
		}

		home, _ := os.UserHomeDir()
		expectedDataDir := filepath.Join(home, ".gemini")
		if s.dataDir != expectedDataDir {
			t.Errorf("dataDir = %q, want %q", s.dataDir, expectedDataDir)
		}
	})
}

func TestNewOpenCodeScanner(t *testing.T) {
	t.Run("returns scanner with configured bin even when dbPath is empty", func(t *testing.T) {
		cfg := Config{
			Platforms: PlatformConfigs{
				OpenCode: PlatformConfig{
					Bin:     "nonexistent-opencode",
					DataDir: ".opencode",
				},
			},
		}

		s := NewOpenCodeScanner(cfg)

		// bin should be what we configured
		if s.bin != "nonexistent-opencode" {
			t.Errorf("bin = %q, want %q", s.bin, "nonexistent-opencode")
		}

		// dbPath will be empty because the bin doesn't exist
		if s.dbPath != "" {
			t.Errorf("dbPath should be empty for nonexistent bin, got %q", s.dbPath)
		}
	})

	t.Run("DataDir returns empty when dbPath is empty", func(t *testing.T) {
		s := &OpenCodeScanner{dbPath: ""}
		if got := s.DataDir(); got != "" {
			t.Errorf("DataDir() = %q, want empty string", got)
		}
	})

	t.Run("DataDir returns dirname of dbPath when set", func(t *testing.T) {
		s := &OpenCodeScanner{dbPath: "/home/user/.opencode/opencode.db"}
		expected := "/home/user/.opencode"
		if got := s.DataDir(); got != expected {
			t.Errorf("DataDir() = %q, want %q", got, expected)
		}
	})
}
