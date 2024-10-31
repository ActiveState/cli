package renderers

import (
	"fmt"
	"strings"

	"github.com/ActiveState/cli/internal/colorize"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/output"
	"github.com/ActiveState/cli/internal/termutils"
)

type bulletList struct {
	prefix  string
	items   []string
	bullets []string
}

// BulletTree outputs a list like:
//
// ├─ one
// ├─ two
// │ wrapped
// └─ three
var BulletTree = []string{output.TreeMid, output.TreeMid, output.TreeLink, output.TreeEnd}

// HeadedBulletTree outputs a list like:
//
// one
// ├─ two
// │ wrapped
// └─ three
var HeadedBulletTree = []string{"", output.TreeMid, output.TreeLink, output.TreeEnd}

// NewBulletList returns a printable list of items prefixed with the given set of bullets.
// The set of bullets should contain four items: the bullet for the first item (e.g. ""); the
// bullet for each subsequent item (e.g. "├─"); the bullet for an item's wrapped lines, if any
// (e.g. "│"); and the bullet for the last item (e.g. "└─").
// The returned list can be passed to a plain printer. It should not be passed to a structured
// printer.
func NewBulletList(prefix string, bullets, items []string) *bulletList {
	if len(bullets) != 4 {
		multilog.Error("Invalid bullet list: 4 bullets required")
		bullets = BulletTree
	}
	return &bulletList{prefix, items, bullets}
}

func (b *bulletList) MarshalOutput(format output.Format) interface{} {
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
		bullet := b.bullets[0]
		if len(b.items) == 1 {
			bullet = b.bullets[3] // special case list length of one; use last bullet
		}

		if i == 0 {
			if bullet != "" {
				bullet += " "
			}
			item = b.prefix + bullet + item
		} else {
			bullet = b.bullets[1]
			continuation := indent + b.bullets[2] + " "
			if i == len(b.items)-1 {
				bullet = b.bullets[3] // this is the last item
				continuation = " "
			}
			wrapped := colorize.Wrap(item, termutils.GetWidth()-len(indent), true, continuation).String()
			item = fmt.Sprintf("%s%s %s", indent, bullet, wrapped)
		}
		out[i] = item
	}

	return strings.Join(out, "\n")
}
