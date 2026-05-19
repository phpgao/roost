package main

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

func renderSessionView(m *Model) string {
	var sb strings.Builder

	// cap layout width so the UI stays readable on wide terminals
	w := min(m.width, maxWidth)

	// breadcrumb header
	projName := ""
	if m.selectedProject != nil {
		projName = projectShortName(m.selectedProject.FullPath)
	}
	crumb := styleSubtle.Render("roost > ")
	// legend: only show installed platforms
	var legendParts []string
	for _, p := range m.installedPlatforms {
		legendParts = append(legendParts, "  "+platformIcon(p))
	}
	legend := styleSubtle.Render(strings.Join(legendParts, ""))
	header := fmt.Sprintf("  %s%s", crumb, styleTitle.Render(projName))
	sb.WriteString(header + "  " + legend + "\n")
	sep := styleSeparator.Render(strings.Repeat("-", w))
	sb.WriteString(sep + "\n")

	list := m.filteredSessions

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
			sb.WriteString(styleSubtle.Render("  no sessions") + "\n")
			sb.WriteString(styleSubtle.Render("  press r to refresh") + "\n")
			contentLines += 2
		}
	}

	// fixed column widths (visible chars)
	const (
		markerW = 3 // "  " or " ●"
		indentW = 2 // cursor indicator
		iconW   = 3 // "● "
		modelW  = 22
		timeW   = 12
		msgW    = 6
		// cap title so short titles don't leave huge whitespace
		maxTitleW = 40
	)
	titleW := w - markerW - indentW - iconW - modelW - timeW - msgW - 2
	if titleW < 10 {
		titleW = 10
	}
	if titleW > maxTitleW {
		titleW = maxTitleW
	}

	// column styles built once per render (titleW depends on terminal width, others are constant)
	colMarker := lipgloss.NewStyle().Width(markerW)
	colIcon := lipgloss.NewStyle().Width(iconW)
	colTitle := lipgloss.NewStyle().Width(titleW)
	colModel := lipgloss.NewStyle().Width(modelW)
	colTime := lipgloss.NewStyle().Width(timeW)
	colMsg := lipgloss.NewStyle().Width(msgW)

	// column header
	headerLine := colMarker.Render("") + "  " + // marker + indicator placeholder
		colIcon.Render("") +
		colTitle.Render(styleSubtle.Render("TITLE")) +
		colModel.Render(styleSubtle.Render("MODEL")) + " " +
		colTime.Render(styleSubtle.Render("LAST ACTIVE")) +
		colMsg.Render(styleSubtle.Render("MSGS"))
	sb.WriteString(headerLine + "\n")

	for i := start; i < end; i++ {
		s := list[i]

		// sanitize title: replace newlines/tabs with a space so the column stays single-line
		title := strings.Map(func(r rune) rune {
			if r == '\n' || r == '\r' || r == '\t' {
				return ' '
			}
			return r
		}, s.Title)
		// append agent type inline when present (avoids a dedicated wide column)
		if s.AgentType != "" && s.AgentType != typeAgentCLI {
			title += " [" + s.AgentType + "]"
		}

		marker := "  "
		if m.selecting {
			if m.selectedSet[s.ID] {
				marker = " ●"
			} else {
				marker = " ○"
			}
		}
		markerRendered := colMarker.Render(marker)

		indicator := styleIndicatorOff.Render("  ")
		if i == m.cursor {
			indicator = styleIndicator.Render("▸ ")
		}

		icon := colIcon.Render(platformDot(s.Platform))
		bold := i == m.cursor || (m.selecting && m.selectedSet[s.ID])
		var titleStr, modelRendered, timeRendered, msgRendered string
		if bold {
			titleStr = colTitle.Bold(true).Render(truncateWidth(title, titleW))
			modelRendered = colModel.Render(styleModel.Bold(true).Render(truncateWidth(s.Model, modelW)))
			timeRendered = colTime.Render(styleTime.Bold(true).Render(relativeTime(s.LastActive)))
			msgRendered = colMsg.Render(styleMsgCount.Bold(true).Render(fmt.Sprintf("%d", s.MsgCount)))
		} else {
			titleStr = colTitle.Render(truncateWidth(title, titleW))
			modelRendered = colModel.Render(styleModel.Render(truncateWidth(s.Model, modelW)))
			timeRendered = colTime.Render(styleTime.Render(relativeTime(s.LastActive)))
			msgRendered = colMsg.Render(styleMsgCount.Render(fmt.Sprintf("%d", s.MsgCount)))
		}

		line := markerRendered + indicator + icon + titleStr + modelRendered + " " + timeRendered + msgRendered

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
			sb.WriteString(renderFooter("[%s] %s %s %s %s %s"+styleSubtle.Render(scrollHint),
				m.platformFilterLabel(),
				styleKey.Render("Enter"), styleKey.Render("n"), styleKey.Render("↑↓"), styleKey.Render("Esc"), styleKey.Render("?"),
			))
		}
	}

	return sb.String()
}
