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

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>
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
void doSetMainMenu(uintptr_t viewID);
uintptr_t doGetMainMenu(uintptr_t viewID);
uintptr_t doGetMainMenuLock(uintptr_t viewID);
void doMainMenuUnlock(uintptr_t menuID);
void doMenuReset(uintptr_t menuID);
uintptr_t doAddSubMenu(uintptr_t menuID, char* mnm);
uintptr_t doAddMenuItem(uintptr_t viewID, uintptr_t submID, char* itmnm, char* sc, bool scShift, bool scCommand, bool scAlt, bool scCtrl, int tag, bool active);
void doAddSeparator(uintptr_t menuID);
uintptr_t doMenuItemByTitle(uintptr_t menuID, char* mnm);
uintptr_t doMenuItemByTag(uintptr_t menuID, int tag);
void doSetMenuItemActive(uintptr_t mitmID, bool active);
*/
import "C"

import (
	"sync"
	"unsafe"

	"github.com/goki/gi/oswin"
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

type mainMenuImpl struct {
	win      *windowImpl
	callback func(win oswin.Window, title string, tag int)
}

func (mm *mainMenuImpl) Window() oswin.Window {
	return mm.win
}

func (mm *mainMenuImpl) SetWindow(win oswin.Window) {
	mm.win = win.(*windowImpl)
}

func (mm *mainMenuImpl) SetFunc(fun func(win oswin.Window, title string, tag int)) {
	mm.callback = fun
}

func (mm *mainMenuImpl) Triggered(win oswin.Window, title string, tag int) {
	if mm.callback == nil {
		return
	}
	mm.callback(win, title, tag)
}

func (mm *mainMenuImpl) Menu() oswin.Menu {
	// mmen := C.doGetMainMenu(C.uintptr_t(uintptr(unsafe.Pointer(mm.win.glw))))
	// 	return oswin.Menu(mmen)
	return oswin.Menu(0)
}

func (mm *mainMenuImpl) SetMenu() {
	// C.doSetMainMenu(C.uintptr_t(uintptr(unsafe.Pointer(mm.win.glw))))
}

func (mm *mainMenuImpl) StartUpdate() oswin.Menu {
	/* mmen := C.doGetMainMenuLock(C.uintptr_t(uintptr(unsafe.Pointer(mm.win.glw))))
	return oswin.Menu(mmen)
	*/
	return oswin.Menu(0)
}

func (mm *mainMenuImpl) EndUpdate(men oswin.Menu) {
	// C.doMainMenuUnlock(C.uintptr_t(uintptr(unsafe.Pointer(mm.win.glw))))
}

func (mm *mainMenuImpl) Reset(men oswin.Menu) {
	// C.doMenuReset(C.uintptr_t(men))
}

func (mm *mainMenuImpl) AddSubMenu(men oswin.Menu, titles string) oswin.Menu {
	/* title := C.CString(titles)
	defer C.free(unsafe.Pointer(title))

	subid := C.doAddSubMenu(C.uintptr_t(men), title)
	return oswin.Menu(subid)
	*/
	return oswin.Menu(0)
}

func (mm *mainMenuImpl) AddItem(men oswin.Menu, titles string, shortcut string, tag int, active bool) oswin.MenuItem {
	/*
		title := C.CString(titles)
		defer C.free(unsafe.Pointer(title))

		sc := ""
		r, mods, err := key.Chord(shortcut).Decode()
		if err == nil {
			sc = strings.ToLower(string(r))
		}

		scShift := (mods&(1<<uint32(key.Shift)) != 0)
		scControl := (mods&(1<<uint32(key.Control)) != 0)
		scAlt := (mods&(1<<uint32(key.Alt)) != 0)
		scCommand := (mods&(1<<uint32(key.Meta)) != 0)

		scs := C.CString(sc)
		defer C.free(unsafe.Pointer(scs))

		mid := C.doAddMenuItem(C.uintptr_t(uintptr(unsafe.Pointer(mm.win.glw))), C.uintptr_t(men), title, scs, C.bool(scShift), C.bool(scCommand), C.bool(scAlt), C.bool(scControl), C.int(tag), C.bool(active))
		return oswin.MenuItem(mid)
	*/
	return oswin.MenuItem(0)
}

func (mm *mainMenuImpl) AddSeparator(men oswin.Menu) {
	// C.doAddSeparator(C.uintptr_t(men))
}

func (mm *mainMenuImpl) ItemByTitle(men oswin.Menu, titles string) oswin.MenuItem {
	/*
		title := C.CString(titles)
		defer C.free(unsafe.Pointer(title))
		mid := C.doMenuItemByTitle(C.uintptr_t(men), title)
		return oswin.MenuItem(mid)
	*/
	return oswin.MenuItem(0)
}

func (mm *mainMenuImpl) ItemByTag(men oswin.Menu, tag int) oswin.MenuItem {
	/* mid := C.doMenuItemByTag(C.uintptr_t(men), C.int(tag))
	return oswin.MenuItem(mid)
	*/
	return oswin.MenuItem(0)
}

func (mm *mainMenuImpl) SetItemActive(mitm oswin.MenuItem, active bool) {
	/* C.doSetMenuItemActive(C.uintptr_t(mitm), C.bool(active))
	 */
}

//export menuFired
func menuFired(id uintptr, title *C.char, tilen C.int, tag C.int) {
	/*	theApp.mu.Lock()
		w := theApp.windows[id]
		theApp.mu.Unlock()
		if w == nil {
			return
		}

		tit := C.GoStringN(title, tilen)
		osmm := w.MainMenu()
		if osmm == nil {
			return
		}
		go osmm.Triggered(w, tit, int(tag))
	*/
}
