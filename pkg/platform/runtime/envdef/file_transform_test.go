package envdef

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRelocateFile(t *testing.T) {
	nullCharacter := "\u0000"
	invalidPad := "too long"
	cases := []struct {
		Name        string
		Replacement string
		PadWith     *string
		HasError    bool
		Expected    string
	}{
		{
			"null-padded", "/ghi", &nullCharacter, false,
			"/ghi/something/else\u0000\u0000\u0000\u0000text\u0000",
		},
		{
			"text-only", "/ghi", nil, false,
			"/ghi/something/else\u0000text\u0000",
		},
		{
			"invalid-padding", "/ghi", &invalidPad, true, "",
		},
		{
			"replacement-too-long", "/too-long", &nullCharacter, true, "",
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(tt *testing.T) {
			ft := &FileTransform{
				Pattern: "/abcdef",
				With:    c.Replacement,
				PadWith: c.PadWith,
			}

			res, err := ft.relocateFile([]byte("/abcdef/something/else\u0000text\u0000"), c.Replacement)
			if c.HasError != (err != nil) {
				tt.Fatalf("relocateFile returned with err: %v", err)
			}
			if err == nil {
				assert.Equal(tt, []byte(c.Expected), res)
			}
		})
	}
}

func TestApplyConstTransforms(t *testing.T) {
	cs := NewConstants(`C:\an\installdir`)

	cases := []struct {
		Name          string
		TransformJSON string
		HasError      bool
		Expected      string
	}{
		{
			"double-slashes", `[{"pattern":
			   "\\",
			"with": "\\\\", "in": ["INSTALLDIR"]}]`,
			false, `C:\\an\\installdir`,
		},
		{
			"unchanged", `[]`, false, "C:\\an\\installdir",
		},
		{
			"invalid-constant", `[{"pattern": "\\", "with": "\\\\", "in": ["INVALID"]}]`,
			true, "",
		},
	}

	//
	for _, c := range cases {
		t.Run(c.Name, func(tt *testing.T) {
			var ct []ConstTransform
			err := json.Unmarshal([]byte(c.TransformJSON), &ct)
			require.NoError(tt, err)
			ft := &FileTransform{
				ConstTransforms: ct,
			}
			res, err := ft.applyConstTransforms(cs)
			if c.HasError != (err != nil) {
				tt.Fatalf("applyConstTransforms returned with err: %v", err)
			}
			if err == nil {
				assert.Equal(tt, c.Expected, res["INSTALLDIR"])
			}
		})
	}

}
