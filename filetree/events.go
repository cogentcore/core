// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package filetree

import (
	"fmt"

	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/giv"
	"goki.dev/gi/v2/keyfun"
	"goki.dev/girl/states"
	"goki.dev/goosi/events"
	"goki.dev/icons"
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

func (fn *Node) FileNodeContextMenu(m *gi.Scene) {
	gi.NewButton(m).SetText("Show file info").SetIcon(icons.Info).
		SetState(!fn.HasSelection(), states.Disabled).
		OnClick(func(e events.Event) {
			fn.ShowFileInfo()
		})
	gi.NewButton(m).SetText("Insert Before").SetIcon(icons.Add).SetState(!fn.HasSelection(), states.Disabled).
		OnClick(func(e events.Event) {
			fn.InsertBefore()
		})
	gi.NewButton(m).SetText("Insert After").SetIcon(icons.Add).SetState(!fn.HasSelection(), states.Disabled).
		OnClick(func(e events.Event) {
			fn.InsertAfter()
		})
	gi.NewButton(m).SetText("Duplicate").SetIcon(icons.TabDuplicate).SetState(!fn.HasSelection(), states.Disabled).
		OnClick(func(e events.Event) {
			fn.Duplicate()
		})
	gi.NewButton(m).SetText("Insert After").SetIcon(icons.Add).SetState(!fn.HasSelection(), states.Disabled).
		OnClick(func(e events.Event) {
			fn.InsertAfter()
		})
	gi.NewButton(m).SetText("Delete").SetIcon(icons.Delete).SetKey(keyfun.Delete).SetState(!fn.HasSelection(), states.Disabled).
		OnClick(func(e events.Event) {
			fn.DeleteNode()
		})
	gi.NewSeparator(m)
	gi.NewButton(m).SetText("Copy").SetIcon(icons.ContentCopy).SetKey(keyfun.Copy).SetState(!fn.HasSelection(), states.Disabled).
		OnClick(func(e events.Event) {
			fn.This().(gi.Clipper).Copy(true)
		})
	gi.NewButton(m).SetText("Cut").SetIcon(icons.ContentCut).SetKey(keyfun.Cut).SetState(!fn.HasSelection(), states.Disabled).
		OnClick(func(e events.Event) {
			fn.This().(gi.Clipper).Cut()
		})
	pbt := gi.NewButton(m).SetText("Paste").SetIcon(icons.ContentPaste).SetKey(keyfun.Paste).
		OnClick(func(e events.Event) {
			fn.This().(gi.Clipper).Paste()
		})
	cb := fn.Sc.EventMgr.ClipBoard()
	if cb != nil {
		pbt.SetState(cb.IsEmpty(), states.Disabled)
	}
	gi.NewSeparator(m)
	gi.NewButton(m).SetText("Edit").SetIcon(icons.Edit).SetState(!fn.HasSelection(), states.Disabled).
		OnClick(func(e events.Event) {
			fn.EditNode()
		})
	gi.NewButton(m).SetText("GoGiEditor").SetIcon(icons.EditDocument).SetState(!fn.HasSelection(), states.Disabled).
		OnClick(func(e events.Event) {
			fn.GoGiEditNode()
		})
	gi.NewSeparator(m)
	gi.NewButton(m).SetText("Open All").SetIcon(icons.Open).SetState(!fn.HasSelection(), states.Disabled).
		OnClick(func(e events.Event) {
			fn.OpenAll()
		})
	gi.NewButton(m).SetText("Close All").SetIcon(icons.Close).SetState(!fn.HasSelection(), states.Disabled).
		OnClick(func(e events.Event) {
			fn.CloseAll()
		})
}

func (fn *Node) ContextMenu(m *gi.Scene) {
	// derived types put native menu code here
	if fn.CustomContextMenu != nil {
		fn.CustomContextMenu(m)
	}
	// TODO(kai/menu): need a replacement for this:
	// if tv.SyncNode != nil && CtxtMenuView(tv.SyncNode, tv.RootIsReadOnly(), tv.Scene, m) { // our viewed obj's menu
	// 	if tv.ShowViewCtxtMenu {
	// 		m.AddSeparator("sep-tvmenu")
	// 		CtxtMenuView(tv.This(), tv.RootIsReadOnly(), tv.Scene, m)
	// 	}
	// } else {
	if fn.IsReadOnly() {
		fn.TreeViewContextMenuReadOnly(m)
	} else {
		fn.TreeViewContextMenu(m)
	}
}
