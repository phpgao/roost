package main

import (
	"context"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

// agentExitMsg agent 子进程退出后发送的消息（suspend 模式）
type agentExitMsg struct {
	err    error
	output string
}

// scanDoneMsg 扫描完成消息
type scanDoneMsg struct {
	projects []Project
}

// deleteDoneMsg 删除完成消息
type deleteDoneMsg struct {
	err error
}

// spinnerTickMsg drives the loading animation
type spinnerTickMsg struct{}

// escHintTimeoutMsg is sent 2 seconds after escHint is set, to auto-clear it
type escHintTimeoutMsg struct {
	gen int // generation ID to avoid stale timeouts
}

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Model 是 Bubble Tea 主 Model
type Model struct {
	scanners           []Scanner
	installedPlatforms []Platform // platforms with binary available on this machine
	cfg                Config

	projects         []Project
	filteredProjects []Project

	screen screen
	cursor int

	selectedProject  *Project
	filteredSessions []Session

	searching   bool
	searchQuery string

	delTarget  deleteTarget
	delSession *Session
	delProject *Project

	loading   bool
	err       error
	errOutput string

	width  int
	height int

	// 平台过滤：-1 表示 All，0/1/2 对应 Platform 枚举
	// 平台过滤：-1 = 全部；0~N-1 = installedPlatforms[index]
	platformFilter int

	// 帮助页显示
	showHelp bool

	// 双击 Esc 退出：记录上次 Esc 时间
	lastEscTime time.Time
	escHint     bool // show "press Esc again to quit" hint after first Esc
	escHintGen  int  // generation ID for timeout messages, to avoid stale timeouts

	// 批量选择模式
	selecting   bool
	selectedSet map[string]bool // key = FullPath(项目) 或 Session.ID(session)

	// 记住各页面的光标位置，Esc 返回时恢复
	projectCursor int
	sessionCursor int

	// spinner animation index
	spinnerIdx int

	// 非 nil 时 main 执行 exec resume
	resumeSession *Session

	// 新建 session：agent 选择页光标 + 选中的 platform
	newAgentCursor    int
	newAgentPlatform  Platform
	newSessionPending bool // true = 退出 TUI 后启动新 session
}

func newModel(scanners []Scanner) Model {
	installedPlatforms := make([]Platform, len(scanners))
	for i, sc := range scanners {
		installedPlatforms[i] = sc.Platform()
	}
	sort.Slice(installedPlatforms, func(i, j int) bool {
		return installedPlatforms[i].Name() < installedPlatforms[j].Name()
	})
	return Model{
		scanners:           scanners,
		installedPlatforms: installedPlatforms,
		loading:            true,
		width:              80,
		height:             24,
		platformFilter:     -1, // All platforms
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.scanCmd(), spinnerTick())
}

func spinnerTick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(_ time.Time) tea.Msg {
		return spinnerTickMsg{}
	})
}

func (m *Model) scanCmd() tea.Cmd {
	scanners := m.scanners
	return func() tea.Msg {
		projects := ScanProjectsParallel(scanners)
		return scanDoneMsg{projects: projects}
	}
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case spinnerTickMsg:
		if m.loading {
			m.spinnerIdx = (m.spinnerIdx + 1) % len(spinnerFrames)
			return m, spinnerTick()
		}
		return m, nil

	case scanDoneMsg:
		m.loading = false
		m.projects = msg.projects
		m.filteredProjects = msg.projects
		m.cursor = 0
		return m, nil

	case deleteDoneMsg:
		if msg.err != nil {
			m.err = msg.err
			m.errOutput = ""
		}
		m.loading = true
		m.screen = screenProject
		m.cursor = 0
		m.selectedProject = nil
		m.selecting = false
		m.selectedSet = nil
		return m, tea.Batch(m.scanCmd(), spinnerTick())

	case agentExitMsg:
		// suspend 模式：agent 退出后回到 TUI，刷新 session 列表
		if msg.err != nil {
			m.err = msg.err
			m.errOutput = msg.output
		} else {
			m.err = nil
			m.errOutput = ""
		}
		m.loading = true
		return m, tea.Batch(m.scanCmd(), spinnerTick())

	case escHintTimeoutMsg:
		// 2 秒后自动清除 escHint（仅当 generation 匹配时）
		if msg.gen == m.escHintGen {
			m.escHint = false
		}
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// 非 esc 按键时清掉退出提示
	if msg.String() != keyEsc {
		m.escHint = false
	}
	// 错误页优先处理，避免继续按原 screen 路由导致按键失效。
	if m.err != nil {
		switch msg.String() {
		case "q", keyCtrlC:
			return m, tea.Quit
		case keyEsc, keyEnter:
			m.err = nil
			m.errOutput = ""
			return m, nil
		default:
			return m, nil
		}
	}
	// 帮助页拦截：任意键关闭
	if m.showHelp {
		switch msg.String() {
		case "q", keyCtrlC:
			return m, tea.Quit
		default:
			m.showHelp = false
			return m, nil
		}
	}
	if m.searching {
		return m.handleSearchKey(msg)
	}
	switch m.screen {
	case screenProject:
		return m.handleProjectKey(msg)
	case screenSession:
		return m.handleSessionKey(msg)
	case screenDeleteConfirm:
		return m.handleDeleteKey(msg)
	case screenNewAgent:
		return m.handleNewAgentKey(msg)
	}
	return m, nil
}

func (m *Model) handleSearchKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyEsc:
		m.searching = false
		m.searchQuery = ""
		m.applyFilter()
		return m, nil
	case "backspace", "ctrl+h":
		if m.searchQuery != "" {
			runes := []rune(m.searchQuery)
			m.searchQuery = string(runes[:len(runes)-1])
			m.applyFilter()
		}
		return m, nil
	case keyEnter:
		m.searching = false
		return m, nil
	default:
		if len(msg.String()) == 1 {
			m.searchQuery += msg.String()
			m.applyFilter()
		}
		return m, nil
	}
}

func (m *Model) applyFilter() {
	q := strings.ToLower(m.searchQuery)

	if m.screen == screenProject {
		var filtered []Project
		for _, p := range m.projects {
			// 平台过滤：platformFilter 是 installedPlatforms 的索引
			if m.platformFilter >= 0 {
				target := m.installedPlatforms[m.platformFilter]
				hasPlat := false
				for _, s := range p.Sessions {
					if s.Platform == target {
						hasPlat = true
						break
					}
				}
				if !hasPlat {
					continue
				}
			}
			// 关键词过滤
			if q != "" && !strings.Contains(strings.ToLower(p.FullPath), q) {
				continue
			}
			filtered = append(filtered, p)
		}
		m.filteredProjects = filtered
	} else if m.selectedProject != nil {
		var filtered []Session
		for _, s := range m.selectedProject.Sessions {
			// 平台过滤：platformFilter 是 installedPlatforms 的索引
			if m.platformFilter >= 0 && s.Platform != m.installedPlatforms[m.platformFilter] {
				continue
			}
			// 关键词过滤
			if q != "" &&
				!strings.Contains(strings.ToLower(s.Title), q) &&
				!strings.Contains(strings.ToLower(s.Model), q) {
				continue
			}
			filtered = append(filtered, s)
		}
		m.filteredSessions = filtered
	}
	m.cursor = 0
}

func (m *Model) handleProjectKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	list := m.filteredProjects
	switch msg.String() {
	case "q", keyCtrlC:
		return m, tea.Quit
	case keyEsc:
		if m.selecting {
			m.selecting = false
			m.selectedSet = nil
			return m, nil
		}
		// 双击 Esc 退出（2s 内连续按两次）
		if m.escHint && time.Since(m.lastEscTime) < 2*time.Second {
			return m, tea.Quit
		}
		m.escHint = true
		m.lastEscTime = time.Now()
		m.escHintGen++
		return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
			return escHintTimeoutMsg{gen: m.escHintGen}
		})
	case " ", keySpace:
		if len(list) > 0 {
			if !m.selecting {
				m.selecting = true
				m.selectedSet = make(map[string]bool)
			}
			key := list[m.cursor].FullPath
			if m.selectedSet[key] {
				delete(m.selectedSet, key)
			} else {
				m.selectedSet[key] = true
			}
		}
	case "D":
		if m.selecting && len(m.selectedSet) > 0 {
			m.delTarget = deleteTargetBatch
			m.screen = screenDeleteConfirm
		}
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(list)-1 {
			m.cursor++
		}
	case "g":
		m.cursor = 0
	case "G":
		if len(list) > 0 {
			m.cursor = len(list) - 1
		}
	case "r":
		m.loading = true
		return m, tea.Batch(m.scanCmd(), spinnerTick())
	case "?":
		m.showHelp = !m.showHelp
	case keyTab:
		m.platformFilter++
		if m.platformFilter >= len(m.installedPlatforms) {
			m.platformFilter = -1
		}
		m.applyFilter()
	case keyEnter:
		if len(list) > 0 {
			p := list[m.cursor]
			m.selectedProject = &p
			m.filteredSessions = p.Sessions
			sort.Slice(m.filteredSessions, func(i, j int) bool {
				return m.filteredSessions[i].LastActive.After(m.filteredSessions[j].LastActive)
			})
			m.projectCursor = m.cursor
			m.screen = screenSession
			m.cursor = clamp(m.sessionCursor, 0, len(m.filteredSessions)-1)
			m.searchQuery = ""
			m.selecting = false
			m.selectedSet = nil
		}
	case "d":
		if !m.selecting && len(list) > 0 {
			m.delProject = new(list[m.cursor])
			m.delTarget = deleteTargetProject
			m.screen = screenDeleteConfirm
		}
	case "/":
		m.searching = true
	}
	return m, nil
}

func (m *Model) handleSessionKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	list := m.filteredSessions
	switch msg.String() {
	case "q", keyCtrlC:
		return m, tea.Quit
	case keyEsc:
		if m.selecting {
			m.selecting = false
			m.selectedSet = nil
			return m, nil
		}
		m.sessionCursor = m.cursor
		m.screen = screenProject
		m.searchQuery = ""
		m.selecting = false
		m.selectedSet = nil
		m.applyFilter()
		m.cursor = clamp(m.projectCursor, 0, len(m.filteredProjects)-1)
	case " ", keySpace:
		if len(list) > 0 {
			if !m.selecting {
				m.selecting = true
				m.selectedSet = make(map[string]bool)
			}
			key := list[m.cursor].ID
			if m.selectedSet[key] {
				delete(m.selectedSet, key)
			} else {
				m.selectedSet[key] = true
			}
		}
	case "D":
		if m.selecting && len(m.selectedSet) > 0 {
			m.delTarget = deleteTargetBatch
			m.screen = screenDeleteConfirm
		}
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case keyDown, "j":
		if m.cursor < len(list)-1 {
			m.cursor++
		}
	case "g":
		m.cursor = 0
	case "G":
		if len(list) > 0 {
			m.cursor = len(list) - 1
		}
	case "r":
		m.loading = true
		return m, tea.Batch(m.scanCmd(), spinnerTick())
	case "?":
		m.showHelp = !m.showHelp
	case keyTab:
		m.platformFilter++
		if m.platformFilter >= len(m.installedPlatforms) {
			m.platformFilter = -1
		}
		m.applyFilter()
	case keyEnter:
		if len(list) > 0 {
			sess := list[m.cursor]
			if m.cfg.GetResumeMode() == ResumeModeSuspend {
				// suspend 模式：用 tea.Exec 启动子进程，退出后回到 TUI
				cmd := m.suspendResume(&sess)
				return m, cmd
			}
			// replace 模式（默认）：退出 TUI 后 syscall.Exec 替换进程
			m.resumeSession = &sess
			return m, tea.Quit
		}
	case "d":
		if !m.selecting && len(list) > 0 {
			m.delSession = new(list[m.cursor])
			m.delTarget = deleteTargetSession
			m.screen = screenDeleteConfirm
		}
	case "x":
		if m.selectedProject != nil {
			m.delProject = m.selectedProject
			m.delTarget = deleteTargetProject
			m.screen = screenDeleteConfirm
		}
	case "n", "N":
		if len(m.installedPlatforms) > 0 {
			m.screen = screenNewAgent
			m.newAgentCursor = 0
		}
	case "/":
		m.searching = true
	}
	return m, nil
}

func (m *Model) handleDeleteKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyEnter:
		cmd := m.doDelete()
		return m, cmd
	case "esc", "n":
		switch m.delTarget {
		case deleteTargetBatch:
			if m.selectedProject != nil {
				m.screen = screenSession
			} else {
				m.screen = screenProject
			}
		case deleteTargetSession:
			m.screen = screenSession
		default:
			m.screen = screenProject
		}
		m.delSession = nil
		m.delProject = nil
	case "q", keyCtrlC:
		return m, tea.Quit
	}
	return m, nil
}

func (m *Model) doDelete() tea.Cmd {
	target := m.delTarget
	sess := m.delSession
	proj := m.delProject
	scanners := m.scanners
	selectedSet := m.selectedSet
	// 收集批量删除需要的上下文
	var batchSessions []Session
	var batchProjects []Project
	if target == deleteTargetBatch {
		if m.screen == screenDeleteConfirm && m.selectedProject != nil {
			// session 列表批量删除
			for _, s := range m.filteredSessions {
				if selectedSet[s.ID] {
					batchSessions = append(batchSessions, s)
				}
			}
		} else {
			// 项目列表批量删除
			for _, p := range m.projects {
				if selectedSet[p.FullPath] {
					batchProjects = append(batchProjects, p)
				}
			}
		}
	}

	return func() tea.Msg {
		var err error
		switch target {
		case deleteTargetSession:
			if sess != nil {
				for _, sc := range scanners {
					if sc.Platform() == sess.Platform {
						err = sc.DeleteSession(*sess)
						break
					}
				}
			}
		case deleteTargetProject:
			if proj != nil {
				byPlatform := make(map[Platform][]Session)
				for _, s := range proj.Sessions {
					byPlatform[s.Platform] = append(byPlatform[s.Platform], s)
				}
				for _, sc := range scanners {
					if sessions, ok := byPlatform[sc.Platform()]; ok {
						tempProj := Project{
							Name:     proj.Name,
							FullPath: proj.FullPath,
							Sessions: sessions,
						}
						if e := sc.DeleteProject(tempProj); e != nil {
							err = e
						}
					}
				}
			}
		case deleteTargetBatch:
			// 批量删除 sessions
			for _, s := range batchSessions {
				for _, sc := range scanners {
					if sc.Platform() == s.Platform {
						if e := sc.DeleteSession(s); e != nil {
							err = e
						}
						break
					}
				}
			}
			// 批量删除 projects
			for _, p := range batchProjects {
				byPlatform := make(map[Platform][]Session)
				for _, s := range p.Sessions {
					byPlatform[s.Platform] = append(byPlatform[s.Platform], s)
				}
				for _, sc := range scanners {
					if sessions, ok := byPlatform[sc.Platform()]; ok {
						tempProj := Project{
							Name:     p.Name,
							FullPath: p.FullPath,
							Sessions: sessions,
						}
						if e := sc.DeleteProject(tempProj); e != nil {
							err = e
						}
					}
				}
			}
		}
		return deleteDoneMsg{err: err}
	}
}

func (m *Model) View() tea.View {
	var s string
	if m.loading {
		frame := spinnerFrames[m.spinnerIdx]
		s = styleTitle.Render("  roost") + "\n" +
			styleSeparator.Render(strings.Repeat("─", m.width)) + "\n\n" +
			fmt.Sprintf("  %s %s", styleSubtle.Render(frame), styleIndicator.Render("scanning sessions…"))
	} else if m.err != nil {
		s = fmt.Sprintf("  error: %v", m.err)
		if m.errOutput != "" {
			s += "\n\n" + indentBlock(m.errOutput, "  ")
		}
		s += "\n\n  press Esc to continue, q to quit"
	} else if m.showHelp {
		s = renderHelpView(m)
	} else {
		switch m.screen {
		case screenProject:
			s = renderProjectView(m)
		case screenSession:
			s = renderSessionView(m)
		case screenDeleteConfirm:
			s = renderDeleteView(m)
		case screenNewAgent:
			s = renderNewAgentView(m)
		}
	}
	v := tea.NewView(s)
	v.AltScreen = true
	return v
}

// platformFilterLabel returns the current platform filter display text.
// It only considers installed platforms; the filter index maps into installedPlatforms.
func (m *Model) platformFilterLabel() string {
	if m.platformFilter < 0 || m.platformFilter >= len(m.installedPlatforms) {
		return "All"
	}
	p := m.installedPlatforms[m.platformFilter]
	return platformIcon(p) + " " + p.Name()
}

func renderHelpView(m *Model) string {
	var sb strings.Builder
	sb.WriteString(styleTitle.Render("  roost — Keyboard Shortcuts") + "\n\n")

	sep := styleSeparator.Render(strings.Repeat("─", m.width))

	// helper to render a key-description pair
	key := func(k string) string { return styleKey.Render(k) }
	desc := func(d string) string { return styleSubtle.Render(d) }
	row := func(k, d string) { fmt.Fprintf(&sb, "  %-12s %s\n", key(k), desc(d)) }

	sb.WriteString("  Navigation\n")
	sb.WriteString(sep + "\n")
	row("↑/k", "Move up")
	row("↓/j", "Move down")
	row("g", "Jump to first")
	row("G", "Jump to last")
	row("Enter", "Open / Resume")
	row("Esc", "Back / Exit select mode")
	sb.WriteString("\n")

	sb.WriteString("  Actions\n")
	sb.WriteString(sep + "\n")
	row("/", "Search (Esc to cancel)")
	row("d", "Delete session/project")
	row("x", "Delete entire project (in session view)")
	row("r", "Refresh (re-scan)")
	row("n", "New session (select agent)")
	// Build dynamic Tab description from installed platforms
	tabDesc := "Cycle platform filter (All"
	for _, p := range m.installedPlatforms {
		tabDesc += " → " + platformIcon(p)
	}
	tabDesc += ")"
	row("Tab", tabDesc)
	row("?", "Toggle this help")
	sb.WriteString("\n")

	sb.WriteString("  Batch Select\n")
	sb.WriteString(sep + "\n")
	row("Space", "Enter select mode / toggle item")
	row("D", "Delete all selected items")
	row("Esc", "Exit select mode")
	sb.WriteString("\n")

	sb.WriteString("  Quit\n")
	sb.WriteString(sep + "\n")
	row("q / Ctrl+C", "Quit")
	row("Esc Esc", "Quit (main screen, press twice within 2s)")
	sb.WriteString("\n")
	sb.WriteString(desc("Press ? or Esc to close this help."))

	return sb.String()
}

// calcViewHeight 计算列表视口高度。固定占用：title(1) + separator(1) + footer(2) = 4 行。
func calcViewHeight(termHeight int) int {
	h := termHeight - 4
	if h < 3 {
		return 3
	}
	return h
}

// calcViewport 计算可见列表的 [start, end) 范围，使 cursor 始终在视口内
func calcViewport(total, cursor, viewHeight int) (int, int) {
	if viewHeight <= 0 || total == 0 {
		return 0, 0
	}
	if total <= viewHeight {
		return 0, total
	}
	// 尽量让 cursor 居中
	start := cursor - viewHeight/2
	if start < 0 {
		start = 0
	}
	end := start + viewHeight
	if end > total {
		end = total
		start = end - viewHeight
	}
	return start, end
}

// clamp 限制 v 在 [lo, hi] 范围内
func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func indentBlock(s, prefix string) string {
	if s == "" {
		return ""
	}
	return prefix + strings.ReplaceAll(s, "\n", "\n"+prefix)
}

func wrapSuspendCommand(p Platform, argv []string) []string {
	if len(argv) == 0 {
		return nil
	}

	wrapped := []string{
		"sh",
		"-c",
		`printf '%s\n' "$1"; shift; exec "$@"`,
		"sh",
		RenderColoredCommand(p, shellQuoteArgs(argv)),
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

func execInteractiveProcess(cmd *exec.Cmd) tea.Cmd {
	// Interactive agent CLIs must inherit the real TTY. If Stdout/Stderr are
	// wrapped with a generic io.Writer, exec.Cmd falls back to pipes and the
	// child process will fail terminal checks like "stdout is not a terminal".
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		return agentExitMsg{err: err}
	})
}

// suspendResume 使用 tea.Exec 启动 agent 子进程，退出后回到 TUI
func (m *Model) suspendResume(session *Session) tea.Cmd {
	var scanner Scanner
	for _, sc := range m.scanners {
		if sc.Platform() == session.Platform {
			scanner = sc
			break
		}
	}
	if scanner == nil {
		return func() tea.Msg {
			return agentExitMsg{err: fmt.Errorf("no scanner for platform %d", session.Platform)}
		}
	}

	argv := scanner.ResumeCmd(*session)
	extraArgs := m.cfg.ArgsFor(session.Platform)
	if len(extraArgs) > 0 {
		argv = append(argv, extraArgs...)
	}
	argv = wrapSuspendCommand(session.Platform, argv)

	cmd := exec.CommandContext(context.Background(), argv[0], argv[1:]...)
	cmd.Dir = session.ProjectPath

	return execInteractiveProcess(cmd)
}

// handleNewAgentKey 处理 agent 选择页的按键
func (m *Model) handleNewAgentKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case keyEsc:
		m.screen = screenSession
	case "up", "k":
		if m.newAgentCursor > 0 {
			m.newAgentCursor--
		}
	case "down", "j":
		if m.newAgentCursor < len(m.installedPlatforms)-1 {
			m.newAgentCursor++
		}
	case "g":
		m.newAgentCursor = 0
	case "G":
		if len(m.installedPlatforms) > 0 {
			m.newAgentCursor = len(m.installedPlatforms) - 1
		}
	case keyEnter:
		p := m.installedPlatforms[m.newAgentCursor]
		cmd := m.launchNewSession(p)
		return m, cmd
	}
	return m, nil
}

// launchNewSession 启动新 session（不 resume）
func (m *Model) launchNewSession(p Platform) tea.Cmd {
	bin := m.cfg.BinFor(p)
	argv := []string{bin}
	extraArgs := m.cfg.ArgsFor(p)
	if len(extraArgs) > 0 {
		argv = append(argv, extraArgs...)
	}

	if m.cfg.GetResumeMode() == ResumeModeSuspend {
		argv = wrapSuspendCommand(p, argv)
		cmd := exec.CommandContext(context.Background(), argv[0], argv[1:]...)
		cmd.Dir = m.selectedProject.FullPath
		return execInteractiveProcess(cmd)
	}

	// replace 模式：通知 main 退出后启动新 session
	m.newAgentPlatform = p
	m.newSessionPending = true
	return tea.Quit
}
