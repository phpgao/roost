# roost

Manage AI coding sessions from Claude, Gemini, Codex, Copilot, and OpenCode in one TUI.

[中文文档](README-zh.md)

## Install

```bash
go install github.com/phpgao/roost@latest
```

Or build from source:

```bash
git clone https://github.com/phpgao/roost
cd roost
go install .
```

## Usage

```bash
roost                  # interactive TUI
roost --list           # list all projects and sessions
roost --list --json    # JSON output (for scripting)
roost --resume <sid>   # resume a session
roost --delete <sid>   # delete a session
```

## Keybindings

| Key | Action |
|-----|--------|
| `↑/k` `↓/j` | Move up / down |
| `g` / `G` | First / last item |
| `Enter` | Enter project / Resume session |
| `n` | New session (select agent) |
| `Esc` | Back / Exit selection / Double-tap to quit |
| `/` | Search |
| `d` | Delete current item |
| `Space` | Batch select / Toggle |
| `D` | Batch delete selected |
| `x` | Delete entire project (in session view) |
| `Tab` | Cycle platform filter (All → CL → GE → CX → CP → OC) |
| `r` | Refresh |
| `?` | Full keybinding list |
| `q` | Quit |

## Configuration

`~/.roost/roost.yaml` (auto-created on first launch)

```yaml
# resume mode:
#   replace - process replacement, returns to shell after agent exits (default)
#   suspend - subprocess mode, returns to roost TUI after agent exits
resume_mode: replace

platforms:
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
```

## Supported Platforms

| Platform | Data Directory | Resume Command |
|----------|---------------|----------------|
| Claude | `~/.claude/` | `claude --resume <sid>` |
| Gemini | `~/.gemini/` | `gemini --resume <sid>` |
| Codex | `~/.codex/` | `codex resume <sid>` |
| Copilot | `~/.copilot/` | `copilot --resume=<sid>` |
| OpenCode | SQLite DB (via `opencode db path`) | `opencode -s <sid>` |

## Tech Stack

- Go 1.26
- [Bubble Tea v2](https://github.com/charmbracelet/bubbletea)
- [Lip Gloss v2](https://github.com/charmbracelet/lipgloss)

## License

MIT
