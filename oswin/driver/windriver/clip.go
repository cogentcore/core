// Copyright 2018 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows
// +build !3d

package windriver

import (
	"fmt"
	"log"
	"syscall"
	"time"
	"unsafe"

	"github.com/goki/gi/oswin/mimedata"
)

// implements clipboard support for Windows
// https://github.com/jtanx/libclipboard/blob/master/src/clipboard_win32.c
// https://msdn.microsoft.com/en-us/library/windows/desktop/ms649016(v=vs.85).aspx
// formats:
// https://msdn.microsoft.com/en-us/library/windows/desktop/ff729168(v=vs.85).aspx
// https://github.com/AllenDang/w32

type clipImpl struct {
}

var theClip = clipImpl{}

// ClipRetries determines how many times to retry in opening the clipboard
var ClipRetries = 5

// ClipRetrySleep determines how long to sleep between retries
var ClipRetrySleep = 5 * time.Millisecond

func (ci *clipImpl) OpenClipboard() bool {
	if len(theApp.winlist) == 0 {
		return false
	}
	win := theApp.winlist[0]
	for retry := 0; retry < ClipRetries; retry++ {
		got := _OpenClipboard(win.hwnd)
		if got {
			return true
		}
		time.Sleep(ClipRetrySleep)
	}
	return false
}

func (ci *clipImpl) IsEmpty() bool {
	if !ci.OpenClipboard() {
		return false
	}
	defer _CloseClipboard()

	avail := _IsClipboardFormatAvailable(_CF_UNICODETEXT) // todo: only checking text now
	return !avail
}

func (ci *clipImpl) Read(types []string) mimedata.Mimes {
	if len(types) == 0 {
		return nil
	}
	if !ci.OpenClipboard() {
		return nil
	}
	defer _CloseClipboard()

	wantText := mimedata.IsText(types[0])

	if wantText {
		hData := _GetClipboardData(_CF_UNICODETEXT)
		if hData == 0 {
			log.Printf("clip.Board.Read couldn't get clip data\n")
			return nil
		}
		wd := _GlobalLock(hData)
		if wd == nil {
			log.Printf("clip.Board.Read couldn't lock clip data\n")
			return nil
		}
		defer _GlobalUnlock(hData)
		txt := ([]byte)(syscall.UTF16ToString((*[1 << 20]uint16)(unsafe.Pointer(wd))[:]))
		isMulti, mediaType, boundary, body := mimedata.IsMultipart(txt)
		if isMulti {
			return mimedata.FromMultipart(body, boundary)
		} else {
			if mediaType != "" { // found a mime type encoding
				return mimedata.NewMime(mediaType, txt)
			} else {
				// we can't really figure out type, so just assume..
				return mimedata.NewMime(types[0], txt)
			}
		}
	} else {
		// todo: deal with image formats etc
	}
	return nil
}

func (ci *clipImpl) WriteText(b []byte) error {
	wc, err := syscall.UTF16FromString(string(b))
	if err != nil {
		return err
	}
	sz := uintptr(len(wc) * 2)
	hData := _GlobalAlloc(_GMEM_MOVEABLE, sz)
	wd := _GlobalLock(hData)
	if wd == nil {
		log.Printf("clip.Board.Write couldn't lock clip data\n")
		return nil
	}
	_CopyMemory(uintptr(unsafe.Pointer(wd)), uintptr(unsafe.Pointer(&wc[0])), sz)
	_GlobalUnlock(hData)

	hRes := _SetClipboardData(_CF_UNICODETEXT, hData)
	if hRes == 0 {
		_GlobalFree(hData)
		return fmt.Errorf("clip.Board.Write Could not set clip data\n")
	}
	return nil
}

func (ci *clipImpl) Write(data mimedata.Mimes) error {
	if len(data) == 0 {
		return nil
	}

	if !ci.OpenClipboard() {
		return fmt.Errorf("clip.Board.Write could not open clipboard\n")
	}
	defer _CloseClipboard()

	if !_EmptyClipboard() {
		return fmt.Errorf("clip.Board.Write could not empty clipboard\n")
	}

	if len(data) > 1 { // multipart
		mpd := data.ToMultipart()
		return ci.WriteText(mpd)
	} else {
		d := data[0]
		if mimedata.IsText(d.Type) {
			return ci.WriteText(d.Data)
		}
	}
	return nil
}

func (ci *clipImpl) Clear() {
	if !ci.OpenClipboard() {
		return
	}
	_EmptyClipboard()
	_CloseClipboard()
}
