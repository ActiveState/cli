// Package conpty provides functions for creating a process attached to a
// ConPTY pseudo-terminal.  This allows the process to call console specific
// API functions without an actual terminal being present.
//
// The concept is best explained in this blog post:
// https://devblogs.microsoft.com/commandline/windows-command-line-introducing-the-windows-pseudo-console-conpty/
package conpty

import (
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ConPty represents a windows pseudo console.
// Attach a process to it by calling the Spawn() method.
// You can send UTF encoded commands to it with Write() and listen to
// its output stream by accessing the output pipe via OutPipe()
type ConPty struct {
	hpCon               *windows.Handle
	pipeFdIn            windows.Handle
	pipeFdOut           windows.Handle
	startupInfo         startupInfoEx
	consoleSize         uintptr
	PipeIn              *os.File
	PipeOut             *os.File
	attributeListBuffer []byte
}

// makeCmdLine builds a command line out of args by escaping "special"
// characters and joining the arguments with spaces.
func makeCmdLine(args []string) string {
	var s string
	for _, v := range args {
		if s != "" {
			s += " "
		}
		s += windows.EscapeArg(v)
	}
	return s
}

func New(X int16, Y int16) *ConPty {
	return &ConPty{
		hpCon:       new(windows.Handle),
		startupInfo: startupInfoEx{},
		consoleSize: uintptr(X) + (uintptr(Y) << 16),
	}
}

func (winpty *ConPty) Close() (err error) {
	err = deleteProcThreadAttributeList(winpty.startupInfo.lpAttributeList)
	if err != nil {
		log.Printf("Failed to free delete proc thread attribute list: %v", err)
	}
	/*
		_, err = syscall.LocalFree(winpty.startupInfo.lpAttributeList)
		if err != nil {
			log.Printf("Failed to free the lpAttributeList")
		}
	*/
	err = closePseudoConsole(*winpty.hpCon)
	if err != nil {
		log.Printf("Failed to close pseudo console: %v", err)
	}
	winpty.PipeIn.Close()
	winpty.PipeOut.Close()
	return
}

func (winpty *ConPty) ReadStdout(buf []byte) (n uint32, err error) {
	err = windows.ReadFile(winpty.pipeFdOut, buf, &n, nil)
	return
}

func (winpty *ConPty) InitializeStartupInfoAttachedToPTY() (err error) {

	var attrListSize uint64
	fmt.Printf("sizeof(startupinfo) = %d\n", unsafe.Sizeof(winpty.startupInfo.startupInfo))
	fmt.Printf("sizeof(startupinfoex) = %d\n", unsafe.Sizeof(winpty.startupInfo))
	winpty.startupInfo.startupInfo.Cb = uint32(unsafe.Sizeof(winpty.startupInfo))

	err = initializeProcThreadAttributeList(0, 1, &attrListSize)
	if err != nil {
		return fmt.Errorf("could not retrieve list size: %v", err)
	}

	winpty.attributeListBuffer = make([]byte, attrListSize)
	// winpty.startupInfo.lpAttributeList, err = localAlloc(attrListSize)
	winpty.startupInfo.lpAttributeList = windows.Handle(unsafe.Pointer(&winpty.attributeListBuffer[0]))
	if err != nil {
		return fmt.Errorf("Could not allocate local memory: %v", err)
	}
	fmt.Printf("attrListSize = %d @ %d\n", attrListSize, winpty.startupInfo.lpAttributeList)
	fmt.Printf("listBuffer: %s\n", hex.EncodeToString(winpty.attributeListBuffer))

	err = initializeProcThreadAttributeList(uintptr(winpty.startupInfo.lpAttributeList), 1, &attrListSize)
	if err != nil {
		return fmt.Errorf("failed to initialize proc attributes: %v", err)
	}

	fmt.Printf("%d\n", procThreadAttributePseudoconsole)
	fmt.Printf("sizeof(HPCON) = %d, %d\n", unsafe.Sizeof(*winpty.hpCon), uintptr(*winpty.hpCon))
	fmt.Printf("listBuffer: %s\n", hex.EncodeToString(winpty.attributeListBuffer))

	err = updateProcThreadAttributeList(
		winpty.startupInfo.lpAttributeList,
		procThreadAttributePseudoconsole,
		*winpty.hpCon,
		unsafe.Sizeof(*winpty.hpCon))
	if err != nil {
		return fmt.Errorf("failed to update proc attributes: %v", err)
	}

	fmt.Printf("listBuffer: %s\n", hex.EncodeToString(winpty.attributeListBuffer))

	// winpty.startupInfo.lpAttributeList = 0

	return
}

func (winpty *ConPty) Spawn(argv []string) (pid int, handle uintptr, err error) {
	/*
		if len(argv0) == 0 {
			return 0, 0, syscall.EWINDOWS
		}
		/*
			if len(attr.Dir) != 0 {
				// StartProcess assumes that argv0 is relative to attr.Dir,
				// because it implies Chdir(attr.Dir) before executing argv0.
				// Windows CreateProcess assumes the opposite: it looks for
				// argv0 relative to the current directory, and, only once the new
				// process is started, it does Chdir(attr.Dir). We are adjusting
				// for that difference here by making argv0 absolute.
				var err error
				argv0, err = joinExeDirAndFName(attr.Dir, argv0)
				if err != nil {
					return 0, 0, err
				}
			}
	*/
	/*
		argv0p, err := syscall.UTF16PtrFromString(argv0)
		if err != nil {
			return 0, 0, err
		}
	*/

	cmdline := makeCmdLine(argv)

	var argvp *uint16
	if len(cmdline) != 0 {
		argvp, err = windows.UTF16PtrFromString(cmdline)
		if err != nil {
			return 0, 0, err
		}
	}

	/*
		var dirp *uint16
		if len(attr.Dir) != 0 {
			dirp, err = UTF16PtrFromString(attr.Dir)
			if err != nil {
				return 0, 0, err
			}
		}
	*/

	/*
		// Acquire the fork lock so that no other threads
		// create new fds that are not yet close-on-exec
		// before we fork.
		ForkLock.Lock()
		defer ForkLock.Unlock()

		p, _ := GetCurrentProcess()
		fd := make([]Handle, len(attr.Files))
		for i := range attr.Files {
			if attr.Files[i] > 0 {
				err := DuplicateHandle(p, Handle(attr.Files[i]), p, &fd[i], 0, true, DUPLICATE_SAME_ACCESS)
				if err != nil {
					return 0, 0, err
				}
				defer CloseHandle(Handle(fd[i]))
			}
		}
	*/
	// si.Cb = uint32(unsafe.Sizeof(*si))
	// si.Flags = STARTF_USESTDHANDLES
	/*
		if sys.HideWindow {
			si.Flags |= STARTF_USESHOWWINDOW
			si.ShowWindow = SW_HIDE
		}
		si.StdInput = fd[0]
		si.StdOutput = fd[1]
		si.StdErr = fd[2]
	*/

	// winpty.startupInfo.startupInfo.Flags = syscall.STARTF_USESTDHANDLES

	pi := new(windows.ProcessInformation)

	flags := uint32(windows.CREATE_UNICODE_ENVIRONMENT) | extendedStartupinfoPresent
	// flags := ExtendedStartupinfoPresent
	// flags := uint32(0)
	fmt.Printf("cb = %d, flags=%d\n", winpty.startupInfo.startupInfo.Cb, flags)

	/*
		var zeroSec syscall.SecurityAttributes
		pSec := &syscall.SecurityAttributes{Length: uint32(unsafe.Sizeof(zeroSec))}
		tSec := &syscall.SecurityAttributes{Length: uint32(unsafe.Sizeof(zeroSec))}
	*/
	fmt.Printf("%d == %d ? \n", uintptr(unsafe.Pointer(&winpty.startupInfo.startupInfo)), uintptr(unsafe.Pointer(&winpty.startupInfo)))

	// winpty.startupInfo.startupInfo.Cb = uint32(unsafe.Sizeof(winpty.startupInfo))
	r1, _, e1 := procCreateProcessW.Call(
		0,
		uintptr(unsafe.Pointer(argvp)),
		0, // process handle not inheritable
		0, // thread handles not inheritable,
		uintptr(0),
		uintptr(flags),
		0, // createEnvBlock(attr.Env),
		0, // use current directory later: dirp,
		uintptr(unsafe.Pointer(&winpty.startupInfo.startupInfo)),
		uintptr(unsafe.Pointer(pi)))
	if r1 == 0 {
		err = e1
	}
	if err != nil {
		return 0, 0, err
	}
	time.Sleep(1 * time.Second)
	/*ev, err := syscall.WaitForSingleObject(pi.Process, 20000)
	fmt.Printf("event was: %d\n", ev)
	*/
	if err != nil {
		fmt.Printf("error waiting for object: %v", err)
	}
	/*
		defer syscall.CloseHandle(syscall.Handle(pi.Thread))
		defer syscall.CloseHandle(syscall.Handle(pi.Process))
	*/

	return int(pi.ProcessId), uintptr(pi.Process), nil
}

func (wpty *ConPty) CreatePseudoConsoleAndPipes() (err error) {
	var hPipePTYIn windows.Handle
	var hPipePTYOut windows.Handle

	if err := windows.CreatePipe(&hPipePTYIn, &wpty.pipeFdIn, nil, 0); err != nil {
		log.Fatalf("Failed to create PTY input pipe: %v", err)
	}
	if err := windows.CreatePipe(&wpty.pipeFdOut, &hPipePTYOut, nil, 0); err != nil {
		log.Fatalf("Failed to create PTY output pipe: %v", err)
	}

	fmt.Printf("pipe handles = %d, %d, invalidHandle=%d\n", uintptr(hPipePTYIn), uintptr(hPipePTYOut), uintptr(syscall.InvalidHandle))

	err = createPseudoConsole(wpty.consoleSize, hPipePTYIn, hPipePTYOut, wpty.hpCon)
	if err != nil {
		return fmt.Errorf("failed to create pseudo console: %d, %v", uintptr(*wpty.hpCon), err)
	}

	fmt.Printf("hpcon = %d\n", uintptr(*wpty.hpCon))

	// Note: We can close the handles to the PTY-end of the pipes here
	// because the handles are dup'ed into the ConHost and will be released
	// when the ConPTY is destroyed.
	if hPipePTYOut != windows.InvalidHandle {
		windows.CloseHandle(hPipePTYOut)
	}
	if hPipePTYIn != windows.InvalidHandle {
		windows.CloseHandle(hPipePTYIn)
	}

	t, err := windows.GetFileType(wpty.pipeFdOut)
	if err != nil {
		fmt.Printf("error get file type: %v", err)
	}
	fmt.Printf("t = %d\n", t)
	wpty.PipeIn = os.NewFile(uintptr(wpty.pipeFdIn), "|0")
	wpty.PipeOut = os.NewFile(uintptr(wpty.pipeFdOut), "|1")

	return
}
