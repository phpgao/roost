package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/pflag"
)

// 版本信息，由 Makefile 的 LDFLAGS 注入
var (
	Version   = "v0.0.0"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

// isCommandAvailable 检测命令是否可用（在 $PATH 中能找到）
func isCommandAvailable(bin string) bool {
	_, err := exec.LookPath(bin)
	return err == nil
}

func main() {
	// command-line flags
	listFlag := pflag.BoolP("list", "l", false, "list all projects and sessions")
	resumeFlag := pflag.StringP("resume", "r", "", "resume the specified session ID")
	deleteFlag := pflag.StringP("delete", "d", "", "delete the specified session ID")
	jsonFlag := pflag.BoolP("json", "j", false, "output in JSON format (use with --list)")
	versionFlag := pflag.BoolP("version", "v", false, "print version information")
	pflag.Parse()

	if *versionFlag {
		fmt.Printf("roost version %s\n", Version)
		fmt.Printf("  build time: %s\n", BuildTime)
		fmt.Printf("  git commit: %s\n", GitCommit)
		return
	}

	cfg := LoadConfig()

	var scanners []Scanner

	// newScannerFor maps each platform to its constructor; nil means not yet supported
	newScannerFor := map[Platform]func(Config) Scanner{
		PlatformCodeBuddy: func(c Config) Scanner { return NewCodeBuddyScanner(c) },
		PlatformClaude:    func(c Config) Scanner { return NewClaudeScanner(c) },
		PlatformGemini:    func(c Config) Scanner { return NewGeminiScanner(c) },
		PlatformCodex:     func(c Config) Scanner { return NewCodexScanner(c) },
		PlatformCopilot:   func(c Config) Scanner { return NewCopilotScanner(c) },
		PlatformOpenCode:  func(c Config) Scanner { return NewOpenCodeScanner(c) },
	}
	for _, p := range []Platform{PlatformCodeBuddy, PlatformClaude, PlatformGemini, PlatformCodex, PlatformCopilot, PlatformOpenCode} {
		if isCommandAvailable(cfg.BinFor(p)) {
			scanners = append(scanners, newScannerFor[p](cfg))
		}
	}

	// 非交互模式
	if *listFlag || *resumeFlag != "" || *deleteFlag != "" {
		runNonInteractive(scanners, cfg, *listFlag, *resumeFlag, *deleteFlag, *jsonFlag)
		return
	}

	// 交互模式（TUI）
	m := newModel(scanners)
	m.cfg = cfg
	p := tea.NewProgram(&m)

	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fm, ok := finalModel.(*Model)
	if !ok {
		return
	}

	if fm.resumeSession != nil {
		execResume(fm.resumeSession, scanners, cfg)
	}

	if fm.newSessionPending && fm.selectedProject != nil {
		execNewSession(fm.newAgentPlatform, fm.selectedProject.FullPath, cfg)
	}
}

func runNonInteractive(scanners []Scanner, cfg Config, list bool, resumeID, deleteID string, jsonOutput bool) {
	projects := ScanProjectsParallel(scanners)

	switch {
	case list:
		cmdList(projects, jsonOutput)
	case resumeID != "":
		cmdResume(projects, scanners, cfg, resumeID)
	case deleteID != "":
		cmdDelete(projects, scanners, deleteID)
	}
}

// jsonSession JSON 输出用的结构
type jsonSession struct {
	ID         string `json:"id"`
	Platform   string `json:"platform"`
	AgentType  string `json:"agent_type,omitempty"`
	Title      string `json:"title"`
	Model      string `json:"model"`
	LastActive string `json:"last_active"`
	MsgCount   int    `json:"msg_count"`
}

type jsonProject struct {
	Path     string        `json:"path"`
	Sessions []jsonSession `json:"sessions"`
}

func cmdList(projects []Project, jsonOutput bool) {
	if jsonOutput {
		var out []jsonProject
		for _, p := range projects {
			jp := jsonProject{Path: p.FullPath}
			for _, s := range p.Sessions {
				jp.Sessions = append(jp.Sessions, jsonSession{
					ID:         s.ID,
					Platform:   s.Platform.Name(),
					AgentType:  s.AgentType,
					Title:      s.Title,
					Model:      s.Model,
					LastActive: s.LastActive.Format("2006-01-02 15:04:05"),
					MsgCount:   s.MsgCount,
				})
			}
			out = append(out, jp)
		}
		data, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(data))
		return
	}

	// 纯文本输出
	for _, p := range projects {
		fmt.Printf("%s (%d sessions)\n", p.FullPath, len(p.Sessions))
		for _, s := range p.Sessions {
			agent := ""
			if s.AgentType != "" && s.AgentType != typeAgentCLI {
				agent = "[" + s.AgentType + "] "
			}
			fmt.Printf("  %-10s %s%-30s  %-18s  %-10s  %d msgs  id:%s\n",
				s.Platform.Name(), agent, s.Title, s.Model,
				relativeTime(s.LastActive), s.MsgCount, s.ID)
		}
	}
}

func cmdResume(projects []Project, scanners []Scanner, cfg Config, sessionID string) {
	sess := findSession(projects, sessionID)
	if sess == nil {
		fmt.Fprintf(os.Stderr, "session not found: %s\n", sessionID)
		os.Exit(1)
	}
	execResume(sess, scanners, cfg)
}

func cmdDelete(projects []Project, scanners []Scanner, sessionID string) {
	sess := findSession(projects, sessionID)
	if sess == nil {
		fmt.Fprintf(os.Stderr, "session not found: %s\n", sessionID)
		os.Exit(1)
	}
	for _, sc := range scanners {
		if sc.Platform() == sess.Platform {
			if err := sc.DeleteSession(*sess); err != nil {
				fmt.Fprintf(os.Stderr, "delete error: %v\n", err)
			}
			fmt.Fprintf(os.Stderr, "deleted: %s (%s)\n", sess.Title, sess.ID)
			return
		}
	}
	// 未找到匹配的 scanner
	fmt.Fprintf(os.Stderr, "no scanner found for platform %d\n", sess.Platform)
	os.Exit(1)
}

// findSession 在全部项目中查找指定 ID 的 session。
func findSession(projects []Project, id string) *Session {
	for _, p := range projects {
		for _, s := range p.Sessions {
			if s.ID == id {
				return &s
			}
		}
	}
	return nil
}

func execResume(sess *Session, scanners []Scanner, cfg Config) {
	if err := os.Chdir(sess.ProjectPath); err != nil {
		fmt.Fprintf(os.Stderr, "chdir error: %v\n", err)
		os.Exit(1)
	}

	var scanner Scanner
	for _, sc := range scanners {
		if sc.Platform() == sess.Platform {
			scanner = sc
			break
		}
	}
	if scanner == nil {
		fmt.Fprintf(os.Stderr, "no scanner for platform %d\n", sess.Platform)
		os.Exit(1)
	}

	argv := scanner.ResumeCmd(*sess)
	// 拼上配置中的额外参数
	extraArgs := cfg.ArgsFor(sess.Platform)
	if len(extraArgs) > 0 {
		argv = append(argv, extraArgs...)
	}

	// 打印执行的命令（带平台颜色）
	fmt.Fprintln(os.Stderr, RenderColoredCommand(sess.Platform, strings.Join(argv, " ")))

	binPath, err := exec.LookPath(argv[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "command not found: %s\n", argv[0])
		os.Exit(1)
	}
	if err := syscall.Exec(binPath, argv, os.Environ()); err != nil {
		fmt.Fprintf(os.Stderr, "exec error: %v\n", err)
		os.Exit(1)
	}
}

// execNewSession 启动新 session（不带 resume arg）
func execNewSession(p Platform, projectPath string, cfg Config) {
	if err := os.Chdir(projectPath); err != nil {
		fmt.Fprintf(os.Stderr, "chdir error: %v\n", err)
		os.Exit(1)
	}

	bin := cfg.BinFor(p)
	argv := []string{bin}
	extraArgs := cfg.ArgsFor(p)
	if len(extraArgs) > 0 {
		argv = append(argv, extraArgs...)
	}

	fmt.Fprintln(os.Stderr, RenderColoredCommand(p, strings.Join(argv, " ")))

	binPath, err := exec.LookPath(argv[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "command not found: %s\n", argv[0])
		os.Exit(1)
	}
	if err := syscall.Exec(binPath, argv, os.Environ()); err != nil {
		fmt.Fprintf(os.Stderr, "exec error: %v\n", err)
		os.Exit(1)
	}
}
