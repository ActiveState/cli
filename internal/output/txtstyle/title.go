package txtstyle

import (
	"unicode/utf8"
)

const (
	// DefaultTitlePadding is the padding character length.
	DefaultTitlePadding = 3
)

// Title represents the config of a styled title. It does not, currently,
// support combining diactics (more info: https://play.golang.org/p/VmHyq3JJ7On).
type Title struct {
	Text    string
	Padding int
}

// NewTitle provides a construction of Title using the default title padding.
func NewTitle(text string) *Title {
	return &Title{
		Text:    text,
		Padding: DefaultTitlePadding,
	}
}

// String implements the fmt.Stringer interface.
func (t *Title) String() string {
	titleLen := utf8.RuneCountInString(t.Text) // NOTE: ignores effects of combining diacritics
	lineLen := titleLen + 2 + 2*t.Padding + 1  // text, border, padding, newline

	rs := make([]rune, 5*lineLen)

	topLf := 0
	topRt := lineLen - 2
	btmLf := lineLen * 4
	btmRt := len(rs) - 2
	titleBgn := lineLen*2 + t.Padding + 1

	rs[topLf], rs[topRt] = '╔', '╗'
	rs[btmLf], rs[btmRt] = '╚', '╝'
	copy(rs[titleBgn:], []rune(t.Text))

	for i := range rs {
		// IS not empty
		if rs[i] != 0 {
			continue
		}

		// IS end (control) of line
		if i%lineLen == lineLen-1 {
			rs[i] = '\n'
			continue
		}

		// IS between top two corners OR between bottom two corners
		if (i > topLf && i < topRt) || (i > btmLf && i < btmRt) {
			rs[i] = '═'
			continue
		}

		// IS not in first line AND not in last line AND start or end (content) of line
		if i > topRt && i < btmLf && (i%lineLen == 0 || i%lineLen == lineLen-2) {
			rs[i] = '║'
			continue
		}

		rs[i] = ' '
	}

	return string(rs[:len(rs)-1]) // drop ending newline
}
