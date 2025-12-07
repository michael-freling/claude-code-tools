package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWorkflowError_Error(t *testing.T) {
	tests := []struct {
		name string
		err  *WorkflowError
		want string
	}{
		{
			name: "returns message",
			err: &WorkflowError{
				Message: "test error message",
			},
			want: "test error message",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			assert.Equal(t, tt.want, got)
		})
	}
}
