//go:build !windows
// +build !windows

package e2e

const (
	basePath             = "/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin:/usr/local:/usr/local/sbin:/usr/local/opt"
	systemHomeEnvVarName = "HOME"
)

func platformSpecificEnv(dirs *Dirs) []string {
	return nil
}

func platformPath() string {
	return basePath
}
