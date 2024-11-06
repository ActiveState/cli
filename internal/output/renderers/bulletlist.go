package renderers

import (
	"strings"
	"unicode/utf8"

	"github.com/ActiveState/cli/internal/colorize"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/termutils"
)

type Bullets struct {
	Start string
	Mid   string
	Link  string
	End   string
}

type bulletList struct {
	prefix  string
	items   []string
	bullets Bullets
}

// BulletTree outputs a list like:
//
// ├─ one
// ├─ two
// │  wrapped
// └─ three
var BulletTree = Bullets{output.TreeMid, output.TreeMid, output.TreeLink + " ", output.TreeEnd}

// BulletTreeDisabled is like BulletTree, but tags the tree glyphs with [DISABLED].
var BulletTreeDisabled = Bullets{
	"[DISABLED]" + output.TreeMid + "[/RESET]",
	"[DISABLED]" + output.TreeMid + "[/RESET]",
	"[DISABLED]" + output.TreeLink + " [/RESET]",
	"[DISABLED]" + output.TreeEnd + "[/RESET]",
}

// HeadedBulletTree outputs a list like:
//
// one
// ├─ two
// │  wrapped
// └─ three
var HeadedBulletTree = Bullets{"", output.TreeMid, output.TreeLink + " ", output.TreeEnd}

func NewBulletList(prefix string, bullets Bullets, items []string) *bulletList {
	return &bulletList{prefix, items, bullets}
}

// str is the business logic for returning a bullet list's string representation for a given
// maximum width. Clients should call String() instead. Only tests should directly call this
// function.
func (b *bulletList) str(maxWidth int) string {
	out := make([]string, len(b.items))

	// Determine the indentation of each item.
	// If the prefix is pure indentation, then the indent is that prefix.
	// If the prefix is not pure indentation, then the indent is the number of characters between
	// the first non-space character and the end of the prefix.
	// For example, both "* " and " * " have and indent of 2 because items should be indented to
	// match the bullet item's left margin (note that wrapping will auto-indent to match the leading
	// space in the second example).
	indent := b.prefix
	if nonIndent := strings.TrimLeft(b.prefix, " "); nonIndent != "" {
		indent = strings.Repeat(" ", len(nonIndent))
	}

	for i, item := range b.items {
		bullet := b.bullets.Start
		if len(b.items) == 1 {
			bullet = b.bullets.End // special case list length of one; use last bullet
		}

		prefix := ""
		continuation := ""
		if i == 0 {
			if bullet != "" {
				bullet += " "
			}
			prefix = b.prefix + bullet
		} else {
			bullet = b.bullets.Mid + " "
			continuation = indent + b.bullets.Link + " "
			if i == len(b.items)-1 {
				bullet = b.bullets.End + " " // this is the last item
				continuation = " "
			}
			prefix = indent + bullet
		}
		wrapped := colorize.Wrap(item, maxWidth-len(indent)-bulletLength(bullet), true, continuation).String()
		out[i] = prefix + wrapped
	}

	return strings.Join(out, "\n")
}

func (b *bulletList) String() string {
	return b.str(termutils.GetWidth())
}

func bulletLength(bullet string) int {
	return utf8.RuneCountInString(colorize.StripColorCodes(bullet))
}
