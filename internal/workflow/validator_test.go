package workflow

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateWorkflowName(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid simple name",
			input:   "auth-feature",
			wantErr: false,
		},
		{
			name:    "valid name with numbers",
			input:   "feature-123",
			wantErr: false,
		},
		{
			name:    "valid name all lowercase",
			input:   "myfeature",
			wantErr: false,
		},
		{
			name:    "valid name all uppercase",
			input:   "MYFEATURE",
			wantErr: false,
		},
		{
			name:    "valid name mixed case",
			input:   "MyFeature",
			wantErr: false,
		},
		{
			name:    "valid name with multiple hyphens",
			input:   "my-auth-feature",
			wantErr: false,
		},
		{
			name:    "valid single character",
			input:   "a",
			wantErr: false,
		},
		{
			name:    "valid two characters",
			input:   "ab",
			wantErr: false,
		},
		{
			name:        "empty name",
			input:       "",
			wantErr:     true,
			errContains: "cannot be empty",
		},
		{
			name:        "name too long",
			input:       strings.Repeat("a", 65),
			wantErr:     true,
			errContains: "too long",
		},
		{
			name:        "name starting with hyphen",
			input:       "-feature",
			wantErr:     true,
			errContains: "cannot start or end with hyphen",
		},
		{
			name:        "name ending with hyphen",
			input:       "feature-",
			wantErr:     true,
			errContains: "cannot start or end with hyphen",
		},
		{
			name:        "name with path traversal",
			input:       "../feature",
			wantErr:     true,
			errContains: "path traversal",
		},
		{
			name:        "name with forward slash",
			input:       "auth/feature",
			wantErr:     true,
			errContains: "path traversal",
		},
		{
			name:        "name with backslash",
			input:       "auth\\feature",
			wantErr:     true,
			errContains: "path traversal",
		},
		{
			name:        "name with special characters",
			input:       "auth_feature",
			wantErr:     true,
			errContains: "alphanumeric",
		},
		{
			name:        "name with spaces",
			input:       "auth feature",
			wantErr:     true,
			errContains: "alphanumeric",
		},
		{
			name:        "name with dot",
			input:       "auth.feature",
			wantErr:     true,
			errContains: "alphanumeric",
		},
		{
			name:    "maximum valid length",
			input:   strings.Repeat("a", 64),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWorkflowName(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.True(t, errors.Is(err, ErrInvalidWorkflowName))
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestValidateWorkflowType(t *testing.T) {
	tests := []struct {
		name        string
		input       WorkflowType
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid feature type",
			input:   WorkflowTypeFeature,
			wantErr: false,
		},
		{
			name:    "valid fix type",
			input:   WorkflowTypeFix,
			wantErr: false,
		},
		{
			name:        "invalid type",
			input:       WorkflowType("invalid"),
			wantErr:     true,
			errContains: "must be 'feature' or 'fix'",
		},
		{
			name:        "empty type",
			input:       WorkflowType(""),
			wantErr:     true,
			errContains: "must be 'feature' or 'fix'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWorkflowType(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				assert.True(t, errors.Is(err, ErrInvalidWorkflowType))
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestValidateDescription(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantErr     bool
		errContains string
	}{
		{
			name:    "valid short description",
			input:   "add authentication",
			wantErr: false,
		},
		{
			name:    "valid long description",
			input:   strings.Repeat("a", 1000),
			wantErr: false,
		},
		{
			name:    "valid description with special characters",
			input:   "add JWT authentication with @#$%^& symbols",
			wantErr: false,
		},
		{
			name:        "empty description",
			input:       "",
			wantErr:     true,
			errContains: "cannot be empty",
		},
		{
			name:        "description too long",
			input:       strings.Repeat("a", 1001),
			wantErr:     true,
			errContains: "too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDescription(tt.input)

			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errContains)
				return
			}

			require.NoError(t, err)
		})
	}
}
