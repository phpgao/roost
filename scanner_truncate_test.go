package main

import (
	"fmt"
	"testing"
	"time"
)

// =====================================================================
// truncateKeepEnd
// =====================================================================

func TestTruncateKeepEnd(t *testing.T) {
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
			name: "exact width unchanged",
			s:    "hello",
			n:    5,
			want: "hello",
		},
		{
			name: "long ascii path keeps tail",
			s:    "/home/user/code/project",
			n:    12,
			want: "…de/project",
		},
		{
			name: "empty string unchanged",
			s:    "",
			n:    5,
			want: "",
		},
		{
			name: "budget zero returns ellipsis",
			s:    "hello",
			n:    1,
			want: "…",
		},
		{
			name: "chinese chars handled by visual width",
			// "…" = 2 cols; budget = 6-2 = 4; from right: "世界" = 4 cols → "…世界" = 6 cols
			s:    "你好世界",
			n:    6,
			want: "…世界",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateKeepEnd(tt.s, tt.n)
			if got != tt.want {
				t.Errorf("truncateKeepEnd(%q, %d) = %q, want %q", tt.s, tt.n, got, tt.want)
			}
		})
	}
}

// =====================================================================
// truncateWidth
// =====================================================================

func TestTruncateWidth(t *testing.T) {
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
			name: "exact width unchanged",
			s:    "hello",
			n:    5,
			want: "hello",
		},
		{
			name: "long ascii string truncated with ellipsis",
			// "…" = 2 cols; budget = 7-2 = 5; "hello" = 5 cols → "hello…" = 7 cols
			s:    "hello world",
			n:    7,
			want: "hello…",
		},
		{
			name: "empty string unchanged",
			s:    "",
			n:    5,
			want: "",
		},
		{
			name: "budget zero returns ellipsis",
			s:    "hello",
			n:    1,
			want: "…",
		},
		{
			name: "chinese chars handled by visual width",
			// "你好世界" = 8 cols; need ≤ 6: keep "你好" (4) + "…" (1) = 5 ≤ 6
			s:    "你好世界",
			n:    6,
			want: "你好…",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateWidth(tt.s, tt.n)
			if got != tt.want {
				t.Errorf("truncateWidth(%q, %d) = %q, want %q", tt.s, tt.n, got, tt.want)
			}
		})
	}
}

// =====================================================================
// ScanProjectsParallel
// =====================================================================

type stubScanner struct {
	platform  Platform
	projects  []Project
	err       error
	deleteErr error
	resumeCmd []string
}

func (s *stubScanner) Platform() Platform { return s.platform }
func (s *stubScanner) DataDir() string    { return "" }
func (s *stubScanner) ScanProjects() ([]Project, error) {
	return s.projects, s.err
}
func (s *stubScanner) DeleteSession(_ Session) error { return s.deleteErr }
func (s *stubScanner) DeleteProject(_ Project) error { return s.deleteErr }
func (s *stubScanner) ResumeCmd(_ Session) []string  { return s.resumeCmd }

func TestScanProjectsParallel_Merges(t *testing.T) {
	now := time.Now()
	sc1 := &stubScanner{
		platform: PlatformCodeBuddy,
		projects: []Project{{Name: "p/a", FullPath: "/a", Sessions: []Session{{ID: "s1", LastActive: now}}}},
	}
	sc2 := &stubScanner{
		platform: PlatformClaude,
		projects: []Project{{Name: "p/b", FullPath: "/b", Sessions: []Session{{ID: "s2", LastActive: now.Add(-time.Hour)}}}},
	}

	result := ScanProjectsParallel([]Scanner{sc1, sc2})
	if len(result) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(result))
	}
	// most recent (/a) should come first
	if result[0].FullPath != "/a" {
		t.Errorf("expected /a first, got %q", result[0].FullPath)
	}
}

func TestScanProjectsParallel_ErrorScanner(t *testing.T) {
	sc := &stubScanner{
		platform: PlatformCodeBuddy,
		err:      fmt.Errorf("scan failed"),
	}
	result := ScanProjectsParallel([]Scanner{sc})
	// error scanner returns nil; mergeProjects gets empty list
	if len(result) != 0 {
		t.Errorf("expected 0 projects for error scanner, got %d", len(result))
	}
}

func TestScanProjectsParallel_Empty(t *testing.T) {
	result := ScanProjectsParallel(nil)
	if len(result) != 0 {
		t.Errorf("expected 0 projects for nil scanners, got %d", len(result))
	}
}

func TestScanProjectsParallel_EmptyProjects(t *testing.T) {
	sc := &stubScanner{platform: PlatformClaude, projects: nil}
	result := ScanProjectsParallel([]Scanner{sc})
	if len(result) != 0 {
		t.Errorf("expected 0 projects, got %d", len(result))
	}
}
