package view

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	lipglosstable "charm.land/lipgloss/v2/table"
	"github.com/phpgao/roost/internal/types"
)

// DefaultSessionRenderer renders the session list using lipgloss table.
type DefaultSessionRenderer struct {
	styles Styles
}

// NewDefaultSessionRenderer creates a new DefaultSessionRenderer.
func NewDefaultSessionRenderer(styles Styles) *DefaultSessionRenderer {
	return &DefaultSessionRenderer{styles: styles}
}

// Render renders the session list screen.
func (r *DefaultSessionRenderer) Render(data SessionViewModel) string {
	var sb strings.Builder

	w := min(data.Width, types.MaxWidth)

	// Breadcrumb header
	crumb := r.styles.Subtle.Render("roost > ")
	header := fmt.Sprintf("  %s%s", crumb, r.styles.Title.Render(data.Breadcrumb))
	legend := r.styles.Subtle.Render(data.Legend)
	sb.WriteString(header + "  " + legend + "\n")

	// Empty state
	list := data.Items
	if len(list) == 0 {
		sep := r.styles.Separator.Render(strings.Repeat("─", w))
		sb.WriteString(sep + "\n")
		if data.Searching {
			sb.WriteString(r.styles.Subtle.Render("  no results for: "+data.SearchQuery) + "\n")
		} else {
			sb.WriteString(r.styles.Subtle.Render("  no sessions") + "\n")
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
		Headers("TITLE", "MODEL", "LAST ACTIVE", "MSGS").
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == lipglosstable.HeaderRow {
				return headerStyle
			}
			if row == cursor {
				return selectedStyle
			}
			isSelected := data.Selecting && data.SelectedSet[list[row].ID]
			if isSelected {
				return baseStyle.Bold(true)
			}
			return baseStyle
		})

	for i, item := range list {
		// Marker + indicator + icon prefix
		prefix := "    "
		if data.Selecting {
			if data.SelectedSet[item.ID] {
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

		titleStr := prefix + r.styles.PlatformDot(item.Platform) + " " + item.Title
		modelStr := r.styles.Model.Render(item.Model)
		timeStr := r.styles.Time.Render(item.LastActive)
		msgStr := r.styles.MsgCount.Render(fmt.Sprintf("%d", item.MsgCount))

		t.Row(titleStr, modelStr, timeStr, msgStr)
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

// renderFooter renders the status bar at the bottom of the session list.
func (r *DefaultSessionRenderer) renderFooter(data SessionViewModel) string {
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
	return r.styles.Help.Render(fmt.Sprintf("  [%s] %s %s %s %s %s%s",
		data.PlatformFilter,
		r.styles.Key.Render("Enter"), r.styles.Key.Render("n"),
		r.styles.Key.Render("↑↓"), r.styles.Key.Render("Esc"),
		r.styles.Key.Render("?"),
		r.styles.Subtle.Render(scrollHint))) + "\n"
}
