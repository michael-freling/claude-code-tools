package workflow

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSessionManager(t *testing.T) {
	tests := []struct {
		name   string
		logger Logger
	}{
		{
			name:   "creates session manager with logger",
			logger: NewLogger(LogLevelNormal),
		},
		{
			name:   "creates session manager without logger",
			logger: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewSessionManager(tt.logger)
			require.NotNil(t, got)
			assert.Equal(t, tt.logger, got.logger)
		})
	}
}

func TestSessionManager_ParseSessionID(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   string
	}{
		{
			name:   "extracts session ID from result chunk",
			output: `{"type":"result","session_id":"abc123","result":"done"}`,
			want:   "abc123",
		},
		{
			name:   "extracts session ID from system init chunk",
			output: `{"type":"system","subtype":"init","session_id":"xyz789"}`,
			want:   "xyz789",
		},
		{
			name: "extracts session ID from multiline stream-json output",
			output: `{"type":"assistant","message":{"content":[{"type":"text","text":"Hello"}]}}
{"type":"system","subtype":"init","session_id":"multi123"}
{"type":"result","result":"done"}`,
			want: "multi123",
		},
		{
			name:   "extracts session ID from first valid chunk when multiple exist",
			output: `{"type":"result","session_id":"first123"}` + "\n" + `{"type":"result","session_id":"second456"}`,
			want:   "first123",
		},
		{
			name:   "returns empty string when no session ID in output",
			output: `{"type":"result","result":"done"}`,
			want:   "",
		},
		{
			name:   "returns empty string for empty output",
			output: "",
			want:   "",
		},
		{
			name:   "handles malformed JSON lines gracefully",
			output: `{invalid json}` + "\n" + `{"type":"result","session_id":"valid123"}`,
			want:   "valid123",
		},
		{
			name:   "extracts session ID using regex fallback - quoted pattern",
			output: `Some text here "session_id":"regex123" more text`,
			want:   "regex123",
		},
		{
			name:   "extracts session ID using regex fallback - unquoted pattern",
			output: `session_id: unquoted456`,
			want:   "unquoted456",
		},
		{
			name:   "extracts UUID format session ID",
			output: `{"type":"result","session_id":"550e8400-e29b-41d4-a716-446655440000"}`,
			want:   "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name:   "ignores empty session_id field",
			output: `{"type":"result","session_id":""}`,
			want:   "",
		},
		{
			name: "extracts from complex nested structure",
			output: `{"type":"assistant","message":{"content":[]}}
{"type":"system","subtype":"init","session_id":"nested789","other":"data"}
{"type":"user","message":"test"}`,
			want: "nested789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewSessionManager(nil)
			got := m.ParseSessionID(tt.output)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSessionManager_BuildCommandArgs(t *testing.T) {
	tests := []struct {
		name            string
		sessionID       string
		forceNewSession bool
		want            []string
	}{
		{
			name:            "returns resume args with valid session ID",
			sessionID:       "session123",
			forceNewSession: false,
			want:            []string{"--resume", "session123"},
		},
		{
			name:            "returns nil when session ID is empty",
			sessionID:       "",
			forceNewSession: false,
			want:            nil,
		},
		{
			name:            "returns nil when forceNewSession is true",
			sessionID:       "session123",
			forceNewSession: true,
			want:            nil,
		},
		{
			name:            "returns nil when both empty session and force new",
			sessionID:       "",
			forceNewSession: true,
			want:            nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewSessionManager(nil)
			got := m.BuildCommandArgs(tt.sessionID, tt.forceNewSession)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSessionManager_GetSessionFromState(t *testing.T) {
	sessionID := "test-session"
	createdAt := time.Now()

	tests := []struct {
		name  string
		state *WorkflowState
		want  *SessionInfo
	}{
		{
			name: "returns session info when session exists",
			state: &WorkflowState{
				SessionID:         &sessionID,
				SessionCreatedAt:  &createdAt,
				SessionReuseCount: 3,
			},
			want: &SessionInfo{
				SessionID:  "test-session",
				CreatedAt:  createdAt,
				ReuseCount: 3,
				IsNew:      false,
			},
		},
		{
			name:  "returns nil when state is nil",
			state: nil,
			want:  nil,
		},
		{
			name:  "returns nil when session ID is nil",
			state: &WorkflowState{},
			want:  nil,
		},
		{
			name: "returns nil when session ID is empty string",
			state: &WorkflowState{
				SessionID: new(string), // empty string pointer
			},
			want: nil,
		},
		{
			name: "returns session info without created time",
			state: &WorkflowState{
				SessionID:         &sessionID,
				SessionReuseCount: 1,
			},
			want: &SessionInfo{
				SessionID:  "test-session",
				CreatedAt:  time.Time{},
				ReuseCount: 1,
				IsNew:      false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewSessionManager(nil)
			got := m.GetSessionFromState(tt.state)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSessionManager_UpdateStateWithSession(t *testing.T) {
	tests := []struct {
		name             string
		initialState     *WorkflowState
		sessionID        string
		isNew            bool
		wantSessionID    *string
		wantReuseCount   int
		wantCreatedAtSet bool
	}{
		{
			name:             "creates new session in state",
			initialState:     &WorkflowState{},
			sessionID:        "new-session",
			isNew:            true,
			wantSessionID:    stringPtr("new-session"),
			wantReuseCount:   0,
			wantCreatedAtSet: true,
		},
		{
			name: "reuses existing session and increments count",
			initialState: &WorkflowState{
				SessionID:         stringPtr("existing-session"),
				SessionReuseCount: 2,
			},
			sessionID:        "existing-session",
			isNew:            false,
			wantSessionID:    stringPtr("existing-session"),
			wantReuseCount:   3,
			wantCreatedAtSet: false,
		},
		{
			name:             "does nothing when state is nil",
			initialState:     nil,
			sessionID:        "test-session",
			isNew:            true,
			wantSessionID:    nil,
			wantReuseCount:   0,
			wantCreatedAtSet: false,
		},
		{
			name:             "does nothing when session ID is empty",
			initialState:     &WorkflowState{},
			sessionID:        "",
			isNew:            true,
			wantSessionID:    nil,
			wantReuseCount:   0,
			wantCreatedAtSet: false,
		},
		{
			name: "updates session ID when reusing with different ID",
			initialState: &WorkflowState{
				SessionID:         stringPtr("old-session"),
				SessionReuseCount: 1,
			},
			sessionID:        "new-session",
			isNew:            false,
			wantSessionID:    stringPtr("new-session"),
			wantReuseCount:   2,
			wantCreatedAtSet: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewSessionManager(nil)
			m.UpdateStateWithSession(tt.initialState, tt.sessionID, tt.isNew)

			if tt.initialState == nil {
				return
			}

			assert.Equal(t, tt.wantSessionID, tt.initialState.SessionID)
			assert.Equal(t, tt.wantReuseCount, tt.initialState.SessionReuseCount)

			if tt.wantCreatedAtSet {
				require.NotNil(t, tt.initialState.SessionCreatedAt)
			}
		})
	}
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}
