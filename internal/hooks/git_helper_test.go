package hooks

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewGitHelper(t *testing.T) {
	helper := NewGitHelper()
	assert.NotNil(t, helper)
	assert.IsType(t, &realGitHelper{}, helper)
}
