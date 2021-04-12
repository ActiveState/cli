// +build !windows

package autostart

func (a *App) Enable() error {
	panic("Not implemented")
	return nil
}

func (a *App) Disable() error {
	panic("Not implemented")
	return nil
}

func (a *App) IsEnabled() bool {
	panic("Not implemented")
	return false
}
