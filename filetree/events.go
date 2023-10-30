// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filetree

import (
	"fmt"

	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/giv"
	"goki.dev/gi/v2/keyfun"
	"goki.dev/goosi/events"
)

func (fn *Node) HandleFileNodeEvents() {
	// note: put all mouse events in parts!
	// note: OnClick is grabbed by the parts first -- we don't see it
	/*
		fn.On(events.KeyChord, func(e events.Event) {
			kt := e.(*events.Key)
			fn.KeyInput(kt)
		})
			ftvwe.AddFunc(goosi.DNDEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d any) {
				de := d.(events.Event)
				tvvi := recv.Embed(TypeNode)
				if tvvi == nil {
					return
				}
				tvv := tvvi.(*Node)
				switch de.Action {
				case events.Start:
					tvv.DragNDropStart()
				case events.DropOnTarget:
					tvv.DragNDropTarget(de)
				case events.DropFmSource:
					tvv.This().(gi.DragNDropper).Dragged(de)
				case events.External:
					tvv.DragNDropExternal(de)
				}
			})
			ftvwe.AddFunc(goosi.DNDFocusEvent, gi.RegPri, func(recv, send ki.Ki, sig int64, d any) {
				de := d.(*events.FocusEvent)
				tvvi := recv.Embed(TypeNode)
				if tvvi == nil {
					return
				}
				tvv := tvvi.(*Node)
				switch de.Action {
				case events.Enter:
					tvv.ParentRenderWin().DNDSetCursor(de.Mod)
				case events.Exit:
					tvv.ParentRenderWin().DNDNotCursor()
				case events.Hover:
					tvv.Open()
				}
			})
	*/
	fn.HandleTreeViewEvents()
}

func (fn *Node) KeyInput(kt events.Event) {
	if gi.KeyEventTrace {
		fmt.Printf("TreeView KeyInput: %v\n", fn.Path())
	}
	kf := keyfun.Of(kt.KeyChord())
	selMode := events.SelectModeBits(kt.Modifiers())

	if selMode == events.SelectOne {
		if fn.SelectMode() {
			selMode = events.ExtendContinuous
		}
	}

	// first all the keys that work for ReadOnly and active
	if !fn.IsReadOnly() && !kt.IsHandled() {
		switch kf {
		case keyfun.Delete:
			fn.DeleteFiles()
			kt.SetHandled()
			// todo: remove when gi issue 237 is resolved
		case keyfun.Backspace:
			fn.DeleteFiles()
			kt.SetHandled()
		case keyfun.Duplicate:
			fn.DuplicateFiles()
			kt.SetHandled()
		case keyfun.Insert: // New File
			giv.NewFuncButton(fn, fn.NewFile).CallFunc()
			kt.SetHandled()
		case keyfun.InsertAfter: // New Folder
			giv.NewFuncButton(fn, fn.NewFolder).CallFunc()
			kt.SetHandled()
		}
	}
	if !kt.IsHandled() {
		fn.HandleTreeViewKeyChord(kt)
	}
}
