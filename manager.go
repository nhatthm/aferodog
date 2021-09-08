package aferodog

import (
	"github.com/godogx/aferosteps"
	"github.com/spf13/afero"
)

// TempDirer creates a temp dir evey time it is called.
//
// Deprecated: Use aferosteps.TempDirer instead.
type TempDirer = aferosteps.TempDirer

// Option is to configure Manager.
//
// Deprecated: Use aferosteps.Option instead.
type Option = aferosteps.Option

// Manager manages a list of file systems and provides steps for godog.
//
// Deprecated: Use aferosteps.Manager instead.
type Manager = aferosteps.Manager

// NewManager initiates a new Manager.
//
// Deprecated: Use aferosteps.NewManager instead.
func NewManager(options ...Option) *Manager {
	return aferosteps.NewManager(options...)
}

// WithFs sets a file system by name.
//
// Deprecated: Use aferosteps.WithFs instead.
func WithFs(name string, fs afero.Fs) Option {
	return aferosteps.WithFs(name, fs)
}

// WithDefaultFs sets the default file system.
//
// Deprecated: Use aferosteps.WithDefaultFs instead.
func WithDefaultFs(fs afero.Fs) Option {
	return aferosteps.WithDefaultFs(fs)
}
