// Copyright 2018 The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based on golang.org/x/exp/shiny:
// Copyright 2015 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build 3d
// +build darwin

package glos

import (
	"sync"
	"unsafe"

	"github.com/goki/gi/oswin/cursor"
	"github.com/goki/gi/oswin/mimedata"
	"github.com/goki/pi/filecat"
)

/////////////////////////////////////////////////////////////////
// clip.Board impl

type clipImpl struct {
	data mimedata.Mimes
}

var theClip = clipImpl{}

// curpMimeData is the current mime data to write to from cocoa side
var curMimeData *mimedata.Mimes

func (ci *clipImpl) IsEmpty() bool {
	ise := C.clipIsEmpty()
	return bool(ise)
}

func (ci *clipImpl) Read(types []string) mimedata.Mimes {
	if len(types) == 0 {
		return nil
	}
	ci.data = nil
	curMimeData = &ci.data

	wantText := mimedata.IsText(types[0])

	if wantText {
		C.clipReadText() // calls addMimeText
		if len(ci.data) == 0 {
			return nil
		}
		dat := ci.data[0].Data
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
	return ci.data
}

func (ci *clipImpl) WriteText(b []byte) {
	sz := len(b)
	cdata := C.malloc(C.size_t(sz))
	copy((*[1 << 30]byte)(cdata)[0:sz], b)
	C.pasteWriteAddText((*C.char)(cdata), C.int(sz))
	C.free(unsafe.Pointer(cdata))
}

func (ci *clipImpl) Write(data mimedata.Mimes) error {
	ci.Clear()
	if len(data) > 1 { // multipart
		mpd := data.ToMultipart()
		ci.WriteText(mpd)
	} else {
		d := data[0]
		if mimedata.IsText(d.Type) {
			ci.WriteText(d.Data)
		}
	}
	C.clipWrite()
	return nil
}

func (ci *clipImpl) Clear() {
	C.clipClear()
}

//export addMimeText
func addMimeText(cdata *C.char, datalen C.int) {
	if *curMimeData == nil {
		*curMimeData = make(mimedata.Mimes, 1)
		(*curMimeData)[0] = &mimedata.Data{Type: filecat.TextPlain}
	}
	md := (*curMimeData)[0]
	if len(md.Type) == 0 {
		md.Type = filecat.TextPlain
	}
	data := C.GoBytes(unsafe.Pointer(cdata), datalen)
	md.Data = append(md.Data, data...)
}

//export addMimeData
func addMimeData(ctyp *C.char, typlen C.int, cdata *C.char, datalen C.int) {
	if *curMimeData == nil {
		*curMimeData = make(mimedata.Mimes, 0)
	}
	typ := C.GoStringN(ctyp, typlen)
	data := C.GoBytes(unsafe.Pointer(cdata), datalen)
	*curMimeData = append(*curMimeData, &mimedata.Data{typ, data})
}

/////////////////////////////////////////////////////////////////
// cursor impl

type cursorImpl struct {
	cursor.CursorBase
	mu sync.Mutex
}

var theCursor = cursorImpl{CursorBase: cursor.CursorBase{Vis: true}}

func (c *cursorImpl) Push(sh cursor.Shapes) {
	c.mu.Lock()
	c.PushStack(sh)
	c.mu.Unlock()
	C.pushCursor(C.int(sh))
}

func (c *cursorImpl) Set(sh cursor.Shapes) {
	c.mu.Lock()
	c.Cur = sh
	c.mu.Unlock()
	C.setCursor(C.int(sh))
}

func (c *cursorImpl) Pop() {
	c.mu.Lock()
	c.PopStack()
	c.mu.Unlock()
	C.popCursor()
}

func (c *cursorImpl) Hide() {
	c.mu.Lock()
	if c.Vis == false {
		c.mu.Unlock()
		return
	}
	c.Vis = false
	c.mu.Unlock()
	C.hideCursor()
}

func (c *cursorImpl) Show() {
	c.mu.Lock()
	if c.Vis {
		c.mu.Unlock()
		return
	}
	c.Vis = true
	c.mu.Unlock()
	C.showCursor()
}

func (c *cursorImpl) PushIfNot(sh cursor.Shapes) bool {
	c.mu.Lock()
	if c.Cur == sh {
		c.mu.Unlock()
		return false
	}
	c.mu.Unlock()
	c.Push(sh)
	return true
}

func (c *cursorImpl) PopIf(sh cursor.Shapes) bool {
	c.mu.Lock()
	if c.Cur == sh {
		c.mu.Unlock()
		c.Pop()
		return true
	}
	c.mu.Unlock()
	return false
}
