package logging

import (
	"fmt"
)

// A formatting interface- it is responsible of taking the arguments and composing a message
type Formatter interface {
	Format(ctx *MessageContext, message string, args ...interface{}) string
}

type SimpleFormatter struct {
	FormatString string
}

func (f *SimpleFormatter) Format(ctx *MessageContext, message string, args ...interface{}) string {
	return fmt.Sprintf(f.FormatString, ctx.Level, ctx.TimeStamp.Format("15:04:05.000"), ctx.File, ctx.Line, fmt.Sprintf(message, args...))
}

var DefaultFormatter Formatter = &SimpleFormatter{
	FormatString: "[%[1]s %[2]s %[3]s:%[4]d] %[5]s",
}
