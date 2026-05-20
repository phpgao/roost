// Package confirm implements the delete confirmation dialog sub-model.
// It is a pure data+logic model with no lipgloss/bubbles dependency;
// rendering is delegated to a view.ConfirmRenderer.
package confirm

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/phpgao/roost/internal/types"
	"github.com/phpgao/roost/internal/view"
)

// ResultMsg is emitted when the user confirms or cancels the dialog.
type ResultMsg struct{ Confirmed bool }

// ConfirmModel is the Bubble Tea sub-model for the delete confirmation dialog.
type ConfirmModel struct {
	renderer  view.ConfirmRenderer
	yes       bool // cursor on Yes (default true)
	target    types.DeleteTarget
	session   *types.Session
	project   *types.Project
	count     int  // batch delete count
	inSession bool // context: Esc returns to session or project
	width     int
	height    int
}

// NewConfirmModel creates a new ConfirmModel with the given renderer.
func NewConfirmModel(renderer view.ConfirmRenderer) ConfirmModel {
	return ConfirmModel{
		renderer: renderer,
		yes:      true,
	}
}

// Init implements tea.Model.
func (m ConfirmModel) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (m ConfirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyPressMsg); ok {
		switch key.String() {
		case "left", "right", "h", "l":
			m.yes = !m.yes
			return m, nil
		case "y":
			return m, func() tea.Msg { return ResultMsg{Confirmed: true} }
		case "n", "esc":
			return m, func() tea.Msg { return ResultMsg{Confirmed: false} }
		case "enter":
			return m, func() tea.Msg { return ResultMsg{Confirmed: m.yes} }
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

// View implements tea.Model.
func (m ConfirmModel) View() tea.View {
	data := m.buildViewModel()
	return tea.NewView(m.renderer.Render(data))
}

// buildViewModel constructs the ConfirmViewModel from the current state.
func (m *ConfirmModel) buildViewModel() view.ConfirmViewModel {
	var action, subject string

	switch m.target {
	case types.DeleteTargetSession:
		if m.session != nil {
			action = "Delete this session?"
			subject = fmt.Sprintf("%q", m.session.Title)
		}
	case types.DeleteTargetProject:
		if m.project != nil {
			count := len(m.project.Sessions)
			action = fmt.Sprintf("Delete entire project? (%d sessions)", count)
			subject = fmt.Sprintf("%q", m.project.Name)
		}
	case types.DeleteTargetBatch:
		if m.inSession {
			action = fmt.Sprintf("Delete %d sessions?", m.count)
		} else {
			action = fmt.Sprintf("Delete %d projects?", m.count)
		}
	}

	return view.ConfirmViewModel{
		Action:     action,
		Subject:    subject,
		Warning:    "This action cannot be undone",
		YesFocused: m.yes,
		Width:      m.width,
		Height:     m.height,
	}
}

// SetSessionTarget sets the target to a single session.
func (m *ConfirmModel) SetSessionTarget(session *types.Session) {
	m.target = types.DeleteTargetSession
	m.session = session
	m.project = nil
	m.count = 0
	m.yes = true
}

// SetProjectTarget sets the target to an entire project.
func (m *ConfirmModel) SetProjectTarget(project *types.Project) {
	m.target = types.DeleteTargetProject
	m.project = project
	m.session = nil
	m.count = 0
	m.yes = true
}

// SetBatchTarget sets the target to a batch delete.
func (m *ConfirmModel) SetBatchTarget(count int, inSession bool) {
	m.target = types.DeleteTargetBatch
	m.count = count
	m.inSession = inSession
	m.session = nil
	m.project = nil
	m.yes = true
}

// SetSize sets the terminal dimensions.
func (m *ConfirmModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SetRenderer replaces the renderer.
func (m *ConfirmModel) SetRenderer(r view.ConfirmRenderer) {
	m.renderer = r
}

// Target returns the current delete target type.
func (m ConfirmModel) Target() types.DeleteTarget {
	return m.target
}

// Session returns the session being deleted, or nil.
func (m ConfirmModel) Session() *types.Session {
	return m.session
}

// Project returns the project being deleted, or nil.
func (m ConfirmModel) Project() *types.Project {
	return m.project
}

// Count returns the batch delete count.
func (m ConfirmModel) Count() int {
	return m.count
}

// InSession returns whether the confirm dialog was opened from the session screen.
func (m ConfirmModel) InSession() bool {
	return m.inSession
}
