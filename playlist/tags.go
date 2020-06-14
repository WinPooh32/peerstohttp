package playlist

import (
	"bufio"
	"strings"
)

type tagScope int

const (
	tagInside tagScope = iota
	tagOutside
	tagAnywhere
)

var tags = map[string]tagScope{
	"instrumental": tagAnywhere,
	"instr":        tagAnywhere,
	"acoustic":     tagAnywhere,
	"interview":    tagAnywhere,
	"orchestral":   tagAnywhere,
	"orch":         tagAnywhere,
	"mix":          tagAnywhere,
	"remix":        tagAnywhere,
	"remixes":      tagAnywhere,
	"demo":         tagInside,
	"capella":      tagInside,
	"acapella":     tagInside,
	"synthesis":    tagInside,
	"live":         tagInside,
	"album":        tagInside,
	"bonus":        tagInside,
	"radio":        tagInside,
	"original":     tagInside,
	"full":         tagInside,
	"heavy":        tagInside,
}

func ExtractPathTags(path []string) []string {
	var b = make([]string, 0, 0)
	for _, s := range path {
		extractTags(replaceDelims(s), &b)
	}
	return b
}

func extractTags(line string, buf *[]string) {
	var non bool
	var scope bool

	var s = bufio.NewScanner(strings.NewReader(line))
	s.Split(bufio.ScanWords)

	for s.Scan() {
		var text = strings.ToLower(s.Text())

		var begin = strings.HasPrefix(text, "(") || strings.HasPrefix(text, "[")

		if begin {
			if !scope {
				scope = true
			}
		}

		if non {
			non = false
			continue
		}

		if text == "non" {
			non = true
			continue
		}

		var filtered = filter(text)

		if tgScope, ok := tags[filtered]; ok {
			if tgScope == tagAnywhere || (scope && tgScope == tagInside) || (!scope && tgScope == tagOutside) {
				*buf = append(*buf, filtered)
			}
		}

		if !begin && (strings.HasSuffix(text, ")") || strings.HasSuffix(text, "]")) {
			if scope {
				scope = false
			}
		}
	}
}

func replaceDelims(s string) string {
	var a [64]byte
	var buf = a[:0]

	for _, v := range []byte(s) {
		switch v {
		case '_', '-', '.':
			buf = append(buf, ' ')
		default:
			buf = append(buf, v)
		}
	}

	return string(buf)
}

func filter(s string) string {
	var a [64]byte
	var buf = a[:0]

	for _, v := range []byte(s) {
		switch v {
		case '-', '(', '[', ')', ']', '.':
			//skip
		default:
			buf = append(buf, v)
		}
	}

	return string(buf)
}
