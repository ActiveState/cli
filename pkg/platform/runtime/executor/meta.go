package executor

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/ActiveState/cli/internal/svcctl"
	"github.com/ActiveState/cli/pkg/project"
)

/*
::sock::/tmp/state-ipc/state-ipts.DX-1060.sock
::bin::/home/daved/code/src/github.com/ActiveState/.home/.cache/28cd50f1/bin
::env::TESTER="test/best"::env::BESTER="example"
::commit-id::1234abcd-1234-abcd-1234-abcd1234
::namespace::ActiveState/Test
::headless::true
*/

var (
	metaFileName = "meta.as"

	sockDelim      = "::sock::"
	binDelim       = "::bin::"
	envDelim       = "::env::"
	commitDelim    = "::commit-id::"
	namespaceDelim = "::namespace::"
	headlessDelim  = "::headless::"
)

type Meta struct {
	SockPath   string
	BinDir     string
	Env        map[string]string
	CommitUUID string
	Namespace  string
	Headless   bool
}

func NewMeta(env map[string]string, t Targeter) *Meta {
	commitID := t.CommitUUID().String()
	return &Meta{
		SockPath:   svcctl.NewIPCSockPathFromGlobals().String(),
		Env:        env,
		CommitUUID: commitID,
		Namespace:  project.NewNamespace(t.Owner(), t.Name(), commitID).String(),
		Headless:   t.Headless(),
	}
}

func NewMetaFromReader(r io.Reader) (*Meta, error) {
	m := Meta{}

	scnr := bufio.NewScanner(r)
	iter := -1
	for scnr.Scan() {
		iter++
		txt := scnr.Text()

		switch iter {
		case 0:
			m.SockPath = strings.TrimPrefix(txt, sockDelim)
		case 1:
			m.BinDir = strings.TrimPrefix(txt, binDelim)
		case 2:
			envMap := make(map[string]string)
			envTxt := strings.TrimPrefix(txt, envDelim)
			envSplit := strings.Split(envTxt, envDelim)
			for _, kv := range envSplit {
				kvSplit := strings.SplitN(kv, "=", 2)
				if len(kvSplit) < 2 {
					return nil, errors.New("env data malformed")
				}
				envMap[kvSplit[0]] = kvSplit[1]
			}
			m.Env = envMap
		case 3:
			m.CommitUUID = strings.TrimPrefix(txt, commitDelim)
		case 4:
			m.Namespace = strings.TrimPrefix(txt, namespaceDelim)
		case 5:
			boolTxt := strings.TrimPrefix(txt, headlessDelim)
			headless, err := strconv.ParseBool(boolTxt)
			if err != nil {
				return nil, err
			}
			m.Headless = headless
		default:
			return nil, errors.New("unexpected line in meta file")
		}

	}
	if err := scnr.Err(); err != nil {
		return nil, err
	}

	return &m, nil
}

func (m *Meta) WriteTo(w io.Writer) (int, error) {
	aw := makeAccumulatingWrite(w)

	aw.fprintf("%s%s\n", sockDelim, m.SockPath)
	aw.fprintf("%s%s\n", binDelim, m.BinDir)
	for k, v := range m.Env {
		aw.fprintf("%s%s=%s", envDelim, k, v)
	}
	aw.fprintf("\n")
	aw.fprintf("%s%s\n", commitDelim, m.CommitUUID)
	aw.fprintf("%s%s\n", namespaceDelim, m.Namespace)
	aw.fprintf("%s%t\n", headlessDelim, m.Headless)

	return aw.total()
}

type accumulatingWrite struct {
	w   io.Writer
	n   int
	err error
}

func makeAccumulatingWrite(w io.Writer) accumulatingWrite {
	return accumulatingWrite{
		w: w,
	}
}

func (aw accumulatingWrite) fprintf(format string, as ...interface{}) {
	if aw.err != nil {
		return
	}
	n, err := fmt.Fprintf(aw.w, format, as...)
	if err != nil {
		aw.err = err
		return
	}
	aw.n += n
}

func (aw accumulatingWrite) total() (int, error) {
	return aw.n, aw.err
}
