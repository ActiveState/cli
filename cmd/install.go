package cmd

import (
  "errors"
  "net/http"

  "github.com/spf13/cobra"
)

func init() {
  rootCmd.AddCommand(installCmd)
}

var installCmd = &cobra.Command{
  Use:   "install",
  Short: "Install the given project in the current working directory",
  Args: func(cmd *cobra.Command, args []string) error {
    if len(args) < 1 {
      return errors.New("requires at least one arg")
    }
    return nil
  },
  Run: func(cmd *cobra.Command, args []string) {
    var url = args[0]
    resp, err := http.Get(url)

    if err != nil {
      return errors.New(err)
    }

    println(resp)
  },
}