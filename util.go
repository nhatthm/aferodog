package aferodog

import (
	"fmt"
	"os"
	"regexp"
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

func fileContentRegexp(s string) string {
	pattern := regexp.MustCompile(`<regexp:[^/]+/>`)
	matches := pattern.FindAllString(s, -1)
	cnt := len(matches)

	if cnt == 0 {
		return regexp.QuoteMeta(s)
	}

	replacementsBefore := make([]string, 0, cnt*2)
	replacementAfter := make([]string, 0, cnt*2)

	for i, match := range matches {
		token := fmt.Sprintf("<regexp%d/>", i)

		replacementsBefore = append(replacementsBefore, match, token)
		replacementAfter = append(replacementAfter, token, strings.TrimPrefix(strings.TrimSuffix(match, "/>"), "<regexp:"))
	}

	replaceBefore := strings.NewReplacer(replacementsBefore...)
	replaceAfter := strings.NewReplacer(replacementAfter...)

	s = replaceBefore.Replace(s)
	s = regexp.QuoteMeta(s)

	return replaceAfter.Replace(s)
}
