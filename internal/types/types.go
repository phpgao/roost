// Package types defines shared domain types and constants for roost.
package types

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/mattn/go-runewidth"
)

// ---------------------------------------------------------------------------
// Resume Mode
// ---------------------------------------------------------------------------

// ResumeMode controls TUI resume behavior.
type ResumeMode string

const (
	// ResumeModeReplace replaces the process (syscall.Exec); returns to shell after agent exits.
	ResumeModeReplace ResumeMode = "replace"
	// ResumeModeSuspend spawns a subprocess (tea.Exec); returns to TUI after agent exits.
	ResumeModeSuspend ResumeMode = "suspend"
)

// ---------------------------------------------------------------------------
// Default Binary Names
// ---------------------------------------------------------------------------

const (
	DefaultBinCodeBuddy = "codebuddy"
	DefaultBinClaude    = "claude"
	DefaultBinGemini    = "gemini"
	DefaultBinCodex     = "codex"
	DefaultBinCopilot   = "copilot"
	DefaultBinOpenCode  = "opencode"
)

// ---------------------------------------------------------------------------
// Default Data Directories
// ---------------------------------------------------------------------------

const (
	DefaultDirCodeBuddy = ".codebuddy"
	DefaultDirClaude    = ".claude"
	DefaultDirGemini    = ".gemini"
	DefaultDirCodex     = ".codex"
	DefaultDirCopilot   = ".copilot"
)

// ---------------------------------------------------------------------------
// Config Template
// ---------------------------------------------------------------------------

// DefaultConfigTemplate is written on first launch when ~/.roost/roost.yaml doesn't exist.
const DefaultConfigTemplate = `# roost configuration

# resume mode:
#   replace - process replacement, returns to shell after agent exits (default)
#   suspend - subprocess mode, returns to roost TUI after agent exits
resume_mode: replace

platforms:
  codebuddy:
    # args: [-y]
  claude:
    bin: claude
    data_dir: .claude
    # args: [--dangerously-skip-permissions]
  gemini:
    bin: gemini
    data_dir: .gemini
    # args: [-y]
  codex:
    bin: codex
    data_dir: .codex
    # args: [--full-auto]
  copilot:
    bin: copilot
    data_dir: .copilot
    # args: []
  opencode:
    bin: opencode
    # args: []
`

// ---------------------------------------------------------------------------
// Screen & Delete Target (TUI state machine enums)
// ---------------------------------------------------------------------------

// Screen identifies the current TUI screen.
type Screen int

const (
	ScreenProject Screen = iota
	ScreenSession
	ScreenConfirm
	ScreenNewAgent
	ScreenHelp
)

// DeleteTarget identifies what is being deleted.
type DeleteTarget int

const (
	DeleteTargetSession DeleteTarget = iota
	DeleteTargetProject
	DeleteTargetBatch
)

// ---------------------------------------------------------------------------
// TUI Key Constants
// ---------------------------------------------------------------------------

const (
	KeyEsc   = "esc"
	KeyCtrlC = "ctrl+c"
	KeyEnter = "enter"
	KeySpace = "space"
	KeyDown  = "down"
	KeyTab   = "tab"
)

// ---------------------------------------------------------------------------
// Platform Enum & Display Names
// ---------------------------------------------------------------------------

// Platform identifies an AI platform.
type Platform int

const (
	PlatformCodeBuddy Platform = iota
	PlatformClaude
	PlatformGemini
	PlatformCodex
	PlatformCopilot
	PlatformOpenCode
)

const (
	NameCodeBuddy = "CodeBuddy"
	NameClaude    = "Claude"
	NameGemini    = "Gemini"
	NameCodex     = "Codex"
	NameCopilot   = "Copilot"
	NameOpenCode  = "OpenCode"
)

// Icon returns the bullet character for the platform.
func (p Platform) Icon() string {
	switch p {
	case PlatformCodeBuddy, PlatformClaude, PlatformGemini, PlatformCodex, PlatformCopilot, PlatformOpenCode:
		return "●"
	default:
		return "○"
	}
}

// ShortName returns the 2-letter abbreviation.
func (p Platform) ShortName() string {
	switch p {
	case PlatformCodeBuddy:
		return "CB"
	case PlatformClaude:
		return "CL"
	case PlatformGemini:
		return "GE"
	case PlatformCodex:
		return "CX"
	case PlatformCopilot:
		return "Co"
	case PlatformOpenCode:
		return "OC"
	default:
		return "??"
	}
}

// Name returns the full display name.
func (p Platform) Name() string {
	switch p {
	case PlatformCodeBuddy:
		return NameCodeBuddy
	case PlatformClaude:
		return NameClaude
	case PlatformGemini:
		return NameGemini
	case PlatformCodex:
		return NameCodex
	case PlatformCopilot:
		return NameCopilot
	case PlatformOpenCode:
		return NameOpenCode
	default:
		return "Unknown"
	}
}

// ---------------------------------------------------------------------------
// Session & Project Types
// ---------------------------------------------------------------------------

// Session represents a single AI session record.
type Session struct {
	ID          string
	Platform    Platform
	AgentType   string
	Title       string
	Model       string
	LastActive  time.Time
	MsgCount    int
	SizeBytes   int64
	ProjectDir  string
	FilePath    string
	ResumeArg   string
	ProjectPath string
}

// Project represents all sessions under a working directory.
type Project struct {
	Name     string
	FullPath string
	Sessions []Session
}

// LastActive returns the most recent activity time across all sessions.
func (p *Project) LastActive() time.Time {
	var t time.Time
	for _, s := range p.Sessions {
		if s.LastActive.After(t) {
			t = s.LastActive
		}
	}
	return t
}

// ---------------------------------------------------------------------------
// Scanner Common Constants
// ---------------------------------------------------------------------------

const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
	UntitledTitle = "(untitled)"
	FlagResume    = "--resume"

	// ScanBufferSize is the max token size for bufio.Scanner (1 MB).
	ScanBufferSize = 1024 * 1024
)

// ---------------------------------------------------------------------------
// Session & Message Type Constants
// ---------------------------------------------------------------------------

const (
	TypeAgentCLI    = "cli"
	TypeMessage     = "message" // CodeBuddy message type
	TypeGemini      = "gemini"  // Gemini platform type
	TypeSessionMeta = "session_meta"
	TypeEventMsg    = "event_msg"
	TypeUserMessage = "user_message"
	CodexCmdResume  = "resume"
)

// ---------------------------------------------------------------------------
// Style Constants
// ---------------------------------------------------------------------------

// MaxWidth caps the layout width so the UI stays readable on wide terminals.
const MaxWidth = 140

// Apple System Colors (Dark Mode) — True Color
const (
	ColorBlue     = "#0A84FF" // system-blue dark
	ColorRed      = "#FF453A" // system-red dark
	ColorYellow   = "#FFD60A" // system-yellow dark
	ColorOrange   = "#be7763" // Claude
	ColorGreen    = "#00ad87" // CodeBuddy
	ColorCopilot  = "#ac52a8" // Copilot
	ColorCodex    = "#9b6ef3" // Codex
	ColorGemini   = "#0067c2" // Gemini
	ColorOpenCode = "#686868" // OpenCode

	// Text hierarchy
	ColorLabelPrimary    = "#FFFFFF"                  // label-primary
	ColorLabelTertiary   = "rgba(235, 235, 245, 0.6)" // label-secondary (60%)
	ColorLabelQuaternary = "rgba(235, 235, 245, 0.3)" // label-tertiary

	// Backgrounds
	ColorBgSelected = "rgba(10, 132, 255, 0.2)" // semi-transparent blue
)

// ---------------------------------------------------------------------------
// Utility Functions
// ---------------------------------------------------------------------------

// Truncate truncates a string to at most n runes (head, append …).
func Truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "…"
}

// TruncateKeepEnd truncates to visual width n, keeping the end (prepend …).
func TruncateKeepEnd(s string, n int) string {
	if StringWidth(s) <= n {
		return s
	}
	const ellipsis = "…"
	budget := n - runewidth.StringWidth(ellipsis)
	if budget <= 0 {
		return ellipsis
	}
	runes := []rune(s)
	w, start := 0, len(runes)
	for i := len(runes) - 1; i >= 0; i-- {
		rw := runewidth.RuneWidth(runes[i])
		if w+rw > budget {
			break
		}
		w += rw
		start = i
	}
	return ellipsis + string(runes[start:])
}

// TruncateWidth truncates to visual width n, keeping the head (append …).
func TruncateWidth(s string, n int) string {
	if StringWidth(s) <= n {
		return s
	}
	const ellipsis = "…"
	budget := n - runewidth.StringWidth(ellipsis)
	if budget <= 0 {
		return ellipsis
	}
	w, keep := 0, 0
	for _, r := range s {
		rw := runewidth.RuneWidth(r)
		if w+rw > budget {
			break
		}
		w += rw
		keep++
	}
	return string([]rune(s)[:keep]) + ellipsis
}

// StringWidth returns the visual width of s.
func StringWidth(s string) int {
	return runewidth.StringWidth(s)
}

// ProjectShortName takes the last two segments of an absolute path as short name.
func ProjectShortName(fullPath string) string {
	fullPath = strings.TrimRight(fullPath, "/")
	if fullPath == "" {
		return "/"
	}
	dir := filepath.Dir(fullPath)
	base := filepath.Base(fullPath)
	parent := filepath.Base(dir)
	if parent == "." || parent == "/" || parent == fullPath {
		return base
	}
	return parent + "/" + base
}

// RelativeTime returns a human-readable relative time string.
func RelativeTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return t.Format("2006-01-02")
	}
}

// Clamp limits v to [lo, hi].
func Clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
