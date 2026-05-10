package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestList(t *testing.T) {
	tests := []struct {
		name    string
		files   map[string]string
		want    []Session
		wantErr bool
	}{
		{
			name: "multiple sessions sorted by most recent first",
			files: map[string]string{
				"session-1.jsonl": `{"type":"system","timestamp":"2026-05-08T14:30:00Z","message":"init"}
{"type":"human","timestamp":"2026-05-08T14:30:01Z","message":"Hello world"}
{"type":"assistant","timestamp":"2026-05-08T14:30:02Z","message":"Hi there"}`,
				"session-2.jsonl": `{"type":"system","timestamp":"2026-05-09T10:00:00Z","message":"init"}
{"type":"human","timestamp":"2026-05-09T10:00:01Z","message":"Fix the bug"}`,
			},
			want: []Session{
				{
					ID:        "session-2",
					CreatedAt: time.Date(2026, 5, 9, 10, 0, 0, 0, time.UTC),
					FirstMsg:  "Fix the bug",
				},
				{
					ID:        "session-1",
					CreatedAt: time.Date(2026, 5, 8, 14, 30, 0, 0, time.UTC),
					FirstMsg:  "Hello world",
				},
			},
		},
		{
			name: "session without human message",
			files: map[string]string{
				"session-1.jsonl": `{"type":"system","timestamp":"2026-05-08T14:30:00Z","message":"init"}`,
			},
			want: []Session{
				{
					ID:        "session-1",
					CreatedAt: time.Date(2026, 5, 8, 14, 30, 0, 0, time.UTC),
					FirstMsg:  "",
				},
			},
		},
		{
			name:  "empty directory",
			files: map[string]string{},
			want:  nil,
		},
		{
			name: "skips non-jsonl files",
			files: map[string]string{
				"readme.txt":      "not a session",
				"session-1.jsonl": `{"type":"system","timestamp":"2026-05-08T14:30:00Z","message":"init"}`,
			},
			want: []Session{
				{
					ID:        "session-1",
					CreatedAt: time.Date(2026, 5, 8, 14, 30, 0, 0, time.UTC),
					FirstMsg:  "",
				},
			},
		},
		{
			name: "skips malformed files without timestamp",
			files: map[string]string{
				"bad.jsonl":  `{"type":"system","message":"no timestamp"}`,
				"good.jsonl": `{"type":"system","timestamp":"2026-05-08T14:30:00Z","message":"init"}`,
			},
			want: []Session{
				{
					ID:        "good",
					CreatedAt: time.Date(2026, 5, 8, 14, 30, 0, 0, time.UTC),
					FirstMsg:  "",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			for name, content := range tt.files {
				err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0o644)
				require.NoError(t, err)
			}

			got, err := List(tmpDir)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestList_NonExistentDirectory(t *testing.T) {
	got, err := List("/nonexistent/directory/path")

	require.NoError(t, err)
	assert.Nil(t, got)
}

func TestList_SkipsMalformedJSON(t *testing.T) {
	tmpDir := t.TempDir()

	// File with invalid JSON on some lines but valid data
	content := `not json at all
{"type":"system","timestamp":"2026-05-08T14:30:00Z","message":"init"}
also not json
{"type":"human","timestamp":"2026-05-08T14:30:01Z","message":"Hello"}
`
	err := os.WriteFile(filepath.Join(tmpDir, "session-1.jsonl"), []byte(content), 0o644)
	require.NoError(t, err)

	got, err := List(tmpDir)

	require.NoError(t, err)
	require.Len(t, got, 1)
	assert.Equal(t, "session-1", got[0].ID)
	assert.Equal(t, "Hello", got[0].FirstMsg)
	assert.Equal(t, time.Date(2026, 5, 8, 14, 30, 0, 0, time.UTC), got[0].CreatedAt)
}

func TestParseSessionFile(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		want        *Session
		wantErr     bool
		errContains string
	}{
		{
			name: "valid session with system and human messages",
			content: `{"type":"system","timestamp":"2026-05-08T14:30:00Z","message":"init"}
{"type":"human","timestamp":"2026-05-08T14:30:01Z","message":"Hello world"}`,
			want: &Session{
				ID:        "test-session",
				CreatedAt: time.Date(2026, 5, 8, 14, 30, 0, 0, time.UTC),
				FirstMsg:  "Hello world",
			},
		},
		{
			name:    "system message only",
			content: `{"type":"system","timestamp":"2026-05-08T14:30:00Z","message":"init"}`,
			want: &Session{
				ID:        "test-session",
				CreatedAt: time.Date(2026, 5, 8, 14, 30, 0, 0, time.UTC),
				FirstMsg:  "",
			},
		},
		{
			name:        "no timestamp at all",
			content:     `{"type":"system","message":"init"}`,
			wantErr:     true,
			errContains: "no timestamp found",
		},
		{
			name:        "empty file",
			content:     "",
			wantErr:     true,
			errContains: "no timestamp found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			filePath := filepath.Join(tmpDir, "test-session.jsonl")
			err := os.WriteFile(filePath, []byte(tt.content), 0o644)
			require.NoError(t, err)

			got, err := parseSessionFile("test-session", filePath)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
