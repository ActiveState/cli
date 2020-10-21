package txtstyle

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/ActiveState/cli/internal/colorize"
	"github.com/ActiveState/cli/internal/output"
)

const (
	// DefaultTitlePadding is the padding character length.
	DefaultTitlePadding = 3
)

// Title represents the config of a styled title. It does not, currently,
// support combining diactics (more info: https://play.golang.org/p/VmHyq3JJ7On).
type Title struct {
	Text      string
	Heading   string
	Padding   int
	ColorCode string
}

// NewTitle provides a construction of Title using the default title padding.
func NewTitle(text string) *Title {
	return &Title{
		Text:      text,
		Heading:   "",
		Padding:   DefaultTitlePadding,
		ColorCode: "DISABLED",
	}
}

// String implements fmt.Stringer.
func (t *Title) String() string {
	if t.Text == "" {
		return ""
	}

	text := colorize.StripColorCodes(t.Text)

	titleLen := utf8.RuneCountInString(text)  // NOTE: ignores effects of combining diacritics
	lineLen := titleLen + 2 + 2*t.Padding + 1 // text, border, padding, newline
	lines := 3

	rs := make([]rune, lines*lineLen)

	topLf := 0
	topRt := lineLen - 2
	btmLf := lineLen * (lines - 1)
	btmRt := len(rs) - 2
	titleBgn := lineLen + t.Padding + 1

	rs[topLf], rs[topRt] = '╔', '╗'
	rs[btmLf], rs[btmRt] = '╚', '╝'
	copy(rs[titleBgn:], []rune(text))

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

	if t.Heading != "" {
		copy(rs[topLf+1:], []rune(t.Heading))
	}

	prefix := "[" + t.ColorCode + "]"
	suffix := "[/RESET]"
	outLines := strings.Split(string(rs), "\n")
	for i, line := range outLines {
		if i == 0 || i == len(outLines)-2 {
			outLines[i] = prefix + line + suffix
		} else {
			re := regexp.MustCompile(`║(.*)║`)
			outLines[i] = re.ReplaceAllString(line, prefix+"║"+suffix+"[HEADING]$1[/RESET]"+prefix+"║")
		}
	}

	out := strings.Join(outLines, "\n")
	out = strings.Replace(out, text, t.Text, 1)

	return strings.TrimSpace(out) + "\n"
}

// MarshalOutput implements output.Marshaller.
func (t *Title) MarshalOutput(format output.Format) interface{} {
	if format == output.PlainFormatName {
		return t.String()
	}
	return output.Suppress
}
