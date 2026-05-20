// Package config handles loading and accessing roost configuration.
package config

import (
	"os"
	"path/filepath"

	"github.com/phpgao/roost/internal/types"
	"gopkg.in/yaml.v3"
)

// Config corresponds to ~/.roost/roost.yaml.
type Config struct {
	ResumeMode types.ResumeMode `yaml:"resume_mode"` // replace (default) | suspend
	Platforms  PlatformConfigs  `yaml:"platforms"`
}

// GetResumeMode returns the effective resume mode, defaulting to replace.
func (c Config) GetResumeMode() types.ResumeMode {
	switch c.ResumeMode {
	case types.ResumeModeSuspend:
		return types.ResumeModeSuspend
	default:
		return types.ResumeModeReplace
	}
}

// PlatformConfigs holds per-platform configuration.
type PlatformConfigs struct {
	CodeBuddy PlatformConfig `yaml:"codebuddy"`
	Claude    PlatformConfig `yaml:"claude"`
	Gemini    PlatformConfig `yaml:"gemini"`
	Codex     PlatformConfig `yaml:"codex"`
	Copilot   PlatformConfig `yaml:"copilot"`
	OpenCode  PlatformConfig `yaml:"opencode"`
}

// PlatformConfig is per-platform configuration.
type PlatformConfig struct {
	Bin     string   `yaml:"bin"`      // binary name, overrides default
	DataDir string   `yaml:"data_dir"` // data directory name (relative to $HOME), overrides default
	Args    []string `yaml:"args"`     // extra command-line arguments on resume
}

// LoadConfig loads from ~/.roost/roost.yaml, creating an empty template if not present.
func LoadConfig() Config {
	var cfg Config

	home, err := os.UserHomeDir()
	if err != nil {
		return cfg
	}

	configDir := filepath.Join(home, ".roost")
	configPath := filepath.Join(configDir, "roost.yaml")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			_ = os.MkdirAll(configDir, 0o755)
			_ = os.WriteFile(configPath, []byte(types.DefaultConfigTemplate), 0o644)
		}
		return cfg
	}

	_ = yaml.Unmarshal(data, &cfg)
	return cfg
}

// platformCfg returns the PlatformConfig for a given platform.
func (c Config) platformCfg(p types.Platform) PlatformConfig {
	switch p {
	case types.PlatformCodeBuddy:
		return c.Platforms.CodeBuddy
	case types.PlatformClaude:
		return c.Platforms.Claude
	case types.PlatformGemini:
		return c.Platforms.Gemini
	case types.PlatformCodex:
		return c.Platforms.Codex
	case types.PlatformCopilot:
		return c.Platforms.Copilot
	case types.PlatformOpenCode:
		return c.Platforms.OpenCode
	default:
		return PlatformConfig{}
	}
}

// ArgsFor returns extra arguments for the given platform.
func (c Config) ArgsFor(p types.Platform) []string {
	return c.platformCfg(p).Args
}

// BinFor returns the binary name for the platform, falling back to default.
func (c Config) BinFor(p types.Platform) string {
	if bin := c.platformCfg(p).Bin; bin != "" {
		return bin
	}
	switch p {
	case types.PlatformCodeBuddy:
		return types.DefaultBinCodeBuddy
	case types.PlatformClaude:
		return types.DefaultBinClaude
	case types.PlatformGemini:
		return types.DefaultBinGemini
	case types.PlatformCodex:
		return types.DefaultBinCodex
	case types.PlatformCopilot:
		return types.DefaultBinCopilot
	case types.PlatformOpenCode:
		return types.DefaultBinOpenCode
	default:
		return ""
	}
}

// DataDirFor returns the data directory name for the platform, falling back to default.
func (c Config) DataDirFor(p types.Platform) string {
	if dir := c.platformCfg(p).DataDir; dir != "" {
		return dir
	}
	switch p {
	case types.PlatformCodeBuddy:
		return types.DefaultDirCodeBuddy
	case types.PlatformClaude:
		return types.DefaultDirClaude
	case types.PlatformGemini:
		return types.DefaultDirGemini
	case types.PlatformCodex:
		return types.DefaultDirCodex
	case types.PlatformCopilot:
		return types.DefaultDirCopilot
	default:
		return ""
	}
}
