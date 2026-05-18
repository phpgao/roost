package main

import "charm.land/lipgloss/v2"

var (
	// 选中行：淡蓝背景 + 前景色保持
	styleSelected = lipgloss.NewStyle().
			Background(lipgloss.Color(colorBgSelected)).
			Foreground(lipgloss.Color(colorLabelPrimary))

	styleNormal = lipgloss.NewStyle()

	styleTime = lipgloss.NewStyle().Foreground(lipgloss.Color(colorLabelTertiary))

	styleModel = lipgloss.NewStyle().Foreground(lipgloss.Color(colorLabelTertiary))

	styleTitle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(colorLabelPrimary))

	styleSubtle = lipgloss.NewStyle().Foreground(lipgloss.Color(colorLabelTertiary))

	styleSearch = lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorYellow)).
			Bold(true)

	styleDeleteBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(colorRed)).
			Padding(1, 2)

	styleDeleteWarning = lipgloss.NewStyle().Foreground(lipgloss.Color(colorRed)).Bold(true)

	styleHelp = lipgloss.NewStyle().Foreground(lipgloss.Color(colorLabelTertiary))

	styleMsgCount = lipgloss.NewStyle().Foreground(lipgloss.Color(colorLabelTertiary))

	// 选中行左侧指示条
	styleIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color(colorBlue)).Bold(true)

	// 未选中行左侧占位
	styleIndicatorOff = lipgloss.NewStyle().Foreground(lipgloss.Color(colorLabelQuaternary))

	// platform icon styles
	stylePlatformCodeBuddy = lipgloss.NewStyle().Foreground(lipgloss.Color(colorGreen)).Bold(true)
	stylePlatformClaude    = lipgloss.NewStyle().Foreground(lipgloss.Color(colorOrange)).Bold(true)
	stylePlatformGemini    = lipgloss.NewStyle().Foreground(lipgloss.Color(colorGemini)).Bold(true)
	stylePlatformCodex     = lipgloss.NewStyle().Foreground(lipgloss.Color(colorCodex)).Bold(true)
	stylePlatformCopilot   = lipgloss.NewStyle().Foreground(lipgloss.Color(colorCopilot)).Bold(true)
	stylePlatformOpenCode  = lipgloss.NewStyle().Foreground(lipgloss.Color(colorOpenCode)).Bold(true)

	// separator line
	styleSeparator = lipgloss.NewStyle().Foreground(lipgloss.Color("rgba(84, 84, 88, 0.6)"))

	// footer key highlight
	styleKey = lipgloss.NewStyle().Foreground(lipgloss.Color(colorBlue)).Bold(true)
)

// platformStyle returns the lipgloss style for a given platform's icon color.
func platformStyle(p Platform) lipgloss.Style {
	switch p {
	case PlatformCodeBuddy:
		return stylePlatformCodeBuddy
	case PlatformClaude:
		return stylePlatformClaude
	case PlatformGemini:
		return stylePlatformGemini
	case PlatformCodex:
		return stylePlatformCodex
	case PlatformCopilot:
		return stylePlatformCopilot
	case PlatformOpenCode:
		return stylePlatformOpenCode
	default:
		return styleSubtle
	}
}

// platformIcon renders a colored bullet + short name for a platform.
func platformIcon(p Platform) string {
	return platformStyle(p).Render(p.Icon() + p.Name())
}

// platformDot renders only a colored bullet (no short name), for inline use.
func platformDot(p Platform) string {
	return platformStyle(p).Render(p.Icon())
}

// RenderColoredCommand renders a command string prefixed with "$ " using the platform's color.
func RenderColoredCommand(p Platform, cmd string) string {
	return platformStyle(p).Render("$ " + cmd)
}
