// Package newagent implements the new agent selection sub-model for the roost TUI.
// It manages the agent selection screen state, key handling, and produces
// NewAgentViewModel for rendering — with no direct lipgloss/bubbles dependency.
package newagent

import (
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

// LaunchMsg signals that a new agent session should be launched.
type LaunchMsg struct {
	Platform    types.Platform
	ProjectPath string
}

// BackMsg signals navigation back to the session list.
type BackMsg struct{}

// ---------------------------------------------------------------------------
// NewAgentModel
// ---------------------------------------------------------------------------

// NewAgentModel manages the new agent selection screen state.
type NewAgentModel struct {
	renderer           view.NewAgentRenderer
	cfg                config.Config
	strategy           resume.ResumeStrategy
	scanners           []scanner.Scanner
	installedPlatforms []types.Platform
	selectedProject    *types.Project
	cursor             int
	width, height      int
}

// NewNewAgentModel creates a new NewAgentModel.
func NewNewAgentModel(renderer view.NewAgentRenderer, scanners []scanner.Scanner, cfg config.Config, platforms []types.Platform) NewAgentModel {
	return NewAgentModel{
		renderer:           renderer,
		cfg:                cfg,
		strategy:           resume.ReplaceStrategy{},
		scanners:           scanners,
		installedPlatforms: platforms,
	}
}

// Init initializes the model (no-op).
func (m NewAgentModel) Init() tea.Cmd { return nil }

// Update handles incoming messages and key events.
func (m NewAgentModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

// View renders the new agent selection screen.
func (m NewAgentModel) View() tea.View {
	if m.renderer == nil {
		return tea.NewView("")
	}
	data := m.buildViewModel()
	return tea.NewView(m.renderer.Render(data))
}

// ---------------------------------------------------------------------------
// Key handling
// ---------------------------------------------------------------------------

func (m NewAgentModel) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.installedPlatforms)-1 {
			m.cursor++
		}
	case "g":
		m.cursor = 0
	case "G":
		m.cursor = len(m.installedPlatforms) - 1
		if m.cursor < 0 {
			m.cursor = 0
		}
	case "enter":
		if len(m.installedPlatforms) == 0 || m.cursor >= len(m.installedPlatforms) {
			return m, nil
		}
		platform := m.installedPlatforms[m.cursor]
		projectPath := ""
		if m.selectedProject != nil {
			projectPath = m.selectedProject.FullPath
		}
		if m.strategy != nil {
			return m, m.strategy.LaunchNew(platform, projectPath, m.cfg)
		}
		return m, func() tea.Msg {
			return LaunchMsg{Platform: platform, ProjectPath: projectPath}
		}
	case "esc":
		return m, func() tea.Msg { return BackMsg{} }
	}
	return m, nil
}

// ---------------------------------------------------------------------------
// View model building
// ---------------------------------------------------------------------------

func (m NewAgentModel) buildViewModel() view.NewAgentViewModel {
	items := make([]view.AgentItem, 0, len(m.installedPlatforms))
	for _, p := range m.installedPlatforms {
		items = append(items, view.AgentItem{
			Icon: p.Icon(),
			Name: p.Name(),
		})
	}

	return view.NewAgentViewModel{
		Items:  items,
		Cursor: m.cursor,
		Width:  m.width,
		Height: m.height,
	}
}

// ---------------------------------------------------------------------------
// Setters
// ---------------------------------------------------------------------------

// SetProject sets the selected project for launching new sessions.
func (m *NewAgentModel) SetProject(p *types.Project) {
	m.selectedProject = p
	m.cursor = 0
}

// SetSize sets the terminal dimensions.
func (m *NewAgentModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SetRenderer replaces the renderer.
func (m *NewAgentModel) SetRenderer(r view.NewAgentRenderer) {
	m.renderer = r
}

// SetInstalledPlatforms updates the available platforms list.
func (m *NewAgentModel) SetInstalledPlatforms(platforms []types.Platform) {
	m.installedPlatforms = platforms
	if m.cursor >= len(platforms) {
		m.cursor = len(platforms) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}
