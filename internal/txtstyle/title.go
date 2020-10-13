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

	rs[0], rs[lineLen-2] = '╔', '╗'
	rs[lineLen*4], rs[len(rs)-2] = '╚', '╝'
	copy(rs[lineLen*2+t.Padding+1:], []rune(t.Text))

	for i := range rs {
		// IS not empty
		if rs[i] != 0 {
			continue
		}

		// IS end of line
		if i%lineLen == lineLen-1 {
			rs[i] = '\n'
			continue
		}

		// IS between top two corners OR between bottom two corners
		if (i > 0 && i < lineLen-2) || (i > lineLen*4 && i < lineLen*5-2) {
			rs[i] = '═'
			continue
		}

		// IS not in first AND not in last line AND start or end of line
		if i > lineLen-1 && i < lineLen*4-1 && (i%lineLen == 0 || i%lineLen == lineLen-2) {
			rs[i] = '║'
			continue
		}

		rs[i] = ' '
	}

	return string(rs[:len(rs)-1]) // drop ending newline
}
