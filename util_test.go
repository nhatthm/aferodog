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

func TestFileContentRegexp(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		scenario string
		content  string
		expected string
	}{
		{
			scenario: "plain text",
			content:  "hello world",
			expected: "hello world",
		},
		{
			scenario: "regexp",
			content: `
[config]
    uuid = "<regexp:\b[0-9a-f]{8}\b-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-\b[0-9a-f]{12}\b/>"
    salt = "<regexp:^[a-f0-9]{32}$/>"
`,
			expected: `
\[config\]
    uuid = "\b[0-9a-f]{8}\b-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-\b[0-9a-f]{12}\b"
    salt = "^[a-f0-9]{32}$"
`,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()

			actual := fileContentRegexp(tc.content)

			assert.Equal(t, tc.expected, actual)
		})
	}
}

func TestFileContentRegexp_Match(t *testing.T) {
	t.Parallel()

	expected := `
[config]
    uuid = "<regexp:\b[0-9a-f]{8}\b-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-\b[0-9a-f]{12}\b/>"
    salt = "<regexp:[a-f0-9]{32}/>"
`
	content := `
[config]
    uuid = "3223cbf8-292f-4829-9a85-9d9f9de675aa"
    salt = "5d41402abc4b2a76b9719d911017c592"
`

	assert.Regexp(t, fileContentRegexp(expected), content)
}
