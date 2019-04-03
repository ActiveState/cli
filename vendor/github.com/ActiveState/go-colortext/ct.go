/*
ct package provides functions to change the color of console text.

Under windows platform, the Console API is used. Under other systems, ANSI text mode is used.
*/
package ct

import (
	"io"
)

// Color is the type of color to be set.
type Color int

const (
	// No change of color
	None = Color(iota)
	Black
	Red
	Green
	Yellow
	Blue
	Magenta
	Cyan
	White
)

// Style is the type of style to be set.
type Style int

const (
	Bold Style = iota
	Underline
)

// Reset resets the foreground, background and style
func Reset(writer io.Writer) {
	reset(writer)
}

// ChangeColor sets the foreground and background colors. If the value of the color is None,
// the corresponding color keeps unchanged.
// If fgBright or bgBright is set true, corresponding color use bright color. bgBright may be
// ignored in some OS environment.
func ChangeColor(writer io.Writer, fg Color, fgBright bool, bg Color, bgBright bool) {
	changeColor(writer, fg, fgBright, bg, bgBright)
}

// Foreground changes the foreground color.
func Foreground(writer io.Writer, cl Color, bright bool) {
	ChangeColor(writer, cl, bright, None, false)
}

// Background changes the background color.
func Background(writer io.Writer, cl Color, bright bool) {
	ChangeColor(writer, None, false, cl, bright)
}

// ChangeStyle changes the style of printed text
func ChangeStyle(writer io.Writer, styles ...Style) {
	changeStyle(writer, styles...)
}
