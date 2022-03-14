package colorstyle

/*
Sourced from: https://github.com/daviddengcn/go-colortext
See license.txt in same directory for license information
*/

import (
	"io"
	"os"
	"syscall"
	"unsafe"

	"github.com/ActiveState/cli/internal/logging"
	"github.com/ActiveState/cli/internal/multilog"
	"github.com/ActiveState/cli/internal/rollbar"
)

var consoleStyleMap = map[Style]uint16{
	Black:   0,
	Red:     consoleRed,
	Green:   consoleGreen,
	Yellow:  consoleRed | consoleGreen,
	Blue:    consoleBlue,
	Magenta: consoleRed | consoleBlue,
	Cyan:    consoleGreen | consoleBlue,
	White:   consoleRed | consoleGreen | consoleBlue}

const (
	consoleBlue      = uint16(0x0001)
	consoleGreen     = uint16(0x0002)
	consoleRed       = uint16(0x0004)
	consoleIntensity = uint16(0x0008)

	consoleColorMask = consoleBlue | consoleGreen | consoleRed | consoleIntensity
)

const (
	stdOutHandle = uint32(-11 & 0xFFFFFFFF)
)

type consoleBufferDimensions struct {
	X, Y int16
}

type consoleBuffer struct {
	DwSize           consoleBufferDimensions
	DwCursorPosition consoleBufferDimensions
	WAttributes      uint16
	SrWindow         struct {
		Left, Top, Right, Bottom int16
	}
	DwMaximumWindowSize consoleBufferDimensions
}

var (
	kernel32 = syscall.NewLazyDLL("kernel32.dll")

	procGetStdHandle               = kernel32.NewProc("GetStdHandle")
	procSetconsoleTextAttribute    = kernel32.NewProc("SetConsoleTextAttribute")
	procGetconsoleScreenBufferInfo = kernel32.NewProc("GetConsoleScreenBufferInfo")

	hStdout    uintptr
	bufferInfo *consoleBuffer
)

func init() {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	procGetStdHandle = kernel32.NewProc("GetStdHandle")
	hStdout, _, _ = procGetStdHandle.Call(uintptr(stdOutHandle))
	bufferInfo = getConsoleScreenBufferInfo(hStdout)
	syscall.LoadDLL("")
}

type Styler struct {
}

func New(writer io.Writer) *Styler {
	return &Styler{}
}

func (st *Styler) SetStyle(s Style, bright bool) {
	if bufferInfo == nil {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			if os.Getenv("CI") == "" {
				multilog.Log(logging.ErrorNoStacktrace, rollbar.Error)("colorstyle.SetStyle failed with: %v", r)
			}
		}
	}()

	if s == Bold || s == Underline {
		return // underline/bold is not supported on windows
	}

	attr := uint16(0)
	if s == Default || s == Reset {
		attr = bufferInfo.WAttributes
	} else if s == Dim {
		attr = attr & ^consoleColorMask | consoleStyleMap[Black]
		bright = true
	} else {
		if style, ok := consoleStyleMap[s]; ok {
			attr = attr & ^consoleColorMask | style
		}
	}
	if bright {
		attr |= consoleIntensity
	}
	setConsoleTextAttribute(hStdout, attr)
}

func setConsoleTextAttribute(hconsoleOutput uintptr, wAttributes uint16) bool {
	ret, _, _ := procSetconsoleTextAttribute.Call(
		hconsoleOutput,
		uintptr(wAttributes))
	return ret != 0
}

func getConsoleScreenBufferInfo(hconsoleOutput uintptr) *consoleBuffer {
	var csbi consoleBuffer
	if ret, _, _ := procGetconsoleScreenBufferInfo.Call(hconsoleOutput, uintptr(unsafe.Pointer(&csbi))); ret == 0 {
		return nil
	}
	return &csbi
}
