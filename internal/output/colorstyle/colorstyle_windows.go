package colorstyle

import "io"

func New(writer io.Writer) ColorStyler {
	if bufferInfo == nil {
		return NewANSI(writer)
	}
	return NewConsole()
}
