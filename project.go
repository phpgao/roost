package main

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

func renderProjectView(m *Model) string {
	var sb strings.Builder

	// cap layout width so the UI stays readable on wide terminals
	w := min(m.width, maxWidth)

	title := styleTitle.Render("  roost")
	// legend: only show installed platforms
	var legendParts []string
	for _, p := range m.installedPlatforms {
		legendParts = append(legendParts, "  "+platformIcon(p))
	}
	legend := styleSubtle.Render(strings.Join(legendParts, ""))
	sb.WriteString(title + "  " + legend + "\n")
	sep := styleSeparator.Render(strings.Repeat("-", w))
	sb.WriteString(sep + "\n")

	list := m.filteredProjects

	// column widths
	const (
		platformColW = 20
		timeColW     = 12
		separators   = 6
	)
	nameColW := w - platformColW - timeColW - separators
	if nameColW < 20 {
		nameColW = 20
	}

	// column header
	colNameStyle := lipgloss.NewStyle().Width(nameColW)
	colPlatformStyle := lipgloss.NewStyle().Width(platformColW)
	colTimeStyle := lipgloss.NewStyle().Width(timeColW)
	headerLine := "    " + // marker(2) + indicator(2)
		colNameStyle.Render(styleSubtle.Render("PATH")) + " " +
		colPlatformStyle.Render(styleSubtle.Render("PLATFORMS")) +
		colTimeStyle.Render(styleSubtle.Render("LAST ACTIVE"))
	sb.WriteString(headerLine + "\n")

	// viewport: terminal - title(1) - separator(1) - footer(2)
	viewHeight := calcViewHeight(m.height)
	start, end := calcViewport(len(list), m.cursor, viewHeight)

	// track how many content lines we've written (to pad up to viewHeight)
	contentLines := 0

	if len(list) == 0 {
		if m.searchQuery != "" {
			sb.WriteString(styleSubtle.Render("  no results for: "+m.searchQuery) + "\n")
			contentLines++
		} else {
			sb.WriteString(styleSubtle.Render("  no sessions found") + "\n")
			sb.WriteString(styleSubtle.Render("  press r to refresh") + "\n")
			contentLines += 2
		}
	}

	// column style built once; nameColW depends on terminal width
	colName := lipgloss.NewStyle().Width(nameColW)

	// ordered platforms for consistent column rendering
	platformOrder := []Platform{PlatformCodeBuddy, PlatformClaude, PlatformGemini, PlatformCodex, PlatformCopilot, PlatformOpenCode}

	for i := start; i < end; i++ {
		p := list[i]

		// count sessions per platform using a fixed-size array (no heap alloc)
		var counts [6]int
		for _, s := range p.Sessions {
			counts[s.Platform]++
		}

		// build platform string and pad to fixed width
		var platformRendered string
		for _, platform := range platformOrder {
			if n := counts[platform]; n > 0 {
				platformRendered += fmt.Sprintf("%s%-2d ", platformDot(platform), n)
			}
		}
		platformPadded := platformRendered + strings.Repeat(" ", max(0, platformColW-lipgloss.Width(platformRendered)))

		timeStr := styleTime.Render(relativeTime(p.LastActive()))

		marker := "  "
		if m.selecting {
			if m.selectedSet[p.FullPath] {
				marker = " ●"
			} else {
				marker = " ○"
			}
		}

		indicator := styleIndicatorOff.Render("  ")
		if i == m.cursor {
			indicator = styleIndicator.Render("▸ ")
		}

		bold := i == m.cursor || (m.selecting && m.selectedSet[p.FullPath])
		var nameCol string
		if bold {
			nameCol = colName.Bold(true).Render(truncateKeepEnd(p.FullPath, nameColW))
			timeStr = styleTime.Bold(true).Render(relativeTime(p.LastActive()))
		} else {
			nameCol = colName.Render(truncateKeepEnd(p.FullPath, nameColW))
		}
		line := fmt.Sprintf("%s%s%s %s %s", marker, indicator, nameCol, platformPadded, timeStr)

		if i == m.cursor {
			sb.WriteString(styleSelected.Render(line) + "\n")
		} else {
			sb.WriteString(styleNormal.Render(line) + "\n")
		}
		contentLines++
	}

	// only pad to viewHeight when list is long enough to scroll;
	// for short lists let the footer follow immediately after the last row
	if len(list) > viewHeight {
		for contentLines < viewHeight {
			sb.WriteString("\n")
			contentLines++
		}
	}

	sb.WriteString(sep + "\n")

	if m.searching {
		sb.WriteString(styleSearch.Render("  / "+m.searchQuery+"_") + "\n")
	} else if m.escHint {
		sb.WriteString(styleDeleteWarning.Render("  press Esc again to quit") + "\n")
	} else {
		scrollHint := renderScrollHint(start, end, len(list), m.cursor)
		if m.selecting {
			sb.WriteString(renderFooter("Space toggle  D delete(%d)  Esc cancel"+styleSubtle.Render(scrollHint), len(m.selectedSet)))
		} else {
			sb.WriteString(renderFooter("[%s] %s %s %s %s"+styleSubtle.Render(scrollHint),
				m.platformFilterLabel(),
				styleKey.Render("Enter"), styleKey.Render("↑↓"), styleKey.Render("Esc"), styleKey.Render("?"),
			))
		}
	}

	return sb.String()
}

// renderFooter formats a footer line with optional key highlights.
func renderFooter(format string, args ...any) string {
	return styleHelp.Render(fmt.Sprintf("  "+format, args...))
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
