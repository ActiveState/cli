package execmeta

import (
	"bytes"
	"reflect"
	"testing"
)

func TestExecMeta(t *testing.T) {
	env := []string{
		"EXAMPLE=value",
		"SAMPLE=other",
		"THIRD=whatever",
	}
	bins := map[string]string{ // map[alias]dest
		"abc":         "/bin/abc",
		"def":         "/tools/def",
		"xyz.bat.exe": "/Scripts/xyz.bat",
		"xyz.exe":     "/Scripts/xyz.bat",
	}
	tgt := Target{
		CommitUUID: "1234abcd-1234-abcd-1234-abcd1234",
		Namespace:  "owner/project",
		Dir:        "/example/bin",
		Headless:   true,
	}
	m := New("/sock-path", env, tgt, bins)
	buf := &bytes.Buffer{}
	if err := m.Encode(buf); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	t.Logf(buf.String())

	mx, err := NewFromReader(buf)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if !reflect.DeepEqual(mx, m) {
		t.Fatalf("got %v, want %v", mx, m)
	}
}
