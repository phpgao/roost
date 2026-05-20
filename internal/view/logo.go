package view

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/phpgao/roost/internal/types"
)

const asciiLogo = `
        _
       //\\
      //  \\   ____  _             _
     //    \\ |  _ \| | __ _ _ __ | |_
    //______\\| |_) | |/ _` + "`" + ` | '_ \| __|
            ||  _ <| | (_| | | | | |_
            ||_| \_\_|\__,_|_| |_|\__|
`

const (
	logoSubtitle = "Interactive AI Session Manager"
	logoGitHub   = "https://github.com/phpgao/roost"
)

// RenderLogo renders the ASCII art logo, subtitle, and GitHub link.
func RenderLogo(s Styles, width int) string {
	var sb strings.Builder
	logo := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(types.ColorBlue)).Render(asciiLogo)
	sb.WriteString(logo)
	sb.WriteString(s.Subtle.Render("  "+logoSubtitle) + "\n")
	sb.WriteString(s.Subtle.Render("  "+logoGitHub) + "\n")
	sb.WriteString(s.Separator.Render(strings.Repeat("-", min(width, types.MaxWidth))) + "\n")
	return sb.String()
}
