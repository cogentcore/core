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

// todo: need a mechanism for nil Context to attach later

// NewDialog returns a new PopupWindow dialog [Stage] in the context
// of the given widget, optionally with the given name.
// See [NewFullDialog] for a full-window dialog.
func (sc *Scene) NewDialog(ctx Widget, name ...string) *Stage {
	sc.DialogStyles()
	sc.Stage = NewMainStage(DialogStage, sc)
	sc.Stage.SetModal(true)
	sc.Stage.SetContext(ctx)
	sc.Stage.Pos = ctx.ContextMenuPos(nil)
	return sc.Stage
}

// NewFullDialog returns a new FullWindow dialog [Stage] in the context
// of the given widget, optionally with the given name.
// See [NewDialog] for a popup-window dialog.
func (sc *Scene) NewFullDialog(ctx Widget, name ...string) *Stage {
	sc.DialogStyles()
	sc.Stage = NewMainStage(DialogStage, sc)
	sc.Stage.SetModal(true)
	sc.Stage.SetContext(ctx)
	sc.Stage.SetFullWindow(true)
	if ctx != nil {
		sc.InheritBarsWidget(ctx)
	}
	return sc.Stage
}

// NewDialog returns a new PopupWindow dialog [Stage] in the context
// of the given widget, optionally with the given name.
// See [NewFullDialog] for a full-window dialog.
func (bd *Body) NewDialog(ctx Widget, name ...string) *Stage {
	return bd.Sc.NewDialog(ctx, name...)
}

// NewFullDialog returns a new FullWindow dialog [Stage] in the context
// of the given widget, optionally with the given name.
// See [NewDialog] for a popup-window dialog.
func (bd *Body) NewFullDialog(ctx Widget, name ...string) *Stage {
	return bd.Sc.NewFullDialog(ctx, name...)
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

// ErrorDialog opens a new Dialog displaying the given error
// in the context of the given widget.  Optional title can be provided.
func ErrorDialog(ctx Widget, err error, title ...string) {
	ttl := "There was an error"
	if len(title) > 0 {
		ttl = title[0]
	}
	NewBody(ctx.Name() + "-error-dialog").AddTitle(ttl).AddText(err.Error()).
		AddOkOnly().NewDialog(ctx).Run()
}

// AddOk adds an OK button to given parent Widget (typically in Bottom
// Bar function), connecting to Close method the Ctrl+Enter keychord event.
// Close sends a Change event to the Scene for listeners there.
// Should add an OnClick listener to this button to perform additional
// specific actions needed beyond Close.
// Name should be passed when there are multiple effective OK buttons.
func (sc *Scene) AddOk(pw Widget, name ...string) *Button {
	nm := "ok"
	if len(name) > 0 {
		nm = name[0]
	}
	bt := NewButton(pw, nm).SetText("OK")
	bt.OnClick(func(e events.Event) {
		e.SetHandled() // otherwise propagates to dead elements
		sc.Close()
	})
	sc.OnKeyChord(func(e events.Event) {
		kf := keyfun.Of(e.KeyChord())
		if kf == keyfun.Accept {
			bt.Send(events.Click, e)
			e.SetHandled()
			sc.Close()
		}
	})
	sc.AddPriorityEvent(events.KeyChord)
	return bt
}

// AddOkOnly just adds an OK button in the BottomBar
// for simple popup dialogs that just need that one button
func (sc *Scene) AddOkOnly() *Scene {
	sc.Bars.Bottom.Add(func(pw Widget) { sc.AddOk(pw) })
	return sc
}

// AddCancel adds Cancel button to given parent Widget
// (typically in Bottom Bar function),
// connecting to Close method and the Esc keychord event.
// Close sends a Change event to the Scene for listeners there.
// Should add an OnClick listener to this button to perform additional
// specific actions needed beyond Close.
// Name should be passed when there are multiple effective Cancel buttons (rare).
func (sc *Scene) AddCancel(pw Widget, name ...string) *Button {
	nm := "cancel"
	if len(name) > 0 {
		nm = name[0]
	}
	bt := NewButton(pw, nm).SetType(ButtonOutlined).SetText("Cancel")
	bt.OnClick(func(e events.Event) {
		e.SetHandled() // otherwise propagates to dead elements
		sc.Close()
	})
	sc.OnKeyChord(func(e events.Event) {
		kf := keyfun.Of(e.KeyChord())
		if kf == keyfun.Abort {
			e.SetHandled()
			bt.Send(events.Click, e)
			sc.Close()
		}
	})
	sc.AddPriorityEvent(events.KeyChord)
	return bt
}

// Close closes the stage associated with this Scene (typically for Dialog)
func (sc *Scene) Close() {
	sc.Send(events.Close, nil)
	if sc.Stage == nil {
		slog.Error("Close: Scene has no Stage")
		return
	}
	mm := sc.Stage.MainMgr
	if mm == nil {
		// slog.Error("Scene has no MainMgr")
		return
	}
	if sc.Stage.NewWindow {
		mm.RenderWin.CloseReq()
		return
	}
	mm.DeleteStage(sc.Stage)
}

// AddOk adds an OK button to given parent Widget (typically in Bottom
// Bar function), connecting to Close method the Ctrl+Enter keychord event.
// Close sends a Change event to the Scene for listeners there.
// Should add an OnClick listener to this button to perform additional
// specific actions needed beyond Close.
// Name should be passed when there are multiple effective OK buttons.
func (bd *Body) AddOk(pw Widget, name ...string) *Button {
	return bd.Sc.AddOk(pw, name...)
}

// AddOkOnly just adds an OK button in the BottomBar
// for simple popup dialogs that just need that one button
func (bd *Body) AddOkOnly() *Body {
	bd.Sc.AddOkOnly()
	return bd
}

// AddCancel adds Cancel button to given parent Widget
// (typically in Bottom Bar function),
// connecting to Close method and the Esc keychord event.
// Close sends a Change event to the Scene for listeners there.
// Should add an OnClick listener to this button to perform additional
// specific actions needed beyond Close.
// Name should be passed when there are multiple effective Cancel buttons (rare).
func (bd *Body) AddCancel(pw Widget, name ...string) *Button {
	return bd.Sc.AddCancel(pw, name...)
}

// Close closes the stage associated with this Scene (typically for Dialog)
func (bd *Body) Close() {
	bd.Sc.Close()
}

// DialogStyles sets default style functions for dialog Scenes
func (sc *Scene) DialogStyles() {
	sc.BarsInherit.Top = true
	sc.Style(func(s *styles.Style) {
		// s.Border.Radius = styles.BorderRadiusExtraLarge
		s.Direction = styles.Column
		s.Color = colors.Scheme.OnSurface
		if !sc.Stage.NewWindow && !sc.Stage.FullWindow {
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
