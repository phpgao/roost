package view

import (
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/compat"
	"github.com/phpgao/roost/internal/types"
)

// Styles holds all lipgloss styles for the application.
type Styles struct {
	// Core
	Title     lipgloss.Style
	Subtle    lipgloss.Style
	Separator lipgloss.Style
	Selected  lipgloss.Style
	Normal    lipgloss.Style

	// Text
	Time         lipgloss.Style
	Model        lipgloss.Style
	MsgCount     lipgloss.Style
	Search       lipgloss.Style
	Help         lipgloss.Style
	Key          lipgloss.Style
	Indicator    lipgloss.Style
	IndicatorOff lipgloss.Style

	// Delete / Confirm
	DeleteBox      lipgloss.Style
	DeleteWarning  lipgloss.Style
	ActiveButton   lipgloss.Style // compat_bubbletea confirm button
	InactiveButton lipgloss.Style

	// Platforms
	Platforms map[types.Platform]lipgloss.Style
}

// NewStyles creates a Styles set adapted to the background color.
func NewStyles(dark bool) Styles {
	if dark {
		return newDarkStyles()
	}
	return newLightStyles()
}

func newDarkStyles() Styles {
	platforms := map[types.Platform]lipgloss.Style{
		types.PlatformCodeBuddy: lipgloss.NewStyle().Foreground(lipgloss.Color(types.ColorGreen)).Bold(true),
		types.PlatformClaude:    lipgloss.NewStyle().Foreground(lipgloss.Color(types.ColorOrange)).Bold(true),
		types.PlatformGemini:    lipgloss.NewStyle().Foreground(lipgloss.Color(types.ColorGemini)).Bold(true),
		types.PlatformCodex:     lipgloss.NewStyle().Foreground(lipgloss.Color(types.ColorCodex)).Bold(true),
		types.PlatformCopilot:   lipgloss.NewStyle().Foreground(lipgloss.Color(types.ColorCopilot)).Bold(true),
		types.PlatformOpenCode:  lipgloss.NewStyle().Foreground(lipgloss.Color(types.ColorOpenCode)).Bold(true),
	}

	activeButton := lipgloss.NewStyle().
		Padding(0, 3).
		Background(compat.AdaptiveColor{Light: lipgloss.Color("#FF6AD2"), Dark: lipgloss.Color("#FF6AD2")}).
		Foreground(compat.AdaptiveColor{Light: lipgloss.Color("#FFFCC2"), Dark: lipgloss.Color("#FFFCC2")})

	return Styles{
		Title:          lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(types.ColorLabelPrimary)),
		Subtle:         lipgloss.NewStyle().Foreground(lipgloss.Color(types.ColorLabelTertiary)),
		Separator:      lipgloss.NewStyle().Foreground(lipgloss.Color("rgba(84, 84, 88, 0.6)")),
		Selected:       lipgloss.NewStyle().Background(lipgloss.Color(types.ColorBgSelected)).Foreground(lipgloss.Color(types.ColorLabelPrimary)),
		Normal:         lipgloss.NewStyle(),
		Time:           lipgloss.NewStyle().Foreground(lipgloss.Color(types.ColorLabelTertiary)),
		Model:          lipgloss.NewStyle().Foreground(lipgloss.Color(types.ColorLabelTertiary)),
		MsgCount:       lipgloss.NewStyle().Foreground(lipgloss.Color(types.ColorLabelTertiary)),
		Search:         lipgloss.NewStyle().Foreground(lipgloss.Color(types.ColorYellow)).Bold(true),
		Help:           lipgloss.NewStyle().Foreground(lipgloss.Color(types.ColorLabelTertiary)),
		Key:            lipgloss.NewStyle().Foreground(lipgloss.Color(types.ColorBlue)).Bold(true),
		Indicator:      lipgloss.NewStyle().Foreground(lipgloss.Color(types.ColorBlue)).Bold(true),
		IndicatorOff:   lipgloss.NewStyle().Foreground(lipgloss.Color(types.ColorLabelQuaternary)),
		DeleteBox:      lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(types.ColorRed)).Padding(1, 2),
		DeleteWarning:  lipgloss.NewStyle().Foreground(lipgloss.Color(types.ColorRed)).Bold(true),
		ActiveButton:   activeButton,
		InactiveButton: activeButton.Background(compat.AdaptiveColor{Light: lipgloss.Color("#988F95"), Dark: lipgloss.Color("#978692")}).Foreground(compat.AdaptiveColor{Light: lipgloss.Color("#FDFCE3"), Dark: lipgloss.Color("#FBFAE7")}),
		Platforms:      platforms,
	}
}

func newLightStyles() Styles {
	s := newDarkStyles()
	s.Title = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#1a1a1a"))
	s.Selected = lipgloss.NewStyle().Background(lipgloss.Color("rgba(10, 132, 255, 0.15)")).Foreground(lipgloss.Color("#1a1a1a"))
	return s
}

// PlatformStyle returns the lipgloss style for a given platform.
func (s Styles) PlatformStyle(p types.Platform) lipgloss.Style {
	if style, ok := s.Platforms[p]; ok {
		return style
	}
	return s.Subtle
}

// PlatformIcon renders a colored bullet + short name for a platform.
func (s Styles) PlatformIcon(p types.Platform) string {
	return s.PlatformStyle(p).Render(p.Icon() + p.Name())
}

// PlatformDot renders only a colored bullet (no short name), for inline use.
func (s Styles) PlatformDot(p types.Platform) string {
	return s.PlatformStyle(p).Render(p.Icon())
}

// RenderColoredCommand renders a command string prefixed with "$ " using the platform's color.
func (s Styles) RenderColoredCommand(p types.Platform, cmd string) string {
	return s.PlatformStyle(p).Render("$ " + cmd)
}
