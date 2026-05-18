package main

import (
	"fmt"
	"strings"
)

func renderDeleteView(m *Model) string {
	var subject, action string

	switch m.delTarget {
	case deleteTargetSession:
		if m.delSession != nil {
			subject = fmt.Sprintf("%q", m.delSession.Title)
			action = "Delete this session?"
		}
	case deleteTargetProject:
		if m.delProject != nil {
			subject = fmt.Sprintf("%q", m.delProject.Name)
			count := len(m.delProject.Sessions)
			action = fmt.Sprintf("Delete entire project? (%d sessions)", count)
		}
	case deleteTargetBatch:
		n := len(m.selectedSet)
		if m.selectedProject != nil {
			action = fmt.Sprintf("Delete %d sessions?", n)
		} else {
			action = fmt.Sprintf("Delete %d projects?", n)
		}
		subject = ""
	}

	content := strings.Join([]string{
		styleDeleteWarning.Render(action),
		"  " + subject,
		"",
		styleSubtle.Render("This action cannot be undone"),
		"",
		"  [" + styleDeleteWarning.Render("Confirm Enter") + "]  [" + styleSubtle.Render("Cancel Esc") + "]",
	}, "\n")

	box := styleDeleteBox.Render(content)

	boxLines := strings.Split(box, "\n")
	maxW := 0
	for _, l := range boxLines {
		if len(l) > maxW {
			maxW = len(l)
		}
	}
	leftPad := (m.width - maxW) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	topPad := (m.height - len(boxLines)) / 2
	if topPad < 0 {
		topPad = 0
	}

	var sb strings.Builder
	sb.WriteString(strings.Repeat("\n", topPad))
	pad := strings.Repeat(" ", leftPad)
	for _, l := range boxLines {
		sb.WriteString(pad + l + "\n")
	}
	return sb.String()
}
