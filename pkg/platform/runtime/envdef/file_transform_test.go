package envdef

import (
	"testing"

	"github.com/autarch/testify/assert"
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
			"/ghi/something/else\u0000\u0000\u0000\u0000",
		},
		{
			"text-only", "/ghi", nil, false,
			"/ghi/something/else\u0000",
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

			res, err := ft.relocateFile([]byte("/abcdef/something/else\u0000"), c.Replacement)
			if c.HasError != (err != nil) {
				tt.Fatalf("relocateFile returned with err: %v", err)
			}
			if err == nil {
				assert.Equal(tt, res, []byte(c.Expected))
			}
		})
	}
}
