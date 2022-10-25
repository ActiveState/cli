//go:build windows
// +build windows

package subshell

func init() {
	supportedShells = []SubShell{
		&cmd.SubShell{},
	}
}
