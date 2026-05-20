package view

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	lipglosstable "charm.land/lipgloss/v2/table"
	"github.com/phpgao/roost/internal/types"
)

// DefaultNewAgentRenderer renders the new agent selection screen.
type DefaultNewAgentRenderer struct {
	styles Styles
}

// NewDefaultNewAgentRenderer creates a new DefaultNewAgentRenderer.
func NewDefaultNewAgentRenderer(styles Styles) *DefaultNewAgentRenderer {
	return &DefaultNewAgentRenderer{styles: styles}
}

// Render renders the new agent selection screen.
func (r *DefaultNewAgentRenderer) Render(data NewAgentViewModel) string {
	var sb strings.Builder

	w := min(data.Width, types.MaxWidth)

	sb.WriteString(r.styles.Title.Render("  New session — select agent") + "\n")

	if len(data.Items) == 0 {
		sep := r.styles.Separator.Render(strings.Repeat("─", w))
		sb.WriteString(sep + "\n")
		sb.WriteString(r.styles.Subtle.Render("  no agents available") + "\n")
		sb.WriteString(sep + "\n")
		return sb.String()
	}

	cursor := data.Cursor
	baseStyle := r.styles.Normal.Padding(0, 1)
	headerStyle := baseStyle.Foreground(r.styles.Subtle.GetForeground()).Bold(true)
	selectedStyle := r.styles.Selected.Padding(0, 1)

	viewHeight := data.Height - 5
	if viewHeight < 3 {
		viewHeight = 3
	}

	t := lipglosstable.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(r.styles.Separator).
		BorderTop(true).
		BorderBottom(true).
		BorderHeader(true).
		BorderColumn(true).
		BorderLeft(false).
		BorderRight(false).
		Width(w).
		Headers("AGENT").
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == lipglosstable.HeaderRow {
				return headerStyle
			}
			if row == cursor {
				return selectedStyle
			}
			return baseStyle
		})

	for i, item := range data.Items {
		prefix := "  "
		if i == cursor {
			prefix = "▸ "
		}

		t.Row(prefix + item.Icon + " " + item.Name)
	}

	yOffset := cursor - viewHeight/2
	if yOffset < 0 {
		yOffset = 0
	}
	t.YOffset(yOffset).Height(viewHeight)

	sb.WriteString(t.String() + "\n")

	scrollHint := renderScrollHint(0, len(data.Items), len(data.Items), data.Cursor)
	sb.WriteString(r.styles.Help.Render(fmt.Sprintf("  %s %s %s%s",
		r.styles.Key.Render("Enter"), r.styles.Key.Render("↑↓"), r.styles.Key.Render("Esc"),
		r.styles.Subtle.Render(scrollHint))) + "\n")

	return sb.String()
}

// renderScrollHint returns the scroll position indicator string.
func renderScrollHint(start, end, total, cursor int) string {
	scrollHint := ""
	if start > 0 {
		scrollHint += "↑"
	}
	if end < total {
		scrollHint += "↓"
	}
	if scrollHint != "" {
		return fmt.Sprintf(" [%d/%d %s]", cursor+1, total, scrollHint)
	}
	if total > 0 {
		return fmt.Sprintf(" [%d/%d]", cursor+1, total)
	}
	return ""
}
