package main

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
// Default Binary Names (used by Config.BinFor)
// ---------------------------------------------------------------------------

const (
	defaultBinCodeBuddy = "codebuddy"
	defaultBinClaude    = "claude"
	defaultBinGemini    = "gemini"
	defaultBinCodex     = "codex"
	defaultBinCopilot   = "copilot"
	defaultBinOpenCode  = "opencode"
)

// ---------------------------------------------------------------------------
// Default Data Directories (used by Config.DataDirFor)
// ---------------------------------------------------------------------------

const (
	defaultDirCodeBuddy = ".codebuddy"
	defaultDirClaude    = ".claude"
	defaultDirGemini    = ".gemini"
	defaultDirCodex     = ".codex"
	defaultDirCopilot   = ".copilot"
)

// ---------------------------------------------------------------------------
// Config Template
// ---------------------------------------------------------------------------

// defaultConfigTemplate is written on first launch when ~/.roost/roost.yaml doesn't exist.
const defaultConfigTemplate = `# roost configuration

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

type screen int

const (
	screenProject screen = iota
	screenSession
	screenDeleteConfirm
)

type deleteTarget int

const (
	deleteTargetSession deleteTarget = iota
	deleteTargetProject
	deleteTargetBatch
)

// ---------------------------------------------------------------------------
// TUI Key Constants
// ---------------------------------------------------------------------------

const (
	keyEsc   = "esc"
	keyCtrlC = "ctrl+c"
	keyEnter = "enter"
	keySpace = "space"
	keyDown  = "down"
	keyTab   = "tab"
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
	nameCodeBuddy = "CodeBuddy"
	nameClaude    = "Claude"
	nameGemini    = "Gemini"
	nameCodex     = "Codex"
	nameCopilot   = "Copilot"
	nameOpenCode  = "OpenCode"
)

// ---------------------------------------------------------------------------
// Scanner Common Constants
// ---------------------------------------------------------------------------

const (
	roleUser      = "user"
	roleAssistant = "assistant"
	untitledTitle = "(untitled)"
	flagResume    = "--resume"

	// scanBufferSize is the max token size for bufio.Scanner (1 MB).
	scanBufferSize = 1024 * 1024
)

// ---------------------------------------------------------------------------
// Session & Message Type Constants
// ---------------------------------------------------------------------------

const (
	typeAgentCLI    = "cli"
	typeMessage     = "message" // CodeBuddy message type
	typeGemini      = "gemini"  // Gemini platform type
	typeSessionMeta = "session_meta"
	typeEventMsg    = "event_msg"
	typeUserMessage = "user_message"
	codexCmdResume  = "resume"
)

// ---------------------------------------------------------------------------
// Style Constants
// ---------------------------------------------------------------------------

// maxWidth caps the layout width so the UI stays readable on wide terminals.
const maxWidth = 140

// Apple System Colors (Dark Mode) — True Color
const (
	colorBlue     = "#0A84FF" // system-blue dark
	colorRed      = "#FF453A" // system-red dark
	colorYellow   = "#FFD60A" // system-yellow dark
	colorOrange   = "#be7763" // Claude
	colorGreen    = "#00ad87" // CodeBuddy
	colorCopilot  = "#ac52a8" // Copilot
	colorCodex    = "#9b6ef3" // Codex
	colorGemini   = "#0067c2" // Gemini
	colorOpenCode = "#686868" // OpenCode

	// Text hierarchy
	colorLabelPrimary    = "#FFFFFF"                  // label-primary
	colorLabelTertiary   = "rgba(235, 235, 245, 0.6)" // label-secondary (60%)
	colorLabelQuaternary = "rgba(235, 235, 245, 0.3)" // label-tertiary

	// Backgrounds
	colorBgSelected = "rgba(10, 132, 255, 0.2)" // semi-transparent blue
)
