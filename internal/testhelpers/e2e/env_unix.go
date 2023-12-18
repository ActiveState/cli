//go:build !windows
// +build !windows

package e2e

const (
	testPath             = "/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin:/usr/local:/usr/local/sbin:/usr/local/opt"
	systemHomeEnvVarName = "HOME"
)

func platformEnv(dirs *Dirs) []string {
	return nil
}
