// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pi

import (
	"sync"

	"github.com/goki/pi/filecat"
)

// FileStates contains two FileState's: one is being processed while the
// other is being used externally.  The FileStates maintains
// a common set of file information set in each of the FileState items when
// they are used.
type FileStates struct {
	Filename string            `desc:"the filename"`
	Sup      filecat.Supported `desc:"the supported file type, if supported (typically only supported files are processed)"`
	BasePath string            `desc:"base path for reporting file names -- this must be set externally e.g., by gide for the project root path"`
	DoneIdx  int               `desc:"index of the state that is done"`
	FsA      FileState         `desc:"one filestate"`
	FsB      FileState         `desc:"one filestate"`
	SwitchMu sync.Mutex        `desc:"mutex locking the switching of Done vs. Proc states"`
	ProcMu   sync.Mutex        `desc:"mutex locking the parsing of Proc state -- reading states can happen fine with this locked, but no switching"`
}

// NewFileStates returns a new FileStates for given filename, basepath,
// and supported file type.
func NewFileStates(fname, basepath string, sup filecat.Supported) *FileStates {
	fs := &FileStates{}
	fs.SetSrc(fname, basepath, sup)
	return fs
}

// SetSrc sets the source that is processed by this FileStates
// if basepath is empty then it is set to the path for the filename.
func (fs *FileStates) SetSrc(fname, basepath string, sup filecat.Supported) {
	fs.ProcMu.Lock() // make sure processing is done
	defer fs.ProcMu.Unlock()
	fs.SwitchMu.Lock()
	defer fs.SwitchMu.Unlock()

	fs.Filename = fname
	fs.BasePath = basepath
	fs.Sup = sup

	fs.FsA.SetSrc(nil, fname, basepath, sup)
	fs.FsB.SetSrc(nil, fname, basepath, sup)
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
		return &fs.FsA
	}
	return &fs.FsB
}

// StartProc should be called when starting to process the file, and returns the
// FileState to use for processing.  It locks the Proc state, sets the current
// source code, and returns the filestate for subsequent processing.
func (fs *FileStates) StartProc(txt []byte) *FileState {
	fs.ProcMu.Lock()
	pfs := fs.ProcNoLock()
	pfs.Src.BasePath = fs.BasePath
	pfs.Src.SetBytes(txt)
	return pfs
}

// EndProc is called when primary processing (parsing) has been completed --
// there still may be ongoing updating of symbols after this point but parse
// is done.  This calls Switch to move Proc over to done, under cover of ProcMu Lock
func (fs *FileStates) EndProc() {
	fs.Switch()
	fs.ProcMu.Unlock()
}

// Switch switches so that the current Proc() filestate is now the Done()
// it is assumed to be called under ProcMu.Locking cover, and also
// does the Swtich locking.
func (fs *FileStates) Switch() {
	fs.SwitchMu.Lock()
	defer fs.SwitchMu.Unlock()
	fs.DoneIdx++
	fs.DoneIdx = fs.DoneIdx % 2
	// fmt.Printf("switched: %v  %v\n", fs.DoneIdx, fs.Filename)
}
