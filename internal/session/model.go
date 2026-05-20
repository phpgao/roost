// Package session implements the session list sub-model for the roost TUI.
// It manages session list state, key handling, and produces SessionViewModel
// for rendering — with no direct lipgloss/bubbles dependency.
package session

import (
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/phpgao/roost/internal/config"
	"github.com/phpgao/roost/internal/resume"
	"github.com/phpgao/roost/internal/scanner"
	"github.com/phpgao/roost/internal/types"
	"github.com/phpgao/roost/internal/view"
)

// ---------------------------------------------------------------------------
// Output messages
// ---------------------------------------------------------------------------

// ResumeMsg signals that a session should be resumed.
type ResumeMsg struct{ Session types.Session }

// DeleteMsg signals that a session should be deleted.
type DeleteMsg struct{ Session types.Session }

// BatchDeleteMsg signals that selected sessions should be deleted.
type BatchDeleteMsg struct{ SelectedSet map[string]bool }

// DeleteProjectMsg signals that all sessions in a project should be deleted.
type DeleteProjectMsg struct{ Project types.Project }

// NewAgentMsg signals navigation to the new-agent screen.
type NewAgentMsg struct{}

// BackMsg signals navigation back to the project list.
type BackMsg struct{}

// RefreshMsg signals that the session list should be re-scanned.
type RefreshMsg struct{}

// ToggleHelpMsg signals that the help overlay should be toggled.
type ToggleHelpMsg struct{}

// ---------------------------------------------------------------------------
// SessionModel
// ---------------------------------------------------------------------------

// SessionModel manages the session list screen state.
type SessionModel struct {
	renderer           view.SessionRenderer
	cfg                config.Config
	strategy           resume.ResumeStrategy
	scanners           []scanner.Scanner
	selectedProject    *types.Project
	filteredSessions   []types.Session
	installedPlatforms []types.Platform
	cursor             int
	searching          bool
	searchQuery        string
	selecting          bool
	selectedSet        map[string]bool
	platformFilter     int
	width, height      int
}

// NewSessionModel creates a new SessionModel.
func NewSessionModel(renderer view.SessionRenderer, scanners []scanner.Scanner, cfg config.Config, platforms []types.Platform) SessionModel {
	return SessionModel{
		renderer:           renderer,
		cfg:                cfg,
		strategy:           resume.ReplaceStrategy{},
		scanners:           scanners,
		installedPlatforms: platforms,
		selectedSet:        make(map[string]bool),
	}
}

// Init initializes the model (no-op).
func (m SessionModel) Init() tea.Cmd { return nil }

// Update handles incoming messages and key events.
func (m SessionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	}
	return m, nil
}

// View renders the session list screen.
func (m SessionModel) View() tea.View {
	if m.renderer == nil {
		return tea.NewView("")
	}
	data := m.buildViewModel()
	return tea.NewView(m.renderer.Render(data))
}

// ---------------------------------------------------------------------------
// Key handling
// ---------------------------------------------------------------------------

func (m SessionModel) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// Searching mode: handle text input
	if m.searching {
		return m.handleSearchKey(msg)
	}

	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.filteredSessions)-1 {
			m.cursor++
		}
	case "g":
		m.cursor = 0
	case "G":
		m.cursor = len(m.filteredSessions) - 1
		if m.cursor < 0 {
			m.cursor = 0
		}
	case "enter":
		if len(m.filteredSessions) == 0 {
			return m, nil
		}
		s := m.filteredSessions[m.cursor]
		if m.strategy != nil {
			return m, m.strategy.Resume(s, m.scanners, m.cfg)
		}
		return m, func() tea.Msg { return ResumeMsg{Session: s} }
	case "d":
		if !m.selecting && len(m.filteredSessions) > 0 {
			s := m.filteredSessions[m.cursor]
			return m, func() tea.Msg { return DeleteMsg{Session: s} }
		}
	case "D":
		if m.selecting {
			return m, func() tea.Msg { return BatchDeleteMsg{SelectedSet: m.selectedSet} }
		}
	case "x":
		if m.selectedProject != nil {
			return m, func() tea.Msg { return DeleteProjectMsg{Project: *m.selectedProject} }
		}
	case types.KeySpace:
		m.toggleSelection()
	case "tab":
		m.platformFilter = (m.platformFilter + 1) % (len(m.installedPlatforms) + 1)
		m.applyFilter()
		m.cursor = 0
	case "/":
		m.searching = true
		m.searchQuery = ""
	case "n", "N":
		return m, func() tea.Msg { return NewAgentMsg{} }
	case "esc":
		if m.selecting {
			m.selecting = false
			m.selectedSet = make(map[string]bool)
			return m, nil
		}
		return m, func() tea.Msg { return BackMsg{} }
	case "r":
		return m, func() tea.Msg { return RefreshMsg{} }
	case "?":
		return m, func() tea.Msg { return ToggleHelpMsg{} }
	}
	return m, nil
}

func (m SessionModel) handleSearchKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.searching = false
		m.searchQuery = ""
		m.applyFilter()
		return m, nil
	case "enter":
		m.searching = false
		return m, nil
	case "backspace":
		if m.searchQuery != "" {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
		} else {
			m.searching = false
		}
		m.applyFilter()
		return m, nil
	default:
		// Printable character
		ch := msg.String()
		if len(ch) == 1 && ch[0] >= 32 && ch[0] < 127 {
			m.searchQuery += ch
			m.applyFilter()
		}
	}
	return m, nil
}

// ---------------------------------------------------------------------------
// Filtering & selection
// ---------------------------------------------------------------------------

func (m *SessionModel) applyFilter() {
	sessions := m.sessions()
	query := strings.ToLower(m.searchQuery)

	var filtered []types.Session
	for _, s := range sessions {
		// Platform filter: 0 = all, 1..n = specific platform
		if m.platformFilter > 0 {
			idx := m.platformFilter - 1
			if idx < len(m.installedPlatforms) && s.Platform != m.installedPlatforms[idx] {
				continue
			}
		}
		// Search filter
		if query != "" {
			title := strings.ToLower(s.Title)
			if !strings.Contains(title, query) {
				continue
			}
		}
		filtered = append(filtered, s)
	}

	m.filteredSessions = filtered
	m.clampCursor()
}

func (m *SessionModel) sessions() []types.Session {
	if m.selectedProject == nil {
		return nil
	}
	return m.selectedProject.Sessions
}

func (m *SessionModel) toggleSelection() {
	if len(m.filteredSessions) == 0 {
		return
	}
	s := m.filteredSessions[m.cursor]
	if m.selectedSet[s.ID] {
		delete(m.selectedSet, s.ID)
	} else {
		m.selectedSet[s.ID] = true
		m.selecting = true
	}
}

func (m *SessionModel) clampCursor() {
	if m.cursor >= len(m.filteredSessions) {
		m.cursor = len(m.filteredSessions) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

// ---------------------------------------------------------------------------
// View model building
// ---------------------------------------------------------------------------

func (m SessionModel) buildViewModel() view.SessionViewModel {
	breadcrumb := "roost"
	if m.selectedProject != nil {
		breadcrumb = "roost > " + types.ProjectShortName(m.selectedProject.FullPath)
	}

	items := make([]view.SessionItem, 0, len(m.filteredSessions))
	for _, s := range m.filteredSessions {
		title := sanitizeTitle(s.Title)
		if s.AgentType != "" {
			title += " [" + s.AgentType + "]"
		}
		items = append(items, view.SessionItem{
			ID:         s.ID,
			Platform:   s.Platform,
			Title:      title,
			Model:      s.Model,
			LastActive: types.RelativeTime(s.LastActive),
			MsgCount:   s.MsgCount,
		})
	}

	// Sort items by last active descending
	sort.Slice(items, func(i, j int) bool {
		// Find the original session for comparison
		si := m.filteredSessions[i]
		sj := m.filteredSessions[j]
		return si.LastActive.After(sj.LastActive)
	})

	platformFilterName := "All"
	if m.platformFilter > 0 && m.platformFilter-1 < len(m.installedPlatforms) {
		p := m.installedPlatforms[m.platformFilter-1]
		platformFilterName = p.Icon() + " " + p.Name()
	}

	return view.SessionViewModel{
		Breadcrumb:     breadcrumb,
		Items:          items,
		Cursor:         m.cursor,
		Searching:      m.searching,
		SearchQuery:    m.searchQuery,
		Selecting:      m.selecting,
		SelectedSet:    m.selectedSet,
		PlatformFilter: platformFilterName,
		Width:          m.width,
		Height:         m.height,
	}
}

// sanitizeTitle replaces newlines and tabs with spaces and trims.
func sanitizeTitle(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	s = strings.TrimSpace(s)
	return s
}

// ---------------------------------------------------------------------------
// Setters
// ---------------------------------------------------------------------------

// SetProject sets the selected project and re-applies filters.
func (m *SessionModel) SetProject(p *types.Project) {
	m.selectedProject = p
	m.cursor = 0
	m.selecting = false
	m.selectedSet = make(map[string]bool)
	m.searchQuery = ""
	m.searching = false
	m.platformFilter = 0
	m.applyFilter()
}

// SetSize sets the terminal dimensions.
func (m *SessionModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SetRenderer replaces the renderer.
func (m *SessionModel) SetRenderer(r view.SessionRenderer) {
	m.renderer = r
}

// Cursor returns the current cursor position.
func (m SessionModel) Cursor() int {
	return m.cursor
}

// Project returns the currently selected project, or nil.
func (m SessionModel) Project() *types.Project {
	return m.selectedProject
}
