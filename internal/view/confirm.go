package view

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// DefaultConfirmRenderer renders the delete confirmation dialog in compat_bubbletea style.
type DefaultConfirmRenderer struct {
	styles Styles
}

// NewDefaultConfirmRenderer creates a new DefaultConfirmRenderer.
func NewDefaultConfirmRenderer(styles Styles) *DefaultConfirmRenderer {
	return &DefaultConfirmRenderer{styles: styles}
}

// Render renders the confirmation dialog.
func (r *DefaultConfirmRenderer) Render(data ConfirmViewModel) string {
	s := r.styles

	yesBtn, noBtn := "Yes", "No"
	if data.YesFocused {
		yesBtn = s.ActiveButton.Render(yesBtn)
		noBtn = s.InactiveButton.Render(noBtn)
	} else {
		yesBtn = s.InactiveButton.Render(yesBtn)
		noBtn = s.ActiveButton.Render(noBtn)
	}

	var parts []string
	parts = append(parts, s.DeleteWarning.Render(data.Action))
	if data.Subject != "" {
		parts = append(parts, "  "+data.Subject)
	}
	parts = append(parts, "", s.Subtle.Render(data.Warning), "", yesBtn+"  "+noBtn)

	content := s.DeleteBox.Render(
		lipgloss.JoinVertical(lipgloss.Center, parts...),
	)

	// Center the dialog
	lines := strings.Split(content, "\n")
	maxW := 0
	for _, l := range lines {
		if w := lipgloss.Width(l); w > maxW {
			maxW = w
		}
	}
	leftPad := (data.Width - maxW) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	topPad := (data.Height - len(lines)) / 2
	if topPad < 0 {
		topPad = 0
	}

	var sb strings.Builder
	sb.WriteString(strings.Repeat("\n", topPad))
	pad := strings.Repeat(" ", leftPad)
	for _, l := range lines {
		sb.WriteString(pad + l + "\n")
	}
	return sb.String()
}
