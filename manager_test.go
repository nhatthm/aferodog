package aferodog

import (
	"errors"
	"io"
	"os"
	"testing"

	"github.com/cucumber/godog"
	"github.com/nhatthm/aferomock"
	"github.com/spf13/afero"
	"github.com/spf13/afero/mem"
	"github.com/stretchr/testify/assert"
)

func newManager(t *testing.T, mockFs aferomock.FsMocker) *Manager {
	t.Helper()

	fs := afero.NewOsFs()

	if mockFs != nil {
		fs = mockFs(t)
	}

	return NewManager(WithDefaultFs(fs))
}

func TestManager_Track(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		scenario       string
		mockFs         aferomock.FsMocker
		path           string
		expectedResult []string
		expectedError  string
	}{
		{
			scenario: "could not stat file",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("Stat", ".github").
					Return(nil, errors.New("stat error"))
			}),
			path:           ".github/workflows/unknown.yaml",
			expectedResult: []string{},
			expectedError:  `could not stat(".github"): stat error`,
		},
		{
			scenario:       "file exists",
			path:           ".github/workflows/test.yaml",
			expectedResult: []string{},
		},
		{
			scenario:       "file does not exist",
			path:           ".github/workflows/unknown.yaml",
			expectedResult: []string{".github/workflows/unknown.yaml"},
		},
		{
			scenario:       "parent does not exist",
			path:           ".github/unknown/test.yaml",
			expectedResult: []string{".github/unknown"},
		},
		{
			scenario:       "parent does not exist (level 3)",
			path:           ".github/workflows/level3/test.yaml",
			expectedResult: []string{".github/workflows/level3"},
		},
		{
			scenario:       "parent does not exist (level 2)",
			path:           ".github/level2/test.yaml",
			expectedResult: []string{".github/level2"},
		},
		{
			scenario:       "level 1 does not exist",
			path:           "level1",
			expectedResult: []string{"level1"},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()

			m := newManager(t, tc.mockFs)
			err := m.track(defaultFs, tc.path)

			assert.Equal(t, tc.expectedResult, m.trackedFiles[defaultFs])

			if tc.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expectedError)
			}
		})
	}
}

func TestManager_Chmod(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		scenario      string
		mockFs        aferomock.FsMocker
		path          string
		perm          string
		expectedError string
	}{
		{
			scenario:      "file not exist",
			path:          "unknown",
			perm:          "0755",
			expectedError: "chmod unknown: no such file or directory",
		},
		{
			scenario:      "wrong perm",
			perm:          "perm",
			expectedError: "strconv.ParseUint: parsing \"perm\": invalid syntax",
		},
		{
			scenario: "success",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("Chmod", "unknown", os.FileMode(0o755)).
					Return(nil)
			}),
			path: "unknown",
			perm: "0755",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()

			m := newManager(t, tc.mockFs)
			err := m.chmod(tc.path, tc.perm)

			if tc.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expectedError)
			}
		})
	}
}

func TestManager_RemoveFile(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		scenario      string
		mockFs        aferomock.FsMocker
		path          string
		expectedError string
	}{
		{
			scenario: "file not exist",
			path:     "unknown",
		},
		{
			scenario: "could not remove",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("RemoveAll", "unknown").
					Return(errors.New("remove error"))
			}),
			path:          "unknown",
			expectedError: "remove error",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()

			m := newManager(t, tc.mockFs)
			err := m.removeFile(tc.path)

			if tc.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expectedError)
			}
		})
	}
}

func TestManager_CreateFile(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		scenario      string
		mockFs        aferomock.FsMocker
		expectedError string
	}{
		{
			scenario: "could not track directory",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("Stat", "level1").
					Return(nil, errors.New("stat error"))
			}),
			expectedError: "could not track file: could not stat(\"level1\"): stat error",
		},
		{
			scenario: "could not create parent directory",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("Stat", "level1").
					Return(nil, os.ErrNotExist)

				fs.On("MkdirAll", "level1/level2", os.FileMode(0o755)).
					Return(errors.New("mkdir error"))
			}),
			expectedError: "could not mkdir \"level1/level2\": mkdir error",
		},
		{
			scenario: "could not create file",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("Stat", "level1").
					Return(nil, os.ErrNotExist)

				fs.On("MkdirAll", "level1/level2", os.FileMode(0o755)).
					Return(nil)

				fs.On("Create", "level1/level2/file.txt").
					Return(nil, errors.New("create error"))
			}),
			expectedError: "could not create \"level1/level2/file.txt\": create error",
		},
		{
			scenario: "success",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("Stat", "level1").
					Return(nil, os.ErrNotExist)

				fs.On("MkdirAll", "level1/level2", os.FileMode(0o755)).
					Return(nil)

				fs.On("Create", "level1/level2/file.txt").
					Return(mem.NewFileHandle(mem.CreateFile("file.txt")), nil)
			}),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()

			m := newManager(t, tc.mockFs)
			err := m.createFile("level1/level2/file.txt")

			if tc.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expectedError)
			}
		})
	}
}

func TestManager_CreateFileWithContent(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		scenario      string
		mockFs        aferomock.FsMocker
		expectedError string
	}{
		{
			scenario: "could not track directory",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("Stat", "level1").
					Return(nil, errors.New("stat error"))
			}),
			expectedError: "could not track file: could not stat(\"level1\"): stat error",
		},
		{
			scenario: "could not create parent directory",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("Stat", "level1").
					Return(nil, os.ErrNotExist)

				fs.On("MkdirAll", "level1/level2", os.FileMode(0o755)).
					Return(errors.New("mkdir error"))
			}),
			expectedError: "could not mkdir \"level1/level2\": mkdir error",
		},
		{
			scenario: "could not create file",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("Stat", "level1").
					Return(nil, os.ErrNotExist)

				fs.On("MkdirAll", "level1/level2", os.FileMode(0o755)).
					Return(nil)

				fs.On("Create", "level1/level2/file.txt").
					Return(nil, errors.New("create error"))
			}),
			expectedError: "could not create \"level1/level2/file.txt\": create error",
		},
		{
			scenario: "could not write file",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("Stat", "level1").
					Return(nil, os.ErrNotExist)

				fs.On("MkdirAll", "level1/level2", os.FileMode(0o755)).
					Return(nil)

				f := mem.NewFileHandle(mem.CreateFile("file.txt"))
				_ = f.Close() // nolint: errcheck

				fs.On("Create", "level1/level2/file.txt").
					Return(f, nil)
			}),
			expectedError: "could not write file \"level1/level2/file.txt\": File is closed",
		},
		{
			scenario: "success",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("Stat", "level1").
					Return(nil, os.ErrNotExist)

				fs.On("MkdirAll", "level1/level2", os.FileMode(0o755)).
					Return(nil)

				f := mem.NewFileHandle(mem.CreateFile("file.txt"))

				fs.On("Create", "level1/level2/file.txt").
					Return(f, nil)
			}),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()

			m := newManager(t, tc.mockFs)
			err := m.createFileWithContent("level1/level2/file.txt", &godog.DocString{Content: "hello world!"})

			if tc.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expectedError)
			}
		})
	}
}

func TestManager_CreateDirectory(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		scenario      string
		mockFs        aferomock.FsMocker
		expectedError string
	}{
		{
			scenario: "could not track directory",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("Stat", "level1").
					Return(nil, errors.New("stat error"))
			}),
			expectedError: "could not track directory: could not stat(\"level1\"): stat error",
		},
		{
			scenario: "could not create directory",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("Stat", "level1").
					Return(aferomock.NewFileInfo(), nil)

				fs.On("Stat", "level1/level2").
					Return(aferomock.NewFileInfo(), nil)

				fs.On("Stat", "level1/level2/level3").
					Return(nil, os.ErrNotExist)

				fs.On("MkdirAll", "level1/level2/level3", os.FileMode(0o755)).
					Return(errors.New("mkdir error"))
			}),
			expectedError: "could not mkdir \"level1/level2/level3\": mkdir error",
		},
		{
			scenario: "success",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("Stat", "level1").
					Return(nil, os.ErrNotExist)

				fs.On("MkdirAll", "level1/level2/level3", os.FileMode(0o755)).
					Return(nil)
			}),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()

			m := newManager(t, tc.mockFs)
			err := m.createDirectory("level1/level2/level3")

			if tc.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expectedError)
			}
		})
	}
}

func TestManager_AssertFileExists(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		scenario      string
		mockFs        aferomock.FsMocker
		expectedError bool
	}{
		{
			scenario: "stat error",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("Stat", "level1/file.txt").
					Return(nil, errors.New("stat error"))
			}),
			expectedError: true,
		},
		{
			scenario: "file not found",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("Stat", "level1/file.txt").
					Return(nil, os.ErrNotExist)
			}),
			expectedError: true,
		},
		{
			scenario: "no error",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("Stat", "level1/file.txt").
					Return(aferomock.NewFileInfo(func(i *aferomock.FileInfo) {
						i.On("IsDir").Return(false)
					}), nil)
			}),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()

			m := newManager(t, tc.mockFs)
			err := m.assertFileExists("level1/file.txt")

			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestManager_AssertDirExists(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		scenario      string
		mockFs        aferomock.FsMocker
		expectedError bool
	}{
		{
			scenario: "stat error",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("Stat", "level1").
					Return(nil, errors.New("stat error"))
			}),
			expectedError: true,
		},
		{
			scenario: "file not found",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("Stat", "level1").
					Return(nil, os.ErrNotExist)
			}),
			expectedError: true,
		},
		{
			scenario: "no error",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("Stat", "level1").
					Return(aferomock.NewFileInfo(func(i *aferomock.FileInfo) {
						i.On("IsDir").Return(true)
					}), nil)
			}),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()

			m := newManager(t, tc.mockFs)
			err := m.assertDirectoryExists("level1")

			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestManager_AssertFileContent(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		scenario      string
		mockFs        aferomock.FsMocker
		expectedError bool
	}{
		{
			scenario: "stat error",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("Stat", "level1/file.txt").
					Return(nil, errors.New("stat error"))
			}),
			expectedError: true,
		},
		{
			scenario: "file not found",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("Stat", "level1/file.txt").
					Return(nil, os.ErrNotExist)
			}),
			expectedError: true,
		},
		{
			scenario: "different content",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("Stat", "level1/file.txt").
					Return(aferomock.NewFileInfo(func(i *aferomock.FileInfo) {
						i.On("IsDir").Return(false)
					}), nil)

				fs.On("Open", "level1/file.txt").
					Return(mem.NewFileHandle(mem.CreateFile("file.txt")), nil)
			}),
			expectedError: true,
		},
		{
			scenario: "success",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("Stat", "level1/file.txt").
					Return(aferomock.NewFileInfo(func(i *aferomock.FileInfo) {
						i.On("IsDir").Return(false)
					}), nil)

				f := mem.NewFileHandle(mem.CreateFile("file.txt"))
				_, _ = f.WriteString("hello world") // nolint: errcheck
				_, _ = f.Seek(0, io.SeekStart)      // nolint: errcheck

				fs.On("Open", "level1/file.txt").
					Return(f, nil)
			}),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()

			m := newManager(t, tc.mockFs)
			err := m.assertFileContent("level1/file.txt", &godog.DocString{Content: "hello world"})

			if tc.expectedError {
				t.Log(err)
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestManager_AssertFileContentRegexp(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		scenario      string
		mockFs        aferomock.FsMocker
		expectedError bool
	}{
		{
			scenario: "stat error",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("Stat", "level1/file.txt").
					Return(nil, errors.New("stat error"))
			}),
			expectedError: true,
		},
		{
			scenario: "file not found",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("Stat", "level1/file.txt").
					Return(nil, os.ErrNotExist)
			}),
			expectedError: true,
		},
		{
			scenario: "different content",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("Stat", "level1/file.txt").
					Return(aferomock.NewFileInfo(func(i *aferomock.FileInfo) {
						i.On("IsDir").Return(false)
					}), nil)

				fs.On("Open", "level1/file.txt").
					Return(mem.NewFileHandle(mem.CreateFile("file.txt")), nil)
			}),
			expectedError: true,
		},
		{
			scenario: "success",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("Stat", "level1/file.txt").
					Return(aferomock.NewFileInfo(func(i *aferomock.FileInfo) {
						i.On("IsDir").Return(false)
					}), nil)

				f := mem.NewFileHandle(mem.CreateFile("file.txt"))
				_, _ = f.WriteString("hello world") // nolint: errcheck
				_, _ = f.Seek(0, io.SeekStart)      // nolint: errcheck

				fs.On("Open", "level1/file.txt").
					Return(f, nil)
			}),
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()

			m := newManager(t, tc.mockFs)
			err := m.assertFileContentRegexp("level1/file.txt", &godog.DocString{Content: "hello <regexp:[a-z]+/>"})

			if tc.expectedError {
				t.Log(err)
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestManager_AssertFilePerm(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		scenario      string
		mockFs        aferomock.FsMocker
		perm          string
		expectedError bool
	}{
		{
			scenario:      "invalid perm",
			perm:          "invalid",
			expectedError: true,
		},
		{
			scenario: "stat error",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("Stat", "level1/file.txt").
					Return(nil, errors.New("stat error"))
			}),
			perm:          "0755",
			expectedError: true,
		},
		{
			scenario: "file not found",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("Stat", "level1/file.txt").
					Return(nil, os.ErrNotExist)
			}),
			perm:          "0755",
			expectedError: true,
		},
		{
			scenario: "different perm",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("Stat", "level1/file.txt").
					Return(aferomock.NewFileInfo(func(i *aferomock.FileInfo) {
						i.On("Mode").Return(os.FileMode(0o644))
					}), nil)
			}),
			perm:          "0755",
			expectedError: true,
		},
		{
			scenario: "success",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("Stat", "level1/file.txt").
					Return(aferomock.NewFileInfo(func(i *aferomock.FileInfo) {
						i.On("Mode").Return(os.FileMode(0o755))
					}), nil)
			}),
			perm: "0755",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()

			m := newManager(t, tc.mockFs)
			err := m.assertFilePerm("level1/file.txt", tc.perm)

			if tc.expectedError {
				t.Log(err)
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestManager_AssertFileTreeEqual(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		scenario      string
		mockFs        aferomock.FsMocker
		expectedError bool
	}{
		{
			scenario: "stat error",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("Stat", ".github").
					Return(nil, errors.New("stat error"))
			}),
			expectedError: true,
		},
		{
			scenario: "success",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()

			expected := `
- workflows:
    - golangci-lint.yaml
    - test.yaml
`

			m := newManager(t, tc.mockFs)
			err := m.assertTreeEqualInPath(".github", &godog.DocString{Content: expected})

			if tc.expectedError {
				t.Log(err)
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestManager_AssertFileTreeContains(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		scenario      string
		mockFs        aferomock.FsMocker
		expectedError bool
	}{
		{
			scenario: "stat error",
			mockFs: aferomock.MockFs(func(fs *aferomock.Fs) {
				fs.On("Stat", ".github").
					Return(nil, errors.New("stat error"))
			}),
			expectedError: true,
		},
		{
			scenario: "success",
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()

			expected := `
- workflows:
    - golangci-lint.yaml
    - test.yaml
`

			m := newManager(t, tc.mockFs)
			err := m.assertTreeContainsInPath(".github", &godog.DocString{Content: expected})

			if tc.expectedError {
				t.Log(err)
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
