// Package view defines Renderer interfaces, ViewModel types, and default
// rendering implementations for the roost TUI. Sub-models depend only on
// the interfaces defined here; the concrete lipgloss/bubbles implementations
// can be swapped out for alternative visual styles.
package view

import "github.com/phpgao/roost/internal/types"

// ---------------------------------------------------------------------------
// ViewModel types — pure data passed from sub-models to renderers
// ---------------------------------------------------------------------------

// ProjectViewModel is the rendering data for the project list screen.
type ProjectViewModel struct {
	Title          string
	Legend         string
	Width          int
	Height         int
	Items          []ProjectItem
	Cursor         int
	Searching      bool
	SearchQuery    string
	Selecting      bool
	SelectedSet    map[string]bool
	PlatformFilter string // "All" or "● Claude" etc.
	ScrollHint     string // "[1/5 ↓]" etc.
	EscHint        bool
}

// PlatformCount holds raw platform session count data for a project item.
// The renderer is responsible for formatting (colored dots, etc.).
type PlatformCount struct {
	Platform types.Platform
	Count    int
}

// ProjectItem is a single row in the project list.
type ProjectItem struct {
	FullPath      string
	SessionCounts []PlatformCount // raw counts, renderer formats
	LastActive    string          // formatted relative time
}

// SessionViewModel is the rendering data for the session list screen.
type SessionViewModel struct {
	Breadcrumb     string // "roost > parent/project"
	Legend         string
	Width          int
	Height         int
	Items          []SessionItem
	Cursor         int
	Searching      bool
	SearchQuery    string
	Selecting      bool
	SelectedSet    map[string]bool
	PlatformFilter string
	ScrollHint     string
	EscHint        bool
}

// SessionItem is a single row in the session list.
type SessionItem struct {
	ID         string         // session ID, used as key for selection
	Platform   types.Platform // raw platform, renderer formats the icon
	Title      string
	Model      string
	LastActive string
	MsgCount   int
}

// ConfirmViewModel is the rendering data for the delete confirmation dialog.
type ConfirmViewModel struct {
	Action     string // "Delete this session?" etc.
	Subject    string // specific name or count
	Warning    string // "This action cannot be undone"
	YesFocused bool   // whether Yes button has focus
	Width      int
	Height     int
}

// NewAgentViewModel is the rendering data for the new agent selection screen.
type NewAgentViewModel struct {
	Items  []AgentItem
	Cursor int
	Width  int
	Height int
}

// AgentItem is a single row in the new agent list.
type AgentItem struct {
	Icon string
	Name string
}

// HelpViewModel is the rendering data for the help screen.
type HelpViewModel struct {
	Platforms []types.Platform
	Width     int
}

// LoadingViewModel is the rendering data for the loading/spinner screen.
type LoadingViewModel struct {
	SpinnerFrame string
	Width        int
}

// ErrorViewModel is the rendering data for the error screen.
type ErrorViewModel struct {
	Error       string
	ErrorOutput string
	Width       int
}

// ---------------------------------------------------------------------------
// Renderer interfaces — one per screen
// ---------------------------------------------------------------------------

// ProjectRenderer renders the project list screen.
type ProjectRenderer interface {
	Render(data ProjectViewModel) string
}

// SessionRenderer renders the session list screen.
type SessionRenderer interface {
	Render(data SessionViewModel) string
}

// ConfirmRenderer renders the delete confirmation dialog.
type ConfirmRenderer interface {
	Render(data ConfirmViewModel) string
}

// NewAgentRenderer renders the new agent selection screen.
type NewAgentRenderer interface {
	Render(data NewAgentViewModel) string
}

// HelpRenderer renders the help screen.
type HelpRenderer interface {
	Render(data HelpViewModel) string
}

// LoadingRenderer renders the loading/spinner screen.
type LoadingRenderer interface {
	Render(data LoadingViewModel) string
}

// ErrorRenderer renders the error screen.
type ErrorRenderer interface {
	Render(data ErrorViewModel) string
}

// ---------------------------------------------------------------------------
// RendererSet — a complete set of renderers, injected into the app
// ---------------------------------------------------------------------------

// RendererSet holds a complete set of screen renderers.
type RendererSet struct {
	Project  ProjectRenderer
	Session  SessionRenderer
	Confirm  ConfirmRenderer
	NewAgent NewAgentRenderer
	Help     HelpRenderer
	Loading  LoadingRenderer
	Error    ErrorRenderer
}

// NewDefaultRendererSet returns a RendererSet using the default lipgloss+bubbles
// implementations.
func NewDefaultRendererSet(styles Styles) RendererSet {
	return RendererSet{
		Project:  NewDefaultProjectRenderer(styles),
		Session:  NewDefaultSessionRenderer(styles),
		Confirm:  NewDefaultConfirmRenderer(styles),
		NewAgent: NewDefaultNewAgentRenderer(styles),
		Help:     NewDefaultHelpRenderer(styles),
		Loading:  NewDefaultLoadingRenderer(styles),
		Error:    NewDefaultErrorRenderer(styles),
	}
}

// UpdateStyles updates styles on all default renderers.
func (s *RendererSet) UpdateStyles(styles Styles) {
	if r, ok := s.Project.(*DefaultProjectRenderer); ok {
		r.styles = styles
	}
	if r, ok := s.Session.(*DefaultSessionRenderer); ok {
		r.styles = styles
	}
	if r, ok := s.Confirm.(*DefaultConfirmRenderer); ok {
		r.styles = styles
	}
	if r, ok := s.NewAgent.(*DefaultNewAgentRenderer); ok {
		r.styles = styles
	}
	if r, ok := s.Help.(*DefaultHelpRenderer); ok {
		r.styles = styles
	}
	if r, ok := s.Loading.(*DefaultLoadingRenderer); ok {
		r.styles = styles
	}
	if r, ok := s.Error.(*DefaultErrorRenderer); ok {
		r.styles = styles
	}
}
