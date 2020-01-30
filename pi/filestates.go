// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pi

import (
	"sync"

	"github.com/goki/pi/filecat"
)

// FileStates contains three FileState's,
// such that one can be updated while another is in use.
type FileStates struct {
	DoneIdx  int        `desc:"index of the state that is done -- Proc is always one after"`
	FsA      FileState  `desc:"one filestate"`
	FsB      FileState  `desc:"one filestate"`
	FsC      FileState  `desc:"one filestate"`
	SwitchMu sync.Mutex `desc:"mutex locking the switching of Done vs. Proc states"`
	ProcMu   sync.Mutex `desc:"mutex locking the parsing of Proc state -- reading states can happen fine with this locked, but no switching"`
}

// Done returns the filestate that is done being updated, and is ready for
// use by external clients etc.  Proc is the other one which is currently
// being processed by the parser and is not ready to be used externally.
// The state is accessed under a lock, and as long as any use of state is
// fast enough, it should be usable over next two switches (typically true).
func (fs *FileStates) Done() *FileState {
	fs.SwitchMu.Lock()
	defer fs.SwitchMu.Unlock()
	return fs.DoneNoLock()
}

// DoneNoLock returns the filestate that is done being updated, and is ready for
// use by external clients etc.  Proc is the other one which is currently
// being processed by the parser and is not ready to be used externally.
// The state is accessed under a lock, and as long as any use of state is
// fast enough, it should be usable over next two switches (typically true).
func (fs *FileStates) DoneNoLock() *FileState {
	switch fs.DoneIdx {
	case 0:
		return &fs.FsA
	case 1:
		return &fs.FsB
	case 2:
		return &fs.FsC
	}
	return &fs.FsA
}

// Proc returns the filestate that is currently being processed by
// the parser etc and is not ready for external use.
// Access is protected by a lock so it will wait if currently switching.
// The state is accessed under a lock, and as long as any use of state is
// fast enough, it should be usable over next two switches (typically true).
func (fs *FileStates) Proc() *FileState {
	fs.SwitchMu.Lock()
	defer fs.SwitchMu.Unlock()
	return fs.ProcNoLock()
}

// ProcNoLock returns the filestate that is currently being processed by
// the parser etc and is not ready for external use.
// Access is protected by a lock so it will wait if currently switching.
// The state is accessed under a lock, and as long as any use of state is
// fast enough, it should be usable over next two switches (typically true).
func (fs *FileStates) ProcNoLock() *FileState {
	switch fs.DoneIdx {
	case 0:
		return &fs.FsB
	case 1:
		return &fs.FsC
	case 2:
		return &fs.FsA
	}
	return &fs.FsB
}

// Switch switches over from one Done Filestate to the next
func (fs *FileStates) Switch() {
	fs.ProcMu.Lock() // make sure processing is done
	defer fs.ProcMu.Unlock()
	fs.SwitchMu.Lock()
	defer fs.SwitchMu.Unlock()
	fs.DoneIdx++
	fs.DoneIdx = fs.DoneIdx % 3
}

// todo: not sure we want this:

// SetSrc sets file for all of the states
func (fs *FileStates) SetSrc(src *[][]rune, fname string, sup filecat.Supported) {
	fs.ProcMu.Lock() // make sure processing is done
	defer fs.ProcMu.Unlock()
	fs.SwitchMu.Lock()
	defer fs.SwitchMu.Unlock()
	fs.FsA.SetSrc(src, fname, sup)
	fs.FsB.SetSrc(src, fname, sup)
	fs.FsC.SetSrc(src, fname, sup)
}
