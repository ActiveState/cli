// +build windows

package prepare

func updateEnvironment(filepath string) error {
	err := updatePath(filepath)
	if err != nil {
		return err
	}

	return os.SetEnv(
		"PATHEXT",
		os.Getenv("PATHEXT")+string(os.PathListSeparator)+".LNK",
	)
}
