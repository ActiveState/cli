//go:build !windows
// +build !windows

package notify

func Send(title, message, actionName, actionLink string) error {
	return nil
}
