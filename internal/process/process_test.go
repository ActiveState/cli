package process

import (
	"path/filepath"
	"testing"
)

func TestIsActivateCamdLineArgs(t *testing.T) {
	stateCmd := filepath.Join("usr", "bin", "state.exe")
	cases := []struct {
		Name     string
		Args     []string
		Expected bool
	}{
		{
			"state activate",
			[]string{stateCmd, "activate"},
			true,
		},
		{
			"state activate with params",
			[]string{stateCmd, "-v", "--output", "plain", "activate"},
			true,
		},
		{
			"state run",
			[]string{stateCmd, "run", "a-script"},
			false,
		},
		{
			"other command",
			[]string{"/bin/bash", "activate", "arg2"},
			false,
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(tt *testing.T) {
			if res := isActivateCmdlineArgs(c.Args); res != c.Expected {
				tt.Errorf("search for 'state activate' in args: %v, expected=%v, got=%v", c.Args, c.Expected, res)
			}
		})
	}
}
