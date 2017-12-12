package cmd

import (
	"encoding/json"
	"log"
	"fmt"
	"io/ioutil"
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
    url := args[0]
    res, err := http.Get(url)

    if err != nil {
      log.Fatal(err)
    }

    body, err := ioutil.ReadAll(res.Body)
    bodyStr := string(body)

    var val map[string]interface{}
    err2 := json.Unmarshal([]byte(bodyStr), &val)

    fmt.Printf("%s", err2.Error())
  },
}