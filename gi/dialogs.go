// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"cogentcore.org/core/colors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/gti"
	"cogentcore.org/core/keyfun"
	"cogentcore.org/core/laser"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
)

// todo: need a mechanism for nil Context to attach later

// NonNilContext returns a non-nil context widget, falling back on the top
// scene of the current window.
func NonNilContext(ctx Widget) Widget {
	if !laser.AnyIsNil(ctx) {
		return ctx
	}
	return CurRenderWin.MainStageMgr.Top().Scene
}

// NewDialog returns a new [DialogStage] that does not take up the
// full window it is created in, in the context of the given widget.
// See [Body.NewFullDialog] for a full-window dialog.
func (bd *Body) NewDialog(ctx Widget) *Stage {
	ctx = NonNilContext(ctx)
	bd.DialogStyles()
	bd.Scene.Stage = NewMainStage(DialogStage, bd.Scene)
	bd.Scene.Stage.SetModal(true)
	bd.Scene.Stage.SetContext(ctx)
	bd.Scene.Stage.Pos = ctx.ContextMenuPos(nil)
	return bd.Scene.Stage
}

// NewFullDialog returns a new [DialogStage] that takes up the full
// window it is created in, in the context of the given widget, optionally
// with the given name. See [Body.NewDialog] for a non-full-window dialog.
func (bd *Body) NewFullDialog(ctx Widget) *Stage {
	bd.DialogStyles()
	bd.Scene.Stage = NewMainStage(DialogStage, bd.Scene)
	bd.Scene.Stage.SetModal(true)
	bd.Scene.Stage.SetContext(ctx)
	bd.Scene.Stage.SetFullWindow(true)
	if ctx != nil {
		bd.Scene.InheritBarsWidget(ctx)
	}
	return bd.Scene.Stage
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

// MessageDialog opens a new Dialog displaying the given message
// in the context of the given widget. An optional title can be provided.
func MessageDialog(ctx Widget, msg string, title ...string) {
	b := NewBody(ctx.Name() + "-message-dialog")
	if len(title) > 0 {
		b.AddTitle(title[0])
	}
	b.AddText(msg).AddOKOnly()
	b.NewDialog(ctx).Run()
}

// ErrorDialog opens a new Dialog displaying the given error
// in the context of the given widget. An optional title can
// be provided; if it is not, the title will default to
// "There was an error". If the given error is nil, no dialog
// is created.
func ErrorDialog(ctx Widget, err error, title ...string) {
	if err == nil {
		return
	}
	ttl := "There was an error"
	if len(title) > 0 {
		ttl = title[0]
	}
	NewBody(ctx.Name() + "-error-dialog").AddTitle(ttl).AddText(err.Error()).
		AddOKOnly().NewDialog(ctx).Run()
}

// AddOK adds an OK button to given parent Widget (typically in Bottom
// Bar function), connecting to Close method the Ctrl+Enter keychord event.
// Close sends a Change event to the Scene for listeners there.
// Should add an OnClick listener to this button to perform additional
// specific actions needed beyond Close.
// Name should be passed when there are multiple effective OK buttons.
func (bd *Body) AddOK(parent Widget, name ...string) *Button {
	nm := "ok"
	if len(name) > 0 {
		nm = name[0]
	}
	bt := NewButton(parent, nm).SetText("OK")
	bt.OnFirst(events.Click, func(e events.Event) { // first de-focus any active editors
		bt.FocusClear()
	})
	bt.OnFinal(events.Click, func(e events.Event) { // then close
		e.SetHandled() // otherwise propagates to dead elements
		bd.Close()
	})
	bd.Scene.OnFirst(events.KeyChord, func(e events.Event) {
		kf := keyfun.Of(e.KeyChord())
		if kf == keyfun.Accept {
			e.SetHandled()
			bt.Send(events.Click, e)
		}
	})
	return bt
}

// AddOKOnly just adds an OK button in the BottomBar
// for simple popup dialogs that just need that one button
func (bd *Body) AddOKOnly() *Body {
	bd.Scene.Bars.Bottom.Add(func(parent Widget) { bd.AddOK(parent) })
	return bd
}

// AddCancel adds Cancel button to given parent Widget
// (typically in Bottom Bar function),
// connecting to Close method and the Esc keychord event.
// Close sends a Change event to the Scene for listeners there.
// Should add an OnClick listener to this button to perform additional
// specific actions needed beyond Close.
// Name should be passed when there are multiple effective Cancel buttons (rare).
func (bd *Body) AddCancel(parent Widget, name ...string) *Button {
	nm := "cancel"
	if len(name) > 0 {
		nm = name[0]
	}
	bt := NewButton(parent, nm).SetType(ButtonOutlined).SetText("Cancel")
	bt.OnClick(func(e events.Event) {
		e.SetHandled() // otherwise propagates to dead elements
		bd.Close()
	})
	bd.OnFirst(events.KeyChord, func(e events.Event) {
		kf := keyfun.Of(e.KeyChord())
		if kf == keyfun.Abort {
			e.SetHandled()
			bt.Send(events.Click, e)
			bd.Close()
		}
	})
	bt.OnFirst(events.KeyChord, func(e events.Event) {
		kf := keyfun.Of(e.KeyChord())
		if kf == keyfun.Abort {
			e.SetHandled()
			bt.Send(events.Click, e)
			bd.Close()
		}
	})
	return bt
}

// Close closes the stage associated with this Body (typically for dialogs)
func (bd *Body) Close() {
	bd.Scene.Close()
}

// DialogStyles sets default stylers for dialog bodies.
// It is automatically called in [Body.NewDialog].
func (bd *Body) DialogStyles() {
	bd.Scene.BarsInherit.Top = true
	bd.Scene.Style(func(s *styles.Style) {
		s.Direction = styles.Column
		s.Color = colors.C(colors.Scheme.OnSurface)
		if !bd.Scene.Stage.NewWindow && !bd.Scene.Stage.FullWindow {
			s.Padding.Set(units.Dp(24))
			s.Border.Radius = styles.BorderRadiusLarge
			s.BoxShadow = styles.BoxShadow3()
			s.Background = colors.C(colors.Scheme.SurfaceContainerLow)
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
