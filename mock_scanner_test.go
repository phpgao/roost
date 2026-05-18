package main

import (
	"time"
)

// makeTestProjects creates predictable test data for integration tests.
func makeTestProjects() []Project {
	now := time.Now()
	return []Project{
		{
			Name:     "user/roost",
			FullPath: "/home/user/roost",
			Sessions: []Session{
				{ID: "s1", Platform: PlatformCodeBuddy, Title: "Fix bug", Model: "sonnet", LastActive: now.Add(-1 * time.Hour), MsgCount: 2, ResumeArg: "s1"},
				{ID: "s2", Platform: PlatformClaude, Title: "Add feature", Model: "opus", LastActive: now.Add(-2 * time.Hour), MsgCount: 4, ResumeArg: "s2"},
			},
		},
		{
			Name:     "team/projA",
			FullPath: "/Users/team/projA",
			Sessions: []Session{
				{ID: "s3", Platform: PlatformGemini, Title: "Deploy", Model: "gemini-pro", LastActive: now.Add(-10 * time.Minute), MsgCount: 1, ResumeArg: "s3"},
			},
		},
	}
}

// makeModelWithProjects creates a Model pre-populated with test projects.
func makeModelWithProjects() *Model {
	projects := makeTestProjects()
	m := &Model{
		scanners:           make([]Scanner, 0), // not needed since we pre-populate
		installedPlatforms: []Platform{PlatformCodeBuddy, PlatformClaude, PlatformGemini, PlatformCodex, PlatformCopilot, PlatformOpenCode},
		projects:           projects,
		filteredProjects:   projects,
		screen:             screenProject,
		cursor:             0,
		platformFilter:     -1,
	}
	return m
}
