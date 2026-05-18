package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/mattn/go-runewidth"
)

func init() {
	// Force East Asian width so ellipsis "…" is consistently treated as 2 cols
	// across all locales (macOS vs Linux CI).
	_ = os.Setenv("RUNEWIDTH_EASTASIAN", "1")
	runewidth.DefaultCondition.EastAsianWidth = true
}

func (p Platform) Icon() string {
	switch p {
	case PlatformCodeBuddy:
		return "●"
	case PlatformClaude:
		return "●"
	case PlatformGemini:
		return "●"
	case PlatformCodex:
		return "●"
	case PlatformCopilot:
		return "●"
	case PlatformOpenCode:
		return "●"
	default:
		return "○"
	}
}

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

func (p Platform) Name() string {
	switch p {
	case PlatformCodeBuddy:
		return nameCodeBuddy
	case PlatformClaude:
		return nameClaude
	case PlatformGemini:
		return nameGemini
	case PlatformCodex:
		return nameCodex
	case PlatformCopilot:
		return nameCopilot
	case PlatformOpenCode:
		return nameOpenCode
	default:
		return "Unknown"
	}
}

// Scanner 定义各平台扫描器接口
type Scanner interface {
	Platform() Platform
	DataDir() string
	ScanProjects() ([]Project, error)
	DeleteSession(s Session) error
	DeleteProject(p Project) error
	ResumeCmd(s Session) []string
}

// Session 代表单条 AI 会话记录
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

// Project 代表一个工作目录下的所有会话
type Project struct {
	Name     string
	FullPath string
	Sessions []Session
}

// LastActive 返回项目内所有 session 中最新的活跃时间
func (p *Project) LastActive() time.Time {
	var t time.Time
	for _, s := range p.Sessions {
		if s.LastActive.After(t) {
			t = s.LastActive
		}
	}
	return t
}

// truncate 截断字符串到最多 n 个 rune（保留头部，末尾加 …）。
// n 以 rune 数计，用于简单英文字段；中文等宽字符请用 truncateWidth。
func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "…"
}

// truncateKeepEnd 截断字符串，使视觉宽度不超过 n（去掉头部，保留末尾，开头加 …）。
// O(n)：从右向左逐字符累加 runewidth，只在最后构造一次结果字符串。
// 重命名说明：原函数名 truncateLeft 易误解为"从左截断"，实际行为是保留末尾。
func truncateKeepEnd(s string, n int) string {
	if lipgloss.Width(s) <= n {
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

// truncateWidth 截断字符串，使视觉宽度不超过 n（保留头部，末尾加 …）。
// O(n)：从左向右逐字符累加 runewidth，只在最后构造一次结果字符串。
func truncateWidth(s string, n int) string {
	if lipgloss.Width(s) <= n {
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

// projectShortName 从绝对路径取最后两段作为短名
func projectShortName(fullPath string) string {
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

// relativeTime 返回相对当前时间的可读字符串
func relativeTime(t time.Time) string {
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

// mergeProjects 将多个 Scanner 返回的 []Project 按 FullPath 合并
func mergeProjects(all [][]Project) []Project {
	m := make(map[string]*Project)
	var order []string
	for _, projects := range all {
		for _, p := range projects {
			if existing, ok := m[p.FullPath]; ok {
				existing.Sessions = append(existing.Sessions, p.Sessions...)
			} else {
				m[p.FullPath] = new(p)
				order = append(order, p.FullPath)
			}
		}
	}
	result := make([]Project, 0, len(m))
	for _, key := range order {
		result = append(result, *m[key])
	}
	// 按最近活跃时间倒序
	sort.Slice(result, func(i, j int) bool {
		return result[i].LastActive().After(result[j].LastActive())
	})
	return result
}

// ScanProjectsParallel 并行扫描所有平台的 projects，返回合并后的列表
func ScanProjectsParallel(scanners []Scanner) []Project {
	var all [][]Project
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, sc := range scanners {
		wg.Add(1)
		go func(sc Scanner) {
			defer wg.Done()
			projects, err := sc.ScanProjects()
			if err != nil || len(projects) == 0 {
				return
			}
			mu.Lock()
			all = append(all, projects)
			mu.Unlock()
		}(sc)
	}
	wg.Wait()

	return mergeProjects(all)
}
