package xpty

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/ActiveState/vt10x"
)

type Xpty struct {
	*impl       // os specific
	Term        *vt10x.VT
	State       *vt10x.State
	rwPipe      *readWritePipe
	termOutPipe io.Reader
}

// readWritePipe is a helper that we use to let the application communicate with a virtual terminal.
type readWritePipe struct {
	r *io.PipeReader
	w *io.PipeWriter
}

func newReadWritePipe() *readWritePipe {
	r, w := io.Pipe()
	return &readWritePipe{r, w}
}

// Read from the reader part of the pipe
func (rw *readWritePipe) Read(buf []byte) (int, error) {
	return rw.r.Read(buf)
}

// Write to the writer part of the pipe
func (rw *readWritePipe) Write(buf []byte) (int, error) {
	return rw.w.Write(buf)
}

// Close all parts of the pipe
func (rw *readWritePipe) Close() error {
	var errMessage string
	e := rw.r.Close()
	if e != nil {
		errMessage += fmt.Sprintf("failed to close read-part of pipe: %v ", e)
	}
	e = rw.w.Close()
	if e != nil {
		errMessage += fmt.Sprintf("failed to close write-part of pipe: %v ", e)
	}
	if len(errMessage) > 0 {
		return fmt.Errorf(errMessage)
	}
	return nil
}

func (xp *Xpty) openVT(cols uint16, rows uint16) (err error) {

	/*
			 We are creating a communication pipe to handle DSR (device status report) and
			 (CPR) cursor position report queries.

			 If an application is sending these queries it is usually expecting a response
		     from the terminal emulator (like xterm). If the response is not send, the
		     application may hang forever waiting for it. The vt10x terminal emulator is able to handle it. If
		     we multiplex the ptm output to a vt10x terminal, the DSR/CPR requests are
		     intercepted and it can inject the responses in the read-write-pipe.

			 The read-part of the read-write-pipe continuously feeds into the ptm device that
			 forwards it to the application.

			      DSR/CPR req                        reply
			 app ------------->  pts/ptm -> vt10x.VT ------> rwPipe --> ptm/pts --> app

			 Note: This is a simplification from github.com/hinshun/vt10x (console.go)
	*/

	xp.rwPipe = newReadWritePipe()

	// Note: the Term instance also closes the rwPipe
	xp.Term, err = vt10x.Create(xp.State, xp.rwPipe)
	if err != nil {
		return err
	}
	xp.Term.Resize(int(cols), int(rows))

	// connect the pipes as described above
	go func() {
		// this drains the rwPipe continuously.  If that didn't happen, we would block on write.
		io.Copy(xp.impl.terminalInPipe(), xp.rwPipe)
	}()

	// duplicate the terminal output pipe: write to vt terminal everything that is being read from it.
	xp.termOutPipe = io.TeeReader(xp.impl.terminalOutPipe(), xp.Term)
	return nil
}

func Open(cols uint16, rows uint16) (*Xpty, error) {
	xpImpl, err := open(cols, rows)
	if err != nil {
		return nil, err
	}
	xp := &Xpty{impl: xpImpl, Term: nil, State: &vt10x.State{}}
	err = xp.openVT(cols, rows)
	if err != nil {
		return nil, err
	}

	return xp, nil
}

func (p *Xpty) TerminalOutPipe() io.Reader {
	return p.termOutPipe
}

func (p *Xpty) TerminalInPipe() io.Writer {
	return p.impl.terminalInPipe()
}

func (p *Xpty) Close() error {
	err := p.impl.close()
	if err != nil {
		return err
	}
	if p.Term == nil {
		return nil
	}
	return p.Term.Close()
}

func (p *Xpty) Tty() *os.File {
	return p.impl.tty()
}

func (p *Xpty) TerminalOutFd() uintptr {
	return p.impl.terminalOutFd()
}
func (p *Xpty) StartProcessInTerminal(cmd *exec.Cmd) error {
	return p.impl.startProcessInTerminal(cmd)
}
