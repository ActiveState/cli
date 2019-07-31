package terminal

func newRuneReaderState(input FileReader) RuneReaderState {
	return newTerminalRuneReaderState(input)
}
