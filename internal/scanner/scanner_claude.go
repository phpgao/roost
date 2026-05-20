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

// ClaudeScanner scans ~/.claude/
type ClaudeScanner struct {
	dataDir    string
	bin        string   // binary name
	knownPaths []string // from .claude.json projects key, for precise encoded path decoding
}

// claudeConfig corresponds to ~/.claude/.claude.json
type claudeConfig struct {
	Projects map[string]json.RawMessage `json:"projects"` // key is absolute path
}

func NewClaudeScanner(cfg config.Config) *ClaudeScanner {
	dataDir := cfg.DataDirFor(types.PlatformClaude)
	if !filepath.IsAbs(dataDir) {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, dataDir)
	}
	s := &ClaudeScanner{dataDir: dataDir, bin: cfg.BinFor(types.PlatformClaude)}
	s.loadKnownPaths()
	return s
}

func (s *ClaudeScanner) loadKnownPaths() {
	data, err := os.ReadFile(filepath.Join(s.dataDir, ".claude.json"))
	if err != nil {
		return
	}
	var cfg claudeConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return
	}
	for path := range cfg.Projects {
		s.knownPaths = append(s.knownPaths, path)
	}
}

func (s *ClaudeScanner) Platform() types.Platform { return types.PlatformClaude }
func (s *ClaudeScanner) DataDir() string          { return s.dataDir }

// claudeEncodePath encodes an absolute path to Claude directory name format.
// /home/user/.dotfiles → -home-user--dotfiles
// Rules: / → -, . → -, _ → -, then prepend -
func claudeEncodePath(path string) string {
	trimmed := strings.TrimPrefix(path, "/")
	result := strings.ReplaceAll(trimmed, "/", "-")
	result = strings.ReplaceAll(result, ".", "-")
	result = strings.ReplaceAll(result, "_", "-")
	return "-" + result
}

// decodeDirName decodes a Claude-encoded directory name back to the real path.
// Strategy: use knownPaths for precise matching (path encoded vs encoded comparison),
// fallback to simple prefix-stripping and - → / replacement.
func (s *ClaudeScanner) decodeDirName(encoded string) string {
	for _, path := range s.knownPaths {
		if claudeEncodePath(path) == encoded {
			return path
		}
	}
	// fallback
	trimmed := strings.TrimPrefix(encoded, "-")
	return "/" + strings.ReplaceAll(trimmed, "-", "/")
}

func (s *ClaudeScanner) ScanProjects() ([]types.Project, error) {
	projectsDir := filepath.Join(s.dataDir, "projects")
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var projects []types.Project
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		fullPath := s.decodeDirName(entry.Name())
		sessions, err := s.scanSessions(filepath.Join(projectsDir, entry.Name()), entry.Name(), fullPath)
		if err != nil {
			continue
		}
		if len(sessions) == 0 {
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

// claudeLine for parsing Claude JSONL lines
type claudeLine struct {
	Type      string          `json:"type"`
	UUID      string          `json:"uuid"`
	Timestamp string          `json:"timestamp"`
	Message   json.RawMessage `json:"message"`
}

type claudeMessage struct {
	Role    string          `json:"role"`
	Model   string          `json:"model"`
	Content json.RawMessage `json:"content"`
}

type claudeContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

func (s *ClaudeScanner) scanSessions(projectDir, encodedName, fullPath string) ([]types.Session, error) {
	entries, err := os.ReadDir(projectDir)
	if err != nil {
		return nil, err
	}

	var sessions []types.Session
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}
		sid := strings.TrimSuffix(entry.Name(), ".jsonl")
		session, err := s.parseSession(filepath.Join(projectDir, entry.Name()), sid, encodedName, fullPath)
		if err != nil {
			continue
		}
		sessions = append(sessions, session)
	}
	return sessions, nil
}

func (s *ClaudeScanner) parseSession(filePath, sid, encodedName, fullPath string) (types.Session, error) {
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

	var title, model string
	var lastActive time.Time
	var firstUserDone bool
	msgCount := 0

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, types.ScanBufferSize), types.ScanBufferSize)
	for sc.Scan() {
		var line claudeLine
		if err := json.Unmarshal(sc.Bytes(), &line); err != nil {
			continue
		}
		// Timestamp (ISO 8601)
		if line.Timestamp != "" {
			if t, err := time.Parse(time.RFC3339, line.Timestamp); err == nil {
				if t.After(lastActive) {
					lastActive = t
				}
			}
		}
		if line.Message == nil {
			continue
		}
		var msg claudeMessage
		if err := json.Unmarshal(line.Message, &msg); err != nil {
			continue
		}
		if msg.Role == types.RoleUser || msg.Role == types.RoleAssistant {
			msgCount++
		}
		// Title: first user message
		if msg.Role == types.RoleUser && !firstUserDone {
			firstUserDone = true
			title = extractClaudeText(msg.Content)
			title = types.Truncate(title, 50)
		}
		// Model: last assistant message
		if msg.Role == types.RoleAssistant && msg.Model != "" {
			model = msg.Model
		}
	}

	displayTitle := title
	if displayTitle == "" {
		displayTitle = types.UntitledTitle
	}

	return types.Session{
		ID:          sid,
		Platform:    types.PlatformClaude,
		Title:       displayTitle,
		Model:       model,
		LastActive:  lastActive,
		MsgCount:    msgCount,
		SizeBytes:   sizeBytes,
		ProjectDir:  encodedName,
		FilePath:    filePath,
		ResumeArg:   sid,
		ProjectPath: fullPath,
	}, nil
}

// extractClaudeText extracts plain text from content (string or array)
func extractClaudeText(raw json.RawMessage) string {
	if raw == nil {
		return ""
	}
	// Try string first
	var str string
	if err := json.Unmarshal(raw, &str); err == nil {
		return str
	}
	// Try array
	var blocks []claudeContentBlock
	if err := json.Unmarshal(raw, &blocks); err == nil {
		for _, b := range blocks {
			if b.Text != "" {
				return b.Text
			}
		}
	}
	return ""
}

func (s *ClaudeScanner) DeleteSession(sess types.Session) error {
	base := filepath.Join(s.dataDir, "projects", sess.ProjectDir)
	_ = os.Remove(filepath.Join(base, sess.ID+".jsonl"))
	_ = os.RemoveAll(filepath.Join(base, sess.ID))
	return nil
}

func (s *ClaudeScanner) DeleteProject(p types.Project) error {
	if len(p.Sessions) == 0 {
		return nil
	}
	projectDir := filepath.Join(s.dataDir, "projects", p.Sessions[0].ProjectDir)
	return os.RemoveAll(projectDir)
}

func (s *ClaudeScanner) ResumeCmd(sess types.Session) []string {
	return []string{s.bin, types.FlagResume, sess.ResumeArg}
}
