// +build windows

func removeSelf() error {
	scriptName := "removePaths"
	box := packr.NewBox("../../../assets/scripts/")
	scriptBlock := box.String(fmt.Sprintf("%s.bat", scriptName))
	sf, err := scriptfile.New(language.Batch, scriptName, scriptBlock)
	if err != nil {
		return locale.WrapError(err, "err_clean_script", "Could not create new scriptfile")
	}

	exe := appinfo.StateApp().Exec()

	args := []string{"/C", sf.Filename(), logFile, fmt.Sprintf("%d", os.Getpid()), filepath.Base(exe), exe}
	_, err = exeutils.ExecuteAndForget("cmd.exe", args)
	if err != nil {
		return locale.WrapError(err, "err_clean_start", "Could not start remove the transitional State Tool")
	}

	return nil
}
