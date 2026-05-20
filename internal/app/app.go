// Package app implements the main Bubble Tea Model that routes between
// sub-models (project, session, confirm, newagent, help) and handles
// global concerns (theme changes, window sizing, scanning, deleting,
// double-esc quit, spinner).
package app

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/phpgao/roost/internal/config"
	"github.com/phpgao/roost/internal/confirm"
	"github.com/phpgao/roost/internal/help"
	"github.com/phpgao/roost/internal/newagent"
	"github.com/phpgao/roost/internal/project"
	"github.com/phpgao/roost/internal/resume"
	"github.com/phpgao/roost/internal/scanner"
	"github.com/phpgao/roost/internal/session"
	"github.com/phpgao/roost/internal/types"
	"github.com/phpgao/roost/internal/view"
)

// ---------------------------------------------------------------------------
// Internal messages
// ---------------------------------------------------------------------------

// scanDoneMsg is sent when a project scan completes.
type scanDoneMsg struct {
	projects []types.Project
}

// deleteDoneMsg is sent when a delete operation completes.
type deleteDoneMsg struct {
	err error
}

// spinnerTickMsg drives the loading animation.
type spinnerTickMsg struct{}

// escHintTimeoutMsg is sent 2 seconds after escHint is set, to auto-clear it.
type escHintTimeoutMsg struct {
	gen int
}

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// ---------------------------------------------------------------------------
// deleteContext holds the context needed to execute a delete operation
// after the user confirms. This is stored in App because the confirm
// sub-model is stateless with respect to selection sets.
// ---------------------------------------------------------------------------

type deleteContext struct {
	target         types.DeleteTarget
	session        *types.Session
	project        *types.Project
	selectedSet    map[string]bool // from the originating sub-model
	batchInSession bool            // batch delete was from session screen
}

// ---------------------------------------------------------------------------
// App — the main router + state machine Model
// ---------------------------------------------------------------------------

// App is the top-level Bubble Tea Model. It delegates rendering to sub-models
// and routes their output messages to coordinate navigation and side effects.
type App struct {
	// Sub-models
	projectModel  project.ProjectModel
	sessionModel  session.SessionModel
	confirmModel  confirm.ConfirmModel
	newAgentModel newagent.NewAgentModel
	helpModel     help.HelpModel

	// Rendering
	renderers view.RendererSet
	styles    view.Styles

	// Infrastructure
	scanners           []scanner.Scanner
	installedPlatforms []types.Platform
	cfg                config.Config
	strategy           resume.ResumeStrategy

	// State
	screen    types.Screen
	loading   bool
	err       error
	errOutput string
	width     int
	height    int

	// Double-esc quit
	lastEscTime time.Time
	escHint     bool
	escHintGen  int

	// Spinner
	spinnerIdx int

	// Help screen return tracking
	screenBeforeHelp types.Screen

	// Pending resume/launch (replace mode)
	resumeSession     *types.Session
	launchPlatform    types.Platform
	launchProjectPath string
	launchPending     bool

	// Delete context — saved when opening confirm, used when confirmed
	pendingDelete deleteContext
}

// NewApp creates a new App with the given scanners and config.
func NewApp(scanners []scanner.Scanner, cfg config.Config) *App {
	installedPlatforms := make([]types.Platform, len(scanners))
	for i, sc := range scanners {
		installedPlatforms[i] = sc.Platform()
	}

	var strategy resume.ResumeStrategy
	switch cfg.GetResumeMode() {
	case types.ResumeModeSuspend:
		strategy = resume.SuspendStrategy{}
	default:
		strategy = resume.ReplaceStrategy{}
	}

	styles := view.NewStyles(true) // default dark; updated on BackgroundColorMsg
	renderers := view.NewDefaultRendererSet(styles)

	a := &App{
		scanners:           scanners,
		installedPlatforms: installedPlatforms,
		cfg:                cfg,
		strategy:           strategy,
		renderers:          renderers,
		styles:             styles,
		loading:            true,
		width:              80,
		height:             24,
		screen:             types.ScreenProject,
		screenBeforeHelp:   types.ScreenProject,
	}

	a.projectModel = project.NewProjectModel(renderers.Project, scanners, cfg, installedPlatforms)
	a.sessionModel = session.NewSessionModel(renderers.Session, scanners, cfg, installedPlatforms)
	a.confirmModel = confirm.NewConfirmModel(renderers.Confirm)
	a.newAgentModel = newagent.NewNewAgentModel(renderers.NewAgent, scanners, cfg, installedPlatforms)
	a.helpModel = help.NewHelpModel(renderers.Help)
	a.helpModel.SetPlatforms(installedPlatforms)

	return a
}

// Init implements tea.Model.
func (a *App) Init() tea.Cmd {
	//nolint:gocritic // tea.RequestBackgroundColor returns tea.Msg, not tea.Cmd — wrapping is required
	return tea.Batch(a.scanCmd(), spinnerTick(), func() tea.Msg { return tea.RequestBackgroundColor() })
}

// Update implements tea.Model.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Global messages
	if cmd := a.handleGlobal(msg); cmd != nil {
		return a, *cmd
	}

	// Sub-model output messages
	if cmd := a.handleSubModelMsg(msg); cmd != nil {
		return a, *cmd
	}

	// Key routing
	if key, ok := msg.(tea.KeyPressMsg); ok {
		if key.String() != types.KeyEsc {
			a.escHint = false
		}
		if a.err != nil {
			return a.handleErrorKey(key)
		}
		return a.routeToSubModel(key)
	}

	return a, nil
}

// handleGlobal processes window size, background color, spinner, scan/delete done,
// agent exit, and esc-hint messages. Returns nil if the message is not handled.
func (a *App) handleGlobal(msg tea.Msg) *tea.Cmd {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.propagateSize()
		return ptrCmd(nil)

	case tea.BackgroundColorMsg:
		a.styles = view.NewStyles(msg.IsDark())
		a.renderers.UpdateStyles(a.styles)
		a.propagateRenderers()
		return ptrCmd(nil)

	case spinnerTickMsg:
		if a.loading {
			a.spinnerIdx = (a.spinnerIdx + 1) % len(spinnerFrames)
			return ptrCmd(spinnerTick())
		}
		return ptrCmd(nil)

	case scanDoneMsg:
		a.loading = false
		a.projectModel.SetProjects(msg.projects)
		return ptrCmd(nil)

	case deleteDoneMsg:
		if msg.err != nil {
			a.err = msg.err
			a.errOutput = ""
		}
		a.loading = true
		a.screen = types.ScreenProject
		a.confirmModel = confirm.NewConfirmModel(a.renderers.Confirm)
		a.confirmModel.SetSize(a.width, a.height)
		cmd := tea.Batch(a.scanCmd(), spinnerTick())
		return &cmd

	case resume.AgentExitMsg:
		if msg.Err != nil {
			a.err = msg.Err
			a.errOutput = msg.Output
		} else {
			a.err = nil
			a.errOutput = ""
		}
		a.loading = true
		cmd := tea.Batch(a.scanCmd(), spinnerTick())
		return &cmd

	case escHintTimeoutMsg:
		if msg.gen == a.escHintGen {
			a.escHint = false
		}
		return ptrCmd(nil)
	}

	return nil
}

// handleSubModelMsg processes output messages from sub-models.
// Returns nil if the message is not a sub-model message.
func (a *App) handleSubModelMsg(msg tea.Msg) *tea.Cmd {
	switch msg := msg.(type) {
	// Replace strategy messages
	case resume.ResumeRequestMsg:
		a.resumeSession = &msg.Session
		return ptrCmd(tea.Quit)

	case resume.LaunchNewRequestMsg:
		a.launchPlatform = msg.Platform
		a.launchProjectPath = msg.ProjectPath
		a.launchPending = true
		return ptrCmd(tea.Quit)

	// Project sub-model
	case project.SelectMsg:
		a.sessionModel.SetProject(&msg.Project)
		a.screen = types.ScreenSession
		return ptrCmd(nil)

	case project.DeleteMsg:
		a.pendingDelete = deleteContext{target: types.DeleteTargetProject, project: &msg.Project}
		a.confirmModel.SetProjectTarget(&msg.Project)
		a.screen = types.ScreenConfirm
		return ptrCmd(nil)

	case project.BatchDeleteMsg:
		a.pendingDelete = deleteContext{
			target: types.DeleteTargetBatch, selectedSet: msg.SelectedSet, batchInSession: false,
		}
		a.confirmModel.SetBatchTarget(len(msg.SelectedSet), false)
		a.screen = types.ScreenConfirm
		return ptrCmd(nil)

	case project.RefreshMsg:
		a.loading = true
		cmd := tea.Batch(a.scanCmd(), spinnerTick())
		return &cmd

	case project.ToggleHelpMsg:
		a.screenBeforeHelp = a.screen
		a.screen = types.ScreenHelp
		return ptrCmd(nil)

	case project.EscMsg:
		return a.handleEscQuit()

	// Session sub-model
	case session.ResumeMsg:
		return ptrCmd(func() tea.Msg { return resume.ResumeRequestMsg{Session: msg.Session} })

	case session.DeleteMsg:
		a.pendingDelete = deleteContext{target: types.DeleteTargetSession, session: &msg.Session}
		a.confirmModel.SetSessionTarget(&msg.Session)
		a.screen = types.ScreenConfirm
		return ptrCmd(nil)

	case session.BatchDeleteMsg:
		a.pendingDelete = deleteContext{
			target: types.DeleteTargetBatch, selectedSet: msg.SelectedSet, batchInSession: true,
		}
		a.confirmModel.SetBatchTarget(len(msg.SelectedSet), true)
		a.screen = types.ScreenConfirm
		return ptrCmd(nil)

	case session.DeleteProjectMsg:
		a.pendingDelete = deleteContext{target: types.DeleteTargetProject, project: &msg.Project}
		a.confirmModel.SetProjectTarget(&msg.Project)
		a.screen = types.ScreenConfirm
		return ptrCmd(nil)

	case session.NewAgentMsg:
		a.newAgentModel.SetProject(a.sessionModel.Project())
		a.screen = types.ScreenNewAgent
		return ptrCmd(nil)

	case session.BackMsg:
		a.screen = types.ScreenProject
		return ptrCmd(nil)

	case session.RefreshMsg:
		a.loading = true
		cmd := tea.Batch(a.scanCmd(), spinnerTick())
		return &cmd

	case session.ToggleHelpMsg:
		a.screenBeforeHelp = a.screen
		a.screen = types.ScreenHelp
		return ptrCmd(nil)

	// Confirm sub-model
	case confirm.ResultMsg:
		model, cmd := a.handleConfirmResult(msg.Confirmed)
		_ = model // always returns a
		return &cmd

	// NewAgent sub-model
	case newagent.LaunchMsg:
		return ptrCmd(func() tea.Msg {
			return resume.LaunchNewRequestMsg{Platform: msg.Platform, ProjectPath: msg.ProjectPath}
		})

	case newagent.BackMsg:
		a.screen = types.ScreenSession
		return ptrCmd(nil)

	// Help sub-model
	case help.CloseMsg:
		a.screen = a.screenBeforeHelp
		return ptrCmd(nil)
	}

	return nil
}

// handleEscQuit implements the double-esc quit logic.
func (a *App) handleEscQuit() *tea.Cmd {
	if a.escHint && time.Since(a.lastEscTime) < 2*time.Second {
		return ptrCmd(tea.Quit)
	}
	a.escHint = true
	a.lastEscTime = time.Now()
	a.escHintGen++
	cmd := tea.Tick(2*time.Second, func(_ time.Time) tea.Msg {
		return escHintTimeoutMsg{gen: a.escHintGen}
	})
	return &cmd
}

// View implements tea.Model.
func (a *App) View() tea.View {
	var content string

	if a.loading {
		vm := view.LoadingViewModel{
			SpinnerFrame: spinnerFrames[a.spinnerIdx],
			Width:        a.width,
		}
		content = a.renderers.Loading.Render(vm)
	} else if a.err != nil {
		vm := view.ErrorViewModel{
			Error:       a.err.Error(),
			ErrorOutput: a.errOutput,
			Width:       a.width,
		}
		content = a.renderers.Error.Render(vm)
	} else {
		switch a.screen {
		case types.ScreenProject:
			content = a.projectModel.View().Content
		case types.ScreenSession:
			content = a.sessionModel.View().Content
		case types.ScreenConfirm:
			content = a.confirmModel.View().Content
		case types.ScreenNewAgent:
			content = a.newAgentModel.View().Content
		case types.ScreenHelp:
			content = a.helpModel.View().Content
		}
	}

	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

// ---------------------------------------------------------------------------
// Key routing
// ---------------------------------------------------------------------------

func (a *App) routeToSubModel(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch a.screen {
	case types.ScreenProject:
		updated, c := a.projectModel.Update(msg)
		a.projectModel = updated.(project.ProjectModel)
		cmd = c
	case types.ScreenSession:
		updated, c := a.sessionModel.Update(msg)
		a.sessionModel = updated.(session.SessionModel)
		cmd = c
	case types.ScreenConfirm:
		updated, c := a.confirmModel.Update(msg)
		a.confirmModel = updated.(confirm.ConfirmModel)
		cmd = c
	case types.ScreenNewAgent:
		updated, c := a.newAgentModel.Update(msg)
		a.newAgentModel = updated.(newagent.NewAgentModel)
		cmd = c
	case types.ScreenHelp:
		updated, c := a.helpModel.Update(msg)
		a.helpModel = updated.(help.HelpModel)
		cmd = c
	default:
		return a, nil
	}

	return a, cmd
}

func (a *App) handleErrorKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", types.KeyCtrlC:
		return a, tea.Quit
	case types.KeyEsc, types.KeyEnter:
		a.err = nil
		a.errOutput = ""
		return a, nil
	default:
		return a, nil
	}
}

// ---------------------------------------------------------------------------
// Confirm result handling
// ---------------------------------------------------------------------------

func (a *App) handleConfirmResult(confirmed bool) (tea.Model, tea.Cmd) {
	if !confirmed {
		a.screen = a.confirmReturnScreen()
		return a, nil
	}
	cmd := a.doDelete()
	return a, cmd
}

// confirmReturnScreen determines which screen to return to after canceling a delete.
func (a *App) confirmReturnScreen() types.Screen {
	if a.pendingDelete.batchInSession {
		return types.ScreenSession
	}
	if a.pendingDelete.target == types.DeleteTargetSession {
		return types.ScreenSession
	}
	return types.ScreenProject
}

// doDelete constructs a tea.Cmd that performs the actual delete operation.
func (a *App) doDelete() tea.Cmd {
	scanners := a.scanners
	ctx := a.pendingDelete

	switch ctx.target {
	case types.DeleteTargetBatch:
		if ctx.batchInSession {
			return a.doBatchDeleteSessions(scanners, ctx)
		}
		return a.doBatchDeleteProjects(scanners, ctx)
	default:
		// single delete — capture in closure
		return func() tea.Msg {
			return deleteDoneMsg{err: a.executeSingleDelete(scanners, ctx)}
		}
	}
}

// executeSingleDelete performs a single session or project delete.
func (a *App) executeSingleDelete(scanners []scanner.Scanner, ctx deleteContext) error {
	switch ctx.target {
	case types.DeleteTargetSession:
		if ctx.session == nil {
			return nil
		}
		for _, sc := range scanners {
			if sc.Platform() == ctx.session.Platform {
				return sc.DeleteSession(*ctx.session)
			}
		}
	case types.DeleteTargetProject:
		if ctx.project != nil {
			return deleteProjectByScanner(scanners, *ctx.project)
		}
	}
	return nil
}

// doBatchDeleteSessions constructs a cmd for batch-deleting sessions.
func (a *App) doBatchDeleteSessions(scanners []scanner.Scanner, ctx deleteContext) tea.Cmd {
	// Capture the session list snapshot at cmd-creation time
	var sessionsToDelete []types.Session
	if proj := a.sessionModel.Project(); proj != nil {
		for _, s := range proj.Sessions {
			if ctx.selectedSet[s.ID] {
				sessionsToDelete = append(sessionsToDelete, s)
			}
		}
	}

	return func() tea.Msg {
		var err error
		for _, s := range sessionsToDelete {
			if e := deleteSessionByScanner(scanners, s); e != nil {
				err = e
			}
		}
		return deleteDoneMsg{err: err}
	}
}

// doBatchDeleteProjects constructs a cmd for batch-deleting projects.
func (a *App) doBatchDeleteProjects(scanners []scanner.Scanner, ctx deleteContext) tea.Cmd {
	// Capture the project list snapshot at cmd-creation time
	var projectsToDelete []types.Project
	for _, p := range a.projectModel.Projects() {
		if ctx.selectedSet[p.FullPath] {
			projectsToDelete = append(projectsToDelete, p)
		}
	}

	return func() tea.Msg {
		var err error
		for _, p := range projectsToDelete {
			if e := deleteProjectByScanner(scanners, p); e != nil {
				err = e
			}
		}
		return deleteDoneMsg{err: err}
	}
}

// deleteSessionByScanner deletes a single session using the matching scanner.
func deleteSessionByScanner(scanners []scanner.Scanner, s types.Session) error {
	for _, sc := range scanners {
		if sc.Platform() == s.Platform {
			return sc.DeleteSession(s)
		}
	}
	return nil
}

// deleteProjectByScanner deletes all sessions in a project by delegating
// to the appropriate scanner for each platform.
func deleteProjectByScanner(scanners []scanner.Scanner, p types.Project) error {
	var err error
	byPlatform := make(map[types.Platform][]types.Session)
	for _, s := range p.Sessions {
		byPlatform[s.Platform] = append(byPlatform[s.Platform], s)
	}
	for _, sc := range scanners {
		if sessions, ok := byPlatform[sc.Platform()]; ok {
			tempProj := types.Project{
				Name:     p.Name,
				FullPath: p.FullPath,
				Sessions: sessions,
			}
			if e := sc.DeleteProject(tempProj); e != nil {
				err = e
			}
		}
	}
	return err
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func (a *App) propagateSize() {
	a.projectModel.SetSize(a.width, a.height)
	a.sessionModel.SetSize(a.width, a.height)
	a.confirmModel.SetSize(a.width, a.height)
	a.newAgentModel.SetSize(a.width, a.height)
	a.helpModel.SetSize(a.width, a.height)
}

func (a *App) propagateRenderers() {
	a.projectModel.SetRenderer(a.renderers.Project)
	a.sessionModel.SetRenderer(a.renderers.Session)
	a.confirmModel.SetRenderer(a.renderers.Confirm)
	a.newAgentModel.SetRenderer(a.renderers.NewAgent)
	a.helpModel.SetRenderer(a.renderers.Help)
}

func (a *App) scanCmd() tea.Cmd {
	scanners := a.scanners
	return func() tea.Msg {
		projects := scanner.ScanProjectsParallel(scanners)
		return scanDoneMsg{projects: projects}
	}
}

func spinnerTick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(_ time.Time) tea.Msg {
		return spinnerTickMsg{}
	})
}

// ptrCmd wraps a tea.Cmd in a pointer for use as an optional return value.
func ptrCmd(cmd tea.Cmd) *tea.Cmd { return &cmd }

// ---------------------------------------------------------------------------
// Accessors for main.go (post-quit logic)
// ---------------------------------------------------------------------------

// ResumeSession returns the session to resume (replace mode), or nil.
func (a *App) ResumeSession() *types.Session { return a.resumeSession }

// LaunchPending returns whether a new session launch is pending (replace mode).
func (a *App) LaunchPending() bool { return a.launchPending }

// LaunchPlatform returns the platform for the pending launch.
func (a *App) LaunchPlatform() types.Platform { return a.launchPlatform }

// LaunchProjectPath returns the project path for the pending launch.
func (a *App) LaunchProjectPath() string { return a.launchProjectPath }
