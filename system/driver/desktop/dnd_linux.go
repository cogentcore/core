// Copyright 2026 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build linux && (x11 || !wayland)

package desktop

/*
#cgo LDFLAGS: -lX11 -lpthread
#include <X11/Xlib.h>
#include <stdbool.h>
#include <stdlib.h>
void registerDrag(Window win);
void unregisterDrag(Window win);
void ungrabGLFW(Display* gd);
bool startDrag(Window win, void* uris, int urislen, void* mime, int mimelen, void* text, int textlen);
*/
import "C"

import (
	"image"
	"net/url"
	"strings"
	"unsafe"

	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/events/key"
	"github.com/go-gl/glfw/v3.4/glfw"
)

// xwindow returns the X11 Window of the window.
func (w *Window) xwindow() C.Window {
	return C.Window(w.Glw.GetX11Window())
}

// initDrag sets up drag-and-drop for the window.
// It must be called on the main thread.
func (w *Window) initDrag() {
	xw := w.xwindow()
	addDragWindow(uintptr(xw), w)
	C.registerDrag(xw)
}

// closeDrag removes the window from drag-and-drop.
// It must be called on the main thread.
func (w *Window) closeDrag() {
	xw := w.xwindow()
	C.unregisterDrag(xw)
	deleteDragWindow(uintptr(xw))
}

func (w *Window) StartDrag(data mimedata.Mimes, files []string, img image.Image) bool {
	if w.Glw == nil {
		return false
	}
	mime, text := dragData(data, files)
	uris := fileURIs(files)
	var xw C.Window
	w.App.RunOnMain(func() {
		if w.Glw == nil {
			return
		}
		xw = w.xwindow()
		C.ungrabGLFW((*C.Display)(unsafe.Pointer(glfw.GetX11Display())))
	})
	if xw == 0 {
		return false
	}
	curis, curislen := cBytes(uris)
	defer C.free(curis)
	cmime, cmimelen := cBytes(mime)
	defer C.free(cmime)
	ctext, ctextlen := cBytes(text)
	defer C.free(ctext)
	return bool(C.startDrag(xw, curis, curislen, cmime, cmimelen, ctext, ctextlen))
}

// fileURIs returns the given files as a text/uri-list.
func fileURIs(files []string) []byte {
	var b strings.Builder
	for _, f := range files {
		u := url.URL{Scheme: "file", Path: f}
		b.WriteString(u.String() + "\r\n")
	}
	return []byte(b.String())
}

// filesFromURIs returns the local files in the given text/uri-list.
func filesFromURIs(uris string) []string {
	var files []string
	for _, line := range strings.Split(uris, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		u, err := url.Parse(line)
		if err != nil || u.Scheme != "file" || u.Path == "" {
			continue
		}
		files = append(files, u.Path)
	}
	return files
}

// cBytes returns a C copy of the given bytes, which must be freed.
func cBytes(b []byte) (unsafe.Pointer, C.int) {
	if len(b) == 0 {
		return nil, 0
	}
	return C.CBytes(b), C.int(len(b))
}

// x11Mods converts the modifier bits from the C code to [key.Modifiers].
func x11Mods(mods C.int) key.Modifiers {
	var m key.Modifiers
	m.SetFlag(mods&1 != 0, key.Shift)
	m.SetFlag(mods&2 != 0, key.Control)
	m.SetFlag(mods&4 != 0, key.Alt)
	m.SetFlag(mods&8 != 0, key.Meta)
	return m
}

// x11DragMove is called for a drag over the window: leave is true when
// the drag leaves the window. x, y are in window pixels.
//
//export x11DragMove
func x11DragMove(win C.Window, leave C.bool, x, y C.int, mods C.int, fromApp C.bool) {
	if w := dragWindow(uintptr(win)); w != nil {
		w.dragMove(bool(leave), image.Pt(int(x), int(y)), x11Mods(mods), bool(fromApp))
	}
}

// x11DragDrop is called for a drop on the window, with either a
// text/uri-list of files, or mime data from [dragData].
//
//export x11DragDrop
func x11DragDrop(win C.Window, x, y C.int, mods C.int, fromApp C.bool, uris *C.char, urislen C.int, mime *C.char, mimelen C.int) {
	w := dragWindow(uintptr(win))
	if w == nil {
		return
	}
	var files []string
	if urislen > 0 {
		files = filesFromURIs(C.GoStringN(uris, urislen))
	}
	var mb []byte
	if mimelen > 0 {
		mb = C.GoBytes(unsafe.Pointer(mime), mimelen)
	}
	w.dragDrop(image.Pt(int(x), int(y)), x11Mods(mods), bool(fromApp), files, mb)
}
