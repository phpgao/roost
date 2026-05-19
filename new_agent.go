package main

import (
	"strings"
)

// renderNewAgentView 渲染 agent 选择页（按 n/N 进入）
func renderNewAgentView(m *Model) string {
	var sb strings.Builder

	w := min(m.width, maxWidth)

	// title
	sb.WriteString(styleTitle.Render("  New session — select agent") + "\n")
	sep := styleSeparator.Render(strings.Repeat("-", w))
	sb.WriteString(sep + "\n")

	// viewport
	viewHeight := calcViewHeight(m.height)
	start, end := calcViewport(len(m.installedPlatforms), m.newAgentCursor, viewHeight)

	contentLines := 0

	if len(m.installedPlatforms) == 0 {
		sb.WriteString(styleSubtle.Render("  no platforms installed") + "\n")
		contentLines = 1
	}

	for i := start; i < end; i++ {
		p := m.installedPlatforms[i]

		indicator := styleIndicatorOff.Render("  ")
		if i == m.newAgentCursor {
			indicator = styleIndicator.Render("▸ ")
		}

		line := indicator + platformIcon(p)

		if i == m.newAgentCursor {
			sb.WriteString(styleSelected.Render(line) + "\n")
		} else {
			sb.WriteString(styleNormal.Render(line) + "\n")
		}
		contentLines++
	}

	// pad to viewHeight
	if len(m.installedPlatforms) > viewHeight {
		for contentLines < viewHeight {
			sb.WriteString("\n")
			contentLines++
		}
	}

	sb.WriteString(sep + "\n")

	scrollHint := renderScrollHint(start, end, len(m.installedPlatforms), m.newAgentCursor)
	sb.WriteString(renderFooter("%s %s %s"+styleSubtle.Render(scrollHint),
		styleKey.Render("↑↓/j/k"), styleKey.Render("Enter"), styleKey.Render("Esc"),
	))

	return sb.String()
}
