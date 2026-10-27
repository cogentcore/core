// Copyright 2026 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build darwin

package desktop

/*
#cgo CFLAGS: -x objective-c -Wno-deprecated-declarations
#cgo LDFLAGS: -framework Cocoa
#include <stdbool.h>
#include <stdlib.h>
void registerDrag(void* nswin);
bool startDrag(void* nswin, char** files, int nfiles, void* mime, int mimelen, void* text, int textlen, void* png, int pnglen, double imgw, double imgh);
*/
import "C"

import (
	"bytes"
	"image"
	"image/png"
	"sync"
	"unsafe"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
)

// dragWindows maps the NSWindow of each window to the window,
// for the drag-and-drop callbacks from Cocoa.
var dragWindows = struct {
	sync.Mutex
	m map[unsafe.Pointer]*Window
}{m: map[unsafe.Pointer]*Window{}}

// initDrag sets up drag-and-drop for the window.
// It must be called on the main thread.
func (w *Window) initDrag() {
	nw := w.Glw.GetCocoaWindow()
	dragWindows.Lock()
	dragWindows.m[nw] = w
	dragWindows.Unlock()
	C.registerDrag(nw)
}

// closeDrag removes the window from drag-and-drop.
// It must be called on the main thread.
func (w *Window) closeDrag() {
	dragWindows.Lock()
	delete(dragWindows.m, w.Glw.GetCocoaWindow())
	dragWindows.Unlock()
}

// dragWindow returns the window for the given NSWindow, or nil.
func dragWindow(nswin unsafe.Pointer) *Window {
	dragWindows.Lock()
	defer dragWindows.Unlock()
	return dragWindows.m[nswin]
}

// macDragWindowOpen returns whether the given NSWindow is still open.
//
//export macDragWindowOpen
func macDragWindowOpen(nswin unsafe.Pointer) C.bool {
	return dragWindow(nswin) != nil
}

func (w *Window) StartDrag(data mimedata.Mimes, files []string, img image.Image) bool {
	if w.Glw == nil {
		return false
	}
	var mime, text []byte
	if len(files) == 0 {
		mime = data.ToMultipart()
		for _, d := range data {
			if mimedata.IsText(d.Type) {
				text = d.Data
				break
			}
		}
	}
	var pngb []byte
	var imgw, imgh float64
	if img != nil {
		var b bytes.Buffer
		if errors.Log(png.Encode(&b, img)) == nil {
			pngb = b.Bytes()
			sz := img.Bounds().Size()
			imgw = float64(sz.X) / float64(w.DevicePixelRatio)
			imgh = float64(sz.Y) / float64(w.DevicePixelRatio)
		}
	}
	cfiles := make([]*C.char, len(files)+1)
	for i, f := range files {
		cfiles[i] = C.CString(f)
	}
	defer func() {
		for _, cf := range cfiles {
			C.free(unsafe.Pointer(cf))
		}
	}()
	cmime, cmimelen := cBytes(mime)
	defer C.free(cmime)
	ctext, ctextlen := cBytes(text)
	defer C.free(ctext)
	cpng, cpnglen := cBytes(pngb)
	defer C.free(cpng)
	ok := false
	w.App.RunOnMain(func() {
		if w.Glw == nil {
			return
		}
		ok = bool(C.startDrag(unsafe.Pointer(w.Glw.GetCocoaWindow()), &cfiles[0], C.int(len(files)), cmime, cmimelen, ctext, ctextlen, cpng, cpnglen, C.double(imgw), C.double(imgh)))
	})
	return ok
}

// cBytes returns a C copy of the given bytes, which must be freed.
func cBytes(b []byte) (unsafe.Pointer, C.int) {
	if len(b) == 0 {
		return nil, 0
	}
	return C.CBytes(b), C.int(len(b))
}

// macMods converts NSEventModifierFlags to [key.Modifiers].
func macMods(flags C.ulong) key.Modifiers {
	var m key.Modifiers
	if flags&(1<<17) != 0 {
		m.SetFlag(true, key.Shift)
	}
	if flags&(1<<18) != 0 {
		m.SetFlag(true, key.Control)
	}
	if flags&(1<<19) != 0 {
		m.SetFlag(true, key.Alt)
	}
	if flags&(1<<20) != 0 {
		m.SetFlag(true, key.Meta)
	}
	return m
}

// macDragMove is called for a drag over the window: leave is true when
// the drag leaves the window. x, y are in window points from the top left.
//
//export macDragMove
func macDragMove(nswin unsafe.Pointer, leave C.bool, x, y C.double, flags C.ulong, fromApp C.bool) {
	w := dragWindow(nswin)
	if w == nil {
		return
	}
	typ := events.DragMove
	if leave {
		typ = events.DragLeave
	}
	where := w.MousePosToPoint(float64(x), float64(y))
	w.Event.DragExternal(typ, where, macMods(flags), nil, bool(fromApp))
}

// macDragDrop is called for a drop on the window, with either files
// or mime data from [Window.StartDrag].
//
//export macDragDrop
func macDragDrop(nswin unsafe.Pointer, x, y C.double, flags C.ulong, fromApp C.bool, files **C.char, nfiles C.int, mime unsafe.Pointer, mimelen C.int) {
	w := dragWindow(nswin)
	if w == nil {
		return
	}
	var md mimedata.Mimes
	if nfiles > 0 {
		for _, f := range unsafe.Slice(files, int(nfiles)) {
			md = append(md, mimedata.NewTextData(C.GoString(f)))
		}
	} else if mimelen > 0 {
		b := C.GoBytes(mime, mimelen)
		if isMulti, _, boundary, body := mimedata.IsMultipart(b); isMulti {
			md = mimedata.FromMultipart(body, boundary)
		}
	}
	if len(md) == 0 {
		return
	}
	where := w.MousePosToPoint(float64(x), float64(y))
	w.Event.DragExternal(events.Drop, where, macMods(flags), md, bool(fromApp))
}
