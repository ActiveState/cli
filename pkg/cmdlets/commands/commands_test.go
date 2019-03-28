package commands

import (
	"errors"
	"fmt"
	"testing"

	"github.com/ActiveState/cli/internal/testhelpers/exiter"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/stretchr/testify/require"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

func TestCreateCommand(t *testing.T) {

	var cmd1 = Command{
		Name:          "foo",
		Description:   "foo_description",
		Aliases:       []string{"blah"},
		Run:           func(cmd *cobra.Command, args []string) {},
		UsageTemplate: "foo_usage_template",
	}

	cmd1.Register()

	var cC *cobra.Command
	cC = cmd1.GetCobraCmd()

	assert.NotNil(t, cC)
}
func TestRunCommand(t *testing.T) {

	ran := false

	var cmd1 = Command{
		Name:          "foo",
		Description:   "foo_description",
		Aliases:       []string{"blah"},
		Run:           func(cmd *cobra.Command, args []string) { ran = true },
		UsageTemplate: "foo_usage_template",
	}

	cmd1.Execute()

	assert.True(t, ran)
}

func TestExitError(t *testing.T) {
	errExpected := errors.New("test")

	var cmd1 = Command{
		Name: "foo",
		Run: func(cmd *cobra.Command, args []string) {
			failures.Handle(errExpected, "Some failure")
		},
	}

	err := cmd1.Execute()
	require.True(t, errExpected == err, "Returns our error")
}

func TestExitCode(t *testing.T) {
	var cmd1 = Command{
		Name: "foo",
		Run: func(cmd *cobra.Command, args []string) {
			failures.Handle(errors.New("test"), "Some failure")
		},
		Exiter: exiter.Exit,
	}

	code := exiter.WaitForExit(func() {
		cmd1.Execute()
		require.FailNow(t, "This should never be reached as exit would have been called")
	})
	require.Equal(t, 1, code, "Exits with code 1")
}

func TestExitCodeZero(t *testing.T) {
	var cmd1 = Command{
		Name: "foo",
		Run: func(cmd *cobra.Command, args []string) {
			failures.FailDeveloper.New("Test failure that shouldn't trip up our exit mechanic")
		},
		Exiter: exiter.Exit,
	}

	code := exiter.WaitForExit(func() {
		cmd1.Execute()
	})
	require.Equal(t, -1, code, "Doesn't trigger an exit")
}

func TestAppend(t *testing.T) {

	var cmd1 = Command{
		Name:        "foo",
		Description: "foo_description",
	}

	var cmd2 = Command{
		Name:        "foo",
		Description: "foo_description",
	}

	cmd1.Append(&cmd2)

	var cC *cobra.Command
	cC = cmd1.GetCobraCmd()

	assert.True(t, cC.HasSubCommands())
}

func TestFlags(t *testing.T) {

	var svar string
	var cmd = Command{
		Name:        "foo",
		Description: "foo_description",
		Flags: []*Flag{
			&Flag{
				Name:        "flag",
				Type:        TypeString,
				StringVar:   &svar,
				StringValue: "value",
			},
		},
	}

	cmd.Register()

	var cc *cobra.Command
	cc = cmd.GetCobraCmd()
	pflags := cc.PersistentFlags()

	pflags.VisitAll(func(pf *pflag.Flag) {
		assert.Equal(t, "flag", pf.Name, "flag is set")
		assert.Equal(t, "value", pf.Value, "flag is set")
	})
}

func TestFlagsUseCb(t *testing.T) {

	var registered int
	var used int
	var onUse = func() {
		used++
	}

	var createCmd = func() *Command {
		registered++
		var bvar bool
		return &Command{
			Name: fmt.Sprintf("foo%d", registered),
			Flags: []*Flag{
				&Flag{
					Name:    fmt.Sprintf("flag%d", registered),
					Type:    TypeBool,
					BoolVar: &bvar,
					Persist: true,
					OnUse:   onUse,
				},
			},
			Run: func(cmd *cobra.Command, args []string) {},
		}
	}

	cmd1 := createCmd()
	cmd2 := createCmd()

	cmd1.Append(cmd2)

	Cc := cmd1.GetCobraCmd()
	Cc.SetArgs([]string{"foo2", "--flag1", "--flag2"})
	cmd1.Execute()

	assert.Equal(t, 2, used, "Should call onUse twice")

}

func TestArgs(t *testing.T) {

	var svar1 string
	var svar2 string
	var cmd1 = Command{
		Name:        "foo",
		Description: "foo_description",
		Run:         func(cmd *cobra.Command, args []string) {},
		Arguments: []*Argument{
			&Argument{
				Name:     "name1",
				Variable: &svar1,
				Required: true,
			},
			&Argument{
				Name:     "name2",
				Variable: &svar2,
			},
		},
	}

	cmd1.Register()

	cc := cmd1.GetCobraCmd()
	cc.SetArgs([]string{"value"})

	err := cmd1.Execute()
	assert.NoError(t, err, "should execute")

	assert.Equal(t, "value", svar1, "argument is set")
}

func TestArgsRequirePanic(t *testing.T) {
	var svar1 string
	var svar2 string
	var cmd2 = Command{
		Name:        "foo",
		Description: "foo_description",
		Run:         func(cmd *cobra.Command, args []string) {},
		Arguments: []*Argument{
			&Argument{
				Name:     "name",
				Variable: &svar1,
			},
			&Argument{
				Name:     "name",
				Variable: &svar2,
				Required: true,
			},
		},
	}

	assert.Panics(t, cmd2.Register, "Should fail because you cannot add a required argument after an optional argument")
}

func TestArgValidator(t *testing.T) {
	var svar string
	var cmd = Command{
		Name:        "foo",
		Description: "foo_description",
		Run:         func(cmd *cobra.Command, args []string) {},
		Arguments: []*Argument{
			&Argument{
				Name:     "name",
				Variable: &svar,
				Validator: func(arg *Argument, value string) error {
					if value != "value" {
						return errors.New("Fail")
					}
					return nil
				},
			},
		},
	}

	cmd.Register()

	cc := cmd.GetCobraCmd()
	cc.SetArgs([]string{"value"})

	err := cmd.Execute()
	assert.NoError(t, err, "Validator is ran properly")
}

func TestAliases(t *testing.T) {
	var al = "alias"
	var cmd = Command{
		Name:        "foo",
		Description: "foo_description",
		Aliases:     []string{al},
		Run:         func(cmd *cobra.Command, args []string) {},
	}

	cmd.Register()

	cc := cmd.GetCobraCmd()

	assert.True(t, cc.HasAlias(al), "Command has alias.")
}
