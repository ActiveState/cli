// Copyright 2016 Alan Shreve github@inconshreveable.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package legacyupd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/ActiveState/cli/internal/logging"
)

// fromStream includes source code from: https://github.com/inconshreveable/go-update/blob/master/apply.go#L48
func (u *Updater) fromStream(path string, updateWith io.Reader) (err error, errRecover error) {
	// Copy the contents of of newbinary to a the new executable file
	updateDir := filepath.Dir(path)
	newPath := filepath.Join(updateDir, fmt.Sprintf(".%s.new", "state"))
	fp, err := os.OpenFile(newPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return
	}
	defer fp.Close()
	_, err = io.Copy(fp, updateWith)
	if err != nil {
		return
	}

	// if we don't call fp.Close(), windows won't let us move the new executable
	// because the file will still be "in use"
	err = fp.Close()
	if err != nil {
		return
	}

	// this is where we'll move the executable to so that we can swap in the updated replacement
	oldPath := filepath.Join(updateDir, fmt.Sprintf(".%s.old", "state"))

	// delete any existing old exec file - this is necessary on Windows for two reasons:
	// 1. after a successful update, Windows can't remove the .old file because the process is still running
	// 2. windows rename operations fail if the destination file already exists
	_ = os.Remove(oldPath)

	// move the existing executable to a new file in the same directory
	err = os.Rename(path, oldPath)
	if err != nil {
		return
	}

	// move the new exectuable in to become the new program
	err = os.Rename(newPath, path)
	if err != nil {
		return
	}

	if err != nil {
		// copy unsuccessful
		errRecover = os.Rename(oldPath, path)
	} else {
		// On macOS if the original binary file is removed we will
		// not be able to start any child processes. Instead we
		// leave the old file and it is up to the caller to use
		// the RemoveOld() function
		if runtime.GOOS != "darwin" {
			// copy successful, remove the old binary
			errRemove := os.Remove(oldPath)
			// windows has trouble with removing old binaries, so hide it instead
			if errRemove != nil {
				errHide := hideFile(oldPath)
				if errHide != nil {
					logging.Error("Encountered error attempting to hide file: %v", err)
				}
			}
		}
	}

	return
}
