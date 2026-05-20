// Package project implements the ProjectModel sub-model for the project list screen.
// It is a pure data+logic model with no lipgloss/bubbles dependency — all rendering
// is delegated to a view.ProjectRenderer.
package project

import (
	"fmt"
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

// SelectMsg is emitted when the user presses Enter on a project.
type SelectMsg struct{ Project types.Project }

// DeleteMsg is emitted when the user presses d on a project (non-selecting mode).
type DeleteMsg struct{ Project types.Project }

// BatchDeleteMsg is emitted when the user presses D in selecting mode with items selected.
type BatchDeleteMsg struct{ SelectedSet map[string]bool }

// RefreshMsg is emitted when the user presses r to refresh.
type RefreshMsg struct{}

// ToggleHelpMsg is emitted when the user presses ?.
type ToggleHelpMsg struct{}

// EscMsg is emitted when Esc is pressed in non-selecting mode.
// The app handles double-esc quit logic.
type EscMsg struct{}

// ---------------------------------------------------------------------------
// ProjectModel
// ---------------------------------------------------------------------------

// ProjectModel is the Bubble Tea sub-model for the project list screen.
type ProjectModel struct {
	renderer           view.ProjectRenderer
	cfg                config.Config
	strategy           resume.ResumeStrategy
	scanners           []scanner.Scanner
	projects           []types.Project
	filteredProjects   []types.Project
	installedPlatforms []types.Platform
	cursor             int
	searching          bool
	searchQuery        string
	selecting          bool
	selectedSet        map[string]bool
	platformFilter     int
	width, height      int
}

// NewProjectModel creates a new ProjectModel.
func NewProjectModel(renderer view.ProjectRenderer, scanners []scanner.Scanner, cfg config.Config, platforms []types.Platform) ProjectModel {
	return ProjectModel{
		renderer:           renderer,
		cfg:                cfg,
		strategy:           resume.ReplaceStrategy{},
		scanners:           scanners,
		installedPlatforms: platforms,
		selectedSet:        make(map[string]bool),
	}
}

// Init implements tea.Model.
func (m ProjectModel) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (m ProjectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		// Searching mode: handle text input
		if m.searching {
			return m.handleSearchInput(msg)
		}

		switch msg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.filteredProjects)-1 {
				m.cursor++
			}
		case "g":
			m.cursor = 0
		case "G":
			m.cursor = len(m.filteredProjects) - 1
			if m.cursor < 0 {
				m.cursor = 0
			}
		case "enter":
			if len(m.filteredProjects) > 0 && m.cursor < len(m.filteredProjects) {
				return m, func() tea.Msg { return SelectMsg{Project: m.filteredProjects[m.cursor]} }
			}
		case "d":
			if !m.selecting && len(m.filteredProjects) > 0 && m.cursor < len(m.filteredProjects) {
				return m, func() tea.Msg { return DeleteMsg{Project: m.filteredProjects[m.cursor]} }
			}
		case "D":
			if m.selecting && len(m.selectedSet) > 0 {
				return m, func() tea.Msg { return BatchDeleteMsg{SelectedSet: m.selectedSet} }
			}
		case types.KeySpace:
			if len(m.filteredProjects) > 0 && m.cursor < len(m.filteredProjects) {
				m.selecting = true
				path := m.filteredProjects[m.cursor].FullPath
				if m.selectedSet[path] {
					delete(m.selectedSet, path)
				} else {
					m.selectedSet[path] = true
				}
			}
		case "tab":
			m.platformFilter = (m.platformFilter + 1) % (len(m.installedPlatforms) + 1)
			m.applyFilter()
			m.clampCursor()
		case "/":
			m.searching = true
			m.searchQuery = ""
		case "esc":
			if m.selecting {
				m.selecting = false
				m.selectedSet = make(map[string]bool)
			} else {
				return m, func() tea.Msg { return EscMsg{} }
			}
		case "r":
			return m, func() tea.Msg { return RefreshMsg{} }
		case "?":
			return m, func() tea.Msg { return ToggleHelpMsg{} }
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}

	return m, nil
}

// handleSearchInput handles key presses while in search mode.
func (m ProjectModel) handleSearchInput(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.searching = false
		m.searchQuery = ""
		m.applyFilter()
		m.clampCursor()
	case "enter":
		m.searching = false
		// Keep the filtered results as-is
	case "backspace":
		if m.searchQuery != "" {
			m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
		}
		m.applyFilter()
		m.clampCursor()
	default:
		// Printable character
		if len(msg.String()) == 1 {
			m.searchQuery += msg.String()
			m.applyFilter()
			m.clampCursor()
		}
	}
	return m, nil
}

// View implements tea.Model.
func (m ProjectModel) View() tea.View {
	vm := m.buildViewModel()
	return tea.NewView(m.renderer.Render(vm))
}

// ---------------------------------------------------------------------------
// ViewModel construction
// ---------------------------------------------------------------------------

// buildViewModel constructs a ProjectViewModel from the current model state.
func (m ProjectModel) buildViewModel() view.ProjectViewModel {
	vm := view.ProjectViewModel{
		Title:       "roost",
		Width:       m.width,
		Height:      m.height,
		Cursor:      m.cursor,
		Searching:   m.searching,
		SearchQuery: m.searchQuery,
		Selecting:   m.selecting,
		SelectedSet: m.selectedSet,
	}

	// Platform filter label
	vm.PlatformFilter = m.platformFilterLabel()

	// Legend: build from installed platforms
	vm.Legend = m.buildLegend()

	// Build items from filteredProjects
	vm.Items = make([]view.ProjectItem, 0, len(m.filteredProjects))
	for _, p := range m.filteredProjects {
		item := view.ProjectItem{
			FullPath:   p.FullPath,
			LastActive: types.RelativeTime(p.LastActive()),
		}
		// Count sessions per platform
		counts := make(map[types.Platform]int)
		for _, s := range p.Sessions {
			counts[s.Platform]++
		}
		for _, plat := range m.installedPlatforms {
			if c, ok := counts[plat]; ok {
				item.SessionCounts = append(item.SessionCounts, view.PlatformCount{
					Platform: plat,
					Count:    c,
				})
			}
		}
		vm.Items = append(vm.Items, item)
	}

	// Scroll hint
	if len(m.filteredProjects) > 0 {
		vm.ScrollHint = fmt.Sprintf("[%d/%d]", m.cursor+1, len(m.filteredProjects))
	}

	// Esc hint
	vm.EscHint = !m.selecting && !m.searching

	return vm
}

// buildLegend builds the legend string from installed platforms.
func (m ProjectModel) buildLegend() string {
	parts := make([]string, 0, len(m.installedPlatforms))
	for _, p := range m.installedPlatforms {
		parts = append(parts, p.Icon()+p.ShortName())
	}
	return strings.Join(parts, " ")
}

// platformFilterLabel returns the display label for the current platform filter.
func (m ProjectModel) platformFilterLabel() string {
	if m.platformFilter == 0 {
		return "All"
	}
	idx := m.platformFilter - 1
	if idx < len(m.installedPlatforms) {
		return m.installedPlatforms[idx].Icon() + " " + m.installedPlatforms[idx].Name()
	}
	return "All"
}

// ---------------------------------------------------------------------------
// Filtering
// ---------------------------------------------------------------------------

// applyFilter applies both platform filter and search query to the project list.
func (m *ProjectModel) applyFilter() {
	filtered := m.projects

	// Platform filter
	if m.platformFilter > 0 {
		idx := m.platformFilter - 1
		if idx < len(m.installedPlatforms) {
			targetPlatform := m.installedPlatforms[idx]
			filtered = filterByPlatform(filtered, targetPlatform)
		}
	}

	// Search filter
	if m.searchQuery != "" {
		filtered = filterBySearch(filtered, m.searchQuery)
	}

	m.filteredProjects = filtered
}

// filterByPlatform returns only projects that have sessions for the given platform.
func filterByPlatform(projects []types.Project, platform types.Platform) []types.Project {
	var result []types.Project
	for _, p := range projects {
		for _, s := range p.Sessions {
			if s.Platform == platform {
				result = append(result, p)
				break
			}
		}
	}
	return result
}

// filterBySearch returns projects whose FullPath contains the query (case-insensitive).
func filterBySearch(projects []types.Project, query string) []types.Project {
	lowerQuery := strings.ToLower(query)
	var result []types.Project
	for _, p := range projects {
		if strings.Contains(strings.ToLower(p.FullPath), lowerQuery) {
			result = append(result, p)
		}
	}
	return result
}

// clampCursor ensures the cursor is within valid range.
func (m *ProjectModel) clampCursor() {
	upper := len(m.filteredProjects) - 1
	if upper < 0 {
		upper = 0
	}
	m.cursor = types.Clamp(m.cursor, 0, upper)
}

// ---------------------------------------------------------------------------
// Setters & Getters
// ---------------------------------------------------------------------------

// SetProjects updates the project list and reapplies filters.
func (m *ProjectModel) SetProjects(projects []types.Project) {
	m.projects = projects
	m.applyFilter()
	m.clampCursor()
}

// SetSize updates the terminal dimensions.
func (m *ProjectModel) SetSize(w, h int) {
	m.width = w
	m.height = h
}

// SetRenderer updates the renderer.
func (m *ProjectModel) SetRenderer(r view.ProjectRenderer) {
	m.renderer = r
}

// Cursor returns the current cursor position.
func (m *ProjectModel) Cursor() int {
	return m.cursor
}

// PlatformFilter returns the current platform filter index.
func (m *ProjectModel) PlatformFilter() int {
	return m.platformFilter
}

// SetPlatformFilter sets the platform filter index and reapplies filtering.
func (m *ProjectModel) SetPlatformFilter(f int) {
	m.platformFilter = f
	m.applyFilter()
	m.clampCursor()
}

// Projects returns the current project list.
func (m ProjectModel) Projects() []types.Project {
	return m.projects
}

// FilteredProjects returns the filtered project list.
func (m ProjectModel) FilteredProjects() []types.Project {
	return m.filteredProjects
}

// ---------------------------------------------------------------------------
// Sort helper (ensure stable ordering)
// ---------------------------------------------------------------------------

// SortProjects sorts projects by LastActive descending for stable ordering.
func SortProjects(projects []types.Project) {
	sort.SliceStable(projects, func(i, j int) bool {
		return projects[i].LastActive().After(projects[j].LastActive())
	})
}
