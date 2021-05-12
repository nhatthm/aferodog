package aferodog

import (
	"os"
	"strconv"
	"strings"
)

func strToPerm(s string) (os.FileMode, error) {
	base := 10
	if strings.HasPrefix(s, "0") {
		base = 8
	}

	mode, err := strconv.ParseUint(s, base, 32)
	if err != nil {
		return 0, err
	}

	return os.FileMode(mode), nil
}

func mustNoError(err error) {
	if err != nil {
		panic(err)
	}
}
