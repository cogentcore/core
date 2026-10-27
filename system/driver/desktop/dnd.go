// Copyright 2026 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package desktop

import (
	"bytes"
	"image"
	"image/png"
	"sync"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
)

// Drag-and-drop with other windows and apps: the platform code starts
// drags from a window with [Window.StartDrag] (with the data from
// [dragData]), and sends drags over the windows of the app with
// [Window.dragMove] and [Window.dragDrop], for the window found with
// [dragWindow] from its native handle.

// dragWindows maps the native handle of each window (NSWindow,
// HWND or X11 Window) to the window, for the drag-and-drop callbacks.
var dragWindows = struct {
	sync.Mutex
	m map[uintptr]*Window
}{m: map[uintptr]*Window{}}

// addDragWindow adds the window with the given native handle.
func addDragWindow(id uintptr, w *Window) {
	dragWindows.Lock()
	dragWindows.m[id] = w
	dragWindows.Unlock()
}

// deleteDragWindow deletes the window with the given native handle.
func deleteDragWindow(id uintptr) {
	dragWindows.Lock()
	delete(dragWindows.m, id)
	dragWindows.Unlock()
}

// dragWindow returns the window with the given native handle, or nil.
func dragWindow(id uintptr) *Window {
	dragWindows.Lock()
	defer dragWindows.Unlock()
	return dragWindows.m[id]
}

// dragData returns the data provided to other apps for a drag of
// the given data: unless there are files, the mime data as multipart
// MIME (for other Cogent Core apps), and its first text item as text.
func dragData(data mimedata.Mimes, files []string) (mime, text []byte) {
	if len(files) > 0 {
		return
	}
	mime = data.ToMultipart()
	for _, d := range data {
		if mimedata.IsText(d.Type) {
			text = d.Data
			break
		}
	}
	return
}

// dragImagePNG returns the given drag image as PNG data.
func dragImagePNG(img image.Image) []byte {
	if img == nil {
		return nil
	}
	var b bytes.Buffer
	if errors.Log(png.Encode(&b, img)) != nil {
		return nil
	}
	return b.Bytes()
}

// dragMove sends a drag over the window to the window, at the given
// position in window (pixel) coordinates, or leaving the window.
func (w *Window) dragMove(leave bool, where image.Point, mods key.Modifiers, fromApp bool) {
	typ := events.DragMove
	if leave {
		typ = events.DragLeave
	}
	w.Event.DragExternal(typ, where, mods, nil, fromApp)
}

// dragDrop sends a drop on the window to the window, with either
// files, or multipart mime data from [dragData].
func (w *Window) dragDrop(where image.Point, mods key.Modifiers, fromApp bool, files []string, mime []byte) {
	var md mimedata.Mimes
	if len(files) > 0 {
		for _, f := range files {
			md = append(md, mimedata.NewTextData(f))
		}
	} else if len(mime) > 0 {
		if isMulti, _, boundary, body := mimedata.IsMultipart(mime); isMulti {
			md = mimedata.FromMultipart(body, boundary)
		}
	}
	if len(md) == 0 {
		return
	}
	w.Event.DragExternal(events.Drop, where, mods, md, fromApp)
}
