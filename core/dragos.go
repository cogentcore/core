// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"image"
	"path/filepath"
	"strings"
	"sync"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/base/fsx"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/tree"
)

// osDrag is the drag that this app most recently handed off to the
// operating system (see [Events.dragToOS]), so that a drop of it back
// into any window of the app can restore the original source widget
// and data, as for a drag within one window.
var osDrag struct {
	sync.Mutex

	// source is the widget that started the drag.
	source Widget

	// data is the drag data given to [Events.DragStart].
	data any

	// files are the files of the drag, from [Events.SetDragFiles].
	files []string
}

// FileDragger is an optional interface for a widget that starts a drag
// with [Events.DragStart] and represents files, which are then provided
// as files to other apps if the drag leaves the window
// (see [Events.SetDragFiles]).
type FileDragger interface {
	// DragFiles returns the absolute paths of the files being dragged.
	DragFiles() []string
}

// SetDragFiles sets the files for the current drag, as absolute paths.
// If the drag leaves the window, they are provided as files to other
// apps, which can then copy or open them. Otherwise, the drag data from
// [Events.DragStart] is provided, if it is [mimedata.Mimes].
// It must be called after [Events.DragStart].
func (em *Events) SetDragFiles(files ...string) {
	em.dragFiles = files
}

// handleDragOS handles a mouse event during a drag that may need to be
// handed off to the operating system, when it leaves the window, and
// mouse events while the operating system has the drag.
// It returns true if the event has been handled.
func (em *Events) handleDragOS(e events.Event) bool {
	et := e.Type()
	if em.dragOS {
		switch et {
		case events.MouseDrag:
			e.SetHandled()
			return true
		case events.MouseUp: // the OS drag has ended
			em.dragOSEnd()
			e.SetHandled()
			return true
		case events.MouseMove: // missed the end of the OS drag
			em.dragOSEnd()
		}
		return false
	}
	if et != events.MouseDrag || tree.IsNil(em.drag) {
		return false
	}
	return em.dragToOS(e)
}

// dragToOS hands the current drag off to the operating system, if the
// mouse is outside of the window and the platform supports it, so that
// the data can be dropped in other windows and apps.
func (em *Events) dragToOS(e events.Event) bool {
	rw := em.RenderWindow()
	if rw == nil || rw.SystemWindow == nil {
		return false
	}
	if e.WindowPos().In(image.Rectangle{Max: rw.SystemWindow.Size()}) {
		return false
	}
	md, _ := em.dragData.(mimedata.Mimes)
	if len(md) == 0 && len(em.dragFiles) == 0 {
		return false
	}
	if !rw.SystemWindow.StartDrag(md, em.dragFiles, em.dragImage) {
		return false
	}
	if DebugSettings.EventTrace {
		fmt.Println(em.drag, "drag handed off to the operating system")
	}
	osDrag.Lock()
	osDrag.source = em.drag
	osDrag.data = em.dragData
	osDrag.files = em.dragFiles
	osDrag.Unlock()
	em.dragOS = true
	em.dragClearSprite()
	em.dragHoversClear(e)
	e.SetHandled()
	return true
}

// dragOSEnd is called when a drag handed off to the operating system ends,
// with the release of the mouse button. Any drop has already been sent
// by then, as an external drop event to the window where it happened.
func (em *Events) dragOSEnd() {
	em.drag = nil
	em.dragData = nil
	em.dragImage = nil
	em.dragFiles = nil
	em.dragOS = false
	em.press = nil
	em.dragPress = nil
	em.scene.WidgetWalkDown(func(cw Widget, cwb *WidgetBase) bool {
		cwb.dragStateReset()
		return tree.Continue
	})
	em.scene.Restyle()
}

// dragHoversClear clears the current drag hovers, sending DragLeave to
// each, and resets the drag state of all widgets.
func (em *Events) dragHoversClear(e events.Event) {
	em.dragHovers = em.updateHovers(nil, em.dragHovers, e, events.DragEnter, events.DragLeave)
	em.scene.WidgetWalkDown(func(cw Widget, cwb *WidgetBase) bool {
		cwb.dragStateReset()
		return tree.Continue
	})
	em.scene.Restyle()
}

// handleExternalDrag handles a drag-and-drop event from the operating
// system (see [events.Source.DragExternal]), for a drag from another
// window or app, or from this window after it left the window.
// DragMove updates the drag hovers as for a drag within the window,
// so that the targets show it, and Drop is sent to the deepest
// droppable widget under the mouse. If there is none, it is sent to all
// of the widgets under the mouse, deepest first, until it is handled.
func (em *Events) handleExternalDrag(de *events.DragDrop) {
	if de.Typ == events.DragLeave {
		em.dragHoversClear(de)
		return
	}
	em.mouseInBBox = nil
	em.getMouseInBBox(em.scene, de.Pos())
	var hovs []Widget
	for _, w := range em.mouseInBBox {
		if w.AsWidget().AbilityIs(abilities.Droppable) {
			hovs = append(hovs, w)
		}
	}
	if de.Typ == events.DragMove {
		em.dragHovers = em.updateHovers(hovs, em.dragHovers, de, events.DragEnter, events.DragLeave)
		de.SetHandled()
		return
	}
	if de.Typ != events.Drop {
		return
	}
	em.dragHoversClear(de)
	var targ Widget
	if len(hovs) > 0 {
		targ = hovs[len(hovs)-1]
	}
	if de.FromApp {
		osDrag.Lock()
		src, data, files := osDrag.source, osDrag.data, osDrag.files
		osDrag.source, osDrag.data, osDrag.files = nil, nil, nil
		osDrag.Unlock()
		if !tree.IsNil(src) && (targ != nil || len(files) == 0) {
			de.Source = src
			de.Data = data
		}
		de.DefaultMod()
	} else {
		de.DropMod = events.DropCopy
	}
	if DebugSettings.EventTrace {
		fmt.Println(targ, "external Drop with mod:", de.DropMod, "source:", de.Source)
	}
	if targ != nil {
		de.Target = targ
		targ.AsWidget().HandleEvent(de)
		return
	}
	for i := len(em.mouseInBBox) - 1; i >= 0 && !de.IsHandled(); i-- {
		em.mouseInBBox[i].AsWidget().HandleEvent(de)
	}
}

// DroppedFiles returns the absolute paths of the existing files in
// the given drop: those of the source if it is a [FileDragger],
// and otherwise the text items of the data that are such paths
// (including file:// URLs), as for files dropped from other apps.
func DroppedFiles(de *events.DragDrop) []string {
	if fd, ok := de.Source.(FileDragger); ok {
		return fd.DragFiles()
	}
	md, _ := de.Data.(mimedata.Mimes)
	var paths []string
	for _, d := range md {
		if d.Type != fileinfo.TextPlain {
			continue
		}
		p := strings.TrimPrefix(string(d.Data), "file://")
		if filepath.IsAbs(p) && errors.Ignore1(fsx.FileExists(p)) {
			paths = append(paths, p)
		}
	}
	return paths
}
