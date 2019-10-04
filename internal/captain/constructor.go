package captain

import (
	"fmt"

	"github.com/ActiveState/cli/internal/failures"
	"github.com/ActiveState/cli/internal/locale"
	"github.com/spf13/cobra"
)

func New(cmd Commander) (*Platoon, error) {
	if err := validateCmd(cmd); err != nil {
		return nil, err
	}

	platoon := &Platoon{}

	cobra, err := setupCobra(platoon, cmd)
	if err != nil {
		return nil, err
	}

	for _, child := range cmd.Children() {
		childCmd, err := New(child)
		if err != nil {
			return nil, err
		}
		cobra.AddCommand(childCmd.cobra)
	}

	return &Platoon{cmd, cobra}, nil
}

func setupCobra(platoon *Platoon, cmd Commander) (*cobra.Command, error) {
	meta := cmd.Meta()
	loc := cmd.Locale()

	cobraCmd := &cobra.Command{
		Use:     meta.Name,
		Aliases: meta.Aliases,
		Short:   loc.Description,
		RunE:    platoon.runner,
		Args:    platoon.argValidator,
	}

	if loc.UsageTemplate != "" {
		setUsageTemplate(cobraCmd, cmd.Arguments(), loc.UsageTemplate)
	}

	for _, flag := range cmd.Flags() {
		if err := addFlag(cobraCmd, flag); err != nil {
			return nil, err
		}
	}

	return cobraCmd, nil
}

func validateCmd(cmd Commander) error {
	args := cmd.Arguments()
	for idx, arg := range args {
		if idx > 0 && arg.Required && !args[idx-1].Required {
			return failures.FailInput.New(
				fmt.Sprintf("Cannot have a non-required argument followed by a required argument.\n\n%v\n\n%v",
					arg, args[len(args)-1]))
		}
	}
	return nil
}

func setUsageTemplate(cobraCmd *cobra.Command, args []*Argument, usageTemplate string) {
	localizedArgs := []map[string]string{}
	for _, arg := range args {
		req := ""
		if arg.Required {
			req = "1"
		}
		localizedArgs = append(localizedArgs, map[string]string{
			"Name":        locale.T(arg.Name),
			"Description": locale.T(arg.Description),
			"Required":    req,
		})
	}
	cobraCmd.SetUsageTemplate(locale.Tt(usageTemplate, map[string]interface{}{
		"Arguments": localizedArgs,
	}))
}

func addFlag(cobraCmd *cobra.Command, flag *Flag) error {
	flagSetter := cobraCmd.Flags
	if flag.Persist {
		flagSetter = cobraCmd.PersistentFlags
	}

	switch flag.Type {
	case TypeString:
		flagSetter().StringVarP(flag.StringVar, flag.Name, flag.Shorthand, flag.StringValue, flag.Description)
	case TypeInt:
		flagSetter().IntVarP(flag.IntVar, flag.Name, flag.Shorthand, flag.IntValue, flag.Description)
	case TypeBool:
		flagSetter().BoolVarP(flag.BoolVar, flag.Name, flag.Shorthand, flag.BoolValue, flag.Description)
	default:
		return failures.FailInput.New("Unknown type:" + string(flag.Type))
	}

	return nil
}
