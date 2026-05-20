package view

import (
	"fmt"
	"strings"
)

// DefaultHelpRenderer renders the help screen.
type DefaultHelpRenderer struct {
	styles Styles
}

// NewDefaultHelpRenderer creates a new DefaultHelpRenderer.
func NewDefaultHelpRenderer(styles Styles) *DefaultHelpRenderer {
	return &DefaultHelpRenderer{styles: styles}
}

// Render renders the help screen.
func (r *DefaultHelpRenderer) Render(data HelpViewModel) string {
	var sb strings.Builder
	sb.WriteString(r.styles.Title.Render("  roost — Keyboard Shortcuts") + "\n\n")

	sep := r.styles.Separator.Render(strings.Repeat("-", data.Width))

	key := func(k string) string { return r.styles.Key.Render(k) }
	desc := func(d string) string { return r.styles.Subtle.Render(d) }
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
	for _, p := range data.Platforms {
		tabDesc += " → " + p.Icon()
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
