package colorstyle

type Style int

const (
	Default = Style(iota)
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
)

type ColorStyler interface {
	SetStyle(s Style, bright bool)
}
