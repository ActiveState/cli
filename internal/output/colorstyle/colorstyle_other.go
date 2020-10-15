// +build !windows

package colorstyle

import "io"

func New(writer io.Writer) ColorStyler {
	return NewANSI(writer)
}
