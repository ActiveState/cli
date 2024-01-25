package colorstyle

type Style int

const (
	Default = Style(iota)
	Dim
	Reset
	Reversed
	Bold
	Underline
	Black
	Red
	Green
	Yellow
	Blue
	Magenta
	Cyan
	White
	Orange
)
