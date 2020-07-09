package fileutils

import (
	"bytes"
	"io"
	"os"

	"github.com/ActiveState/cli/internal/errs"
)

type tokenStreamScanner struct {
	find    []byte
	replace []byte
	r       io.Reader
	w       io.Writer
	bufs    [][]byte
	rpos    []int
	gpos    []int
	wpos    int
	ns      []int
	nw      int
	ci      int
	atEOF   bool
}

func newTokenStreamScanner(r io.Reader, w io.Writer, find, replace []byte) *tokenStreamScanner {
	bufLen := 100 * 1024
	buf := make([]byte, 2*bufLen)
	return &tokenStreamScanner{
		find:    find,
		replace: replace,
		r:       r,
		w:       w,
		bufs:    [][]byte{buf[0:bufLen], buf[bufLen:]},
		rpos:    make([]int, 2),
		gpos:    make([]int, 2),
		ns:      make([]int, 2),
	}

}

func (tss *tokenStreamScanner) nextByte() (int, error) {
	if tss.rpos[tss.ci] < tss.ns[tss.ci] {
		i, err := tss.matchesStringAtPos()
		if i < 0 {
			tss.rpos[tss.ci]++
		}
		return i, err
	}

	// fmt.Printf("writing 4: %q\n", tss.bufs[tss.ci][tss.wpos:tss.ns[tss.ci]])
	n, err := tss.w.Write(tss.bufs[tss.ci][tss.wpos:tss.ns[tss.ci]])
	tss.wpos = 0
	if err != nil {
		return -1, err
	}
	tss.nw += n
	nci := 1 - tss.ci
	err = tss.readBuffer(nci)
	if err != nil {
		return -1, err
	}

	tss.ci = nci
	i, err := tss.matchesStringAtPos()
	return i, err
}

func (tss *tokenStreamScanner) readBuffer(ci int) error {
	oci := 1 - ci

	if tss.atEOF {
		return io.EOF
	}

	if tss.gpos[ci] > tss.gpos[oci] {
		return nil
	}

	n, err := tss.r.Read(tss.bufs[ci])
	if n == 0 && err == io.EOF {
		tss.atEOF = true
	}
	if err != nil {
		return err
	}
	tss.ns[ci] = n
	tss.gpos[ci] = tss.gpos[oci] + tss.ns[oci]
	tss.rpos[ci] = 0
	return nil
}

func (tss *tokenStreamScanner) matchesStringAtPos() (int, error) {
	i := 0
	oci := tss.ci
	orpos := tss.rpos[oci]
	ci := oci
	rpos := orpos
	n := tss.ns[ci]
	lastChSlash := false
	for {
		if rpos == n {
			ci = 1 - ci
			err := tss.readBuffer(ci)
			if err != nil {
				if err == io.EOF {
					return -1, err
				}
				return -1, err
			}
			rpos = 0
			n = tss.ns[ci]
		}
		if i == len(tss.find) {
			tss.ci = ci
			// fmt.Printf("writing 5: %q\n", tss.bufs[oci][tss.wpos:orpos])
			n, err := tss.w.Write(tss.bufs[oci][tss.wpos:orpos])
			tss.rpos[ci] = rpos
			tss.wpos = rpos
			tss.nw += n
			return i, err
		}
		b := tss.bufs[ci][rpos]
		rpos++
		if b == '\\' {
			if lastChSlash {
				continue
			}
			lastChSlash = true
		} else {
			lastChSlash = false
		}
		if b != tss.find[i] {
			return -1, nil
		}
		i++
	}
}

func (tss *tokenStreamScanner) advanceTillNulByte() error {
	for {
		rpos := tss.rpos[tss.ci]

		b := tss.bufs[tss.ci][rpos]
		if b == nullByte {
			// fmt.Printf("writing 1: %q\n", tss.bufs[tss.ci][tss.wpos:rpos])
			n, err := tss.w.Write(tss.bufs[tss.ci][tss.wpos:rpos])
			tss.nw += n
			tss.wpos = rpos
			tss.rpos[tss.ci] = rpos
			return err
		}
		if rpos < tss.ns[tss.ci]-1 {
			tss.rpos[tss.ci]++
			continue
		}

		// fmt.Printf("writing 2: %q\n", tss.bufs[tss.ci][tss.wpos:tss.ns[tss.ci]])
		n, err := tss.w.Write(tss.bufs[tss.ci][tss.wpos:tss.ns[tss.ci]])
		tss.wpos = 0
		if err != nil {
			return err
		}
		tss.nw += n

		nci := 1 - tss.ci
		err = tss.readBuffer(nci)
		if err != nil {
			return err
		}

		tss.ci = nci
	}
}

func (tss *tokenStreamScanner) padWithNulBytes(num int) error {
	zeros := make([]byte, num)
	// fmt.Printf("writing %d zeros\n", num)
	n, err := tss.w.Write(zeros)
	tss.nw += n
	return err
}

func (tss *tokenStreamScanner) scan() (bool, int, error) {
	var replaced bool

	n, err := tss.r.Read(tss.bufs[0])
	if n == 0 && err == io.EOF {
		tss.atEOF = true
	}
	if err != nil {
		return replaced, tss.nw, err
	}
	tss.ns[0] = n
	for {
		l, err := tss.nextByte()
		if err == io.EOF {
			/*
				fmt.Printf("writing 3: %q\n", tss.bufs[tss.ci][tss.wpos:tss.rpos[tss.ci]])
				n, _ := tss.w.Write(tss.bufs[tss.ci][tss.wpos:tss.rpos[tss.ci]])
				tss.nw += n
			*/
		}
		if err != nil {
			return replaced, tss.nw, err
		}
		if l == -1 {
			continue
		}

		replaced = true

		if err != nil {
			return replaced, tss.nw, err
		}

		if l < len(tss.replace) {
			return replaced, tss.nw, errs.New("Replacement string is too long")
		}

		// write replacement string
		// fmt.Printf("writing replacement %q\n", replace)
		n, err = tss.w.Write(tss.replace)
		tss.nw += n
		if err != nil {
			return replaced, tss.nw, err
		}

		err = tss.advanceTillNulByte()
		if err != nil {
			return replaced, tss.nw, err
		}

		err = tss.padWithNulBytes(l - len(tss.replace))
		if err != nil {
			return replaced, tss.nw, err
		}
	}
}
func ReplaceNulTerminatedPathStream(r io.Reader, oldpath, newpath string) (int, []byte, error) {
	init := make([]byte, 0, 10000000)
	w := bytes.NewBuffer(init)

	tss := newTokenStreamScanner(r, w, []byte(oldpath), []byte(newpath))

	_, _, err := tss.scan()
	if err != io.EOF {
		return 0, nil, err
	}

	return 1, w.Bytes(), nil
}

func isBinaryStream(f *os.File) bool {
	b := make([]byte, 1024)
	for {
		n, err := f.Read(b)
		if n == 0 && err == io.EOF {
			return false
		}
		if bytes.IndexByte(b[:n], nullByte) != -1 {
			return true
		}
	}
}
