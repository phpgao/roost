package resume

import (
	tea "charm.land/bubbletea/v2"
	"github.com/phpgao/roost/internal/config"
	"github.com/phpgao/roost/internal/scanner"
	"github.com/phpgao/roost/internal/types"
)

// ReplaceStrategy exits the TUI process and replaces it with the agent (syscall.Exec).
type ReplaceStrategy struct{}

// Resume sends a ResumeRequestMsg to signal the app to quit and exec-resume.
func (ReplaceStrategy) Resume(session types.Session, _ []scanner.Scanner, _ config.Config) tea.Cmd {
	return func() tea.Msg { return ResumeRequestMsg{Session: session} }
}

// LaunchNew sends a LaunchNewRequestMsg to signal the app to quit and launch a new session.
func (ReplaceStrategy) LaunchNew(platform types.Platform, projectPath string, _ config.Config) tea.Cmd {
	return func() tea.Msg { return LaunchNewRequestMsg{Platform: platform, ProjectPath: projectPath} }
}
