# roost

一个 TUI，统一管理 Claude、Gemini、Codex、Copilot、OpenCode 平台的 AI coding session。

[English](README.md)

## 安装

```bash
go install github.com/phpgao/roost@latest
```

或本地构建：

```bash
git clone https://github.com/phpgao/roost
cd roost
go install .
```

## 使用

```bash
roost                  # 交互式 TUI
roost --list           # 列出所有项目和 session
roost --list --json    # JSON 输出（给脚本用）
roost --resume <sid>   # 直接 resume
roost --delete <sid>   # 直接删除
```

## 快捷键

| 按键 | 效果 |
|------|------|
| `↑/k` `↓/j` | 上下移动 |
| `g` / `G` | 首项 / 末项 |
| `Enter` | 进入项目 / Resume session |
| `Esc` | 返回 / 退出选择 / 双击退出 |
| `/` | 搜索 |
| `d` | 删除当前项 |
| `Space` | 批量选择 / 切换选中 |
| `D` | 批量删除 |
| `x` | 删除整个项目（session 列表里） |
| `Tab` | 切换平台过滤 (All → CL → GE → CX → CP → OC) |
| `r` | 刷新 |
| `?` | 完整快捷键列表 |
| `q` | 退出 |

## 配置

`~/.roost/roost.yaml`（首次启动自动创建）

```yaml
# resume mode:
#   replace - 进程替换，agent 退出后回到 shell（默认）
#   suspend - 子进程模式，agent 退出后回到 roost TUI
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

## 支持平台

| 平台 | 数据目录 | Resume 命令 |
|------|----------|-------------|
| Claude | `~/.claude/` | `claude --resume <sid>` |
| Gemini | `~/.gemini/` | `gemini --resume <sid>` |
| Codex | `~/.codex/` | `codex resume <sid>` |
| Copilot | `~/.copilot/` | `copilot --resume=<sid>` |
| OpenCode | SQLite DB（通过 `opencode db path` 获取） | `opencode -s <sid>` |

## 技术栈

- Go 1.26
- [Bubble Tea v2](https://github.com/charmbracelet/bubbletea)
- [Lip Gloss v2](https://github.com/charmbracelet/lipgloss)

## License

MIT
