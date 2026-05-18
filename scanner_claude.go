package main

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ClaudeScanner 扫描 ~/.claude/
type ClaudeScanner struct {
	dataDir    string
	bin        string   // 二进制名称
	knownPaths []string // 从 .claude.json projects 键读取，用于精确还原编码路径
}

// claudeConfig 对应 ~/.claude/.claude.json
type claudeConfig struct {
	Projects map[string]json.RawMessage `json:"projects"` // key 是绝对路径
}

func NewClaudeScanner(cfg Config) *ClaudeScanner {
	home, _ := os.UserHomeDir()
	dataDir := filepath.Join(home, cfg.DataDirFor(PlatformClaude))
	s := &ClaudeScanner{dataDir: dataDir, bin: cfg.BinFor(PlatformClaude)}
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

func (s *ClaudeScanner) Platform() Platform { return PlatformClaude }
func (s *ClaudeScanner) DataDir() string    { return s.dataDir }

// claudeEncodePath 将绝对路径编码为 Claude 目录名格式
// /home/user/.dotfiles → -home-user--dotfiles
// 规则：/ → -，. → -，_ → -，再加前缀 -
func claudeEncodePath(path string) string {
	// 去掉前缀 /
	trimmed := strings.TrimPrefix(path, "/")
	// / → -
	result := strings.ReplaceAll(trimmed, "/", "-")
	// . → -
	result = strings.ReplaceAll(result, ".", "-")
	// _ → - （Claude 实际行为）
	result = strings.ReplaceAll(result, "_", "-")
	return "-" + result
}

// decodeDirName 将 Claude 编码的目录名还原为真实路径
// 策略：用 knownPaths 做精确匹配（路径编码后与 encoded 比对），
// 无匹配时 fallback 为简单 去前缀- 再 - → / 替换
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

func (s *ClaudeScanner) ScanProjects() ([]Project, error) {
	projectsDir := filepath.Join(s.dataDir, "projects")
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var projects []Project
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
		projects = append(projects, Project{
			Name:     projectShortName(fullPath),
			FullPath: fullPath,
			Sessions: sessions,
		})
	}
	return projects, nil
}

// claudeLine 用于解析 Claude JSONL 行
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

func (s *ClaudeScanner) scanSessions(projectDir, encodedName, fullPath string) ([]Session, error) {
	entries, err := os.ReadDir(projectDir)
	if err != nil {
		return nil, err
	}

	var sessions []Session
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

func (s *ClaudeScanner) parseSession(filePath, sid, encodedName, fullPath string) (Session, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return Session{}, err
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
	sc.Buffer(make([]byte, scanBufferSize), scanBufferSize)
	for sc.Scan() {
		var line claudeLine
		if err := json.Unmarshal(sc.Bytes(), &line); err != nil {
			continue
		}
		// 时间戳（ISO 8601）
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
		if msg.Role == roleUser || msg.Role == roleAssistant {
			msgCount++
		}
		// 标题：首条 user 消息
		if msg.Role == roleUser && !firstUserDone {
			firstUserDone = true
			title = extractClaudeText(msg.Content)
			title = truncate(title, 50)
		}
		// 模型：最后一条 assistant 消息
		if msg.Role == roleAssistant && msg.Model != "" {
			model = msg.Model
		}
	}

	displayTitle := title
	if displayTitle == "" {
		displayTitle = untitledTitle
	}

	return Session{
		ID:          sid,
		Platform:    PlatformClaude,
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

// extractClaudeText 从 content（字符串或数组）提取纯文本
func extractClaudeText(raw json.RawMessage) string {
	if raw == nil {
		return ""
	}
	// 先尝试字符串
	var str string
	if err := json.Unmarshal(raw, &str); err == nil {
		return str
	}
	// 再尝试数组
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

func (s *ClaudeScanner) DeleteSession(sess Session) error {
	base := filepath.Join(s.dataDir, "projects", sess.ProjectDir)
	_ = os.Remove(filepath.Join(base, sess.ID+".jsonl"))
	_ = os.RemoveAll(filepath.Join(base, sess.ID))
	return nil
}

func (s *ClaudeScanner) DeleteProject(p Project) error {
	if len(p.Sessions) == 0 {
		return nil
	}
	projectDir := filepath.Join(s.dataDir, "projects", p.Sessions[0].ProjectDir)
	return os.RemoveAll(projectDir)
}

func (s *ClaudeScanner) ResumeCmd(sess Session) []string {
	return []string{s.bin, flagResume, sess.ResumeArg}
}
