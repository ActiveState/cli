package rollbar

import (
	"strings"
	"testing"
)

func TestBuildStack(t *testing.T) {
	frame := buildStack(getCallersFrames(0))[0]

	if !strings.HasSuffix(frame.Filename, "rollbar-go/stack_test.go") {
		t.Errorf("got: %s", frame.Filename)
	}
	if frame.Method != "rollbar-go.TestBuildStack" {
		t.Errorf("got: %s", frame.Method)
	}
	if frame.Line != 9 {
		t.Errorf("got: %d", frame.Line)
	}
}

func TestStackFingerprint(t *testing.T) {
	tests := []struct {
		Fingerprint string
		Stack       stack
	}{
		{
			"9344290d",
			stack{
				frame{"foo.go", "Oops", 1},
			},
		},
		{
			"a4d78b7",
			stack{
				frame{"foo.go", "Oops", 2},
			},
		},
		{
			"50e0fcb3",
			stack{
				frame{"foo.go", "Oops", 1},
				frame{"foo.go", "Oops", 2},
			},
		},
	}

	for i, test := range tests {
		fingerprint := test.Stack.Fingerprint()
		if fingerprint != test.Fingerprint {
			t.Errorf("tests[%d]: got %s", i, fingerprint)
		}
	}
}

func TestShortenFilePath(t *testing.T) {
	tests := []struct {
		Given    string
		Expected string
	}{
		{"", ""},
		{"foo.go", "foo.go"},
		{"/usr/local/go/src/pkg/runtime/proc.c", "pkg/runtime/proc.c"},
		{"/home/foo/go/src/github.com/stvp/rollbar.go", "github.com/stvp/rollbar.go"},
	}
	for i, test := range tests {
		got := shortenFilePath(test.Given)
		if got != test.Expected {
			t.Errorf("tests[%d]: got %s", i, got)
		}
	}
}
