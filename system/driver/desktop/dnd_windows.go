// Copyright 2026 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows

package desktop

/*
#cgo LDFLAGS: -lole32 -lshell32 -luuid
#include <windows.h>
#include <stdbool.h>
#include <stdlib.h>
void registerDrag(HWND hwnd);
void unregisterDrag(HWND hwnd);
void startDrag(HWND hwnd, char** files, int nfiles, void* mime, int mimelen, char* text, int textlen, void* bgra, int imgw, int imgh);
*/
import "C"

import (
	"image"
	"image/draw"
	"unsafe"

	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/events/key"
)

// hwnd returns the HWND of the window.
func (w *Window) hwnd() C.HWND {
	return C.HWND(unsafe.Pointer(w.Glw.GetWin32Window()))
}

// initDrag sets up drag-and-drop for the window.
// It must be called on the main thread.
func (w *Window) initDrag() {
	h := w.hwnd()
	addDragWindow(uintptr(unsafe.Pointer(h)), w)
	C.registerDrag(h)
}

// closeDrag removes the window from drag-and-drop.
// It must be called on the main thread.
func (w *Window) closeDrag() {
	h := w.hwnd()
	C.unregisterDrag(h)
	deleteDragWindow(uintptr(unsafe.Pointer(h)))
}

func (w *Window) StartDrag(data mimedata.Mimes, files []string, img image.Image) bool {
	if w.Glw == nil {
		return false
	}
	mime, text := dragData(data, files)
	var bgra []byte
	var imgw, imgh int
	if img != nil {
		// premultiplied BGRA, top down
		sz := img.Bounds().Size()
		rgba := image.NewRGBA(image.Rectangle{Max: sz})
		draw.Draw(rgba, rgba.Bounds(), img, img.Bounds().Min, draw.Src)
		bgra = rgba.Pix
		for i := 0; i < len(bgra); i += 4 {
			bgra[i], bgra[i+2] = bgra[i+2], bgra[i]
		}
		imgw, imgh = sz.X, sz.Y
	}
	cfiles := make([]*C.char, len(files)+1)
	for i, f := range files {
		cfiles[i] = C.CString(f)
	}
	cmime, cmimelen := cBytes(mime)
	ctext, ctextlen := cBytes(text)
	cimg, _ := cBytes(bgra)
	// DoDragDrop runs until the drop, so this does not wait for it
	go w.App.RunOnMain(func() {
		defer func() {
			for _, cf := range cfiles {
				C.free(unsafe.Pointer(cf))
			}
			C.free(cmime)
			C.free(ctext)
			C.free(cimg)
		}()
		if w.Glw == nil {
			return
		}
		C.startDrag(w.hwnd(), &cfiles[0], C.int(len(files)), cmime, cmimelen, (*C.char)(ctext), ctextlen, cimg, C.int(imgw), C.int(imgh))
	})
	return true
}

// cBytes returns a C copy of the given bytes, which must be freed.
func cBytes(b []byte) (unsafe.Pointer, C.int) {
	if len(b) == 0 {
		return nil, 0
	}
	return C.CBytes(b), C.int(len(b))
}

// winMods converts the modifier bits from the C code to [key.Modifiers].
func winMods(mods C.int) key.Modifiers {
	var m key.Modifiers
	m.SetFlag(mods&1 != 0, key.Shift)
	m.SetFlag(mods&2 != 0, key.Control)
	m.SetFlag(mods&4 != 0, key.Alt)
	m.SetFlag(mods&8 != 0, key.Meta)
	return m
}

// winDragMove is called for a drag over the window: leave is true when
// the drag leaves the window. x, y are in client pixels.
//
//export winDragMove
func winDragMove(hwnd C.HWND, leave C.bool, x, y C.int, mods C.int, fromApp C.bool) {
	if w := dragWindow(uintptr(unsafe.Pointer(hwnd))); w != nil {
		w.dragMove(bool(leave), image.Pt(int(x), int(y)), winMods(mods), bool(fromApp))
	}
}

// winDragDrop is called for a drop on the window, with either files
// or mime data from [dragData].
//
//export winDragDrop
func winDragDrop(hwnd C.HWND, x, y C.int, mods C.int, fromApp C.bool, files **C.char, nfiles C.int, mime unsafe.Pointer, mimelen C.int) {
	w := dragWindow(uintptr(unsafe.Pointer(hwnd)))
	if w == nil {
		return
	}
	var fs []string
	for _, f := range unsafe.Slice(files, int(nfiles)) {
		fs = append(fs, C.GoString(f))
	}
	var mb []byte
	if mimelen > 0 {
		mb = C.GoBytes(mime, mimelen)
	}
	w.dragDrop(image.Pt(int(x), int(y)), winMods(mods), bool(fromApp), fs, mb)
}

// winDragPoll runs the functions waiting for the main thread
// (see [base.App.RunOnMain]), during a drag started by [Window.StartDrag],
// which runs its own event loop on the main thread until the drop.
//
//export winDragPoll
func winDragPoll() {
	for {
		select {
		case f := <-TheApp.MainQueue:
			f.F()
			if f.Done != nil {
				f.Done <- struct{}{}
			}
		default:
			return
		}
	}
}
