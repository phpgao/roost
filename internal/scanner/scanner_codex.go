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
)

// CodexScanner scans ~/.codex/sessions/
type CodexScanner struct {
	dataDir string
	bin     string
}

func NewCodexScanner(cfg config.Config) *CodexScanner {
	dataDir := cfg.DataDirFor(types.PlatformCodex)
	if !filepath.IsAbs(dataDir) {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, dataDir)
	}
	return &CodexScanner{
		dataDir: dataDir,
		bin:     cfg.BinFor(types.PlatformCodex),
	}
}

func (s *CodexScanner) Platform() types.Platform { return types.PlatformCodex }
func (s *CodexScanner) DataDir() string          { return s.dataDir }

// codexSessionMeta corresponds to the session_meta line payload
type codexSessionMeta struct {
	ID            string `json:"id"`
	CWD           string `json:"cwd"`
	Timestamp     string `json:"timestamp"`
	ModelProvider string `json:"model_provider"`
	CLIVersion    string `json:"cli_version"`
}

// codexLine is a generic JSONL line structure
type codexLine struct {
	Timestamp string          `json:"timestamp"`
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload"`
}

// codexEventPayload for parsing event_msg payload
type codexEventPayload struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// codexTurnContext for parsing turn_context payload to get model
type codexTurnContext struct {
	Model string `json:"model"`
}

func (s *CodexScanner) ScanProjects() ([]types.Project, error) {
	sessionsDir := filepath.Join(s.dataDir, "sessions")
	if _, err := os.Stat(sessionsDir); os.IsNotExist(err) {
		return nil, nil
	}

	// Aggregate sessions by cwd
	projectMap := make(map[string][]types.Session)

	// Walk sessions/YYYY/MM/DD/*.jsonl
	err := filepath.WalkDir(sessionsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip inaccessible paths
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), ".jsonl") {
			return nil
		}
		sess, err := s.parseSession(path)
		if err != nil {
			return nil // skip files that fail to parse
		}
		projectMap[sess.ProjectPath] = append(projectMap[sess.ProjectPath], sess)
		return nil
	})
	if err != nil {
		return nil, err
	}

	var projects []types.Project
	for fullPath, sessions := range projectMap {
		projects = append(projects, types.Project{
			Name:     types.ProjectShortName(fullPath),
			FullPath: fullPath,
			Sessions: sessions,
		})
	}
	return projects, nil
}

func (s *CodexScanner) parseSession(filePath string) (types.Session, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return types.Session{}, err
	}
	defer func() { _ = f.Close() }()

	info, _ := f.Stat()
	var sizeBytes int64
	if info != nil {
		sizeBytes = info.Size()
	}

	var meta codexSessionMeta
	var title, model string
	var lastActive time.Time
	var firstUserDone bool
	msgCount := 0

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, types.ScanBufferSize), types.ScanBufferSize)

	for sc.Scan() {
		var line codexLine
		if err := json.Unmarshal(sc.Bytes(), &line); err != nil {
			continue
		}

		// Update last active time
		if line.Timestamp != "" {
			if t, err := time.Parse(time.RFC3339, line.Timestamp); err == nil {
				if t.After(lastActive) {
					lastActive = t
				}
			}
		}

		switch line.Type {
		case types.TypeSessionMeta:
			_ = json.Unmarshal(line.Payload, &meta)

		case types.TypeEventMsg:
			var ev codexEventPayload
			if err := json.Unmarshal(line.Payload, &ev); err != nil {
				continue
			}
			switch ev.Type {
			case types.TypeUserMessage:
				msgCount++
				if !firstUserDone {
					firstUserDone = true
					// Take the first line as title
					msg := ev.Message
					if idx := strings.IndexByte(msg, '\n'); idx >= 0 {
						msg = msg[:idx]
					}
					title = types.Truncate(strings.TrimSpace(msg), 50)
				}
			case "agent_message":
				msgCount++
			}

		case "turn_context":
			var tc codexTurnContext
			if err := json.Unmarshal(line.Payload, &tc); err == nil && tc.Model != "" {
				model = tc.Model
			}
		}
	}

	if meta.ID == "" {
		return types.Session{}, os.ErrInvalid
	}

	displayTitle := title
	if displayTitle == "" {
		displayTitle = types.UntitledTitle
	}

	return types.Session{
		ID:          meta.ID,
		Platform:    types.PlatformCodex,
		Title:       displayTitle,
		Model:       model,
		LastActive:  lastActive,
		MsgCount:    msgCount,
		SizeBytes:   sizeBytes,
		FilePath:    filePath,
		ResumeArg:   meta.ID,
		ProjectPath: meta.CWD,
	}, nil
}

func (s *CodexScanner) DeleteSession(sess types.Session) error {
	return os.Remove(sess.FilePath)
}

func (s *CodexScanner) DeleteProject(p types.Project) error {
	for _, sess := range p.Sessions {
		_ = os.Remove(sess.FilePath)
	}
	return nil
}

func (s *CodexScanner) ResumeCmd(sess types.Session) []string {
	return []string{s.bin, types.CodexCmdResume, sess.ResumeArg}
}
