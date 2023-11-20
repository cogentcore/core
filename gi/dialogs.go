// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"log/slog"

	"goki.dev/colors"
	"goki.dev/gi/v2/keyfun"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi/events"
	"goki.dev/gti"
)

// NewDialog returns a new [Dialog] in the context of the given widget,
// optionally with the given name.
func NewDialog(sc *Scene) Stage {
	// if ctx == nil {
	// 	ctx = CurRenderWin.MainScene()
	// }
	// d := &Dialog{}
	// nm := ""
	// if len(name) > 0 {
	// 	nm = name[0]
	// } else if ctx != nil {
	// 	nm = ctx.Name() + "-dialog"
	// }
	//
	// 	d.InitName(d, nm)
	// 	d.EventMgr.Scene = &d.Scene
	// 	d.BgColor.SetSolid(colors.Transparent)
	// 	d.DialogStyles()
	sc.DialogStyles()
	sc.Stage = NewMainStage(DialogStage, sc)
	sc.Stage.SetModal(true)
	return sc.Stage
}

// RecycleDialog looks for a dialog with the given data. If it
// finds it, it shows it and returns true. Otherwise, it returns false.
func RecycleDialog(data any) bool {
	rw, got := DialogRenderWins.FindData(data)
	if !got {
		return false
	}
	rw.Raise()
	return true
}

// ErrorDialog returns a new Dialog [Stage] displaying the given error
// in the context of the given widget.  Optional title can be provided.
func ErrorDialog(ctx Widget, err error, title ...string) Stage {
	ttl := "There was an error"
	if len(title) > 0 {
		ttl = title[0]
	}
	sc := NewScene(NewBody(ctx.Name() + "-error-dialog").AddTitle(ttl).AddText(err.Error()))
	sc.Footer.Add(func(par Widget) { sc.AddOk(par) })
	return NewDialog(sc).SetContext(ctx)
}

// AddOk adds an OK button to given parent Widget (typically in Bottom
// function), connecting to Close method the Ctrl+Enter keychord event.
// Close sends a Change event to the Scene for listeners there.
// Should add an OnClick listener to this button to perform additional
// specific actions needed beyond Close.
// Name should be passed when there are multiple effective OK buttons.
func (sc *Scene) AddOk(par Widget, name ...string) *Button {
	nm := "ok"
	if len(name) > 0 {
		nm = name[0]
	}
	bt := NewButton(par, nm).SetText("OK")
	bt.OnClick(func(e events.Event) {
		e.SetHandled() // otherwise propagates to dead elements
		sc.Close()
	})
	sc.OnKeyChord(func(e events.Event) {
		kf := keyfun.Of(e.KeyChord())
		if kf == keyfun.Accept {
			e.SetHandled()
			sc.Close()
		}
	})
	return bt
}

// AddCancel adds Cancel button to given parent Widget
// (typically in Bottom function),
// connecting to Close method and the Esc keychord event.
// Close sends a Change event to the Scene for listeners there.
// Should add an OnClick listener to this button to perform additional
// specific actions needed beyond Close.
// Name should be passed when there are multiple effective Cancel buttons (rare).
func (sc *Scene) AddCancel(par Widget, name ...string) *Button {
	nm := "cancel"
	if len(name) > 0 {
		nm = name[0]
	}
	bt := NewButton(par, nm).SetType(ButtonOutlined).SetText("Cancel")
	bt.OnClick(func(e events.Event) {
		e.SetHandled() // otherwise propagates to dead elements
		sc.Close()
	})
	sc.OnKeyChord(func(e events.Event) {
		kf := keyfun.Of(e.KeyChord())
		if kf == keyfun.Abort {
			e.SetHandled()
			sc.Close()
		}
	})
	return bt
}

// Close closes the stage associated with this Scene (typically for Dialog)
func (sc *Scene) Close() {
	sc.Send(events.Change)
	mm := sc.Stage.AsMain().StageMgr
	if mm == nil {
		slog.Error("dialog has no MainMgr")
		return
	}
	if sc.Stage.AsBase().NewWindow {
		mm.RenderWin.CloseReq()
		return
	}
	mm.PopDeleteType(DialogStage) // todo: this is probably not right
}

// DialogStyles sets default style functions for dialog Scenes
func (sc *Scene) DialogStyles() {
	sc.Style(func(s *styles.Style) {
		// s.Border.Radius = styles.BorderRadiusExtraLarge
		s.Direction = styles.Column
		s.Color = colors.Scheme.OnSurface
		if !sc.Stage.AsBase().NewWindow && !sc.Stage.AsBase().FullWindow {
			s.Padding.Set(units.Dp(24))
			// s.Justify.Content = styles.Center // vert
			// s.Align.Content = styles.Center // horiz
			s.Border.Radius = styles.BorderRadiusLarge
			s.BoxShadow = styles.BoxShadow3()
			// material likes SurfaceContainerHigh here, but that seems like too much; STYTODO: maybe figure out a better background color setup for dialogs?
			s.BackgroundColor.SetSolid(colors.Scheme.SurfaceContainerLow)
		}
	})
}

// NewItemsData contains the data necessary to make a certain
// number of items of a certain type, which can be used with a
// StructView in new item dialogs.
type NewItemsData struct {
	// Number is the number of elements to create
	Number int
	// Type is the type of elements to create
	Type *gti.Type
}
