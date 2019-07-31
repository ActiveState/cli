// +build windows

package expect

import (
	"bufio"
	"bytes"
	"log"
	"os"
	"unicode/utf8"

	"github.com/iamacarpet/go-winpty"
	"github.com/kballard/go-shellquote"
)

func (p *Process) start() error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	args := append([]string{p.command}, p.args...)
	p.winpty, err = winpty.OpenWithOptions(winpty.Options{
		DLLPrefix: "C:\\Users\\nrijk\\go\\src\\github.com\\ActiveState\\cli\\pkg\\expect",
		Command:   shellquote.Join(args...),
		Dir:       wd,
		Env:       p.env,
	})
	if err != nil {
		return err
	}
	p.winpty.SetSize(800, 60)

	// Wait for output
	go func() {
		buf := make([]byte, 8192)
		reader := bufio.NewReader(p.winpty.StdOut)
		var buffer bytes.Buffer
		for {
			n, err := reader.Read(buf)
			if err != nil {
				log.Printf("Failed to read from pty master: %s", err)
				return
			}

			//read byte array as Unicode code points (rune in go)
			bufferBytes := buffer.Bytes()
			runeReader := bufio.NewReader(bytes.NewReader(append(bufferBytes[:], buf[:n]...)))
			buffer.Reset()

			i := 0
			for i < n {
				char, charLen, e := runeReader.ReadRune()
				if e != nil {
					log.Printf("Failed to read from pty master: %s", err)
					return
				}

				if char == utf8.RuneError {
					runeReader.UnreadRune()
					break
				}

				i += charLen
				buffer.WriteRune(char)
			}

			p.stdout = p.stdout + string(buffer.Bytes())
			p.combined = p.combined + string(buffer.Bytes())
			p.onOutput(buffer.Bytes())
			p.onStdout(buffer.Bytes())

			buffer.Reset()
			if i < n {
				buffer.Write(buf[i:n])
			}
		}
	}()

	return nil
}

func (p *Process) setupStdin() {
}

func (p *Process) setupStderr() {
}

func (p *Process) setupStdout() {
}

func (p *Process) wait() error {
	for p.running {
	}
	return nil
}

func (p *Process) close() error {
	p.winpty.Close()
	return nil
}

func (p *Process) quit() error {
	return p.close()
}

func (p *Process) exit() error {
	return p.close()
}

func (p *Process) exitCode() int {
	return 0
}

func (p *Process) Write(input string) error {
	_, err := p.winpty.StdIn.WriteString(input)
	return err
}
