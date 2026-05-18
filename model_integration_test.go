package main

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// =====================================================================
// Init / NewModel
// =====================================================================

func TestNewModel(t *testing.T) {
	m := newModel(nil)
	if m.screen != screenProject {
		t.Errorf("initial screen = %v, want screenProject", m.screen)
	}
	if m.cursor != 0 {
		t.Errorf("initial cursor = %d, want 0", m.cursor)
	}
	if m.platformFilter != -1 {
		t.Errorf("initial platformFilter = %d, want -1", m.platformFilter)
	}
	if m.width != 80 {
		t.Errorf("initial width = %d, want 80", m.width)
	}
}

func TestInit(t *testing.T) {
	m := newModel(nil)
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("Init() returned nil cmd")
	}
	// calling the cmd should produce a non-nil message
	msg := cmd()
	if msg == nil {
		t.Error("Init() cmd returned nil msg")
	}
}

// =====================================================================
// View rendering
// =====================================================================

func TestView_ProjectView(t *testing.T) {
	m := makeModelWithProjects()
	m.width = 80
	m.height = 24

	v := m.View()
	if !strings.Contains(v.Content, "roost") {
		t.Error("View should contain 'roost'")
	}
	if !strings.Contains(v.Content, "user/roost") {
		t.Error("View should contain project path")
	}
}

func TestView_SessionView(t *testing.T) {
	now := time.Now()
	m := makeModelWithProjects()
	m.width = 80
	m.height = 24
	m.screen = screenSession
	m.selectedProject = &Project{
		Name:     "user/roost",
		FullPath: "/home/user/roost",
		Sessions: []Session{
			{ID: "s1", Platform: PlatformCodeBuddy, Title: "Fix bug", Model: "sonnet", LastActive: now},
		},
	}
	m.filteredSessions = m.selectedProject.Sessions

	v := m.View()
	if !strings.Contains(v.Content, "Fix bug") {
		t.Error("session view should contain session title")
	}
}

func TestView_DeleteConfirm(t *testing.T) {
	m := makeModelWithProjects()
	m.width = 80
	m.height = 24
	m.screen = screenDeleteConfirm
	m.delTarget = deleteTargetSession
	m.delSession = &Session{ID: "s1", Title: "Test"}

	v := m.View()
	// delete view shows "Delete this session?" (capital D)
	if !strings.Contains(v.Content, "Delete") {
		t.Error("delete view should mention 'Delete'")
	}
	if !strings.Contains(v.Content, "cannot be undone") {
		t.Error("delete view should mention 'cannot be undone'")
	}
}

func TestView_HelpView(t *testing.T) {
	m := makeModelWithProjects()
	m.width = 80
	m.height = 24
	m.showHelp = true

	v := m.View()
	if !strings.Contains(v.Content, "Keyboard") {
		t.Error("help view should contain 'Keyboard'")
	}
}

// =====================================================================
// Update: project view key handling
// =====================================================================

func TestUpdate_Project_UpDown(t *testing.T) {
	m := makeModelWithProjects()

	// down (j) → cursor = 1
	nm, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	mm := nm.(*Model)
	if mm.cursor != 1 {
		t.Errorf("j: cursor = %d, want 1", mm.cursor)
	}

	// up (k) → cursor back to 0
	nm2, _ := mm.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	mm2 := nm2.(*Model)
	if mm2.cursor != 0 {
		t.Errorf("k: cursor = %d, want 0", mm2.cursor)
	}

	// k at top → no wrap (stays at 0)
	nm3, _ := mm2.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	mm3 := nm3.(*Model)
	if mm3.cursor != 0 {
		t.Errorf("k at top: cursor = %d, want 0 (no wrap)", mm3.cursor)
	}
}

func TestUpdate_Project_GG(t *testing.T) {
	m := makeModelWithProjects()

	// g → cursor = 0
	nm, _ := m.Update(tea.KeyPressMsg{Code: 'g'})
	mm := nm.(*Model)
	if mm.cursor != 0 {
		t.Errorf("g: cursor = %d, want 0", mm.cursor)
	}

	// G → cursor = last
	nm2, _ := mm.Update(tea.KeyPressMsg{Code: 'G'})
	mm2 := nm2.(*Model)
	if mm2.cursor != len(mm2.filteredProjects)-1 {
		t.Errorf("G: cursor = %d, want %d", mm2.cursor, len(mm2.filteredProjects)-1)
	}
}

func TestUpdate_Project_Enter(t *testing.T) {
	m := makeModelWithProjects()
	// move to second project which has sessions
	m.cursor = 1

	nm, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	mm := nm.(*Model)
	if mm.screen != screenSession {
		t.Errorf("after enter: screen = %v, want screenSession", mm.screen)
	}
	if mm.selectedProject == nil {
		t.Error("after enter: selectedProject should not be nil")
	}
}

func TestUpdate_Project_Tab(t *testing.T) {
	m := makeModelWithProjects()

	// tab → platformFilter = 0
	nm, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	mm := nm.(*Model)
	if mm.platformFilter != 0 {
		t.Errorf("tab: platformFilter = %d, want 0", mm.platformFilter)
	}

	// 6 more tabs → back to -1 (total 7 tabs: -1→0→1→2→3→4→5→-1)
	for i := 0; i < 6; i++ {
		nm, _ = mm.Update(tea.KeyPressMsg{Code: tea.KeyTab})
		mm = nm.(*Model)
	}
	if mm.platformFilter != -1 {
		t.Errorf("tab x6: platformFilter = %d, want -1", mm.platformFilter)
	}
}

func TestUpdate_Project_Slash(t *testing.T) {
	m := makeModelWithProjects()

	nm, _ := m.Update(tea.KeyPressMsg{Code: '/'})
	mm := nm.(*Model)
	if !mm.searching {
		t.Error("after /: searching should be true")
	}
}

func TestUpdate_Project_D(t *testing.T) {
	m := makeModelWithProjects()

	nm, _ := m.Update(tea.KeyPressMsg{Code: 'd'})
	mm := nm.(*Model)
	if mm.screen != screenDeleteConfirm {
		t.Errorf("after d: screen = %v, want screenDeleteConfirm", mm.screen)
	}
	if mm.delTarget != deleteTargetProject {
		t.Errorf("after d: delTarget = %v, want deleteTargetProject", mm.delTarget)
	}
}

func TestUpdate_Project_SpaceD(t *testing.T) {
	m := makeModelWithProjects()

	// space → selecting = true
	nm, _ := m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	mm := nm.(*Model)
	if !mm.selecting {
		t.Error("after space: selecting should be true")
	}
	if len(mm.selectedSet) != 1 {
		t.Errorf("after space: selectedSet size = %d, want 1", len(mm.selectedSet))
	}

	// D → batch delete confirm
	nm2, _ := mm.Update(tea.KeyPressMsg{Code: 'D'})
	mm2 := nm2.(*Model)
	if mm2.screen != screenDeleteConfirm {
		t.Error("after D: should show delete confirm")
	}
	if mm2.delTarget != deleteTargetBatch {
		t.Errorf("after D: delTarget = %v, want deleteTargetBatch", mm2.delTarget)
	}
}

func TestUpdate_Project_Esc(t *testing.T) {
	m := makeModelWithProjects()

	// first esc → escHint = true
	nm, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	mm := nm.(*Model)
	if !mm.escHint {
		t.Error("after first esc: escHint should be true")
	}

	// second esc (within 2s) → returns tea.Quit cmd
	m2 := makeModelWithProjects()
	m2.escHint = true
	m2.lastEscTime = time.Now().Add(-1 * time.Second) // within 2s window
	nm2, cmd := m2.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	_ = nm2.(*Model)
	if cmd == nil {
		t.Error("second esc: should return tea.Quit cmd")
	}
}

func TestUpdate_Project_Question(t *testing.T) {
	m := makeModelWithProjects()

	nm, _ := m.Update(tea.KeyPressMsg{Code: '?'})
	mm := nm.(*Model)
	if !mm.showHelp {
		t.Error("after ?: showHelp should be true")
	}

	// any key hides help
	nm2, _ := mm.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	mm2 := nm2.(*Model)
	if mm2.showHelp {
		t.Error("after esc in help: showHelp should be false")
	}
}

func TestUpdate_Project_Refresh(t *testing.T) {
	m := makeModelWithProjects()

	nm, cmd := m.Update(tea.KeyPressMsg{Code: 'r'})
	mm := nm.(*Model)
	if !mm.loading {
		t.Error("after r: loading should be true")
	}
	if cmd == nil {
		t.Error("after r: should return non-nil cmds")
	}
}

// =====================================================================
// Update: session view key handling
// =====================================================================

func TestUpdate_Session_Esc(t *testing.T) {
	now := time.Now()
	m := makeModelWithProjects()
	m.screen = screenSession
	m.selectedProject = &Project{
		Name:     "user/roost",
		FullPath: "/home/user/roost",
		Sessions: []Session{
			{ID: "s1", Platform: PlatformCodeBuddy, Title: "T", LastActive: now},
		},
	}
	m.filteredSessions = m.selectedProject.Sessions
	m.cursor = 0

	nm, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	mm := nm.(*Model)
	if mm.screen != screenProject {
		t.Errorf("after esc in session: screen = %v, want screenProject", mm.screen)
	}
}

func TestUpdate_Session_D(t *testing.T) {
	now := time.Now()
	m := makeModelWithProjects()
	m.screen = screenSession
	m.selectedProject = &Project{
		Name:     "user/roost",
		FullPath: "/home/user/roost",
		Sessions: []Session{
			{ID: "s1", Platform: PlatformCodeBuddy, Title: "T", LastActive: now},
		},
	}
	m.filteredSessions = m.selectedProject.Sessions

	nm, _ := m.Update(tea.KeyPressMsg{Code: 'd'})
	mm := nm.(*Model)
	if mm.screen != screenDeleteConfirm {
		t.Errorf("after d in session: screen = %v, want screenDeleteConfirm", mm.screen)
	}
}

func TestUpdate_Session_X(t *testing.T) {
	now := time.Now()
	m := makeModelWithProjects()
	m.screen = screenSession
	m.selectedProject = &Project{
		Name:     "user/roost",
		FullPath: "/home/user/roost",
		Sessions: []Session{
			{ID: "s1", Platform: PlatformCodeBuddy, Title: "T", LastActive: now},
		},
	}
	m.filteredSessions = m.selectedProject.Sessions

	nm, _ := m.Update(tea.KeyPressMsg{Code: 'x'})
	mm := nm.(*Model)
	if mm.screen != screenDeleteConfirm {
		t.Errorf("after x: screen = %v, want screenDeleteConfirm", mm.screen)
	}
}

func TestUpdate_Session_Tab(t *testing.T) {
	now := time.Now()
	m := makeModelWithProjects()
	m.screen = screenSession
	m.selectedProject = &Project{
		Name:     "user/roost",
		FullPath: "/home/user/roost",
		Sessions: []Session{
			{ID: "s1", Platform: PlatformCodeBuddy, Title: "T", LastActive: now},
		},
	}
	m.filteredSessions = m.selectedProject.Sessions

	nm, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	mm := nm.(*Model)
	if mm.platformFilter != 0 {
		t.Errorf("tab in session: platformFilter = %d, want 0", mm.platformFilter)
	}
}

func TestUpdate_Session_Slash(t *testing.T) {
	now := time.Now()
	m := makeModelWithProjects()
	m.screen = screenSession
	m.selectedProject = &Project{
		Name:     "user/roost",
		FullPath: "/home/user/roost",
		Sessions: []Session{
			{ID: "s1", Platform: PlatformCodeBuddy, Title: "T", LastActive: now},
		},
	}
	m.filteredSessions = m.selectedProject.Sessions

	nm, _ := m.Update(tea.KeyPressMsg{Code: '/'})
	mm := nm.(*Model)
	if !mm.searching {
		t.Error("after / in session: searching should be true")
	}
}

// =====================================================================
// Update: delete confirm
// =====================================================================

func TestUpdate_Delete_Enter(t *testing.T) {
	m := makeModelWithProjects()
	m.screen = screenDeleteConfirm
	m.delTarget = deleteTargetSession
	m.delSession = &Session{ID: "s1"}

	nm, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	mm := nm.(*Model)
	// screen stays at screenDeleteConfirm until deleteDoneMsg is processed
	if mm.screen != screenDeleteConfirm {
		t.Errorf("after enter: screen = %v, want screenDeleteConfirm", mm.screen)
	}
	// cmd should not be nil (triggers async delete)
	if cmd == nil {
		t.Error("after enter delete: should return non-nil cmd")
	}
	// process deleteDoneMsg to complete the flow
	msg := cmd()
	nm2, _ := mm.Update(msg)
	mm2 := nm2.(*Model)
	if mm2.screen != screenProject {
		t.Errorf("after deleteDoneMsg: screen = %v, want screenProject", mm2.screen)
	}
}

func TestUpdate_Delete_Esc(t *testing.T) {
	m := makeModelWithProjects()
	m.screen = screenDeleteConfirm
	m.delTarget = deleteTargetSession
	m.delSession = &Session{ID: "s1"}

	nm, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	mm := nm.(*Model)
	if mm.screen == screenDeleteConfirm {
		t.Error("after esc delete: should not be in deleteConfirm")
	}
}

func TestUpdate_Delete_N(t *testing.T) {
	m := makeModelWithProjects()
	m.screen = screenDeleteConfirm
	m.delTarget = deleteTargetSession
	m.delSession = &Session{ID: "s1"}

	nm, _ := m.Update(tea.KeyPressMsg{Code: 'n'})
	mm := nm.(*Model)
	if mm.screen == screenDeleteConfirm {
		t.Error("after n delete: should not be in deleteConfirm")
	}
}

// =====================================================================
// Update: search mode
// =====================================================================

func TestUpdate_Search_TypeBackspaceEnterEsc(t *testing.T) {
	m := makeModelWithProjects()
	m.searching = true

	// type "bug"
	for _, ch := range "bug" {
		nm, _ := m.Update(tea.KeyPressMsg{Code: ch})
		m = nm.(*Model)
	}
	if m.searchQuery != "bug" {
		t.Errorf("after typing: searchQuery = %q, want 'bug'", m.searchQuery)
	}

	// backspace → "bu"
	nm, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	mm := nm.(*Model)
	if mm.searchQuery != "bu" {
		t.Errorf("after backspace: searchQuery = %q, want 'bu'", mm.searchQuery)
	}

	// enter → confirm search
	nm2, _ := mm.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	mm2 := nm2.(*Model)
	if mm2.searching {
		t.Error("after enter: searching should be false")
	}

	// re-enter search, then esc → cancel
	mm2.searching = true
	mm2.searchQuery = "test"
	nm3, _ := mm2.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	mm3 := nm3.(*Model)
	if mm3.searching {
		t.Error("after esc: searching should be false")
	}
	if mm3.searchQuery != "" {
		t.Errorf("after esc: searchQuery = %q, want empty", mm3.searchQuery)
	}
}

// =====================================================================
// scanDoneMsg / deleteDoneMsg
// =====================================================================

func TestUpdate_ScanDoneMsg(t *testing.T) {
	m := newModel(nil)
	m.loading = true
	projects := makeTestProjects()

	nm, _ := m.Update(scanDoneMsg{projects: projects})
	mm := nm.(*Model)
	if mm.loading {
		t.Error("after scanDoneMsg: loading should be false")
	}
	if len(mm.projects) != len(projects) {
		t.Errorf("after scanDoneMsg: projects len = %d, want %d", len(mm.projects), len(projects))
	}
}

func TestUpdate_DeleteDoneMsg(t *testing.T) {
	m := newModel(nil)
	m.loading = true
	m.screen = screenDeleteConfirm

	nm, cmd := m.Update(deleteDoneMsg{})
	mm := nm.(*Model)
	// deleteDoneMsg sets loading=true (triggers re-scan)
	if !mm.loading {
		t.Error("after deleteDoneMsg: loading should be true (re-scan triggered)")
	}
	if mm.screen != screenProject {
		t.Errorf("after deleteDoneMsg: screen = %v, want screenProject", mm.screen)
	}
	// cmd should not be nil (scanCmd + spinnerTick)
	if cmd == nil {
		t.Error("after deleteDoneMsg: should return non-nil cmd")
	}
}

// =====================================================================
// applyFilter
// =====================================================================

func TestApplyFilter_Platform(t *testing.T) {
	m := makeModelWithProjects()
	m.platformFilter = 0 // CodeBuddy only
	m.applyFilter()

	// applyFilter keeps projects that have at least one session matching the platform
	// It does NOT remove non-matching sessions from the project
	// So we check that filteredProjects only contains projects with CodeBuddy sessions
	for _, p := range m.filteredProjects {
		hasCodeBuddy := false
		for _, s := range p.Sessions {
			if s.Platform == PlatformCodeBuddy {
				hasCodeBuddy = true
				break
			}
		}
		if !hasCodeBuddy {
			t.Errorf("applyFilter platformFilter=0: project %q has no CodeBuddy session", p.Name)
		}
	}
	// should have at least one project (user/roost has CodeBuddy session)
	if len(m.filteredProjects) < 1 {
		t.Error("applyFilter platformFilter=0: should have at least one project")
	}
}

// TestApplyFilter_PartialPlatforms verifies that applyFilter() uses installedPlatforms[index]
// not the raw Platform int value. This is a regression test for the bug where
// platformFilter (an index into installedPlatforms) was wrongly compared directly
// against Platform enum values.
func TestApplyFilter_PartialPlatforms(t *testing.T) {
	// Only CodeBuddy (0) and OpenCode (5) are "installed".
	// After sorting: installedPlatforms = [PlatformCodeBuddy, PlatformOpenCode]
	m := makeModelWithProjects()
	m.installedPlatforms = []Platform{PlatformCodeBuddy, PlatformOpenCode}

	// filter=0 → should show only CodeBuddy sessions (Platform=0)
	// OpenCode has Platform=5, which != installedPlatforms[0]=0
	m.platformFilter = 0
	m.applyFilter()

	for _, p := range m.filteredProjects {
		hasCodeBuddy := false
		for _, s := range p.Sessions {
			if s.Platform == PlatformCodeBuddy {
				hasCodeBuddy = true
				break
			}
		}
		if !hasCodeBuddy {
			t.Errorf("partial platforms filter=0: project %q should have CodeBuddy session", p.Name)
		}
	}
}

func TestApplyFilter_Search(t *testing.T) {
	m := makeModelWithProjects()
	m.searchQuery = "fix"
	m.applyFilter()

	found := false
	for _, p := range m.filteredProjects {
		if strings.Contains(strings.ToLower(p.FullPath), "fix") {
			found = true
			break
		}
		for _, s := range p.Sessions {
			if strings.Contains(strings.ToLower(s.Title), "fix") {
				found = true
				break
			}
		}
	}
	if !found && len(m.filteredProjects) > 0 {
		t.Error("applyFilter search='fix': should match something")
	}
}

// =====================================================================
// platformFilterLabel / renderScrollHint / renderFooter
// =====================================================================

func TestPlatformFilterLabel_AllAndEach(t *testing.T) {
	// Build scanners for all platforms
	allPlatforms := []Platform{PlatformCodeBuddy, PlatformClaude, PlatformGemini, PlatformCodex, PlatformCopilot, PlatformOpenCode}
	var scs []Scanner
	for _, p := range allPlatforms {
		scs = append(scs, &stubScanner{platform: p})
	}
	m := newModel(scs)
	if got := m.platformFilterLabel(); got != "All" {
		t.Errorf("platformFilterLabel() = %q, want 'All'", got)
	}
	for i := range m.installedPlatforms {
		m.platformFilter = i
		got := m.platformFilterLabel()
		if !strings.Contains(got, "●") {
			t.Errorf("platformFilterLabel() with filter=%d = %q, should contain bullet", i, got)
		}
	}
}

func TestRenderScrollHint(t *testing.T) {
	// no scroll needed, but still shows [cursor+1/total]
	s := renderScrollHint(0, 1, 1, 0)
	if !strings.Contains(s, "[1/1]") {
		t.Errorf("scrollHint no scroll = %q, want [1/1]", s)
	}
	// can scroll down
	s = renderScrollHint(0, 1, 5, 0)
	if !strings.Contains(s, "↓") {
		t.Errorf("scrollHint can go down = %q, should contain ↓", s)
	}
	// can scroll up
	s = renderScrollHint(2, 5, 5, 4)
	if !strings.Contains(s, "↑") {
		t.Errorf("scrollHint can go up = %q, should contain ↑", s)
	}
	// total = 0 → empty
	s = renderScrollHint(0, 0, 0, 0)
	if s != "" {
		t.Errorf("scrollHint total=0 = %q, want empty", s)
	}
}

func TestRenderFooter(t *testing.T) {
	s := renderFooter("hello %s", "world")
	if !strings.Contains(s, "hello") {
		t.Errorf("renderFooter output = %q, should contain 'hello'", s)
	}
}

// =====================================================================
// calcViewport / clamp
// =====================================================================

func TestCalcViewport_Integration(t *testing.T) {
	tests := []struct {
		name      string
		total     int
		cursor    int
		h         int
		wantStart int
		wantEnd   int
	}{
		{"empty", 0, 0, 5, 0, 0},
		{"all visible", 3, 0, 10, 0, 3},
		{"cursor at top", 10, 0, 5, 0, 5},
		{"cursor at bottom", 10, 9, 5, 5, 10},
		{"cursor in middle", 10, 4, 5, 2, 7},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start, end := calcViewport(tt.total, tt.cursor, tt.h)
			if start != tt.wantStart || end != tt.wantEnd {
				t.Errorf("calcViewport(%d,%d,%d) = (%d,%d), want (%d,%d)",
					tt.total, tt.cursor, tt.h, start, end, tt.wantStart, tt.wantEnd)
			}
		})
	}
}

func TestClamp(t *testing.T) {
	if got := clamp(5, 0, 10); got != 5 {
		t.Errorf("clamp(5,0,10) = %d, want 5", got)
	}
	if got := clamp(-1, 0, 10); got != 0 {
		t.Errorf("clamp(-1,0,10) = %d, want 0", got)
	}
	if got := clamp(20, 0, 10); got != 10 {
		t.Errorf("clamp(20,0,10) = %d, want 10", got)
	}
}

// =====================================================================
// agentExitMsg
// =====================================================================

func TestAgentExitMsg(t *testing.T) {
	m := newModel(nil)
	msg := agentExitMsg{err: nil}
	nm, _ := m.Update(msg)
	_ = nm.(*Model)
	// should not panic
}

// =====================================================================
// q / ctrl+c handling
// =====================================================================

func TestUpdate_Project_Q(t *testing.T) {
	m := makeModelWithProjects()
	nm, cmd := m.Update(tea.KeyPressMsg{Code: 'q'})
	_ = nm.(*Model)
	if cmd == nil {
		t.Error("q should return tea.Quit cmd")
	}
}

func TestUpdate_Session_Q(t *testing.T) {
	now := time.Now()
	m := makeModelWithProjects()
	m.screen = screenSession
	m.selectedProject = &Project{
		Name:     "p",
		FullPath: "/p",
		Sessions: []Session{
			{ID: "s1", Title: "T", LastActive: now},
		},
	}
	m.filteredSessions = m.selectedProject.Sessions

	nm, cmd := m.Update(tea.KeyPressMsg{Code: 'q'})
	_ = nm.(*Model)
	if cmd == nil {
		t.Error("q in session view should return tea.Quit cmd")
	}
}

func TestUpdate_CtrlC(t *testing.T) {
	m := makeModelWithProjects()
	nm, cmd := m.Update(tea.KeyPressMsg{Mod: tea.ModCtrl, Code: 'c'})
	_ = nm.(*Model)
	if cmd == nil {
		t.Error("ctrl+c should return tea.Quit cmd")
	}
}
