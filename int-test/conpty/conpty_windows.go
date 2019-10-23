package conpty

import (
	"fmt"
	"log"
	"os"
	"syscall"
	"unsafe"
)

var (
	kernel32                              = syscall.NewLazyDLL("kernel32.dll")
	procCreatePseudoConsole               = kernel32.NewProc("CreatePseudoConsole")
	procClosePseudoConsole               = kernel32.NewProc("ClosePseudoConsole")
	procInitializeProcThreadAttributeList = kernel32.NewProc("InitializeProcThreadAttributeList")
	procUpdateProcThreadAttribute         = kernel32.NewProc("UpdateProcThreadAttribute")
	procLocalAlloc                       = kernel32.NewProc("LocalAlloc")
)

type COORD struct {
	X uint16
	Y uint16
}

type WinPtyPipe struct {
	hpCon            syscall.Handle
	pipeFdIn         syscall.Handle
	pipeFdOut        syscall.Handle
	startupInfo      StartupInfoEx
	PipeIn           *os.File
	PipeOut          *os.File
	attributesBuffer syscall.Handle
}

type StartupInfoEx struct {
	startupInfo     syscall.StartupInfo
	lpAttributeList syscall.Handle
}

type ProcThreadAttribute uintptr

const ExtendedStartupinfoPresent uint32 = 0x00080000

const (
	ProcThreadAttributePseudoconsole ProcThreadAttribute = 22 | 0x00020000 // this is the only one we support right now
)

func localAlloc(size uint64) (handle syscall.Handle, err error) {
	r1, _, e1 := syscall.Syscall(procLocalAlloc.Addr(), 2, uintptr(0x0040), uintptr(size), 0)
	if r1 == 0 {
		if e1 != 0 {
			err = e1
		} else {
			err = syscall.EINVAL
		}
		handle = syscall.InvalidHandle
		return
	}
	handle = syscall.Handle(r1)
	return
}

// makeCmdLine builds a command line out of args by escaping "special"
// characters and joining the arguments with spaces.
func makeCmdLine(args []string) string {
	var s string
	for _, v := range args {
		if s != "" {
			s += " "
		}
		s += syscall.EscapeArg(v)
	}
	return s
}

func updateProcThreadAttributeList(attributeList syscall.Handle, attribute ProcThreadAttribute, lpValue *syscall.Handle, lpSize uintptr) (err error) {

	r1, _, e1 := syscall.Syscall9(procUpdateProcThreadAttribute.Addr(), 7, uintptr(attributeList), 0, uintptr(attribute), uintptr(unsafe.Pointer(lpValue)), lpSize, 0, 0, 0, 0)
	if r1 == 0 {
		if e1 != 0 {
			err = e1
		} else {
			err = syscall.EINVAL
		}
	}

	return
}

func initializeProcThreadAttributeList(attributeList *syscall.Handle, attributeCount uint32, listSize *uint64) (err error) {

	if attributeList == nil {
		syscall.Syscall6(procInitializeProcThreadAttributeList.Addr(), 4, 0, uintptr(attributeCount), 0, uintptr(unsafe.Pointer(listSize)), 0, 0)
		return
	}
	r1, _, e1 := syscall.Syscall6(procInitializeProcThreadAttributeList.Addr(), 4, uintptr(*attributeList), uintptr(attributeCount), 0, uintptr(unsafe.Pointer(listSize)), 0, 0)
	if r1 == 0 {
		if e1 != 0 {
			err = e1
		} else {
			err = syscall.EINVAL
		}
	}

	return
}

func closePseudoConsole(handle syscall.Handle) (err error) {
	r1, _, e1 := syscall.Syscall(procClosePseudoConsole.Addr(), 1, uintptr(handle), 0, 0)
	if r1 == 0 {
		if e1 != 0 {
			err = e1
		} else {
			err = syscall.EINVAL
		}
	}

	return
}

func New() *WinPtyPipe {
	return &WinPtyPipe{hpCon: syscall.InvalidHandle, startupInfo: StartupInfoEx{}}
}

func (winpty *WinPtyPipe) Close() {
	err := closePseudoConsole(winpty.hpCon)
	if err != nil {
		log.Fatalf("Failed to close pseudo console: %v", err)
	}
	winpty.PipeIn.Close()
	winpty.PipeOut.Close()
}

func (winpty *WinPtyPipe) ReadStdout(buf []byte) (n uint32, err error) {
	err = syscall.ReadFile(winpty.pipeFdOut, buf, &n, nil)
	return
}

func (winpty *WinPtyPipe) InitializeStartupInfoAttachedToPTY() (err error) {

	var attrListSize uint64
	fmt.Printf("sizeof(startupinfo) = %d\n", unsafe.Sizeof(winpty.startupInfo.startupInfo))
	fmt.Printf("sizeof(startupinfoex) = %d\n", unsafe.Sizeof(winpty.startupInfo))
	winpty.startupInfo.startupInfo.Cb = uint32(unsafe.Sizeof(winpty.startupInfo))

	err = initializeProcThreadAttributeList(nil, 1, &attrListSize)
	if err != nil {
		return fmt.Errorf("could not retrieve list size: %v", err)
	}

	winpty.startupInfo.lpAttributeList, err = localAlloc(attrListSize)  // make([]byte, attrListSize)
	if err != nil {
		return fmt.Errorf("Could not allocate local memory: %v", err)
	}
	fmt.Printf("attrListSize = %d\n", attrListSize)

	err = initializeProcThreadAttributeList(&winpty.startupInfo.lpAttributeList, 1, &attrListSize)
	if err != nil {
		return fmt.Errorf("failed to initialize proc attributes: %v", err)
	}

	fmt.Printf("sizeof(HPCON) = %d\n", unsafe.Sizeof(winpty.hpCon))

	err = updateProcThreadAttributeList(winpty.startupInfo.lpAttributeList, ProcThreadAttributePseudoconsole, &winpty.hpCon, unsafe.Sizeof(winpty.hpCon))

	// winpty.startupInfo.lpAttributeList = 0

	return
}

func (winpty *WinPtyPipe) Spawn(argv []string) (pid int, handle uintptr, err error) {
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
		argvp, err = syscall.UTF16PtrFromString(cmdline)
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

	pi := new(syscall.ProcessInformation)

	flags := uint32(syscall.CREATE_UNICODE_ENVIRONMENT) | ExtendedStartupinfoPresent
	// flags := ExtendedStartupinfoPresent
	// flags := uint32(0)
	fmt.Printf("cb = %d\n", winpty.startupInfo.startupInfo.Cb)
	winpty.startupInfo.startupInfo.Cb = uint32(unsafe.Sizeof(winpty.startupInfo))
	err = syscall.CreateProcess(
		nil,
		argvp,
		nil, // process handle not inheritable
		nil, // thread handles not inheritable,
		false,
		flags,
		nil, // createEnvBlock(attr.Env),
		nil, // use current directory later: dirp,
		&winpty.startupInfo.startupInfo,
		pi)
	if err != nil {
		return 0, 0, err
	}
	log.Printf("before waiting for thread")
	ev, err := syscall.WaitForSingleObject(pi.Process, 20000)
	log.Printf("event was: %d\n", ev)
	if err != nil {
		fmt.Printf("error waiting for object: %v", err)
	}
	defer syscall.CloseHandle(syscall.Handle(pi.Thread))

	return int(pi.ProcessId), uintptr(pi.Process), nil
}

func (winpty *WinPtyPipe) createPseudoConsole(consoleSize *COORD, ptyIn *syscall.Handle, ptyOut *syscall.Handle) (err error) {
	winpty.hpCon = syscall.InvalidHandle
	r1, _, e1 := syscall.Syscall6(procCreatePseudoConsole.Addr(), 5, uintptr(unsafe.Pointer(consoleSize)), uintptr(unsafe.Pointer(ptyIn)), uintptr(unsafe.Pointer(ptyOut)), 0, uintptr(unsafe.Pointer(&winpty.hpCon)), 0)

	if r1 == 0 { // !S_OK
		if e1 != 0 {
			err = fmt.Errorf("Could not create pseudo console: Windows error code: %d", e1)
		} else {
			err = fmt.Errorf("Could not create pseudo console: Unknown windows error")
		}
	}
	return
}

func (wpty *WinPtyPipe) CreatePseudoConsoleAndPipes() (err error) {
	var hPipePTYIn syscall.Handle
	var hPipePTYOut syscall.Handle

	if err := syscall.CreatePipe(&hPipePTYIn, &wpty.pipeFdIn, nil, 0); err != nil {
		log.Fatalf("Failed to create pipe to vt output: %v", err)
	}
	if err := syscall.CreatePipe(&wpty.pipeFdOut, &hPipePTYOut, nil, 0); err != nil {
		log.Fatalf("Failed to create vt input pipe: %v", err)
	}

	consoleSize := &COORD{X: 120, Y: 40}

	err = wpty.createPseudoConsole(consoleSize, &hPipePTYIn, &hPipePTYOut)
	if err != nil {
		return fmt.Errorf("failed to create pseudo console: %v", err)
	}

	// Note: We can close the handles to the PTY-end of the pipes here
	// because the handles are dup'ed into the ConHost and will be released
	// when the ConPTY is destroyed.
	if hPipePTYOut != syscall.InvalidHandle {
		syscall.CloseHandle(hPipePTYOut)
	}
	if hPipePTYIn != syscall.InvalidHandle {
		syscall.CloseHandle(hPipePTYIn)
	}

	t, err := syscall.GetFileType(wpty.pipeFdOut)
	if err != nil {
		fmt.Printf("error get file type: %v", err)
	}
	fmt.Printf("t = %d\n", t)
	wpty.PipeIn = os.NewFile(uintptr(wpty.pipeFdIn), "|0")
	wpty.PipeOut = os.NewFile(uintptr(wpty.pipeFdOut), "|1")

	return
}

func spawn() {

	var sI syscall.StartupInfo
	var pI syscall.ProcessInformation

	argv := syscall.StringToUTF16Ptr(".\\build\\state.exe")
	syscall.CreateProcess(nil, argv, nil, nil, false, 0, nil, nil, &sI, &pI)
}
