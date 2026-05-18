package main

import (
	"reflect"
	"testing"
)

func TestCalcViewport(t *testing.T) {
	tests := []struct {
		name       string
		total      int
		cursor     int
		viewHeight int
		wantStart  int
		wantEnd    int
	}{
		{
			name:       "list fits in viewport, no scroll needed",
			total:      5,
			cursor:     2,
			viewHeight: 10,
			wantStart:  0,
			wantEnd:    5,
		},
		{
			name:       "cursor at top, viewport starts at 0",
			total:      50,
			cursor:     0,
			viewHeight: 10,
			wantStart:  0,
			wantEnd:    10,
		},
		{
			name:       "cursor in middle, centered in viewport",
			total:      50,
			cursor:     25,
			viewHeight: 10,
			wantStart:  20,
			wantEnd:    30,
		},
		{
			name:       "cursor near bottom, viewport clamped to end",
			total:      50,
			cursor:     48,
			viewHeight: 10,
			wantStart:  40,
			wantEnd:    50,
		},
		{
			name:       "cursor at last item",
			total:      50,
			cursor:     49,
			viewHeight: 10,
			wantStart:  40,
			wantEnd:    50,
		},
		{
			name:       "viewport height zero returns empty range",
			total:      50,
			cursor:     5,
			viewHeight: 0,
			wantStart:  0,
			wantEnd:    0,
		},
		{
			name:       "empty list",
			total:      0,
			cursor:     0,
			viewHeight: 10,
			wantStart:  0,
			wantEnd:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := calcViewport(tt.total, tt.cursor, tt.viewHeight)
			if start != tt.wantStart || end != tt.wantEnd {
				t.Errorf("calcViewport(%d, %d, %d) = (%d, %d), want (%d, %d)",
					tt.total, tt.cursor, tt.viewHeight, start, end, tt.wantStart, tt.wantEnd)
			}
		})
	}
}

func TestWrapSuspendCommand(t *testing.T) {
	tests := []struct {
		name string
		p    Platform
		argv []string
		want []string
	}{
		{
			name: "empty argv",
			p:    PlatformClaude,
			argv: nil,
			want: nil,
		},
		{
			name: "simple command",
			p:    PlatformCodex,
			argv: []string{"codex", codexCmdResume, "abc-123"},
			want: []string{
				"sh",
				"-c",
				`printf '%s\n' "$1"; shift; exec "$@"`,
				"sh",
				RenderColoredCommand(PlatformCodex, "codex resume abc-123"),
				"codex",
				codexCmdResume,
				"abc-123",
			},
		},
		{
			name: "quotes shell sensitive args",
			p:    PlatformClaude,
			argv: []string{"claude", "--resume", "id with space", "it's"},
			want: []string{
				"sh",
				"-c",
				`printf '%s\n' "$1"; shift; exec "$@"`,
				"sh",
				RenderColoredCommand(PlatformClaude, "claude --resume 'id with space' 'it'\"'\"'s'"),
				"claude",
				"--resume",
				"id with space",
				"it's",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wrapSuspendCommand(tt.p, tt.argv)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("wrapSuspendCommand(%v) = %#v, want %#v", tt.argv, got, tt.want)
			}
		})
	}
}

func TestShellQuote(t *testing.T) {
	tests := []struct {
		name string
		arg  string
		want string
	}{
		{name: "safe arg", arg: "codex", want: "codex"},
		{name: "empty arg", arg: "", want: "''"},
		{name: "space", arg: "hello world", want: "'hello world'"},
		{name: "single quote", arg: "it's", want: `'it'"'"'s'`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shellQuote(tt.arg)
			if got != tt.want {
				t.Fatalf("shellQuote(%q) = %q, want %q", tt.arg, got, tt.want)
			}
		})
	}
}
