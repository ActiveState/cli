// Unit tests for the logging module

package logging

import (
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
)

type Test1Handler struct {
	formatter Formatter
	file      *os.File
	messages  []string
}

func (l *Test1Handler) SetFormatter(f Formatter) {
	l.formatter = f
}

func (l *Test1Handler) SetVerbose(v bool) {
}

func (l *Test1Handler) Output() io.Writer {
	return nil
}

func (l *Test1Handler) Emit(ctx *MessageContext, message string, args ...interface{}) error {
	l.messages = append(l.messages, message)

	return nil
}

func (l *Test1Handler) Printf(msg string, args ...interface{}) {}

func (l *Test1Handler) Reset() {
	l.messages = make([]string, 0)
}

func (l *Test1Handler) Len() int {
	return len(l.messages)
}

func (l *Test1Handler) Close() {}

func logAllLevels(msg string) {
	Debug(msg)
	Info(msg)
	Warning(msg)
	Error(msg)
	// we do not test critical as it calls GoExit
	// Critical(msg)
}

func Test_SetLevelByString(t *testing.T) {

	w := &Test1Handler{DefaultFormatter, nil, nil}
	SetHandler(w)

	w.Reset()
	// test levels
	e := SetMinimalLevelByName("DEBUG")
	fmt.Println(level)
	if e != nil {
		t.Fatalf("Could not set level by name: %s", e)
	}
	logAllLevels("Hello world")

	if w.Len() != 4 {
		t.Errorf("Not all levels logged - got  %d messages", w.Len())
	}
	e = SetMinimalLevelByName("ERROR")
	if e != nil {
		t.Fatalf("Could not set level by name: %s", e)
	}
	w.Reset()
	logAllLevels("Hello world")
	if w.Len() != 1 {
		t.Errorf("Not all levels logged - got  %d messages", w.Len())
	}

	e = SetMinimalLevelByName("FOOBAR")
	if e == nil {
		t.Fatalf("This should have raised an error...")
	}

}
func Test_Logging(t *testing.T) {

	w := &Test1Handler{DefaultFormatter, nil, nil}
	SetHandler(w)

	w.Reset()
	// test levels
	SetLevel(0)
	logAllLevels("Hello world")

	if w.Len() > 0 {
		t.Errorf("Got messages for level 0")
	}

	SetLevel(ALL)
	w.Reset()
	logAllLevels("Hello world")
	fmt.Printf("Received %d messages\n", w.Len())
	if w.Len() != 4 {
		fmt.Println(w.messages)
		t.Errorf("Did not log all errors (%d)", w.Len())
	}

	levels := []int{DEBUG, INFO, WARNING, ERROR}

	for l := range levels {

		w.Reset()
		SetLevel(levels[l])
		logAllLevels("Testing")

		if !(w.Len() == 1 || (levels[l] == CRITICAL && w.Len() == 2)) {
			t.Errorf("Wrong number of messages written: %d. level: %d", w.Len(), levels[l])
		}
	}

}

type TestHandler struct {
	output    [][]interface{}
	formatter Formatter
	t         *testing.T
}

func (t *TestHandler) Emit(ctx *MessageContext, message string, args ...interface{}) error {
	t.output = append(t.output, []interface{}{ctx.Level, ctx.File, message, ctx.Line, args})
	fmt.Println(*ctx)
	if ctx.Line <= 0 || ctx.Level == "" {
		t.t.Fatalf("Invalid args")
	}
	return nil
}

func (t *TestHandler) SetFormatter(fmt Formatter) {
	t.formatter = fmt
}

func (l *TestHandler) Output() io.Writer {
	return nil
}

func (l *TestHandler) SetVerbose(v bool) {
}

func (l *TestHandler) Printf(msg string, args ...interface{}) {
}

func (l *TestHandler) Close() {}

func Test_Handler(t *testing.T) {

	handler := &TestHandler{
		make([][]interface{}, 0),
		DefaultFormatter,
		t,
	}
	SetHandler(handler)

	SetLevel(ALL)
	Info("Foo Bar %s", 1)
	Warning("Bar Baz %s", 2)

	if len(handler.output) != 2 {
		t.Fatal("Wrong len of output handler ")
	}

	fmt.Println("Passed testHandler")
}

func TestLogTail(t *testing.T) {
	handler := &TestHandler{
		make([][]interface{}, 0),
		DefaultFormatter,
		t,
	}
	SetHandler(handler)

	SetLevel(ALL)
	Info("Foo Bar %d", 1)
	Warning("Bar Baz %d", 2)

	contents := ReadTail()
	fmt.Println(contents)
	if !strings.Contains(contents, "[INF ") {
		t.Fatal("Tail does not contain '[INF '")
	}
	if !strings.Contains(contents, "] Foo Bar 1") {
		t.Fatal("Tail does not contain '] Foo Bar 1'")
	}
	if !strings.Contains(contents, "[WRN ") {
		t.Fatal("Tail does not contain '[WRN '")
	}
	if !strings.Contains(contents, "] Bar Baz 2") {
		t.Fatal("Tail does not contain '] Bar Baz 2'")
	}
}
