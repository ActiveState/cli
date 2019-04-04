package print_test

import (
	"testing"

	"github.com/ActiveState/cli/internal/print"
	"github.com/kami-zh/go-capturer"
	"github.com/stretchr/testify/suite"
)

type printMethod func(format string, a ...interface{})
type captureMethod func(f func()) string

type PrintTestSuite struct {
	suite.Suite
}

type Writer struct {
	Out string
}

func (w *Writer) Write(p []byte) (n int, err error) {
	w.Out = w.Out + string(p)
	return 0, nil
}

func (suite *PrintTestSuite) assertMethod(method printMethod, capturer captureMethod, suffix string) {
	out := capturer(func() {
		method("Hello %s", "World")
	})
	suite.Contains(out, "Hello World"+suffix)
}

func (suite *PrintTestSuite) TestLine() {
	suite.assertMethod(print.Line, capturer.CaptureStdout, "\n")
}

func (suite *PrintTestSuite) TestLineSingleArg() {
	out := capturer.CaptureStdout(func() {
		print.Line("Hello World")
	})
	suite.Contains(out, "Hello World\n")
}

func (suite *PrintTestSuite) TestError() {
	suite.assertMethod(print.Error, capturer.CaptureStderr, "\n")
}

func (suite *PrintTestSuite) TestWarning() {
	suite.assertMethod(print.Warning, capturer.CaptureStdout, "\n")
}

func (suite *PrintTestSuite) TestInfo() {
	suite.assertMethod(print.Info, capturer.CaptureStdout, "\n")
}

func (suite *PrintTestSuite) TestBold() {
	suite.assertMethod(print.Bold, capturer.CaptureStdout, "\n")
}

func (suite *PrintTestSuite) TestBoldInline() {
	suite.assertMethod(print.BoldInline, capturer.CaptureStdout, "")
}

func (suite *PrintTestSuite) TestStdout() {
	suite.assertMethod(func(f string, a ...interface{}) { print.Stdout().Line(f, a...) }, capturer.CaptureStdout, "\n")
}

func (suite *PrintTestSuite) TestStderr() {
	suite.assertMethod(func(f string, a ...interface{}) { print.Stderr().Line(f, a...) }, capturer.CaptureStderr, "\n")
}

func (suite *PrintTestSuite) TestCustomOutput() {
	writer := &Writer{}
	printer := print.New(writer, true)

	printer.Line("Hello %s", "World")
	suite.Contains(writer.Out, "Hello World\n")
}

func (suite *PrintTestSuite) TestCustomOutputWithColors() {
	writer := &Writer{}
	printer := print.New(writer, false)

	// This test is a little pointless in terms of its assertions, I'm mainly adding it to test that colors aren't causing
	// panics. It might be good to test for the actual color output, but that has cross-platform implications that
	// I don't want to tackle right now.
	printer.Warning("Hello %s", "World")
	printer.Error("Hello %s", "World")
	printer.Info("Hello %s", "World")
	printer.Bold("Hello %s", "World")
	printer.BoldInline("Hello %s", "World")
	suite.Contains(writer.Out, "Hello World\n")
}

func TestPrintTestSuite(t *testing.T) {
	suite.Run(t, new(PrintTestSuite))
}
