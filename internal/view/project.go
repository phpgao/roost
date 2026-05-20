package view

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	lipglosstable "charm.land/lipgloss/v2/table"
	"github.com/phpgao/roost/internal/types"
)

// DefaultProjectRenderer renders the project list using lipgloss table.
type DefaultProjectRenderer struct {
	styles Styles
}

// NewDefaultProjectRenderer creates a new DefaultProjectRenderer.
func NewDefaultProjectRenderer(styles Styles) *DefaultProjectRenderer {
	return &DefaultProjectRenderer{styles: styles}
}

// Render renders the project list screen.
func (r *DefaultProjectRenderer) Render(data ProjectViewModel) string {
	var sb strings.Builder

	w := min(data.Width, types.MaxWidth)

	// Header
	title := r.styles.Title.Render("  " + data.Title)
	legend := r.styles.Subtle.Render(data.Legend)
	sb.WriteString(title + "  " + legend + "\n")

	// Empty state
	list := data.Items
	if len(list) == 0 {
		sep := r.styles.Separator.Render(strings.Repeat("─", w))
		sb.WriteString(sep + "\n")
		if data.Searching {
			sb.WriteString(r.styles.Subtle.Render("  no results for: "+data.SearchQuery) + "\n")
		} else {
			sb.WriteString(r.styles.Subtle.Render("  no sessions found") + "\n")
			sb.WriteString(r.styles.Subtle.Render("  press r to refresh") + "\n")
		}
		sb.WriteString(sep + "\n")
		sb.WriteString(r.renderFooter(data))
		return sb.String()
	}

	// Viewport: terminal - title(1) - header(2) - footer(2)
	viewHeight := data.Height - 5
	if viewHeight < 3 {
		viewHeight = 3
	}

	// Build lipgloss table
	cursor := data.Cursor
	baseStyle := r.styles.Normal.Padding(0, 1)
	headerStyle := baseStyle.Foreground(r.styles.Subtle.GetForeground()).Bold(true)
	selectedStyle := r.styles.Selected.Padding(0, 1)

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
		Headers("PATH", "PLATFORMS", "LAST ACTIVE").
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == lipglosstable.HeaderRow {
				return headerStyle
			}
			if row == cursor {
				return selectedStyle
			}
			isSelected := data.Selecting && data.SelectedSet[list[row].FullPath]
			if isSelected {
				return baseStyle.Bold(true)
			}
			return baseStyle
		})

	for i, item := range list {
		// Marker + indicator prefix
		prefix := "    "
		if data.Selecting {
			if data.SelectedSet[item.FullPath] {
				prefix = "● "
			} else {
				prefix = "○ "
			}
		}
		if i == cursor {
			prefix += "▸ "
		} else {
			prefix += "  "
		}

		pathStr := prefix + item.FullPath

		// Platforms
		platformStr := r.formatSessionCounts(item.SessionCounts)

		// Time
		timeStr := item.LastActive

		t.Row(pathStr, platformStr, timeStr)
	}

	// YOffset: center cursor in viewport
	yOffset := cursor - viewHeight/2
	if yOffset < 0 {
		yOffset = 0
	}
	t.YOffset(yOffset).Height(viewHeight)

	sb.WriteString(t.String() + "\n")
	sb.WriteString(r.renderFooter(data))

	return sb.String()
}

// formatSessionCounts renders platform session counts as colored dots + counts.
func (r *DefaultProjectRenderer) formatSessionCounts(counts []PlatformCount) string {
	var sb strings.Builder
	for i, pc := range counts {
		if i > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(r.styles.PlatformDot(pc.Platform))
		fmt.Fprintf(&sb, "%-2d", pc.Count)
	}
	return sb.String()
}

// renderFooter renders the status bar at the bottom of the project list.
func (r *DefaultProjectRenderer) renderFooter(data ProjectViewModel) string {
	if data.Searching {
		return r.styles.Search.Render("  / "+data.SearchQuery+"_") + "\n"
	}
	if data.EscHint {
		return r.styles.DeleteWarning.Render("  press Esc again to quit") + "\n"
	}

	scrollHint := data.ScrollHint
	if data.Selecting {
		return r.styles.Help.Render(fmt.Sprintf("  [%s] Space toggle  D delete(%d)  Esc cancel%s",
			data.PlatformFilter, len(data.SelectedSet), r.styles.Subtle.Render(scrollHint))) + "\n"
	}
	return r.styles.Help.Render(fmt.Sprintf("  [%s] %s %s %s %s%s",
		data.PlatformFilter,
		r.styles.Key.Render("Enter"), r.styles.Key.Render("↑↓"),
		r.styles.Key.Render("Esc"), r.styles.Key.Render("?"),
		r.styles.Subtle.Render(scrollHint))) + "\n"
}
