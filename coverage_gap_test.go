package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

// =====================================================================
// calcViewHeight
// =====================================================================

func TestCalcViewHeight(t *testing.T) {
	tests := []struct {
		height int
		want   int
	}{
		{24, 20},  // normal terminal
		{10, 6},   // small terminal
		{4, 3},    // minimum clamp (4-4=0 → 3)
		{3, 3},    // below minimum → 3
		{0, 3},    // zero → 3
		{100, 96}, // large terminal
	}
	for _, tt := range tests {
		got := calcViewHeight(tt.height)
		if got != tt.want {
			t.Errorf("calcViewHeight(%d) = %d, want %d", tt.height, got, tt.want)
		}
	}
}

// =====================================================================
// config.go — platformCfg / BinFor / DataDirFor default branch
// =====================================================================

func TestConfig_PlatformCfg_Default(t *testing.T) {
	cfg := Config{}
	// unknown platform → empty PlatformConfig
	got := cfg.platformCfg(Platform(99))
	if got.Bin != "" || got.DataDir != "" || got.Args != nil {
		t.Errorf("platformCfg(unknown) should return zero PlatformConfig, got %+v", got)
	}
}

func TestConfig_BinFor_Unknown(t *testing.T) {
	cfg := Config{}
	got := cfg.BinFor(Platform(99))
	if got != "" {
		t.Errorf("BinFor(unknown) = %q, want empty", got)
	}
}

func TestConfig_DataDirFor_Unknown(t *testing.T) {
	cfg := Config{}
	got := cfg.DataDirFor(Platform(99))
	if got != "" {
		t.Errorf("DataDirFor(unknown) = %q, want empty", got)
	}
}

// =====================================================================
// styles.go — platformStyle default branch
// =====================================================================

func TestPlatformStyle_Default(t *testing.T) {
	// Platform(99) hits default branch in platformStyle
	unknown := Platform(99)
	// platformDot / platformIcon use platformStyle; just check no panic and non-empty output
	dot := platformDot(unknown)
	if dot == "" {
		t.Error("platformDot(unknown) should return non-empty string")
	}
	icon := platformIcon(unknown)
	if icon == "" {
		t.Error("platformIcon(unknown) should return non-empty string")
	}
}

// =====================================================================
// scanner_opencode.go — Platform() and DeleteSession bad dbPath
// =====================================================================

func TestOpenCodeScanner_Platform(t *testing.T) {
	sc := &OpenCodeScanner{bin: "opencode", dbPath: ""}
	if sc.Platform() != PlatformOpenCode {
		t.Errorf("Platform() = %v, want PlatformOpenCode", sc.Platform())
	}
}

func TestOpenCodeScanner_DeleteSession_BadPath(t *testing.T) {
	sc := &OpenCodeScanner{bin: "opencode", dbPath: "/nonexistent/bad.db"}
	err := sc.DeleteSession(Session{ID: "s1"})
	if err == nil {
		t.Error("DeleteSession with bad dbPath should return error")
	}
}

// =====================================================================
// delete.go — renderDeleteView all branches
// =====================================================================

func TestRenderDeleteView_Session(t *testing.T) {
	m := makeModelWithProjects()
	m.width = 80
	m.height = 24
	m.screen = screenDeleteConfirm
	m.delTarget = deleteTargetSession
	m.delSession = &Session{ID: "s1", Title: "Fix bug"}

	s := renderDeleteView(m)
	if !strings.Contains(s, "Delete this session?") {
		t.Errorf("renderDeleteView session: missing action text, got %q", s)
	}
	if !strings.Contains(s, "Fix bug") {
		t.Errorf("renderDeleteView session: missing title, got %q", s)
	}
}

func TestRenderDeleteView_Project(t *testing.T) {
	m := makeModelWithProjects()
	m.width = 80
	m.height = 24
	m.screen = screenDeleteConfirm
	m.delTarget = deleteTargetProject
	m.delProject = &Project{
		Name:     "user/roost",
		FullPath: "/home/user/roost",
		Sessions: []Session{{ID: "s1"}, {ID: "s2"}},
	}

	s := renderDeleteView(m)
	if !strings.Contains(s, "Delete entire project?") {
		t.Errorf("renderDeleteView project: missing action text, got %q", s)
	}
	if !strings.Contains(s, "2 sessions") {
		t.Errorf("renderDeleteView project: missing session count, got %q", s)
	}
}

func TestRenderDeleteView_Batch_Sessions(t *testing.T) {
	m := makeModelWithProjects()
	m.width = 80
	m.height = 24
	m.screen = screenDeleteConfirm
	m.delTarget = deleteTargetBatch
	m.selectedSet = map[string]bool{"s1": true, "s2": true}
	m.selectedProject = &Project{Name: "p", FullPath: "/p"} // in session view

	s := renderDeleteView(m)
	if !strings.Contains(s, "Delete 2 sessions?") {
		t.Errorf("renderDeleteView batch sessions: missing text, got %q", s)
	}
}

func TestRenderDeleteView_Batch_Projects(t *testing.T) {
	m := makeModelWithProjects()
	m.width = 80
	m.height = 24
	m.screen = screenDeleteConfirm
	m.delTarget = deleteTargetBatch
	m.selectedSet = map[string]bool{"/a": true, "/b": true, "/c": true}
	m.selectedProject = nil // in project view

	s := renderDeleteView(m)
	if !strings.Contains(s, "Delete 3 projects?") {
		t.Errorf("renderDeleteView batch projects: missing text, got %q", s)
	}
}

// =====================================================================
// model.go — handleSessionKey: navigation, select, enter replace mode
// =====================================================================

func makeSessionModel() *Model {
	now := time.Now()
	m := makeModelWithProjects()
	m.screen = screenSession
	m.selectedProject = &Project{
		Name:     "user/roost",
		FullPath: "/home/user/roost",
		Sessions: []Session{
			{ID: "s1", Platform: PlatformCodeBuddy, Title: "A", LastActive: now},
			{ID: "s2", Platform: PlatformClaude, Title: "B", LastActive: now.Add(-time.Hour)},
			{ID: "s3", Platform: PlatformGemini, Title: "C", LastActive: now.Add(-2 * time.Hour)},
		},
	}
	m.filteredSessions = m.selectedProject.Sessions
	m.cursor = 0
	m.cfg = Config{ResumeMode: ResumeModeReplace}
	return m
}

func TestUpdate_Session_NavUpDown(t *testing.T) {
	m := makeSessionModel()
	m.cursor = 1

	nm, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	mm := nm.(*Model)
	if mm.cursor != 0 {
		t.Errorf("up: cursor = %d, want 0", mm.cursor)
	}

	nm2, _ := mm.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	mm2 := nm2.(*Model)
	if mm2.cursor != 1 {
		t.Errorf("down: cursor = %d, want 1", mm2.cursor)
	}
}

func TestUpdate_Session_NavVim(t *testing.T) {
	m := makeSessionModel()
	m.cursor = 1

	nm, _ := m.Update(tea.KeyPressMsg{Code: 'k'})
	mm := nm.(*Model)
	if mm.cursor != 0 {
		t.Errorf("k: cursor = %d, want 0", mm.cursor)
	}

	nm2, _ := mm.Update(tea.KeyPressMsg{Code: 'j'})
	mm2 := nm2.(*Model)
	if mm2.cursor != 1 {
		t.Errorf("j: cursor = %d, want 1", mm2.cursor)
	}
}

func TestUpdate_Session_JumpGG(t *testing.T) {
	m := makeSessionModel()
	m.cursor = 1

	nm, _ := m.Update(tea.KeyPressMsg{Code: 'G'})
	mm := nm.(*Model)
	if mm.cursor != 2 {
		t.Errorf("G: cursor = %d, want 2 (last)", mm.cursor)
	}

	nm2, _ := mm.Update(tea.KeyPressMsg{Code: 'g'})
	mm2 := nm2.(*Model)
	if mm2.cursor != 0 {
		t.Errorf("g: cursor = %d, want 0", mm2.cursor)
	}
}

func TestUpdate_Session_SelectSpaceD(t *testing.T) {
	m := makeSessionModel()

	// space → enter selecting, toggle s1
	nm, _ := m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	mm := nm.(*Model)
	if !mm.selecting {
		t.Error("space: selecting should be true")
	}
	if !mm.selectedSet["s1"] {
		t.Error("space: s1 should be selected")
	}

	// space again → deselect s1
	nm2, _ := mm.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	mm2 := nm2.(*Model)
	if mm2.selectedSet["s1"] {
		t.Error("second space: s1 should be deselected")
	}

	// re-select, then D → batch delete confirm
	mm2.selectedSet["s1"] = true
	nm3, _ := mm2.Update(tea.KeyPressMsg{Code: 'D'})
	mm3 := nm3.(*Model)
	if mm3.screen != screenDeleteConfirm {
		t.Errorf("D: screen = %v, want screenDeleteConfirm", mm3.screen)
	}
	if mm3.delTarget != deleteTargetBatch {
		t.Errorf("D: delTarget = %v, want deleteTargetBatch", mm3.delTarget)
	}
}

func TestUpdate_Session_EscWhileSelecting(t *testing.T) {
	m := makeSessionModel()
	m.selecting = true
	m.selectedSet = map[string]bool{"s1": true}

	nm, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	mm := nm.(*Model)
	if mm.selecting {
		t.Error("esc while selecting: selecting should be false")
	}
	if mm.screen != screenSession {
		t.Errorf("esc while selecting: should stay in session view, got %v", mm.screen)
	}
}

func TestUpdate_Session_Enter_Replace(t *testing.T) {
	m := makeSessionModel()
	m.cfg = Config{ResumeMode: ResumeModeReplace}

	nm, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	mm := nm.(*Model)
	if mm.resumeSession == nil {
		t.Error("enter (replace): resumeSession should be set")
	}
	if mm.resumeSession.ID != "s1" {
		t.Errorf("enter (replace): resumeSession.ID = %q, want s1", mm.resumeSession.ID)
	}
	if cmd == nil {
		t.Error("enter (replace): should return tea.Quit cmd")
	}
}

func TestUpdate_Session_Refresh(t *testing.T) {
	m := makeSessionModel()

	nm, cmd := m.Update(tea.KeyPressMsg{Code: 'r'})
	mm := nm.(*Model)
	if !mm.loading {
		t.Error("r: loading should be true")
	}
	if cmd == nil {
		t.Error("r: should return non-nil cmd")
	}
}

func TestUpdate_Session_Question(t *testing.T) {
	m := makeSessionModel()

	nm, _ := m.Update(tea.KeyPressMsg{Code: '?'})
	mm := nm.(*Model)
	if !mm.showHelp {
		t.Error("?: showHelp should be true")
	}
}

// =====================================================================
// model.go — handleDeleteKey: batch→session branch, batch→project branch
// =====================================================================

func TestUpdate_DeleteKey_BatchInSession(t *testing.T) {
	m := makeModelWithProjects()
	m.screen = screenDeleteConfirm
	m.delTarget = deleteTargetBatch
	m.selectedProject = &Project{Name: "p", FullPath: "/p"} // in session context

	nm, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	mm := nm.(*Model)
	if mm.screen != screenSession {
		t.Errorf("esc batch-in-session: screen = %v, want screenSession", mm.screen)
	}
}

func TestUpdate_DeleteKey_BatchInProject(t *testing.T) {
	m := makeModelWithProjects()
	m.screen = screenDeleteConfirm
	m.delTarget = deleteTargetBatch
	m.selectedProject = nil // in project context

	nm, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	mm := nm.(*Model)
	if mm.screen != screenProject {
		t.Errorf("esc batch-in-project: screen = %v, want screenProject", mm.screen)
	}
}

func TestUpdate_DeleteKey_Q(t *testing.T) {
	m := makeModelWithProjects()
	m.screen = screenDeleteConfirm

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'q'})
	if cmd == nil {
		t.Error("q in delete confirm: should return tea.Quit cmd")
	}
}

// =====================================================================
// model.go — doDelete: deleteTargetProject and deleteTargetBatch (sessions)
// =====================================================================

func TestDoDelete_Project(t *testing.T) {
	m := makeModelWithProjects()
	m.screen = screenDeleteConfirm
	m.delTarget = deleteTargetProject
	m.delProject = new(m.filteredProjects[0])

	nm, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	mm := nm.(*Model)
	_ = mm
	if cmd == nil {
		t.Error("enter delete project: should return non-nil cmd")
	}
	// fire the async cmd → should produce deleteDoneMsg (no real scanner, err=nil)
	msg := cmd()
	if _, ok := msg.(deleteDoneMsg); !ok {
		t.Errorf("delete project cmd produced %T, want deleteDoneMsg", msg)
	}
}

func TestDoDelete_BatchSessions(t *testing.T) {
	now := time.Now()
	m := makeModelWithProjects()
	m.screen = screenDeleteConfirm
	m.delTarget = deleteTargetBatch
	m.selectedProject = &Project{Name: "p", FullPath: "/p"} // session-level batch
	m.filteredSessions = []Session{
		{ID: "s1", Platform: PlatformCodeBuddy, LastActive: now},
		{ID: "s2", Platform: PlatformClaude, LastActive: now},
	}
	m.selectedSet = map[string]bool{"s1": true}

	nm, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	_ = nm.(*Model)
	if cmd == nil {
		t.Error("enter batch delete sessions: should return non-nil cmd")
	}
	msg := cmd()
	if _, ok := msg.(deleteDoneMsg); !ok {
		t.Errorf("batch delete cmd produced %T, want deleteDoneMsg", msg)
	}
}

func TestDoDelete_BatchProjects(t *testing.T) {
	m := makeModelWithProjects()
	m.screen = screenDeleteConfirm
	m.delTarget = deleteTargetBatch
	m.selectedProject = nil // project-level batch
	m.selectedSet = map[string]bool{"/home/user/roost": true}

	nm, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	_ = nm.(*Model)
	if cmd == nil {
		t.Error("enter batch delete projects: should return non-nil cmd")
	}
	msg := cmd()
	if _, ok := msg.(deleteDoneMsg); !ok {
		t.Errorf("batch delete projects cmd produced %T, want deleteDoneMsg", msg)
	}
}

// =====================================================================
// main.go — findSession, isCommandAvailable
// =====================================================================

func TestFindSession(t *testing.T) {
	now := time.Now()
	projects := []Project{
		{FullPath: "/a", Sessions: []Session{{ID: "s1", LastActive: now}}},
		{FullPath: "/b", Sessions: []Session{{ID: "s2", LastActive: now}}},
	}

	s := findSession(projects, "s2")
	if s == nil {
		t.Fatal("findSession(s2): should find session")
	}
	if s.ID != "s2" {
		t.Errorf("findSession(s2).ID = %q, want s2", s.ID)
	}

	missing := findSession(projects, "nonexistent")
	if missing != nil {
		t.Error("findSession(nonexistent): should return nil")
	}
}

func TestIsCommandAvailable(t *testing.T) {
	// "sh" is available on all Unix systems
	if !isCommandAvailable("sh") {
		t.Error("isCommandAvailable(sh): should be true")
	}
	if isCommandAvailable("__roost_no_such_binary_xyz__") {
		t.Error("isCommandAvailable(nonexistent): should be false")
	}
}

// =====================================================================
// main.go — cmdList (plain text output, no panic)
// =====================================================================

func TestCmdList_PlainText(t *testing.T) {
	now := time.Now()
	projects := []Project{
		{
			FullPath: "/home/user/roost",
			Sessions: []Session{
				{
					ID: "s1", Platform: PlatformCodeBuddy, Title: "Fix bug", Model: "sonnet",
					LastActive: now.Add(-time.Hour), MsgCount: 3,
				},
			},
		},
	}
	// should not panic; output goes to stdout but we just verify no panic
	cmdList(projects, false)
}

func TestCmdList_JSON(t *testing.T) {
	now := time.Now()
	projects := []Project{
		{
			FullPath: "/p",
			Sessions: []Session{
				{
					ID: "s1", Platform: PlatformClaude, Title: "T", Model: "opus",
					AgentType: "agent", LastActive: now, MsgCount: 1,
				},
			},
		},
	}
	// should not panic
	cmdList(projects, true)
}

// =====================================================================
// Update — uncovered message types
// =====================================================================

func TestUpdate_WindowSizeMsg(t *testing.T) {
	m := makeModelWithProjects()
	nm, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	mm := nm.(*Model)
	if mm.width != 120 || mm.height != 40 {
		t.Errorf("WindowSizeMsg: got %dx%d, want 120x40", mm.width, mm.height)
	}
}

func TestUpdate_SpinnerTickMsg_NotLoading(t *testing.T) {
	m := makeModelWithProjects()
	m.loading = false
	nm, cmd := m.Update(spinnerTickMsg{})
	mm := nm.(*Model)
	if cmd != nil {
		t.Error("spinnerTickMsg when not loading: should return nil cmd")
	}
	_ = mm
}

func TestUpdate_AgentExitMsg_WithError(t *testing.T) {
	m := makeModelWithProjects()
	testErr := fmt.Errorf("agent crashed")
	nm, _ := m.Update(agentExitMsg{err: testErr})
	mm := nm.(*Model)
	if mm.err == nil || mm.err.Error() != "agent crashed" {
		t.Errorf("agentExitMsg with error: err = %v, want 'agent crashed'", mm.err)
	}
	if !mm.loading {
		t.Error("agentExitMsg: should set loading=true")
	}
}

func TestUpdate_EscHintTimeoutMsg_GenMismatch(t *testing.T) {
	m := makeModelWithProjects()
	m.escHint = true
	m.escHintGen = 5
	nm, _ := m.Update(escHintTimeoutMsg{gen: 3}) // mismatched gen
	mm := nm.(*Model)
	if !mm.escHint {
		t.Error("escHintTimeoutMsg with mismatched gen: should NOT clear escHint")
	}
}

func TestUpdate_EscHintTimeoutMsg_GenMatch(t *testing.T) {
	m := makeModelWithProjects()
	m.escHint = true
	m.escHintGen = 5
	nm, _ := m.Update(escHintTimeoutMsg{gen: 5}) // matched gen
	mm := nm.(*Model)
	if mm.escHint {
		t.Error("escHintTimeoutMsg with matched gen: should clear escHint")
	}
}

func TestUpdate_DefaultMsg(t *testing.T) {
	m := makeModelWithProjects()
	// Send an unhandled message type → should hit default return
	type dummyMsg struct{}
	nm, cmd := m.Update(dummyMsg{})
	mm := nm.(*Model)
	if cmd != nil {
		t.Error("unhandled msg type: should return nil cmd")
	}
	_ = mm
}

// =====================================================================
// View — error branch
// =====================================================================

func TestView_Error(t *testing.T) {
	m := makeModelWithProjects()
	m.loading = false
	m.err = fmt.Errorf("something broke")
	v := m.View()
	if !strings.Contains(v.Content, "error:") {
		t.Error("View with err: should contain 'error:'")
	}
}

// =====================================================================
// doDelete — with real scanners
// =====================================================================

func TestDoDelete_Session_WithScanner(t *testing.T) {
	sc := &stubScanner{platform: PlatformCodeBuddy, deleteErr: nil}
	m := makeModelWithProjects()
	m.scanners = []Scanner{sc}
	m.screen = screenDeleteConfirm
	m.delTarget = deleteTargetSession
	m.delSession = &Session{ID: "s1", Platform: PlatformCodeBuddy, Title: "Fix bug"}

	cmd := m.doDelete()
	msg := cmd()
	done, ok := msg.(deleteDoneMsg)
	if !ok {
		t.Fatal("doDelete should return deleteDoneMsg")
	}
	if done.err != nil {
		t.Errorf("doDelete session: unexpected error %v", done.err)
	}
}

func TestDoDelete_Session_ScannerError(t *testing.T) {
	sc := &stubScanner{platform: PlatformCodeBuddy, deleteErr: fmt.Errorf("delete failed")}
	m := makeModelWithProjects()
	m.scanners = []Scanner{sc}
	m.screen = screenDeleteConfirm
	m.delTarget = deleteTargetSession
	m.delSession = &Session{ID: "s1", Platform: PlatformCodeBuddy, Title: "Fix bug"}

	cmd := m.doDelete()
	msg := cmd()
	done := msg.(deleteDoneMsg)
	if done.err == nil {
		t.Error("doDelete session with scanner error: should propagate error")
	}
}

func TestDoDelete_Project_WithScanner(t *testing.T) {
	sc := &stubScanner{platform: PlatformCodeBuddy, deleteErr: nil}
	m := makeModelWithProjects()
	m.scanners = []Scanner{sc}
	m.screen = screenDeleteConfirm
	m.delTarget = deleteTargetProject
	m.delProject = &Project{
		Name: "user/roost", FullPath: "/home/user/roost",
		Sessions: []Session{{ID: "s1", Platform: PlatformCodeBuddy}},
	}

	cmd := m.doDelete()
	msg := cmd()
	done := msg.(deleteDoneMsg)
	if done.err != nil {
		t.Errorf("doDelete project: unexpected error %v", done.err)
	}
}

func TestDoDelete_BatchSessions_WithScanner(t *testing.T) {
	sc := &stubScanner{platform: PlatformCodeBuddy, deleteErr: nil}
	m := makeModelWithProjects()
	m.scanners = []Scanner{sc}
	m.screen = screenDeleteConfirm
	m.selectedProject = &Project{Name: "p", FullPath: "/p"}
	m.delTarget = deleteTargetBatch
	m.selectedSet = map[string]bool{"s1": true}
	m.filteredSessions = []Session{{ID: "s1", Platform: PlatformCodeBuddy, Title: "T"}}

	cmd := m.doDelete()
	msg := cmd()
	done := msg.(deleteDoneMsg)
	if done.err != nil {
		t.Errorf("doDelete batch sessions: unexpected error %v", done.err)
	}
}

func TestDoDelete_BatchProjects_WithScanner(t *testing.T) {
	sc := &stubScanner{platform: PlatformCodeBuddy, deleteErr: nil}
	m := makeModelWithProjects()
	m.scanners = []Scanner{sc}
	m.screen = screenDeleteConfirm
	m.delTarget = deleteTargetBatch
	m.selectedSet = map[string]bool{"/home/user/roost": true}
	// No selectedProject → project view batch delete

	cmd := m.doDelete()
	msg := cmd()
	done := msg.(deleteDoneMsg)
	if done.err != nil {
		t.Errorf("doDelete batch projects: unexpected error %v", done.err)
	}
}

// =====================================================================
// renderSessionView — uncovered branches
// =====================================================================

func TestRenderSessionView_EmptyWithSearch(t *testing.T) {
	m := makeModelWithProjects()
	m.screen = screenSession
	m.selectedProject = &Project{Name: "p", FullPath: "/p"}
	m.filteredSessions = nil
	m.searchQuery = "nonexistent"
	s := renderSessionView(m)
	if !strings.Contains(s, "no results for:") {
		t.Error("renderSessionView empty+search: should contain 'no results for:'")
	}
}

func TestRenderSessionView_EmptyNoSearch(t *testing.T) {
	m := makeModelWithProjects()
	m.screen = screenSession
	m.selectedProject = &Project{Name: "p", FullPath: "/p"}
	m.filteredSessions = nil
	s := renderSessionView(m)
	if !strings.Contains(s, "no sessions") {
		t.Error("renderSessionView empty: should contain 'no sessions'")
	}
}

func TestRenderSessionView_Selecting(t *testing.T) {
	m := makeSessionModel()
	m.selecting = true
	m.selectedSet = map[string]bool{"s1": true}
	s := renderSessionView(m)
	if !strings.Contains(s, "●") {
		t.Error("renderSessionView selecting: should contain bullet")
	}
	if !strings.Contains(s, "○") {
		t.Error("renderSessionView selecting: should contain empty circle for unselected")
	}
}

func TestRenderSessionView_EscHint(t *testing.T) {
	m := makeSessionModel()
	m.escHint = true
	s := renderSessionView(m)
	if !strings.Contains(s, "press Esc again to quit") {
		t.Error("renderSessionView escHint: should contain esc hint")
	}
}

func TestRenderSessionView_Searching(t *testing.T) {
	m := makeSessionModel()
	m.searching = true
	m.searchQuery = "fix"
	s := renderSessionView(m)
	if !strings.Contains(s, "/ fix_") {
		t.Error("renderSessionView searching: should contain search prompt")
	}
}

func TestRenderSessionView_AgentType(t *testing.T) {
	now := time.Now()
	m := &Model{
		installedPlatforms: []Platform{PlatformClaude},
		selectedProject:    &Project{Name: "p", FullPath: "/p"},
		filteredSessions: []Session{
			{
				ID: "s1", Platform: PlatformClaude, Title: "T", Model: "opus",
				AgentType: "code-review", LastActive: now, MsgCount: 1,
			},
		},
		width: 80, height: 24,
	}
	s := renderSessionView(m)
	if !strings.Contains(s, "[code-review]") {
		t.Error("renderSessionView with AgentType: should show [code-review]")
	}
}

// =====================================================================
// renderProjectView — uncovered branches
// =====================================================================

func TestRenderProjectView_EmptyWithSearch(t *testing.T) {
	m := makeModelWithProjects()
	m.filteredProjects = nil
	m.searchQuery = "nonexistent"
	s := renderProjectView(m)
	if !strings.Contains(s, "no results for:") {
		t.Error("renderProjectView empty+search: should contain 'no results for:'")
	}
}

func TestRenderProjectView_EmptyNoSearch(t *testing.T) {
	m := makeModelWithProjects()
	m.filteredProjects = nil
	s := renderProjectView(m)
	if !strings.Contains(s, "no sessions found") {
		t.Error("renderProjectView empty: should contain 'no sessions found'")
	}
}

func TestRenderProjectView_Selecting(t *testing.T) {
	m := makeModelWithProjects()
	m.selecting = true
	m.selectedSet = map[string]bool{m.filteredProjects[0].FullPath: true}
	s := renderProjectView(m)
	if !strings.Contains(s, "●") {
		t.Error("renderProjectView selecting: should contain bullet")
	}
	if !strings.Contains(s, "○") {
		t.Error("renderProjectView selecting: should contain empty circle")
	}
}

func TestRenderProjectView_EscHint(t *testing.T) {
	m := makeModelWithProjects()
	m.escHint = true
	s := renderProjectView(m)
	if !strings.Contains(s, "press Esc again to quit") {
		t.Error("renderProjectView escHint: should contain esc hint")
	}
}

func TestRenderProjectView_Searching(t *testing.T) {
	m := makeModelWithProjects()
	m.searching = true
	m.searchQuery = "fix"
	s := renderProjectView(m)
	if !strings.Contains(s, "/ fix_") {
		t.Error("renderProjectView searching: should contain search prompt")
	}
}

// =====================================================================
// handleProjectKey / handleSessionKey — edge cases
// =====================================================================

func TestHandleProjectKey_UpAtZero(t *testing.T) {
	m := makeModelWithProjects()
	m.cursor = 0
	nm, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	mm := nm.(*Model)
	if mm.cursor != 0 {
		t.Errorf("up at 0: cursor = %d, want 0", mm.cursor)
	}
}

func TestHandleProjectKey_DWithoutSelecting(t *testing.T) {
	m := makeModelWithProjects()
	m.cursor = 0
	nm, _ := m.Update(tea.KeyPressMsg{Code: 'D'})
	mm := nm.(*Model)
	if mm.screen != screenProject {
		t.Error("D without selecting: should stay on project screen")
	}
}

func TestHandleProjectKey_SpaceTogglesSelect(t *testing.T) {
	m := makeModelWithProjects()
	// First space enters select mode and selects current item
	nm, _ := m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	mm := nm.(*Model)
	if !mm.selecting {
		t.Error("first space: should enter selecting mode")
	}
	key := m.filteredProjects[0].FullPath
	if !mm.selectedSet[key] {
		t.Error("first space: should select current item")
	}
	// Second space toggles (deselects) the same item
	nm2, _ := mm.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	mm2 := nm2.(*Model)
	if mm2.selectedSet[key] {
		t.Error("second space: should deselect current item")
	}
}

func TestHandleSessionKey_UpAtZero(t *testing.T) {
	m := makeSessionModel()
	m.cursor = 0
	nm, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	mm := nm.(*Model)
	if mm.cursor != 0 {
		t.Errorf("up at 0: cursor = %d, want 0", mm.cursor)
	}
}

func TestHandleSessionKey_EnterReplace(t *testing.T) {
	m := makeSessionModel()
	nm, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	mm := nm.(*Model)
	if mm.resumeSession == nil {
		t.Error("enter on session: should set resumeSession")
	}
	if cmd == nil {
		t.Error("enter on session: should return tea.Quit cmd")
	}
}

// =====================================================================
// suspendResume
// =====================================================================

func TestSuspendResume_WithScanner(t *testing.T) {
	sc := &stubScanner{
		platform:  PlatformClaude,
		resumeCmd: []string{"/bin/echo", "test"},
	}
	m := makeModelWithProjects()
	m.scanners = []Scanner{sc}
	m.cfg = Config{ResumeMode: ResumeModeSuspend}

	sess := &Session{ID: "s1", Platform: PlatformClaude, ProjectPath: "/tmp"}
	cmd := m.suspendResume(sess)
	if cmd == nil {
		t.Error("suspendResume: should return non-nil cmd")
	}
}

func TestSuspendResume_NoScanner(t *testing.T) {
	m := makeModelWithProjects()
	m.scanners = nil
	m.cfg = Config{ResumeMode: ResumeModeSuspend}

	sess := &Session{ID: "s1", Platform: PlatformClaude, ProjectPath: "/tmp"}
	cmd := m.suspendResume(sess)
	if cmd == nil {
		t.Error("suspendResume with no scanner: should return error cmd")
	}
	msg := cmd()
	exitMsg, ok := msg.(agentExitMsg)
	if !ok {
		t.Fatal("expected agentExitMsg")
	}
	if exitMsg.err == nil {
		t.Error("suspendResume with no scanner: should return error")
	}
}

// =====================================================================
// spinnerTickMsg + scanCmd — loading=true branches
// =====================================================================

func TestUpdate_SpinnerTickMsg_Loading(t *testing.T) {
	m := makeModelWithProjects()
	m.loading = true
	nm, cmd := m.Update(spinnerTickMsg{})
	mm := nm.(*Model)
	if !mm.loading {
		t.Error("should still be loading")
	}
	if cmd == nil {
		t.Error("spinnerTickMsg while loading: should return spinnerTick cmd")
	}
}

func TestScanCmd_Executes(t *testing.T) {
	sc := &stubScanner{platform: PlatformClaude, projects: []Project{{FullPath: "/test"}}}
	m := newModel([]Scanner{sc})
	cmd := m.scanCmd()
	msg := cmd()
	done, ok := msg.(scanDoneMsg)
	if !ok {
		t.Fatal("scanCmd should return scanDoneMsg")
	}
	if len(done.projects) != 1 || done.projects[0].FullPath != "/test" {
		t.Errorf("scanCmd: got %v, want project with FullPath /test", done.projects)
	}
}

// =====================================================================
// View — loading and help branches
// =====================================================================

func TestView_Loading(t *testing.T) {
	m := makeModelWithProjects()
	m.loading = true
	v := m.View()
	if !strings.Contains(v.Content, "scanning") {
		t.Error("View loading: should contain 'scanning'")
	}
}

func TestView_Help(t *testing.T) {
	m := makeModelWithProjects()
	m.loading = false
	m.showHelp = true
	v := m.View()
	if !strings.Contains(v.Content, "Keyboard Shortcuts") {
		t.Error("View help: should contain 'Keyboard Shortcuts'")
	}
}

// =====================================================================
// handleKey — help page quit, esc clears hint
// =====================================================================

func TestHandleKey_HelpPage_Quit(t *testing.T) {
	m := makeModelWithProjects()
	m.showHelp = true
	nm, cmd := m.Update(tea.KeyPressMsg{Code: 'q'})
	_ = nm.(*Model)
	if cmd == nil {
		t.Error("q in help: should return tea.Quit")
	}
}

func TestHandleKey_HelpPage_AnyKey(t *testing.T) {
	m := makeModelWithProjects()
	m.showHelp = true
	nm, _ := m.Update(tea.KeyPressMsg{Code: 'a'})
	mm := nm.(*Model)
	if mm.showHelp {
		t.Error("any key in help: should close help")
	}
}

func TestHandleKey_EscClearsHint(t *testing.T) {
	m := makeModelWithProjects()
	m.escHint = true
	nm, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	_ = nm.(*Model)
	// Esc doesn't clear escHint immediately (it sets it again for double-press logic)
	// But a non-Esc key should clear it
	m2 := makeModelWithProjects()
	m2.escHint = true
	nm2, _ := m2.Update(tea.KeyPressMsg{Code: 'j'})
	mm2 := nm2.(*Model)
	if mm2.escHint {
		t.Error("non-esc key should clear escHint")
	}
}

// =====================================================================
// renderDeleteView — escape path (n key)
// =====================================================================

func TestHandleDeleteKey_NKey(t *testing.T) {
	m := makeModelWithProjects()
	m.screen = screenDeleteConfirm
	m.delTarget = deleteTargetSession
	m.delSession = &Session{ID: "s1"}
	nm, _ := m.Update(tea.KeyPressMsg{Code: 'n'})
	mm := nm.(*Model)
	if mm.screen == screenDeleteConfirm {
		t.Error("n key in delete: should leave confirm screen")
	}
}

// =====================================================================
// handleDeleteKey — q / ctrl+c
// =====================================================================

func TestHandleDeleteKey_Q(t *testing.T) {
	m := makeModelWithProjects()
	m.screen = screenDeleteConfirm
	m.delTarget = deleteTargetSession
	nm, cmd := m.Update(tea.KeyPressMsg{Code: 'q'})
	_ = nm.(*Model)
	if cmd == nil {
		t.Error("q in delete confirm: should return tea.Quit")
	}
}

// =====================================================================
// main.go — cmdResume / cmdDelete / execResume / execNewSession
// Sub-process testing pattern for os.Exit functions
// =====================================================================

func TestCmdResume_NotFound(t *testing.T) {
	if os.Getenv("ROOST_TEST_CMDRESUME") == "1" {
		cmdResume(nil, nil, Config{}, "nonexistent")
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestCmdResume_NotFound") //nolint:gosec // test binary path is safe // test binary path is safe
	cmd.Env = append(os.Environ(), "ROOST_TEST_CMDRESUME=1")
	err := cmd.Run()
	var e *exec.ExitError
	if errors.As(err, &e) && !e.Success() {
		return // expected: non-zero exit
	}
	t.Fatalf("cmdResume with nonexistent session: expected non-zero exit, got %v", err)
}

func TestCmdDelete_NotFound(t *testing.T) {
	if os.Getenv("ROOST_TEST_CMDDELETE") == "1" {
		cmdDelete(nil, nil, "nonexistent")
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestCmdDelete_NotFound") //nolint:gosec // test binary path is safe // test binary path is safe
	cmd.Env = append(os.Environ(), "ROOST_TEST_CMDDELETE=1")
	err := cmd.Run()
	var e *exec.ExitError
	if errors.As(err, &e) && !e.Success() {
		return // expected: non-zero exit
	}
	t.Fatalf("cmdDelete with nonexistent session: expected non-zero exit, got %v", err)
}

func TestCmdDelete_WithScanner(t *testing.T) {
	now := time.Now()
	projects := []Project{
		{FullPath: "/p", Sessions: []Session{{ID: "s1", Platform: PlatformCodeBuddy, Title: "T", LastActive: now}}},
	}
	sc := &stubScanner{platform: PlatformCodeBuddy}
	// This should not exit, just print to stderr
	cmdDelete(projects, []Scanner{sc}, "s1")
}

func TestCmdDelete_NoMatchingScanner(t *testing.T) {
	if os.Getenv("ROOST_TEST_CMDDELETE_NOSCANNER") == "1" {
		now := time.Now()
		projects := []Project{
			{FullPath: "/p", Sessions: []Session{{ID: "s1", Platform: PlatformClaude, Title: "T", LastActive: now}}},
		}
		cmdDelete(projects, []Scanner{&stubScanner{platform: PlatformCodeBuddy}}, "s1") // no matching scanner for Claude
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestCmdDelete_NoMatchingScanner") //nolint:gosec // test binary path is safe // test binary path is safe
	cmd.Env = append(os.Environ(), "ROOST_TEST_CMDDELETE_NOSCANNER=1")
	err := cmd.Run()
	var e *exec.ExitError
	if errors.As(err, &e) && !e.Success() {
		return // expected: non-zero exit
	}
	t.Fatalf("cmdDelete with no matching scanner: expected non-zero exit, got %v", err)
}

func TestExecResume_ChdirError(t *testing.T) {
	if os.Getenv("ROOST_TEST_EXECRESUME_CHDIR") == "1" {
		sess := &Session{ID: "s1", Platform: PlatformClaude, ProjectPath: "/nonexistent/path/that/does/not/exist"}
		execResume(sess, nil, Config{})
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestExecResume_ChdirError") //nolint:gosec // test binary path is safe
	cmd.Env = append(os.Environ(), "ROOST_TEST_EXECRESUME_CHDIR=1")
	err := cmd.Run()
	var e *exec.ExitError
	if errors.As(err, &e) && !e.Success() {
		return // expected: non-zero exit
	}
	t.Fatalf("execResume with bad chdir: expected non-zero exit, got %v", err)
}

func TestExecResume_NoScanner(t *testing.T) {
	if os.Getenv("ROOST_TEST_EXECRESUME_NOSCANNER") == "1" {
		sess := &Session{ID: "s1", Platform: PlatformClaude, ProjectPath: "/tmp"}
		execResume(sess, nil, Config{})
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestExecResume_NoScanner") //nolint:gosec // test binary path is safe
	cmd.Env = append(os.Environ(), "ROOST_TEST_EXECRESUME_NOSCANNER=1")
	err := cmd.Run()
	var e *exec.ExitError
	if errors.As(err, &e) && !e.Success() {
		return // expected: non-zero exit
	}
	t.Fatalf("execResume with no scanner: expected non-zero exit, got %v", err)
}

func TestExecNewSession_ChdirError(t *testing.T) {
	if os.Getenv("ROOST_TEST_EXECNEW_CHDIR") == "1" {
		execNewSession(PlatformClaude, "/nonexistent/path/that/does/not/exist", Config{})
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestExecNewSession_ChdirError") //nolint:gosec // test binary path is safe
	cmd.Env = append(os.Environ(), "ROOST_TEST_EXECNEW_CHDIR=1")
	err := cmd.Run()
	var e *exec.ExitError
	if errors.As(err, &e) && !e.Success() {
		return // expected: non-zero exit
	}
	t.Fatalf("execNewSession with bad chdir: expected non-zero exit, got %v", err)
}

func TestExecResume_BinNotFound(t *testing.T) {
	if os.Getenv("ROOST_TEST_EXECRESUME_NOBIN") == "1" {
		sess := &Session{ID: "s1", Platform: PlatformClaude, ProjectPath: "/tmp"}
		sc := &stubScanner{platform: PlatformClaude, resumeCmd: []string{"__nonexistent_binary_xyz__"}}
		execResume(sess, []Scanner{sc}, Config{})
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestExecResume_BinNotFound") //nolint:gosec // test binary path is safe
	cmd.Env = append(os.Environ(), "ROOST_TEST_EXECRESUME_NOBIN=1")
	err := cmd.Run()
	var e *exec.ExitError
	if errors.As(err, &e) && !e.Success() {
		return // expected: non-zero exit
	}
	t.Fatalf("execResume with missing binary: expected non-zero exit, got %v", err)
}

func TestExecNewSession_BinNotFound(t *testing.T) {
	if os.Getenv("ROOST_TEST_EXECNEW_NOBIN") == "1" {
		cfg := Config{
			Platforms: PlatformConfigs{
				Claude: PlatformConfig{Bin: "__nonexistent_binary_xyz__"},
			},
		}
		execNewSession(PlatformClaude, "/tmp", cfg)
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestExecNewSession_BinNotFound") //nolint:gosec // test binary path is safe
	cmd.Env = append(os.Environ(), "ROOST_TEST_EXECNEW_NOBIN=1")
	err := cmd.Run()
	var e *exec.ExitError
	if errors.As(err, &e) && !e.Success() {
		return // expected: non-zero exit
	}
	t.Fatalf("execNewSession with missing binary: expected non-zero exit, got %v", err)
}
