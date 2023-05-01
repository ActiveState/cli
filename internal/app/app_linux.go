package app

func (a *App) Path() string {
	return a.Exec
}

func (a *App) install() error {
	return nil
}

func (a *App) uninstall() error {
	return nil
}
