package execmeta

import (
	"bytes"
	"reflect"
	"testing"
)

/* example meta.as
::sock::/tmp/state-ipc/state-ipts.DX-123.sock
::env::TESTER="test/best"::env::BESTER="example"
::bins::/home/.cache/deadbeef/bin/python3::bins::/home/.cache/deadbeef/bin/cython
::commit-id::1234abcd-1234-abcd-1234-abcd1234
::namespace::SomeOrg/Test
::headless::true
*/

func TestExecMeta(t *testing.T) {
	env := map[string]string{
		"EXAMPLE": "value",
		"SAMPLE":  "other",
		"THIRD":   "whatever",
	}
	bins := []string{
		"/example/bin/abc",
		"/example/bin/def",
		"/example/bin/xyz",
	}
	tgt := Target{
		CommitUUID: "1234abcd-1234-abcd-1234-abcd1234",
		Namespace:  "owner/project",
		Dir:        "/example/bin",
		Headless:   true,
	}
	m := New("/sock-path", env, tgt, bins)
	buf := &bytes.Buffer{}
	_, err := m.WriteTo(buf)
	if err != nil {
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
