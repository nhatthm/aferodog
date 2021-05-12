package aferodog

import "fmt"

type tError struct {
	err error
}

func (t *tError) Errorf(format string, args ...interface{}) {
	t.err = fmt.Errorf(format, args...) // nolint: goerr113
}

func (t *tError) LastError() error {
	return t.err
}

func teeError() *tError {
	return &tError{}
}
