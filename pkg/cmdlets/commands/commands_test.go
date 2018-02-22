package commands

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestCreateCommand(t *testing.T) {

	var cmd1 = Command{
		Name:          "foo",
		Description:   "foo_description",
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
		Run:           func(cmd *cobra.Command, args []string) { ran = true },
		UsageTemplate: "foo_usage_template",
	}

	cmd1.Execute()

	assert.True(t, ran)
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
