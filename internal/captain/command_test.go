package captain

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestFindChildrenTable(t *testing.T) {
	tests := []struct {
		name     string
		cmds     []string
		aliases  map[string]string
		args     []string
		expected []string
	}{
		{
			name:     "Find children",
			cmds:     []string{"cmd1", "cmd2", "cmd3"},
			args:     []string{"cmd2", "cmd3"},
			expected: []string{"cmd2", "cmd3"},
		},
		{
			name:     "Find children with alias",
			cmds:     []string{"cmd1", "cmd2", "cmd3"},
			aliases:  map[string]string{"cmd2": "cmd2alias"},
			args:     []string{"cmd2alias", "cmd3"},
			expected: []string{"cmd2", "cmd3"},
		},
		{
			name:     "Find children not found",
			cmds:     []string{"cmd1", "cmd2", "cmd3"},
			args:     []string{"cmd2", "cmd4"},
			expected: []string{"cmd2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmds := make([]*Command, len(tt.cmds))
			for i, name := range tt.cmds {
				cmds[i] = newTestCommand(name)
			}

			for i := 0; i < len(cmds)-1; i++ {
				cmds[i].AddChildren(cmds[i+1])
			}

			for name, alias := range tt.aliases {
				for _, cmd := range cmds {
					if cmd.Name() != name {
						continue
					}
					cmd.SetAliases(alias)
				}
			}

			children, err := cmds[0].FindChildren(tt.args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(children) != len(tt.expected) {
				t.Fatalf("length of children is not equal to expected")
			}

			for i, child := range children {
				if child.Name() != tt.expected[i] {
					t.Fatalf("unexpected child, got: %s, want: %s", child.Name(), tt.expected[i])
				}
			}
		})
	}
}

func newTestCommand(name string) *Command {
	cmd := &Command{
		name: name,
		cobra: &cobra.Command{
			Use: name,
		},
	}

	cobraMapping[cmd.cobra] = cmd

	return cmd
}
