// +build darwin

package constants

// MacOSApplicationName is the name of the ActiveState Desktop app on MacOS.
// This is currently shared by the installer and uninstall mechanisms
// Todo With https://www.pivotaltracker.com/story/show/177600107 the constant
// should only be needed by the installer, so we could consider moving it there.
const MacOSApplicationName = "ActiveState Desktop.app"
