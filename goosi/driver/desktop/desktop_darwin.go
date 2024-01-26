// Copyright 2018 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build darwin

package desktop

/*
#cgo CFLAGS: -x objective-c -Wno-deprecated-declarations
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>
int setThreadPri(double p);
void clipClear();
bool clipIsEmpty();
void clipReadText();
void pasteWriteAddText(char* data, int dlen);
void clipWrite();
void pushCursor(int);
void popCursor();
void setCursor(int);
void hideCursor();
void showCursor();
*/
import "C"

import (
	"fmt"
	"os/exec"
	"os/user"
	"path/filepath"
	"unsafe"

	"cogentcore.org/core/fi"
	"cogentcore.org/core/goosi"
	"cogentcore.org/core/grr"
	"cogentcore.org/core/mimedata"
)

/////////////////////////////////////////////////////////////////
// OS-specific methods

func SetThreadPri(p float64) error {
	rv := C.setThreadPri(C.double(p))
	if rv != 0 {
		err := fmt.Errorf("SetThreadPri failed: %v\n", rv)
		fmt.Println(err)
		return err
	}
	return nil
}

func (a *App) Platform() goosi.Platforms {
	return goosi.MacOS
}

func (a *App) OpenURL(url string) {
	cmd := exec.Command("open", url)
	grr.Log(cmd.Run())
}

func (a *App) DataDir() string {
	usr, err := user.Current()
	if grr.Log(err) != nil {
		return "/tmp"
	}
	return filepath.Join(usr.HomeDir, "Library")
}

func (w *Window) Handle() any {
	return uintptr(w.Glw.GetCocoaWindow())
}

/////////////////////////////////////////////////////////////////
// goosi.Clipboard impl

// TheClip is the single [goosi.Clipboard] for MacOS
var TheClip = &Clip{}

// Clip is the [goosi.Clipboard] implementation for MacOS
type Clip struct {
	// Data is the current clipboard data
	Data mimedata.Mimes
}

// CurMimeData is the current mime data to write to from cocoa side
var CurMimeData *mimedata.Mimes

func (cl *Clip) IsEmpty() bool {
	ise := C.clipIsEmpty()
	return bool(ise)
}

func (cl *Clip) Read(types []string) mimedata.Mimes {
	if len(types) == 0 {
		return nil
	}
	cl.Data = nil
	CurMimeData = &cl.Data

	wantText := mimedata.IsText(types[0])

	if wantText {
		C.clipReadText() // calls addMimeText
		if len(cl.Data) == 0 {
			return nil
		}
		dat := cl.Data[0].Data
		isMulti, mediaType, boundary, body := mimedata.IsMultipart(dat)
		if isMulti {
			return mimedata.FromMultipart(body, boundary)
		} else {
			if mediaType != "" { // found a mime type encoding
				return mimedata.NewMime(mediaType, dat)
			} else {
				// we can't really figure out type, so just assume..
				return mimedata.NewMime(types[0], dat)
			}
		}
	} else {
		// todo: deal with image formats etc
	}
	return cl.Data
}

func (cl *Clip) WriteText(b []byte) {
	sz := len(b)
	cdata := C.malloc(C.size_t(sz))
	copy((*[1 << 30]byte)(cdata)[0:sz], b)
	C.pasteWriteAddText((*C.char)(cdata), C.int(sz))
	C.free(unsafe.Pointer(cdata))
}

func (cl *Clip) Write(data mimedata.Mimes) error {
	cl.Clear()
	if len(data) > 1 { // multipart
		mpd := data.ToMultipart()
		cl.WriteText(mpd)
	} else {
		d := data[0]
		if mimedata.IsText(d.Type) {
			cl.WriteText(d.Data)
		}
	}
	C.clipWrite()
	return nil
}

func (cl *Clip) Clear() {
	C.clipClear()
}

//export addMimeText
func addMimeText(cdata *C.char, datalen C.int) {
	if *CurMimeData == nil {
		*CurMimeData = make(mimedata.Mimes, 1)
		(*CurMimeData)[0] = &mimedata.Data{Type: fi.TextPlain}
	}
	md := (*CurMimeData)[0]
	if len(md.Type) == 0 {
		md.Type = fi.TextPlain
	}
	data := C.GoBytes(unsafe.Pointer(cdata), datalen)
	md.Data = append(md.Data, data...)
}

//export addMimeData
func addMimeData(ctyp *C.char, typlen C.int, cdata *C.char, datalen C.int) {
	if *CurMimeData == nil {
		*CurMimeData = make(mimedata.Mimes, 0)
	}
	typ := C.GoStringN(ctyp, typlen)
	data := C.GoBytes(unsafe.Pointer(cdata), datalen)
	*CurMimeData = append(*CurMimeData, &mimedata.Data{typ, data})
}

// TODO(kai): macOpenFile

//export macOpenFile
func macOpenFile(fname *C.char, flen C.int) {
	/*
		ofn := C.GoString(fname)
		// fmt.Printf("open file: %s\n", ofn)
		if theApp.NWindows() == 0 {
			theApp.openFiles = append(theApp.openFiles, ofn)
		} else {
			// win := theApp.Window(0)
			// win.EventMgr.NewOS(events.OSEvent, []string{ofn})
		}
	*/
}
