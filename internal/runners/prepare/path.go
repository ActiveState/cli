package prepare

import "os"

func updatePath(filepath string) error {
	return os.Setenv(
		"PATH",
		os.Getenv("PATH")+string(os.PathListSeparator)+filepath,
	)
}
