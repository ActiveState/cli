package output

import (
	"strings"
	"testing"
)

func TestStyledTitleString(t *testing.T) {
	tests := []struct {
		test  string
		title string
		want  string
	}{
		{test: "basic non-empty", title: "Platforms", want: nonemptyBasic},
		{test: "basic empty", title: "", want: emptyBasic},
		{
			test:  "basic long",
			title: `Activating the "ActiveState/cli" runtime`,
			want:  nonemptyLong,
		},
	}

	for _, tt := range tests {
		got := NewStyledTitle(tt.title).String()
		if got != tt.want {
			t.Errorf("%s:\ngot\n%v\nwant\n%v", tt.test, got, tt.want)
		}
	}
}

var nonemptyBasic = strings.TrimSpace(`
╔═══════════════╗
║               ║
║   Platforms   ║
║               ║
╚═══════════════╝
`)

var emptyBasic = strings.TrimSpace(`
╔══════╗
║      ║
║      ║
║      ║
╚══════╝
`)

var nonemptyLong = strings.TrimSpace(`
╔══════════════════════════════════════════════╗
║                                              ║
║   Activating the "ActiveState/cli" runtime   ║
║                                              ║
╚══════════════════════════════════════════════╝
`)
