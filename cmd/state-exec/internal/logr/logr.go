package logr

type (
	LogFunc func(format string, args ...interface{})
)

var (
	Debug    LogFunc = Null
	debugSet bool
)

func SetDebug(fn LogFunc) {
	if debugSet {
		return
	}
	Debug = fn
	debugSet = true
}

func Null(format string, args ...interface{}) {}

func CallIfDebugIsSet(fn func()) {
	if !debugSet {
		return
	}
	fn()
}
