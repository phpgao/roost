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

// CodeBuddyScanner scans ~/.codebuddy/
type CodeBuddyScanner struct {
	dataDir     string
	bin         string
	trustedDirs []string // from settings.json trustedDirectories, for precise encoded path decoding
}

// cbSettings corresponds to ~/.codebuddy/settings.json
type cbSettings struct {
	TrustedDirectories []string `json:"trustedDirectories"`
}

func NewCodeBuddyScanner(cfg config.Config) *CodeBuddyScanner {
	dataDir := cfg.DataDirFor(types.PlatformCodeBuddy)
	if !filepath.IsAbs(dataDir) {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, dataDir)
	}
	s := &CodeBuddyScanner{
		dataDir: dataDir,
		bin:     cfg.BinFor(types.PlatformCodeBuddy),
	}
	s.loadTrustedDirs()
	return s
}

func (s *CodeBuddyScanner) loadTrustedDirs() {
	data, err := os.ReadFile(filepath.Join(s.dataDir, "settings.json"))
	if err != nil {
		return
	}
	var settings cbSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return
	}
	// Filter out wildcard entries (e.g. /path/**)
	for _, d := range settings.TrustedDirectories {
		if !strings.Contains(d, "*") {
			s.trustedDirs = append(s.trustedDirs, d)
		}
	}
}

func (s *CodeBuddyScanner) Platform() types.Platform { return types.PlatformCodeBuddy }
func (s *CodeBuddyScanner) DataDir() string          { return s.dataDir }

// encodePath encodes an absolute path to CodeBuddy directory name format.
// /home/user/code → home-user-code
func encodePath(path string) string {
	path = strings.TrimPrefix(path, "/")
	return strings.ReplaceAll(path, "/", "-")
}

// decodeDirName decodes a CodeBuddy-encoded directory name back to the real path.
// Strategy: use trustedDirs for precise matching (path encoded vs encoded comparison),
// take longest match; fallback to simple - → / replacement.
func (s *CodeBuddyScanner) decodeDirName(encoded string) string {
	var bestMatch string
	for _, dir := range s.trustedDirs {
		enc := encodePath(dir)
		if enc == encoded {
			// Exact match
			return dir
		}
		// Prefix match: encoded starts with the encoding of a known path
		if strings.HasPrefix(encoded, enc+"-") || strings.HasPrefix(encoded, enc) {
			if len(dir) > len(bestMatch) {
				bestMatch = dir
			}
		}
	}
	if bestMatch != "" && encodePath(bestMatch) == encoded {
		return bestMatch
	}
	// Fallback: simple replacement
	return "/" + strings.ReplaceAll(encoded, "-", "/")
}

func (s *CodeBuddyScanner) ScanProjects() ([]types.Project, error) {
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

// cbLine for parsing CodeBuddy JSONL lines
type cbLine struct {
	Type         string          `json:"type"`
	Role         string          `json:"role"`
	Timestamp    int64           `json:"timestamp"`
	AITitle      string          `json:"aiTitle"`
	SessionID    string          `json:"sessionId"`
	ProviderData json.RawMessage `json:"providerData"`
}

type cbProviderData struct {
	Model string `json:"model"`
	Agent string `json:"agent"`
}

func (s *CodeBuddyScanner) scanSessions(projectDir, encodedName, fullPath string) ([]types.Session, error) {
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

func (s *CodeBuddyScanner) parseSession(filePath, sid, encodedName, fullPath string) (types.Session, error) {
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

	var title, model, agentType string
	var lastActive time.Time
	msgCount := 0

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, types.ScanBufferSize), types.ScanBufferSize)
	for scanner.Scan() {
		var line cbLine
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			continue
		}
		// Timestamp (millisecond Unix)
		if line.Timestamp > 0 {
			t := time.UnixMilli(line.Timestamp)
			if t.After(lastActive) {
				lastActive = t
			}
		}
		// Title
		if line.Type == "ai-title" && line.AITitle != "" {
			title = line.AITitle
		}
		// Message count (only user/assistant)
		if line.Type == types.TypeMessage && (line.Role == types.RoleUser || line.Role == types.RoleAssistant) {
			msgCount++
		}
		// Model and agent (from last assistant message)
		if line.Type == types.TypeMessage && line.Role == types.RoleAssistant && line.ProviderData != nil {
			var pd cbProviderData
			if err := json.Unmarshal(line.ProviderData, &pd); err == nil {
				if pd.Model != "" {
					model = pd.Model
				}
				if pd.Agent != "" {
					agentType = pd.Agent
				}
			}
		}
	}

	displayTitle := title
	if displayTitle == "" {
		displayTitle = types.UntitledTitle
	}

	return types.Session{
		ID:          sid,
		Platform:    types.PlatformCodeBuddy,
		AgentType:   agentType,
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

func (s *CodeBuddyScanner) DeleteSession(sess types.Session) error {
	base := filepath.Join(s.dataDir, "projects", sess.ProjectDir)
	_ = os.Remove(filepath.Join(base, sess.ID+".jsonl"))
	_ = os.RemoveAll(filepath.Join(base, sess.ID))
	_ = os.RemoveAll(filepath.Join(s.dataDir, "tasks", sess.ID))
	_ = os.RemoveAll(filepath.Join(s.dataDir, "file-history", sess.ID))
	return nil
}

func (s *CodeBuddyScanner) DeleteProject(p types.Project) error {
	if len(p.Sessions) == 0 {
		return nil
	}
	projectDir := filepath.Join(s.dataDir, "projects", p.Sessions[0].ProjectDir)
	_ = os.RemoveAll(projectDir)
	for _, sess := range p.Sessions {
		_ = os.RemoveAll(filepath.Join(s.dataDir, "tasks", sess.ID))
		_ = os.RemoveAll(filepath.Join(s.dataDir, "file-history", sess.ID))
	}
	return nil
}

func (s *CodeBuddyScanner) ResumeCmd(sess types.Session) []string {
	return []string{s.bin, types.FlagResume, sess.ResumeArg}
}
