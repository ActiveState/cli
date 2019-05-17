// +build !windows

package shimming

var (
	binarySuffixes = []string{"", ".sh"}
	forwardHeader  = "#!/usr/bin/env bash\n"
	forwardSuffix  = ""
	forwardArgs    = "$@"
)
