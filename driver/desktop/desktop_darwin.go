// Copyright 2018 The GoKi Authors. All rights reserved.
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
uintptr_t doNewMainMenu(uintptr_t viewID);
uintptr_t mainMenuFromDelegate(uintptr_t delID);
void doSetMainMenu(uintptr_t viewID, uintptr_t menID);
void doMenuReset(uintptr_t menuID);
uintptr_t doAddSubMenu(uintptr_t menuID, char* mnm);
uintptr_t doAddMenuItem(uintptr_t viewID, uintptr_t submID, uintptr_t delID, char* itmnm, char* sc, bool scShift, bool scCommand, bool scAlt, bool scCtrl, int tag, bool active);
void doAddSeparator(uintptr_t menuID);
uintptr_t doMenuItemByTitle(uintptr_t menuID, char* mnm);
uintptr_t doMenuItemByTag(uintptr_t menuID, int tag);
void doSetMenuItemActive(uintptr_t mitmID, bool active);
*/
import "C"

import (
	"fmt"
	"log"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"sync"
	"unsafe"

	"goki.dev/goosi"
	"goki.dev/goosi/key"
	"goki.dev/goosi/mimedata"
	"goki.dev/goosi/osevent"
	"goki.dev/pi/v2/filecat"
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

func (app *appImpl) Platform() goosi.Platforms {
	return goosi.MacOS
}

func (app *appImpl) OpenURL(url string) {
	cmd := exec.Command("open", url)
	cmd.Run()
}

func (app *appImpl) PrefsDir() string {
	usr, err := user.Current()
	if err != nil {
		log.Print(err)
		return "/tmp"
	}
	return filepath.Join(usr.HomeDir, "Library")
}

// this is the main call to create the main menu if not exist
func (w *windowImpl) MainMenu() goosi.MainMenu {
	if w.mainMenu == nil {
		mm := &mainMenuImpl{win: w}
		w.mainMenu = mm
		vid := uintptr(mm.win.glw.GetCocoaWindow())
		mm.delID = uintptr(C.doNewMainMenu(C.uintptr_t(vid)))
		mm.menID = uintptr(C.mainMenuFromDelegate(C.uintptr_t(mm.delID)))
	}
	return w.mainMenu.(*mainMenuImpl)
}

func (w *windowImpl) OSHandle() uintptr {
	return uintptr(w.glw.GetCocoaWindow())
}

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

///////////////////////////////////////////////////////
//  MainMenu

type mainMenuImpl struct {
	win      *windowImpl
	menID    uintptr // mainmenu id
	delID    uintptr // delegate id
	callback func(win goosi.Window, title string, tag int)
	mu       sync.Mutex
}

func (mm *mainMenuImpl) Window() goosi.Window {
	return mm.win
}

func (mm *mainMenuImpl) SetWindow(win goosi.Window) {
	mm.win = win.(*windowImpl)
}

func (mm *mainMenuImpl) SetFunc(fun func(win goosi.Window, title string, tag int)) {
	mm.callback = fun
}

func (mm *mainMenuImpl) Triggered(win goosi.Window, title string, tag int) {
	if mm.callback == nil {
		return
	}
	fw := theApp.WindowInFocus()
	if win != fw {
		if fw == nil {
			fmt.Printf("vkos main menu event focus window is nil!  window: %v\n", win.Name())
		} else {
			fmt.Printf("vkos main menu event window: %v != focus window: %v\n", win.Name(), fw.Name())
			// win = fw // this doesn't work because focus window doesn't have menu most of time
		}
	}
	mm.callback(win, title, tag)
}

func (mm *mainMenuImpl) Menu() goosi.Menu {
	return goosi.Menu(mm.menID)
}

func (mm *mainMenuImpl) SetMenu() {
	mm.mu.Lock()
	vid := uintptr(mm.win.glw.GetCocoaWindow())
	C.doSetMainMenu(C.uintptr_t(vid), C.uintptr_t(mm.menID))
	mm.mu.Unlock()
}

func (mm *mainMenuImpl) StartUpdate() goosi.Menu {
	mm.mu.Lock()
	return goosi.Menu(mm.menID)
}

func (mm *mainMenuImpl) EndUpdate(men goosi.Menu) {
	mm.mu.Unlock()
}

// Reset must be called within StartUpdate window
func (mm *mainMenuImpl) Reset(men goosi.Menu) {
	C.doMenuReset(C.uintptr_t(men))
}

func (mm *mainMenuImpl) AddSubMenu(men goosi.Menu, titles string) goosi.Menu {
	title := C.CString(titles)
	defer C.free(unsafe.Pointer(title))

	subid := C.doAddSubMenu(C.uintptr_t(men), title)
	return goosi.Menu(subid)
}

func (mm *mainMenuImpl) AddItem(men goosi.Menu, titles string, shortcut string, tag int, active bool) goosi.MenuItem {
	title := C.CString(titles)
	defer C.free(unsafe.Pointer(title))

	sc := ""
	scShift := false
	scControl := false
	scAlt := false
	scCommand := false
	// don't register shortcuts on main menu -- just causes problems!
	if false {
		r, mods, err := key.Chord(shortcut).Decode()
		if err == nil {
			sc = strings.ToLower(string(r))
		}
		scShift = mods.HasFlag(goosi.Shift)
		scControl = mods.HasFlag(goosi.Control)
		scAlt = mods.HasFlag(goosi.Alt)
		scCommand = mods.HasFlag(goosi.Meta)
	}

	scs := C.CString(sc)
	defer C.free(unsafe.Pointer(scs))

	vid := uintptr(mm.win.glw.GetCocoaWindow())
	mid := C.doAddMenuItem(C.uintptr_t(vid), C.uintptr_t(men), C.uintptr_t(mm.delID), title, scs, C.bool(scShift), C.bool(scCommand), C.bool(scAlt), C.bool(scControl), C.int(tag), C.bool(active))
	return goosi.MenuItem(mid)
}

func (mm *mainMenuImpl) AddSeparator(men goosi.Menu) {
	C.doAddSeparator(C.uintptr_t(men))
}

func (mm *mainMenuImpl) ItemByTitle(men goosi.Menu, titles string) goosi.MenuItem {
	title := C.CString(titles)
	defer C.free(unsafe.Pointer(title))
	mid := C.doMenuItemByTitle(C.uintptr_t(men), title)
	return goosi.MenuItem(mid)
}

func (mm *mainMenuImpl) ItemByTag(men goosi.Menu, tag int) goosi.MenuItem {
	mid := C.doMenuItemByTag(C.uintptr_t(men), C.int(tag))
	return goosi.MenuItem(mid)
}

func (mm *mainMenuImpl) SetItemActive(mitm goosi.MenuItem, active bool) {
	C.doSetMenuItemActive(C.uintptr_t(mitm), C.bool(active))
}

//export menuFired
func menuFired(id uintptr, title *C.char, tilen C.int, tag C.int) {
	theApp.mu.Lock()
	w, ok := theApp.oswindows[id]
	theApp.mu.Unlock()
	if !ok || w == nil {
		return
	}

	tit := C.GoStringN(title, tilen)
	osmm := w.MainMenu()
	if osmm == nil {
		return
	}
	go osmm.Triggered(w, tit, int(tag))
}

//export macOpenFile
func macOpenFile(fname *C.char, flen C.int) {
	ofn := C.GoString(fname)
	// fmt.Printf("open file: %s\n", ofn)
	if theApp.NWindows() == 0 {
		theApp.openFiles = append(theApp.openFiles, ofn)
	} else {
		win := theApp.Window(0)
		osev := osevent.NewOpenFilesEvent([]string{ofn})
		osev.Init()
		win.Send(osev)
	}
}
