package main

import (
	"testing"
	"time"
)

func TestRelativeTime(t *testing.T) {
	tests := []struct {
		name string
		t    time.Time
		want string
	}{
		{
			name: "zero time returns dash",
			t:    time.Time{},
			want: "-",
		},
		{
			name: "less than a minute ago",
			t:    time.Now().Add(-30 * time.Second),
			want: "just now",
		},
		{
			name: "minutes ago",
			t:    time.Now().Add(-5 * time.Minute),
			want: "5m ago",
		},
		{
			name: "hours ago",
			t:    time.Now().Add(-3 * time.Hour),
			want: "3h ago",
		},
		{
			name: "days ago",
			t:    time.Now().Add(-2 * 24 * time.Hour),
			want: "2d ago",
		},
		{
			name: "more than a week shows date",
			t:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local),
			want: "2026-01-01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := relativeTime(tt.t)
			if got != tt.want {
				t.Errorf("relativeTime() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestProjectShortName(t *testing.T) {
	tests := []struct {
		name     string
		fullPath string
		want     string
	}{
		{
			name:     "normal two-segment path",
			fullPath: "/home/user/roost",
			want:     "user/roost",
		},
		{
			name:     "trailing slash stripped",
			fullPath: "/home/user/roost/",
			want:     "user/roost",
		},
		{
			name:     "single segment path",
			fullPath: "/roost",
			want:     "roost",
		},
		{
			name:     "root path",
			fullPath: "/",
			want:     "/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := projectShortName(tt.fullPath)
			if got != tt.want {
				t.Errorf("projectShortName(%q) = %q, want %q", tt.fullPath, got, tt.want)
			}
		})
	}
}

func TestMergeProjects(t *testing.T) {
	now := time.Now()

	t.Run("merges same FullPath from different scanners", func(t *testing.T) {
		fromCB := []Project{
			{Name: "user/roost", FullPath: "/home/user/roost", Sessions: []Session{
				{ID: "s1", Platform: 0, LastActive: now.Add(-1 * time.Hour)},
			}},
		}
		fromClaude := []Project{
			{Name: "user/roost", FullPath: "/home/user/roost", Sessions: []Session{
				{ID: "s2", Platform: 1, LastActive: now.Add(-30 * time.Minute)},
			}},
		}

		result := mergeProjects([][]Project{fromCB, fromClaude})

		if len(result) != 1 {
			t.Fatalf("expected 1 merged project, got %d", len(result))
		}
		if len(result[0].Sessions) != 2 {
			t.Errorf("expected 2 sessions, got %d", len(result[0].Sessions))
		}
	})

	t.Run("sorts by most recent active time descending", func(t *testing.T) {
		older := []Project{
			{Name: "proj/old", FullPath: "/old", Sessions: []Session{
				{ID: "s1", LastActive: now.Add(-10 * time.Hour)},
			}},
		}
		newer := []Project{
			{Name: "proj/new", FullPath: "/new", Sessions: []Session{
				{ID: "s2", LastActive: now.Add(-1 * time.Minute)},
			}},
		}

		result := mergeProjects([][]Project{older, newer})

		if len(result) != 2 {
			t.Fatalf("expected 2 projects, got %d", len(result))
		}
		if result[0].FullPath != "/new" {
			t.Errorf("expected /new first (most recent), got %q", result[0].FullPath)
		}
	})

	t.Run("empty input returns empty", func(t *testing.T) {
		result := mergeProjects(nil)
		if len(result) != 0 {
			t.Errorf("expected empty, got %d", len(result))
		}
	})
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name string
		s    string
		n    int
		want string
	}{
		{
			name: "short string unchanged",
			s:    "hello",
			n:    10,
			want: "hello",
		},
		{
			name: "exact length unchanged",
			s:    "hello",
			n:    5,
			want: "hello",
		},
		{
			name: "long string truncated with ellipsis",
			s:    "hello world",
			n:    5,
			want: "hello…",
		},
		{
			name: "empty string",
			s:    "",
			n:    5,
			want: "",
		},
		{
			name: "chinese characters count as one rune each",
			s:    "你好世界测试",
			n:    4,
			want: "你好世界…",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.s, tt.n)
			if got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.s, tt.n, got, tt.want)
			}
		})
	}
}

func TestPlatformMethods(t *testing.T) {
	tests := []struct {
		platform  Platform
		icon      string
		name      string
		shortName string
	}{
		{PlatformCodeBuddy, "●", "CodeBuddy", "CB"},
		{PlatformClaude, "●", "Claude", "CL"},
		{PlatformGemini, "●", "Gemini", "GE"},
		{PlatformCodex, "●", "Codex", "CX"},
		{PlatformCopilot, "●", "Copilot", "Co"},
		{PlatformOpenCode, "●", "OpenCode", "OC"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.platform.Icon(); got != tt.icon {
				t.Errorf("Icon() = %q, want %q", got, tt.icon)
			}
			if got := tt.platform.Name(); got != tt.name {
				t.Errorf("Name() = %q, want %q", got, tt.name)
			}
			if got := tt.platform.ShortName(); got != tt.shortName {
				t.Errorf("ShortName() = %q, want %q", got, tt.shortName)
			}
		})
	}
}

func TestPlatformUnknown(t *testing.T) {
	unknown := Platform(99)
	if got := unknown.Icon(); got != "○" {
		t.Errorf("unknown Icon() = %q, want ○", got)
	}
	if got := unknown.Name(); got != "Unknown" {
		t.Errorf("unknown Name() = %q, want Unknown", got)
	}
	if got := unknown.ShortName(); got != "??" {
		t.Errorf("unknown ShortName() = %q, want ??", got)
	}
}

func TestProjectLastActive(t *testing.T) {
	t1 := time.Now().Add(-1 * time.Hour)
	t2 := time.Now().Add(-30 * time.Minute)

	p := Project{
		Sessions: []Session{
			{LastActive: t1},
			{LastActive: t2},
		},
	}
	if !p.LastActive().Equal(t2) {
		t.Errorf("LastActive() = %v, want %v", p.LastActive(), t2)
	}
}

func TestProjectLastActiveEmpty(t *testing.T) {
	p := Project{Sessions: []Session{}}
	if !p.LastActive().IsZero() {
		t.Errorf("LastActive() with no sessions should be zero time, got %v", p.LastActive())
	}
}
