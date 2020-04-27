// +build !windows

package conpty

func InitTerminal() (func(), error) {
	return func() {}, nil
}
