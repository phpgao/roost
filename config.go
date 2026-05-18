package main

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config 对应 ~/.roost/roost.yaml
type Config struct {
	ResumeMode ResumeMode      `yaml:"resume_mode"` // replace（默认）| suspend
	Platforms  PlatformConfigs `yaml:"platforms"`
}

// GetResumeMode 返回生效的 resume 模式，未配置时默认 replace
func (c Config) GetResumeMode() ResumeMode {
	switch c.ResumeMode {
	case ResumeModeSuspend:
		return ResumeModeSuspend
	default:
		return ResumeModeReplace
	}
}

// PlatformConfigs 各平台配置
type PlatformConfigs struct {
	CodeBuddy PlatformConfig `yaml:"codebuddy"`
	Claude    PlatformConfig `yaml:"claude"`
	Gemini    PlatformConfig `yaml:"gemini"`
	Codex     PlatformConfig `yaml:"codex"`
	Copilot   PlatformConfig `yaml:"copilot"`
	OpenCode  PlatformConfig `yaml:"opencode"`
}

// PlatformConfig 单平台配置
type PlatformConfig struct {
	Bin     string   `yaml:"bin"`      // 二进制名称，覆盖默认值
	DataDir string   `yaml:"data_dir"` // 数据目录名（$HOME 下的相对名称），覆盖默认值
	Args    []string `yaml:"args"`     // resume 时附加的额外命令行参数
}

// LoadConfig 从 ~/.roost/roost.yaml 加载配置，不存在则创建空配置文件
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
			// 创建目录和默认配置文件
			_ = os.MkdirAll(configDir, 0o755)
			_ = os.WriteFile(configPath, []byte(defaultConfigTemplate), 0o644)
		}
		return cfg
	}

	_ = yaml.Unmarshal(data, &cfg)
	return cfg
}

// platformCfg returns the PlatformConfig for a given platform.
func (c Config) platformCfg(p Platform) PlatformConfig {
	switch p {
	case PlatformCodeBuddy:
		return c.Platforms.CodeBuddy
	case PlatformClaude:
		return c.Platforms.Claude
	case PlatformGemini:
		return c.Platforms.Gemini
	case PlatformCodex:
		return c.Platforms.Codex
	case PlatformCopilot:
		return c.Platforms.Copilot
	case PlatformOpenCode:
		return c.Platforms.OpenCode
	default:
		return PlatformConfig{}
	}
}

// ArgsFor 根据 Platform 枚举获取该平台的额外参数
func (c Config) ArgsFor(p Platform) []string {
	return c.platformCfg(p).Args
}

// BinFor 获取平台的二进制名称，未配置则返回默认值
func (c Config) BinFor(p Platform) string {
	if bin := c.platformCfg(p).Bin; bin != "" {
		return bin
	}
	switch p {
	case PlatformCodeBuddy:
		return defaultBinCodeBuddy
	case PlatformClaude:
		return defaultBinClaude
	case PlatformGemini:
		return defaultBinGemini
	case PlatformCodex:
		return defaultBinCodex
	case PlatformCopilot:
		return defaultBinCopilot
	case PlatformOpenCode:
		return defaultBinOpenCode
	default:
		return ""
	}
}

// DataDirFor 获取平台的数据目录名（$HOME 下的相对名称），未配置则返回默认值
func (c Config) DataDirFor(p Platform) string {
	if dir := c.platformCfg(p).DataDir; dir != "" {
		return dir
	}
	switch p {
	case PlatformCodeBuddy:
		return defaultDirCodeBuddy
	case PlatformClaude:
		return defaultDirClaude
	case PlatformGemini:
		return defaultDirGemini
	case PlatformCodex:
		return defaultDirCodex
	case PlatformCopilot:
		return defaultDirCopilot
	default:
		return ""
	}
}
