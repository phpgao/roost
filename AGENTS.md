# roost AI 协作指南

> 本文是注入到所有对话上下文的底层记忆，**只放跨会话恒等的硬约束和强偏好**。

---

## 1. 项目概述

- **项目名称**：roost
- **主要语言/框架**：Go 1.26 / Bubble Tea v2
- **定位**：交互式 AI Session 管理 TUI，统一管理 Claude / Gemini / Codex / Copilot / OpenCode 平台会话
- **维护者**：jimmygao

---

## 2. 技术栈

| 依赖 | 用途 |
|------|------|
| `charm.land/bubbletea/v2` | TUI 框架 |
| `charm.land/lipgloss/v2` | 样式渲染 |
| `gopkg.in/yaml.v3` | 配置文件解析 |
| `modernc.org/sqlite` | OpenCode SQLite 数据库 |

---

## 3. 文件结构

```
├── main.go                 # 入口，CLI 参数解析，exec resume
├── config.go               # ~/.roost/roost.yaml 配置加载
├── model.go                # Bubble Tea Model + 状态机 + Update
├── project.go              # 项目列表 View
├── session.go              # Session 列表 View
├── delete.go               # 删除确认框 View
├── scanner.go              # Scanner 接口 + 公共工具
├── scanner_claude.go       # Claude 扫描实现
├── scanner_gemini.go       # Gemini 扫描实现
├── scanner_codex.go        # Codex 扫描实现
├── scanner_copilot.go      # Copilot 扫描实现
├── scanner_opencode.go      # OpenCode 扫描实现
├── styles.go               # Lipgloss 样式常量
├── *_test.go               # 测试文件
└── docs/superpowers/       # 设计文档
```

---

## 4. 工程要求

- TDD 优先：先写失败测试，再写最小实现
- 编译通过 + `go test ./...` 全绿才可提交
- `golangci-lint` 零 issue（unused 在 TDD 中间过程允许）
- Scanner 数据目录通过 struct 字段注入，方便测试用 `t.TempDir()` 模拟

---

## 5. 关键约定

### 路径解码

- Claude: 编码规则为 `/` → `-`、`.` → `-`、`_` → `-`，通过 `~/.claude/.claude.json` 的 `projects` key 精确匹配
- Gemini: 通过 `~/.gemini/projects.json` 映射
- Codex: 目录结构为 `sessions/YYYY/MM/DD/*.jsonl`，从 `session_meta` 行提取 `cwd`
- Copilot: 通过 `~/.copilot/session-state/<id>/workspace.yaml` 获取 `cwd`
- OpenCode: 通过 SQLite 数据库 `session` 表和 `project` 表关联获取 `worktree`

### Resume

- 退出 TUI 后用 `syscall.Exec` 替换进程（replace 模式）
- 或用 `tea.Exec` 启动子进程（suspend 模式），退出后回到 TUI
- exec 前 `os.Chdir` 到项目目录
- 从 `~/.roost/roost.yaml` 读取各平台额外参数

### 配置文件

- 路径：`~/.roost/roost.yaml`
- 不存在时自动创建空模板
- 支持配置：`resume_mode`、`platforms[].bin`、`platforms[].data_dir`、`platforms[].args`

---

## 6. CLI 参数

```
roost                    # 交互模式（TUI）
roost --list             # 列出所有项目和 session
roost --list --json      # JSON 格式输出
roost --resume <sid>     # 直接 resume
roost --delete <sid>     # 直接删除
```

---

## 7. 输出规范

**所有程序输出必须为英文**，包括：
- UI 文本（帮助页、状态提示、错误信息）
- CLI 输出（`--list`、`--list --json`）
- 日志和调试信息

例外：`AGENTS.md`、`README-zh.md` 等文档可以中文。
