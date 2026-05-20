// Package resume defines the ResumeStrategy interface and message types
// for abstracting how the TUI resumes/launches agent sessions.
package resume

import (
	tea "charm.land/bubbletea/v2"
	"github.com/phpgao/roost/internal/config"
	"github.com/phpgao/roost/internal/scanner"
	"github.com/phpgao/roost/internal/types"
)

// ResumeStrategy abstracts how the TUI resumes/launches agent sessions.
type ResumeStrategy interface {
	// Resume resumes an existing session.
	Resume(session types.Session, scanners []scanner.Scanner, cfg config.Config) tea.Cmd
	// LaunchNew starts a new session for the given platform in the given project.
	LaunchNew(platform types.Platform, projectPath string, cfg config.Config) tea.Cmd
}

// ResumeRequestMsg signals the app to quit and exec-resume (replace mode).
type ResumeRequestMsg struct {
	Session types.Session
}

// LaunchNewRequestMsg signals the app to quit and launch a new session (replace mode).
type LaunchNewRequestMsg struct {
	Platform    types.Platform
	ProjectPath string
}
