package scanner

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/phpgao/roost/internal/config"
	"github.com/phpgao/roost/internal/types"
	"gopkg.in/yaml.v3"
)

// CopilotScanner scans ~/.copilot/session-state/
type CopilotScanner struct {
	dataDir string
	bin     string
}

// copilotWorkspace corresponds to workspace.yaml in each session directory
type copilotWorkspace struct {
	ID         string `yaml:"id"`
	Cwd        string `yaml:"cwd"`
	GitRoot    string `yaml:"git_root"`
	Repository string `yaml:"repository"`
	HostType   string `yaml:"host_type"`
	Branch     string `yaml:"branch"`
	Name       string `yaml:"name"`
	CreatedAt  string `yaml:"created_at"`
	UpdatedAt  string `yaml:"updated_at"`
}

func NewCopilotScanner(cfg config.Config) *CopilotScanner {
	dataDir := cfg.DataDirFor(types.PlatformCopilot)
	if !filepath.IsAbs(dataDir) {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, dataDir)
	}
	return &CopilotScanner{
		dataDir: dataDir,
		bin:     cfg.BinFor(types.PlatformCopilot),
	}
}

func (s *CopilotScanner) Platform() types.Platform { return types.PlatformCopilot }
func (s *CopilotScanner) DataDir() string          { return s.dataDir }

func (s *CopilotScanner) ScanProjects() ([]types.Project, error) {
	sessionsDir := filepath.Join(s.dataDir, "session-state")
	entries, err := os.ReadDir(sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	// Group sessions by cwd
	projectMap := make(map[string][]types.Session)
	var cwdOrder []string

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		wsPath := filepath.Join(sessionsDir, entry.Name(), "workspace.yaml")
		data, err := os.ReadFile(wsPath)
		if err != nil {
			continue
		}
		var ws copilotWorkspace
		if err := yaml.Unmarshal(data, &ws); err != nil {
			continue
		}
		if ws.Cwd == "" {
			continue
		}

		session := s.parseSession(sessionsDir, ws)
		if _, exists := projectMap[ws.Cwd]; !exists {
			cwdOrder = append(cwdOrder, ws.Cwd)
		}
		projectMap[ws.Cwd] = append(projectMap[ws.Cwd], session)
	}

	var projects []types.Project
	for _, cwd := range cwdOrder {
		sessions := projectMap[cwd]
		projects = append(projects, types.Project{
			Name:     types.ProjectShortName(cwd),
			FullPath: cwd,
			Sessions: sessions,
		})
	}
	return projects, nil
}

func (s *CopilotScanner) parseSession(sessionsDir string, ws copilotWorkspace) types.Session {
	var lastActive time.Time
	var msgCount int
	var model string

	if ws.UpdatedAt != "" {
		if t, err := time.Parse(time.RFC3339, ws.UpdatedAt); err == nil {
			lastActive = t
		}
	}
	if ws.CreatedAt != "" {
		if t, err := time.Parse(time.RFC3339, ws.CreatedAt); err == nil {
			if t.After(lastActive) {
				lastActive = t
			}
		}
	}

	// Count turns from events.jsonl and extract model
	eventsPath := filepath.Join(sessionsDir, ws.ID, "events.jsonl")
	msgCount, model = s.parseEvents(eventsPath)

	title := ws.Name
	if title == "" {
		title = types.UntitledTitle
	}

	// Calculate session directory size
	var sizeBytes int64
	sessionDir := filepath.Join(sessionsDir, ws.ID)
	_ = filepath.Walk(sessionDir, func(_ string, info os.FileInfo, err error) error {
		if err == nil && info != nil {
			sizeBytes += info.Size()
		}
		return nil
	})

	return types.Session{
		ID:          ws.ID,
		Platform:    types.PlatformCopilot,
		Title:       types.Truncate(title, 50),
		Model:       model,
		LastActive:  lastActive,
		MsgCount:    msgCount,
		SizeBytes:   sizeBytes,
		ProjectDir:  ws.Cwd,
		FilePath:    sessionDir,
		ResumeArg:   ws.ID,
		ProjectPath: ws.Cwd,
	}
}

// copilotEvent is a minimal struct for parsing events.jsonl
type copilotEvent struct {
	Type string `json:"type"`
	Data struct {
		NewModel string `json:"newModel"`
	} `json:"data"`
}

func (s *CopilotScanner) parseEvents(path string) (msgCount int, model string) {
	f, err := os.Open(path)
	if err != nil {
		return 0, ""
	}
	defer func() { _ = f.Close() }()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, types.ScanBufferSize), types.ScanBufferSize)
	for sc.Scan() {
		line := sc.Text()
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Count user and assistant turns
		if strings.Contains(line, `"type":"turn"`) || strings.Contains(line, `"type":"message"`) {
			msgCount++
		}
		// Extract model from session.model_change
		if strings.Contains(line, `"session.model_change"`) {
			var evt copilotEvent
			if err := json.Unmarshal([]byte(line), &evt); err == nil && evt.Data.NewModel != "" {
				model = evt.Data.NewModel
			}
		}
	}
	return msgCount, model
}

func (s *CopilotScanner) DeleteSession(sess types.Session) error {
	return os.RemoveAll(sess.FilePath)
}

func (s *CopilotScanner) DeleteProject(p types.Project) error {
	for _, sess := range p.Sessions {
		_ = s.DeleteSession(sess)
	}
	return nil
}

func (s *CopilotScanner) ResumeCmd(sess types.Session) []string {
	return []string{s.bin, "--resume=" + sess.ResumeArg}
}
