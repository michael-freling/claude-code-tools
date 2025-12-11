package hooks

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewGhHelper(t *testing.T) {
	helper := NewGhHelper()
	assert.NotNil(t, helper)
	assert.IsType(t, &realGhHelper{}, helper)
}
