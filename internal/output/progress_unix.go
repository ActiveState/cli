//go:build linux || darwin
// +build linux darwin

package output

func (d *Spinner) moveCaretBackInCommandPrompt(n int) {
	// No-op (logging to Rollbar every tick would be a disaster)
	// Rely on manual and unit testing to catch any errors in display.
}
