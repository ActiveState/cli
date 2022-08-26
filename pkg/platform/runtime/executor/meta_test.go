package executor

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/ActiveState/cli/pkg/platform/runtime/target"
	"github.com/go-openapi/strfmt"
)

func TestMeta(t *testing.T) {
	env := map[string]string{
		"EXAMPLE": "value",
		"SAMPLE":  "other",
		"THIRD":   "whatever",
	}
	tgt := target.NewCustomTarget("owner", "name", strfmt.UUID("1234abcd-1234-abcd-1234-abcd1234"), "./", target.TriggerActivate, true)
	m := NewMeta(env, tgt)
	buf := &bytes.Buffer{}
	_, err := m.WriteTo(buf)
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	fmt.Println(buf.String())

}
