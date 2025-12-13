package workflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetMaxDescriptionLength(t *testing.T) {
	tests := []struct {
		name   string
		envVal string
		setEnv bool
		want   int
	}{
		{
			name:   "returns default when env var not set",
			setEnv: false,
			want:   DefaultMaxDescriptionLength,
		},
		{
			name:   "returns valid env var override",
			envVal: "65536",
			setEnv: true,
			want:   65536,
		},
		{
			name:   "returns default for non-numeric env var",
			envVal: "invalid",
			setEnv: true,
			want:   DefaultMaxDescriptionLength,
		},
		{
			name:   "returns default for zero env var",
			envVal: "0",
			setEnv: true,
			want:   DefaultMaxDescriptionLength,
		},
		{
			name:   "returns default for negative env var",
			envVal: "-100",
			setEnv: true,
			want:   DefaultMaxDescriptionLength,
		},
		{
			name:   "returns default for empty string env var",
			envVal: "",
			setEnv: true,
			want:   DefaultMaxDescriptionLength,
		},
		{
			name:   "returns valid small positive value",
			envVal: "1024",
			setEnv: true,
			want:   1024,
		},
		{
			name:   "returns valid large positive value",
			envVal: "1048576",
			setEnv: true,
			want:   1048576,
		},
		{
			name:   "returns default for floating point value",
			envVal: "123.45",
			setEnv: true,
			want:   DefaultMaxDescriptionLength,
		},
		{
			name:   "returns default for value with whitespace",
			envVal: " 1000 ",
			setEnv: true,
			want:   DefaultMaxDescriptionLength,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv {
				t.Setenv(EnvMaxDescriptionLength, tt.envVal)
			}

			got := GetMaxDescriptionLength()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConstants(t *testing.T) {
	tests := []struct {
		name string
		got  interface{}
		want interface{}
	}{
		{
			name: "DefaultMaxDescriptionLength is 32768",
			got:  DefaultMaxDescriptionLength,
			want: 32768,
		},
		{
			name: "MinDescriptionLength is 1",
			got:  MinDescriptionLength,
			want: 1,
		},
		{
			name: "EnvMaxDescriptionLength is correct",
			got:  EnvMaxDescriptionLength,
			want: "CLAUDE_WORKFLOW_MAX_DESCRIPTION_LENGTH",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.got)
		})
	}
}
