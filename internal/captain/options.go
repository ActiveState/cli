package captain

import "github.com/spf13/cobra"

type Option func(platoon *Platoon) error

func OptionHidden() Option {
	return func(platoon *Platoon) error {
		platoon.cobra.Hidden = true
		return nil
	}
}

func OptionPersistentPreRun(preRunFunc func([]string) error) Option {
	return func(platoon *Platoon) error {
		platoon.cobra.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
			return preRunFunc(args)
		}
		return nil
	}
}
