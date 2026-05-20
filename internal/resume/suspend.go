package resume

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/phpgao/roost/internal/config"
	"github.com/phpgao/roost/internal/scanner"
	"github.com/phpgao/roost/internal/types"
)

// SuspendStrategy spawns the agent as a subprocess (tea.ExecProcess);
// returns to TUI when the agent exits.
type SuspendStrategy struct{}

// AgentExitMsg is sent when an agent subprocess exits (suspend mode).
type AgentExitMsg struct {
	Err    error
	Output string
}

// Resume uses tea.ExecProcess to spawn the agent subprocess.
func (SuspendStrategy) Resume(session types.Session, scanners []scanner.Scanner, cfg config.Config) tea.Cmd {
	var sc scanner.Scanner
	for _, s := range scanners {
		if s.Platform() == session.Platform {
			sc = s
			break
		}
	}
	if sc == nil {
		return func() tea.Msg {
			return AgentExitMsg{Err: fmt.Errorf("no scanner for platform %d", session.Platform)}
		}
	}

	argv := sc.ResumeCmd(session)
	extraArgs := cfg.ArgsFor(session.Platform)
	if len(extraArgs) > 0 {
		argv = append(argv, extraArgs...)
	}
	argv = wrapSuspendCommand(session.Platform, argv)

	cmd := exec.CommandContext(context.Background(), argv[0], argv[1:]...)
	cmd.Dir = session.ProjectPath

	return execInteractiveProcess(cmd)
}

// LaunchNew spawns a new agent subprocess via tea.ExecProcess.
func (SuspendStrategy) LaunchNew(platform types.Platform, projectPath string, cfg config.Config) tea.Cmd {
	bin := cfg.BinFor(platform)
	argv := []string{bin}
	extraArgs := cfg.ArgsFor(platform)
	if len(extraArgs) > 0 {
		argv = append(argv, extraArgs...)
	}
	argv = wrapSuspendCommand(platform, argv)

	cmd := exec.CommandContext(context.Background(), argv[0], argv[1:]...)
	cmd.Dir = projectPath

	return execInteractiveProcess(cmd)
}

func execInteractiveProcess(cmd *exec.Cmd) tea.Cmd {
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return AgentExitMsg{Err: err}
	})
}

func wrapSuspendCommand(p types.Platform, argv []string) []string {
	if len(argv) == 0 {
		return nil
	}

	wrapped := []string{
		"sh",
		"-c",
		`printf '%s\n' "$1"; shift; exec "$@"`,
		"sh",
		renderColoredCommand(p, shellQuoteArgs(argv)),
	}

	return append(wrapped, argv...)
}

func shellQuoteArgs(argv []string) string {
	quoted := make([]string, len(argv))
	for i, arg := range argv {
		quoted[i] = shellQuote(arg)
	}
	return strings.Join(quoted, " ")
}

func shellQuote(arg string) string {
	if arg == "" {
		return "''"
	}

	const safe = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_@%+=:,./-"
	for i := 0; i < len(arg); i++ {
		if !strings.ContainsRune(safe, rune(arg[i])) {
			return "'" + strings.ReplaceAll(arg, "'", `'"'"'`) + "'"
		}
	}

	return arg
}

func renderColoredCommand(p types.Platform, cmd string) string {
	// Minimal colored command rendering without full Styles dependency.
	// The app layer can override this with a styled version if needed.
	return "$ " + cmd
}
