package updatemocks

import (
	"fmt"

	"github.com/ActiveState/cli/internal/constants"
)

func CreateRequestPath(branch, append string) string {
	return fmt.Sprintf("%s/%s/%s", constants.CommandName, branch, append)
}
