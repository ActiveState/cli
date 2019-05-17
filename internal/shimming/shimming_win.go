// +build windows

package shimming

var (
	binarySuffixes = []string{".exe"}
	forwardHeader  = ""
	forwardSuffix  = ".cmd"
	forwardArgs    = "%*"
)
