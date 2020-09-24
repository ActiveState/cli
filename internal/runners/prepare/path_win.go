// +build windows

package prepare

import "os"

func updateEnvironment(filepath string) error {
	err := updatePath(filepath)
	if err != nil {
		return err
	}

	return os.Setenv(
		"PATHEXT",
		os.Getenv("PATHEXT")+string(os.PathListSeparator)+".LNK",
	)
}
