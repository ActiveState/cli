package condition

import (
	"os"
	"strings"
)

var inTest = strings.HasSuffix(os.Args[0], ".test")

// InTest returns true when the app is being tested
func InTest() bool { return inTest }
