package scanner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/phpgao/roost/internal/config"
	"github.com/phpgao/roost/internal/types"
)

// GeminiScanner scans ~/.gemini/
type GeminiScanner struct {
	dataDir string
	bin     string
}

func NewGeminiScanner(cfg config.Config) *GeminiScanner {
	dataDir := cfg.DataDirFor(types.PlatformGemini)
	if !filepath.IsAbs(dataDir) {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, dataDir)
	}
	return &GeminiScanner{
		dataDir: dataDir,
		bin:     cfg.BinFor(types.PlatformGemini),
	}
}

func (s *GeminiScanner) Platform() types.Platform { return types.PlatformGemini }
func (s *GeminiScanner) DataDir() string          { return s.dataDir }

type geminiProjects struct {
	Projects map[string]string `json:"projects"`
}

type geminiSession struct {
	SessionID   string          `json:"sessionId"`
	StartTime   string          `json:"startTime"`
	LastUpdated string          `json:"lastUpdated"`
	Messages    []geminiMessage `json:"messages"`
}

type geminiMessage struct {
	Type    string          `json:"type"`
	Content json.RawMessage `json:"content"`
	Model   string          `json:"model"`
}

func (s *GeminiScanner) ScanProjects() ([]types.Project, error) {
	projectsFile := filepath.Join(s.dataDir, "projects.json")
	data, err := os.ReadFile(projectsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var gp geminiProjects
	if err := json.Unmarshal(data, &gp); err != nil {
		return nil, err
	}

	var projects []types.Project
	for fullPath, name := range gp.Projects {
		chatsDir := filepath.Join(s.dataDir, "tmp", name, "chats")
		sessions, err := s.scanSessions(chatsDir, name, fullPath)
		if err != nil || len(sessions) == 0 {
			continue
		}
		projects = append(projects, types.Project{
			Name:     types.ProjectShortName(fullPath),
			FullPath: fullPath,
			Sessions: sessions,
		})
	}
	return projects, nil
}

func (s *GeminiScanner) scanSessions(chatsDir, projectName, fullPath string) ([]types.Session, error) {
	entries, err := os.ReadDir(chatsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var sessions []types.Session
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		session, err := s.parseSession(filepath.Join(chatsDir, entry.Name()), projectName, fullPath)
		if err != nil {
			continue
		}
		sessions = append(sessions, session)
	}
	return sessions, nil
}

func (s *GeminiScanner) parseSession(filePath, projectName, fullPath string) (types.Session, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return types.Session{}, err
	}

	var gs geminiSession
	if err := json.Unmarshal(data, &gs); err != nil {
		return types.Session{}, err
	}

	var lastActive time.Time
	if gs.LastUpdated != "" {
		if t, err := time.Parse(time.RFC3339, gs.LastUpdated); err == nil {
			lastActive = t
		}
	}
	if lastActive.IsZero() && gs.StartTime != "" {
		if t, err := time.Parse(time.RFC3339, gs.StartTime); err == nil {
			lastActive = t
		}
	}

	var title string
	for _, msg := range gs.Messages {
		if msg.Type == types.RoleUser {
			title = types.Truncate(extractGeminiText(msg.Content), 50)
			break
		}
	}

	var model string
	for i := len(gs.Messages) - 1; i >= 0; i-- {
		if gs.Messages[i].Type == types.TypeGemini && gs.Messages[i].Model != "" {
			model = gs.Messages[i].Model
			break
		}
	}

	displayTitle := title
	if displayTitle == "" {
		displayTitle = types.UntitledTitle
	}

	return types.Session{
		ID:          gs.SessionID,
		Platform:    types.PlatformGemini,
		Title:       displayTitle,
		Model:       model,
		LastActive:  lastActive,
		MsgCount:    len(gs.Messages),
		SizeBytes:   int64(len(data)),
		ProjectDir:  projectName,
		FilePath:    filePath,
		ResumeArg:   gs.SessionID,
		ProjectPath: fullPath,
	}, nil
}

func extractGeminiText(raw json.RawMessage) string {
	if raw == nil {
		return ""
	}
	var str string
	if err := json.Unmarshal(raw, &str); err == nil {
		return str
	}
	var blocks []struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(raw, &blocks); err == nil {
		for _, b := range blocks {
			if b.Text != "" {
				return b.Text
			}
		}
	}
	return ""
}

func (s *GeminiScanner) DeleteSession(sess types.Session) error {
	return os.Remove(sess.FilePath)
}

func (s *GeminiScanner) DeleteProject(p types.Project) error {
	if len(p.Sessions) == 0 {
		return nil
	}
	chatsDir := filepath.Dir(p.Sessions[0].FilePath)
	projectDir := filepath.Dir(chatsDir)
	return os.RemoveAll(projectDir)
}

func (s *GeminiScanner) ResumeCmd(sess types.Session) []string {
	return []string{s.bin, types.FlagResume, sess.ResumeArg}
}
