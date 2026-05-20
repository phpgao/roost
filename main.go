package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	tea "charm.land/bubbletea/v2"
	"github.com/phpgao/roost/internal/app"
	"github.com/phpgao/roost/internal/config"
	"github.com/phpgao/roost/internal/scanner"
	"github.com/phpgao/roost/internal/types"
	"github.com/phpgao/roost/internal/view"
	"github.com/spf13/pflag"
)

// Version info, injected by Makefile LDFLAGS.
var (
	Version   = "v0.0.0"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

func main() {
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

	cfg := config.LoadConfig()
	scanners := buildScanners(cfg)

	// Non-interactive mode
	if *listFlag || *resumeFlag != "" || *deleteFlag != "" {
		runNonInteractive(scanners, cfg, *listFlag, *resumeFlag, *deleteFlag, *jsonFlag)
		return
	}

	// Interactive TUI mode
	a := app.NewApp(scanners, cfg)
	p := tea.NewProgram(a)

	finalModel, err := p.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	fm, ok := finalModel.(*app.App)
	if !ok {
		return
	}

	if sess := fm.ResumeSession(); sess != nil {
		execResume(sess, scanners, cfg)
	}

	if fm.LaunchPending() {
		execNewSession(fm.LaunchPlatform(), fm.LaunchProjectPath(), cfg)
	}
}

// buildScanners creates scanner instances for all platforms with available binaries.
func buildScanners(cfg config.Config) []scanner.Scanner {
	type platformCtor struct {
		platform types.Platform
		newScan  func(config.Config) scanner.Scanner
	}

	ctors := []platformCtor{
		{types.PlatformCodeBuddy, func(c config.Config) scanner.Scanner { return scanner.NewCodeBuddyScanner(c) }},
		{types.PlatformClaude, func(c config.Config) scanner.Scanner { return scanner.NewClaudeScanner(c) }},
		{types.PlatformGemini, func(c config.Config) scanner.Scanner { return scanner.NewGeminiScanner(c) }},
		{types.PlatformCodex, func(c config.Config) scanner.Scanner { return scanner.NewCodexScanner(c) }},
		{types.PlatformCopilot, func(c config.Config) scanner.Scanner { return scanner.NewCopilotScanner(c) }},
		{types.PlatformOpenCode, func(c config.Config) scanner.Scanner { return scanner.NewOpenCodeScanner(c) }},
	}

	var scanners []scanner.Scanner
	for _, pc := range ctors {
		if isCommandAvailable(cfg.BinFor(pc.platform)) {
			scanners = append(scanners, pc.newScan(cfg))
		}
	}
	return scanners
}

func isCommandAvailable(bin string) bool {
	_, err := exec.LookPath(bin)
	return err == nil
}

// ---------------------------------------------------------------------------
// Non-interactive commands
// ---------------------------------------------------------------------------

func runNonInteractive(scanners []scanner.Scanner, cfg config.Config, list bool, resumeID, deleteID string, jsonOutput bool) {
	projects := scanner.ScanProjectsParallel(scanners)

	switch {
	case list:
		cmdList(projects, jsonOutput)
	case resumeID != "":
		cmdResume(projects, scanners, cfg, resumeID)
	case deleteID != "":
		cmdDelete(projects, scanners, deleteID)
	}
}

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

func cmdList(projects []types.Project, jsonOutput bool) {
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

	for _, p := range projects {
		fmt.Printf("%s (%d sessions)\n", p.FullPath, len(p.Sessions))
		for _, s := range p.Sessions {
			agent := ""
			if s.AgentType != "" && s.AgentType != types.TypeAgentCLI {
				agent = "[" + s.AgentType + "] "
			}
			fmt.Printf("  %-10s %s%-30s  %-18s  %-10s  %d msgs  id:%s\n",
				s.Platform.Name(), agent, s.Title, s.Model,
				types.RelativeTime(s.LastActive), s.MsgCount, s.ID)
		}
	}
}

func cmdResume(projects []types.Project, scanners []scanner.Scanner, cfg config.Config, sessionID string) {
	sess := findSession(projects, sessionID)
	if sess == nil {
		fmt.Fprintf(os.Stderr, "session not found: %s\n", sessionID)
		os.Exit(1)
	}
	execResume(sess, scanners, cfg)
}

func cmdDelete(projects []types.Project, scanners []scanner.Scanner, sessionID string) {
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
	fmt.Fprintf(os.Stderr, "no scanner found for platform %d\n", sess.Platform)
	os.Exit(1)
}

func findSession(projects []types.Project, id string) *types.Session {
	for _, p := range projects {
		for _, s := range p.Sessions {
			if s.ID == id {
				return &s
			}
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Exec helpers (replace mode)
// ---------------------------------------------------------------------------

func execResume(sess *types.Session, scanners []scanner.Scanner, cfg config.Config) {
	if err := os.Chdir(sess.ProjectPath); err != nil {
		fmt.Fprintf(os.Stderr, "chdir error: %v\n", err)
		os.Exit(1)
	}

	var sc scanner.Scanner
	for _, s := range scanners {
		if s.Platform() == sess.Platform {
			sc = s
			break
		}
	}
	if sc == nil {
		fmt.Fprintf(os.Stderr, "no scanner for platform %d\n", sess.Platform)
		os.Exit(1)
	}

	argv := sc.ResumeCmd(*sess)
	extraArgs := cfg.ArgsFor(sess.Platform)
	if len(extraArgs) > 0 {
		argv = append(argv, extraArgs...)
	}

	styles := view.NewStyles(true)
	fmt.Fprintln(os.Stderr, styles.RenderColoredCommand(sess.Platform, strings.Join(argv, " ")))

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

func execNewSession(p types.Platform, projectPath string, cfg config.Config) {
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

	styles := view.NewStyles(true)
	fmt.Fprintln(os.Stderr, styles.RenderColoredCommand(p, strings.Join(argv, " ")))

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
