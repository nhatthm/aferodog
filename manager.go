package aferodog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/cucumber/godog"
	"github.com/cucumber/messages-go/v10"
	"github.com/nhatthm/aferoassert"
	"github.com/spf13/afero"
)

const defaultFs = "_default"

// TempDirer creates a temp dir evey time it is called.
type TempDirer interface {
	TempDir() string
}

// Option is to configure Manager.
type Option func(m *Manager)

// Manager manages a list of file systems and provides steps for godog.
type Manager struct {
	td TempDirer

	fss          map[string]afero.Fs
	testDir      string
	trackedFiles map[string][]string

	mu sync.Mutex
}

// RegisterContext registers all the steps.
func (m *Manager) RegisterContext(td TempDirer, ctx *godog.ScenarioContext) {
	ctx.BeforeScenario(func(*godog.Scenario) {
		m.WithTempDirer(td)
		_ = m.resetDir() // nolint: errcheck
	})

	ctx.AfterScenario(func(*godog.Scenario, error) {
		m.cleanup()
		_ = m.resetDir() // nolint: errcheck
	})

	ctx.BeforeStep(m.expandVariables)

	// Utils.
	ctx.Step(`(?:current|working) directory is temporary`, m.chTempDir)
	ctx.Step(`(?:current|working) directory is "([^"]+)"`, m.chDir)
	ctx.Step(`changes? (?:current|working) directory to "([^"]+)"`, m.chDir)
	ctx.Step(`resets? (?:current|working) directory`, m.resetDir)

	// Default FS.
	ctx.Step(`^there is no (?:file|directory) "([^"]+)"$`, m.removeFile)
	ctx.Step(`^there is a file "([^"]+)"$`, m.createFile)
	ctx.Step(`^there is a directory "([^"]+)"$`, m.createDirectory)
	ctx.Step(`^there is a file "([^"]+)" with content:`, m.createFileWithContent)
	ctx.Step(`changes? "([^"]+)" permission to ([0-9]+)$`, m.chmod)
	ctx.Step(`^(?:file|directory) "([^"]+)" permission is ([0-9]+)$`, m.chmod)

	ctx.Step(`^there should be a file "([^"]+)"$`, m.assertFileExists)
	ctx.Step(`^there should be a directory "([^"]+)"$`, m.assertDirectoryExists)
	ctx.Step(`^there should be a file "([^"]+)" with content:`, m.assertFileContent)
	ctx.Step(`^there should be a file "([^"]+)" with content matches:`, m.assertFileContentRegexp)
	ctx.Step(`^(?:file|directory) "([^"]+)" permission should be ([0-9]+)$`, m.assertFilePerm)
	ctx.Step(`^there should be only these files:`, m.assertTreeEqual)
	ctx.Step(`^there should be these files:`, m.assertTreeContains)
	ctx.Step(`^there should be only these files in "([^"]+)":`, m.assertTreeEqualInPath)
	ctx.Step(`^there should be these files in "([^"]+)":`, m.assertTreeContainsInPath)

	// Another FS.
	ctx.Step(`^there is no (?:file|directory) "([^"]+)" in "([^"]+)" (?:fs|filesystem|file system)$`, m.removeFileInFs)
	ctx.Step(`^there is a file "([^"]+)" in "([^"]+)" (?:fs|filesystem|file system)$`, m.createFileInFs)
	ctx.Step(`^there is a directory "([^"]+)" in "([^"]+)" (?:fs|filesystem|file system)$`, m.createDirectoryInFs)
	ctx.Step(`^there is a file "([^"]+)" in "([^"]+)" (?:fs|filesystem|file system) with content:`, m.createFileInFsWithContent)
	ctx.Step(`changes? "([^"]+)" permission in "([^"]+)" (?:fs|filesystem|file system) to ([0-9]+)$`, m.chmodInFs)
	ctx.Step(`^(?:file|directory) "([^"]+)" permission in "([^"]+)" (?:fs|filesystem|file system) is ([0-9]+)$`, m.chmodInFs)

	ctx.Step(`^there should be a file "([^"]+)" in "([^"]+)" (?:fs|filesystem|file system)$`, m.assertFileExistsInFs)
	ctx.Step(`^there should be a directory "([^"]+)" in "([^"]+)" (?:fs|filesystem|file system)$`, m.assertDirectoryExistsInFs)
	ctx.Step(`^there should be a file "([^"]+)" in "([^"]+)" (?:fs|filesystem|file system) with content:`, m.assertFileContentInFs)
	ctx.Step(`^there should be a file "([^"]+)" in "([^"]+)" (?:fs|filesystem|file system) with content matches:`, m.assertFileContentRegexpInFs)
	ctx.Step(`^(?:file|directory) "([^"]+)" permission in "([^"]+)" (?:fs|filesystem|file system) should be ([0-9]+)$`, m.assertFilePermInFs)
	ctx.Step(`^there should be only these files in "([^"]+)" (?:fs|filesystem|file system):`, m.assertTreeEqualInFs)
	ctx.Step(`^there should be these files in "([^"]+)" (?:fs|filesystem|file system):`, m.assertTreeContainsInFs)
	ctx.Step(`^there should be only these files in "([^"]+)" in "([^"]+)" (?:fs|filesystem|file system):`, m.assertTreeEqualInPathInFs)
	ctx.Step(`^there should be these files in "([^"]+)" in "([^"]+)" (?:fs|filesystem|file system):`, m.assertTreeContainsInPathInFs)
}

// WithTempDirer sets the TempDirer.
func (m *Manager) WithTempDirer(td TempDirer) *Manager {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.td = td

	return m
}

func (m *Manager) cleanup() {
	for id, files := range m.trackedFiles {
		fs := m.fs(id)

		for _, f := range files {
			_ = fs.RemoveAll(f) // nolint: errcheck
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.trackedFiles = make(map[string][]string)
}

func (m *Manager) expandVariables(st *godog.Step) {
	cwd, err := os.Getwd()
	mustNoError(err)

	replacer := strings.NewReplacer(
		"$TEST_DIR", m.testDir,
		"$CWD", cwd,
		"$WORKING_DIR", cwd,
	)

	st.Text = replacer.Replace(st.Text)

	if st.Argument == nil {
		return
	}

	if msg, ok := st.Argument.Message.(*messages.PickleStepArgument_DocString); ok {
		msg.DocString.Content = replacer.Replace(msg.DocString.Content)
	}
}

func (m *Manager) trackPath(fs afero.Fs, path string) (string, error) {
	parent := filepath.Dir(path)

	if parent != "." {
		track, err := m.trackPath(fs, parent)
		if err != nil {
			return "", err
		}

		if track != "" {
			return track, nil
		}
	}

	if _, err := fs.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return path, nil
		}

		return "", fmt.Errorf("could not stat(%q): %w", path, err)
	}

	return "", nil
}

func (m *Manager) track(fs string, path string) error {
	if _, ok := m.trackedFiles[fs]; !ok {
		m.trackedFiles[fs] = make([]string, 0)
	}

	path, err := m.trackPath(m.fs(fs), filepath.Clean(path))
	if err != nil {
		return err
	}

	if path == "" {
		return nil
	}

	m.trackedFiles[fs] = append(m.trackedFiles[fs], path)

	return nil
}

func (m *Manager) fs(name string) afero.Fs {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.fss[name]
}

func (m *Manager) chTempDir() error {
	// TempDir will be deleted automatically, we don't need to track it manually.
	return m.chDir(m.td.TempDir())
}

func (m *Manager) chDir(dir string) error {
	return os.Chdir(dir)
}

func (m *Manager) resetDir() error {
	return m.chDir(m.testDir)
}

func (m *Manager) chmod(path, perm string) error {
	return m.chmodInFs(path, defaultFs, perm)
}

func (m *Manager) removeFile(path string) error {
	return m.removeFileInFs(path, defaultFs)
}

func (m *Manager) createFile(path string) error {
	return m.createFileInFs(path, defaultFs)
}

func (m *Manager) createDirectory(path string) error {
	return m.createDirectoryInFs(path, defaultFs)
}

func (m *Manager) createFileWithContent(path string, body *godog.DocString) error {
	return m.createFileInFsWithContent(path, defaultFs, body)
}

func (m *Manager) assertFileExists(path string) error {
	return m.assertFileExistsInFs(path, defaultFs)
}

func (m *Manager) assertDirectoryExists(path string) error {
	return m.assertDirectoryExistsInFs(path, defaultFs)
}

func (m *Manager) assertFileContent(path string, body *godog.DocString) error {
	return m.assertFileContentInFs(path, defaultFs, body)
}

func (m *Manager) assertFileContentRegexp(path string, body *godog.DocString) error {
	return m.assertFileContentRegexpInFs(path, defaultFs, body)
}

func (m *Manager) assertFilePerm(path string, perm string) error {
	return m.assertFilePermInFs(path, defaultFs, perm)
}

func (m *Manager) assertTreeEqual(body *godog.DocString) error {
	return m.assertTreeEqualInFs(defaultFs, body)
}

func (m *Manager) assertTreeEqualInPath(path string, body *godog.DocString) error {
	return m.assertTreeEqualInPathInFs(path, defaultFs, body)
}

func (m *Manager) assertTreeContains(body *godog.DocString) error {
	return m.assertTreeContainsInFs(defaultFs, body)
}

func (m *Manager) assertTreeContainsInPath(path string, body *godog.DocString) error {
	return m.assertTreeContainsInPathInFs(path, defaultFs, body)
}

func (m *Manager) chmodInFs(path string, fs string, permStr string) error {
	perm, err := strToPerm(permStr)
	if err != nil {
		return err
	}

	return m.fs(fs).Chmod(path, perm)
}

func (m *Manager) removeFileInFs(path, fs string) error {
	return m.fs(fs).RemoveAll(path)
}

func (m *Manager) createFileInFs(path, fs string) error {
	return m.createFileInFsWithContent(path, fs, nil)
}

func (m *Manager) createDirectoryInFs(path, fs string) error {
	if err := m.track(fs, path); err != nil {
		return fmt.Errorf("could not track directory: %w", err)
	}

	path = filepath.Clean(path)

	if err := m.fs(fs).MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("could not mkdir %q: %w", path, err)
	}

	return nil
}

func (m *Manager) createFileInFsWithContent(path, fsID string, body *godog.DocString) error {
	if err := m.track(fsID, path); err != nil {
		return fmt.Errorf("could not track file: %w", err)
	}

	fs := m.fs(fsID)
	parent := filepath.Dir(path)

	if err := fs.MkdirAll(parent, 0755); err != nil {
		return fmt.Errorf("could not mkdir %q: %w", parent, err)
	}

	path = filepath.Clean(path)

	f, err := fs.Create(path)
	if err != nil {
		return fmt.Errorf("could not create %q: %w", path, err)
	}

	defer f.Close() // nolint: errcheck

	if body != nil {
		if _, err = f.WriteString(body.Content); err != nil {
			return fmt.Errorf("could not write file %q: %w", path, err)
		}
	}

	return nil
}

func (m *Manager) assertFileExistsInFs(path string, fs string) error {
	t := teeError()

	if !aferoassert.FileExists(t, m.fs(fs), path) {
		return t.LastError()
	}

	return nil
}

func (m *Manager) assertDirectoryExistsInFs(path string, fs string) error {
	t := teeError()

	if !aferoassert.DirExists(t, m.fs(fs), path) {
		return t.LastError()
	}

	return nil
}

func (m *Manager) assertFileContentInFs(path string, fs string, body *godog.DocString) error {
	t := teeError()

	if !aferoassert.FileContent(t, m.fs(fs), path, body.Content) {
		return t.LastError()
	}

	return nil
}

func (m *Manager) assertFileContentRegexpInFs(path string, fs string, body *godog.DocString) error {
	t := teeError()

	if !aferoassert.FileContentRegexp(t, m.fs(fs), path, fileContentRegexp(body.Content)) {
		return t.LastError()
	}

	return nil
}

func (m *Manager) assertFilePermInFs(path string, fs string, permStr string) error {
	perm, err := strToPerm(permStr)
	if err != nil {
		return err
	}

	t := teeError()

	if !aferoassert.Perm(t, m.fs(fs), path, perm) {
		return t.LastError()
	}

	return nil
}

func (m *Manager) assertTreeEqualInFs(fs string, body *godog.DocString) error {
	return m.assertTreeEqualInPathInFs("", fs, body)
}

func (m *Manager) assertTreeEqualInPathInFs(path, fs string, body *godog.DocString) error {
	t := teeError()

	if !aferoassert.YAMLTreeEqual(t, m.fs(fs), body.Content, path) {
		return t.LastError()
	}

	return nil
}

func (m *Manager) assertTreeContainsInFs(fs string, body *godog.DocString) error {
	return m.assertTreeContainsInPathInFs("", fs, body)
}

func (m *Manager) assertTreeContainsInPathInFs(path, fs string, body *godog.DocString) error {
	t := teeError()

	if !aferoassert.YAMLTreeContains(t, m.fs(fs), body.Content, path) {
		return t.LastError()
	}

	return nil
}

// NewManager initiates a new Manager.
func NewManager(options ...Option) *Manager {
	cwd, err := os.Getwd()
	mustNoError(err)

	m := &Manager{
		fss: map[string]afero.Fs{
			defaultFs: afero.NewOsFs(),
		},
		testDir:      cwd,
		trackedFiles: make(map[string][]string),
	}

	for _, o := range options {
		o(m)
	}

	return m
}

// WithFs sets a file system by name.
func WithFs(name string, fs afero.Fs) Option {
	return func(m *Manager) {
		m.fss[name] = fs
	}
}

// WithDefaultFs sets the default file system.
func WithDefaultFs(fs afero.Fs) Option {
	return func(m *Manager) {
		m.fss[defaultFs] = fs
	}
}
