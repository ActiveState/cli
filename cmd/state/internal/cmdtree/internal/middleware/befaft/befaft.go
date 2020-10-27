package befaft

import (
	"fmt"

	"github.com/ActiveState/cli/internal/captain"
)

func Wrap(next captain.ExecuteFunc) captain.ExecuteFunc {
	return func(cmd *captain.Command, args []string) error {
		fmt.Println("pre")

		if err := next(cmd, args); err != nil {
			return err
		}

		fmt.Println("post")

		return nil
	}
}
