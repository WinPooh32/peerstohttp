package name

import (
	"strings"

	"github.com/asaskevich/govalidator"
)

type Music string

func (m Music) Name() string {
	var s string

	s = takeLast(string(m))
	s = trimTrackNum(s)
	s = strings.TrimSpace(s)

	return s
}

func trimTrackNum(name string) string {
	var tokens = strings.SplitN(name, " ", 2)
	if len(tokens) < 2 {
		return name
	}

	var s = strings.ReplaceAll(tokens[0], ".", "")

	if govalidator.IsUTFNumeric(s) {
		return tokens[1]
	}

	return name
}

func takeLast(name string) string {
	var size = len(name)
	if size <= 3 {
		return name
	}

	var begin int
	var scope int
	var buf = []byte(name)

	for i := size - 1; i >= 0; i-- {
		switch buf[i] {
		case ')':
			scope++
		case '(':
			scope--
		case '-':
			if scope == 0 && i+1 < size {
				// skip -
				begin = i + 1
			}
		default:
		}
	}

	if begin == 0 {
		return name
	}

	return string(buf[begin:])
}
