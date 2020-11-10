package updater

import "fmt"

func PIDFileName(n int) string {
	return fmt.Sprintf("updated-%d", n)
}
