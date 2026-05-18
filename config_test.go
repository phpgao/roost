package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfig_GetResumeMode(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want ResumeMode
	}{
		{"default empty", Config{}, ResumeModeReplace},
		{"explicit replace", Config{ResumeMode: ResumeModeReplace}, ResumeModeReplace},
		{"suspend", Config{ResumeMode: ResumeModeSuspend}, ResumeModeSuspend},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cfg.GetResumeMode(); got != tt.want {
				t.Errorf("GetResumeMode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_BinFor(t *testing.T) {
	cfg := Config{
		Platforms: PlatformConfigs{
			Claude:   PlatformConfig{Bin: "my-claude"},
			Gemini:   PlatformConfig{},
			OpenCode: PlatformConfig{Bin: "/usr/local/bin/opencode"},
		},
	}

	tests := []struct {
		platform Platform
		want     string
	}{
		{PlatformClaude, "my-claude"},
		{PlatformGemini, "gemini"},
		{PlatformCodex, "codex"},
		{PlatformCopilot, "copilot"},
		{PlatformOpenCode, "/usr/local/bin/opencode"},
		{PlatformCodeBuddy, "codebuddy"},
	}
	for _, tt := range tests {
		t.Run(tt.platform.Name(), func(t *testing.T) {
			if got := cfg.BinFor(tt.platform); got != tt.want {
				t.Errorf("BinFor(%s) = %q, want %q", tt.platform.Name(), got, tt.want)
			}
		})
	}
}

func TestConfig_DataDirFor(t *testing.T) {
	cfg := Config{
		Platforms: PlatformConfigs{
			Claude:  PlatformConfig{DataDir: ".custom-claude"},
			Copilot: PlatformConfig{},
		},
	}

	tests := []struct {
		platform Platform
		want     string
	}{
		{PlatformClaude, ".custom-claude"},
		{PlatformGemini, ".gemini"},
		{PlatformCodex, ".codex"},
		{PlatformCopilot, ".copilot"},
		{PlatformCodeBuddy, ".codebuddy"},
		{PlatformOpenCode, ""},
	}
	for _, tt := range tests {
		t.Run(tt.platform.Name(), func(t *testing.T) {
			if got := cfg.DataDirFor(tt.platform); got != tt.want {
				t.Errorf("DataDirFor(%s) = %q, want %q", tt.platform.Name(), got, tt.want)
			}
		})
	}
}

func TestConfig_ArgsFor(t *testing.T) {
	cfg := Config{
		Platforms: PlatformConfigs{
			Claude:   PlatformConfig{Args: []string{"--dangerously-skip-permissions"}},
			Gemini:   PlatformConfig{Args: nil},
			OpenCode: PlatformConfig{Args: []string{"-y"}},
		},
	}

	if got := cfg.ArgsFor(PlatformClaude); len(got) != 1 || got[0] != "--dangerously-skip-permissions" {
		t.Errorf("ArgsFor(Claude) = %v, want [--dangerously-skip-permissions]", got)
	}
	if got := cfg.ArgsFor(PlatformGemini); got != nil {
		t.Errorf("ArgsFor(Gemini) = %v, want nil", got)
	}
	if got := cfg.ArgsFor(PlatformCodeBuddy); got != nil {
		t.Errorf("ArgsFor(CodeBuddy) = %v, want nil", got)
	}
	if got := cfg.ArgsFor(PlatformOpenCode); len(got) != 1 || got[0] != "-y" {
		t.Errorf("ArgsFor(OpenCode) = %v, want [-y]", got)
	}
	// default 分支：PlatformCodeBuddy 未在 cfg 中配置，应返回 nil
	if got := cfg.ArgsFor(PlatformCodeBuddy); got != nil {
		t.Errorf("ArgsFor(CodeBuddy) = %v, want nil", got)
	}
}

func TestLoadConfig_CreatesDefault(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".roost")
	configPath := filepath.Join(configDir, "roost.yaml")

	// Override home for this test
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	cfg := LoadConfig()

	// Config file should have been created
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("config file should be created when it doesn't exist")
	}

	// Default resume mode should be replace
	if cfg.GetResumeMode() != ResumeModeReplace {
		t.Errorf("GetResumeMode() = %v, want ResumeModeReplace", cfg.GetResumeMode())
	}
}

func TestLoadConfig_ReadsExisting(t *testing.T) {
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ".roost")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}

	yamlContent := `resume_mode: suspend
platforms:
  claude:
    bin: my-claude
    args: [--test]
`
	if err := os.WriteFile(filepath.Join(configDir, "roost.yaml"), []byte(yamlContent), 0o644); err != nil {
		t.Fatal(err)
	}

	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", originalHome)

	cfg := LoadConfig()

	if cfg.GetResumeMode() != ResumeModeSuspend {
		t.Errorf("GetResumeMode() = %v, want ResumeModeSuspend", cfg.GetResumeMode())
	}
	if cfg.BinFor(PlatformClaude) != "my-claude" {
		t.Errorf("BinFor(Claude) = %q, want %q", cfg.BinFor(PlatformClaude), "my-claude")
	}
	args := cfg.ArgsFor(PlatformClaude)
	if len(args) != 1 || args[0] != "--test" {
		t.Errorf("ArgsFor(Claude) = %v, want [--test]", args)
	}
}
