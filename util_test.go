package aferodog

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMustNoError(t *testing.T) {
	t.Parallel()

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		assert.Panics(t, func() {
			mustNoError(errors.New("error"))
		})
	})

	t.Run("no error", func(t *testing.T) {
		t.Parallel()

		assert.NotPanics(t, func() {
			mustNoError(nil)
		})
	})
}
