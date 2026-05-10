package session

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Session represents a claude-forge session.
type Session struct {
	ID        string
	CreatedAt time.Time
	FirstMsg  string
}

// jsonLine represents a single line in a session JSONL file.
type jsonLine struct {
	Type      string `json:"type"`
	Timestamp string `json:"timestamp"`
	Message   string `json:"message"`
}

// List reads JSONL session files from the project's session directory.
// sessionDir is the path like ~/.claude-forge/<project-id>/
// Each .jsonl file is a session. The filename (without extension) is the session ID.
// It reads the first lines of each file to extract the timestamp and first user message.
// Returns sessions sorted by creation time (most recent first).
func List(sessionDir string) ([]Session, error) {
	entries, err := os.ReadDir(sessionDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read session directory: %w", err)
	}

	var sessions []Session
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".jsonl") {
			continue
		}

		sessionID := strings.TrimSuffix(entry.Name(), ".jsonl")
		filePath := filepath.Join(sessionDir, entry.Name())

		session, err := parseSessionFile(sessionID, filePath)
		if err != nil {
			// Skip malformed files
			continue
		}

		sessions = append(sessions, *session)
	}

	// Sort by creation time, most recent first
	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].CreatedAt.After(sessions[j].CreatedAt)
	})

	return sessions, nil
}

// parseSessionFile parses a JSONL session file and extracts metadata.
func parseSessionFile(sessionID string, filePath string) (*Session, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open session file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	var createdAt time.Time
	var firstMsg string
	foundTimestamp := false

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var entry jsonLine
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		// Extract timestamp from the first valid entry
		if !foundTimestamp && entry.Timestamp != "" {
			parsed, err := time.Parse(time.RFC3339, entry.Timestamp)
			if err == nil {
				createdAt = parsed
				foundTimestamp = true
			}
		}

		// Extract first user message
		if entry.Type == "human" && firstMsg == "" {
			firstMsg = entry.Message
		}

		// Stop once we have both
		if foundTimestamp && firstMsg != "" {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read session file: %w", err)
	}

	if !foundTimestamp {
		return nil, fmt.Errorf("no timestamp found in session file")
	}

	return &Session{
		ID:        sessionID,
		CreatedAt: createdAt,
		FirstMsg:  firstMsg,
	}, nil
}
