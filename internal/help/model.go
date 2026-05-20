// Package help implements the HelpModel sub-model for the help overlay screen.
// It is a pure data+logic model with no lipgloss/bubbles dependency;
// rendering is delegated to a view.HelpRenderer.
package help

import (
	tea "charm.land/bubbletea/v2"
	"github.com/phpgao/roost/internal/types"
	"github.com/phpgao/roost/internal/view"
)

// CloseMsg is emitted when the user presses ? or Esc to close the help overlay.
type CloseMsg struct{}

// HelpModel is the Bubble Tea sub-model for the help overlay screen.
type HelpModel struct {
	renderer  view.HelpRenderer
	platforms []types.Platform
	width     int
	height    int
}

// NewHelpModel creates a new HelpModel.
func NewHelpModel(renderer view.HelpRenderer) HelpModel {
	return HelpModel{
		renderer: renderer,
	}
}

// Init implements tea.Model.
func (m HelpModel) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (m HelpModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		switch msg.String() {
		case "?", "esc":
			return m, func() tea.Msg { return CloseMsg{} }
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

// View implements tea.Model.
func (m HelpModel) View() tea.View {
	vm := view.HelpViewModel{
		Platforms: m.platforms,
		Width:     m.width,
	}
	return tea.NewView(m.renderer.Render(vm))
}

// SetPlatforms updates the platform list shown in the help screen.
func (m *HelpModel) SetPlatforms(platforms []types.Platform) {
	m.platforms = platforms
}

// SetSize updates the terminal dimensions.
func (m *HelpModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SetRenderer replaces the renderer.
func (m *HelpModel) SetRenderer(r view.HelpRenderer) {
	m.renderer = r
}
