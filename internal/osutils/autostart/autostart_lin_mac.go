//go:build !windows
// +build !windows

package autostart

type Options struct {
	LaunchFileName string
	IconFileName   string
	IconFileSource string
	GenericName    string
	Comment        string
	Keywords       string
	ConfigKey      string
	SetConfig      bool
}
