//go:build linux || darwin
// +build linux darwin

package output

func (d *Spinner) moveCaretBackInCommandPrompt(n int) {
	// No-op
}
