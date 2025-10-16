package resilience

import (
"testing"

"github.com/stretchr/testify/assert"
)

func TestModule(t *testing.T) {
	t.Run("module is defined", func(t *testing.T) {
assert.NotNil(t, Module)
})
}
