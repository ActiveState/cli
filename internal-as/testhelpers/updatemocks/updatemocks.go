package updatemocks

import (
	"fmt"
)

func CreateRequestPath(branch, append string) string {
	return fmt.Sprintf("%s/%s", branch, append)
}
