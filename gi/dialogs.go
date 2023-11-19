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

// Dialog is a scene with methods for configuring a dialog
type Dialog struct { //goki:no-new
	Scene

	// Accepted means that the dialog was accepted -- else canceled
	Accepted bool `set:"-"`

	// Buttons go here when added
	Btns *Layout `set:"-"`
}

// NewDialog returns a new [Dialog] in the context of the given widget,
// optionally with the given name.
func NewDialog(ctx Widget, name ...string) *Dialog {
	if ctx == nil {
		ctx = CurRenderWin.MainScene()
	}
	d := &Dialog{}
	nm := ""
	if len(name) > 0 {
		nm = name[0]
	} else if ctx != nil {
		nm = ctx.Name() + "-dialog"
	}

	d.InitName(d, nm)
	d.EventMgr.Scene = &d.Scene
	d.BgColor.SetSolid(colors.Transparent)
	d.DialogStyles()

	d.Stage = NewMainStage(DialogStage, &d.Scene, ctx)
	d.Modal(true)
	return d
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

// ErrorDialog returns a new [Dialog] displaying the given error
// in the context of the given widget.
func ErrorDialog(ctx Widget, err error) *Dialog {
	return NewDialog(ctx, ctx.Name()+"-error-dialog").Title("There was an error").Prompt(err.Error()).Ok()
}

// Title adds the given title to the dialog
func (d *Dialog) Title(title string) *Dialog {
	d.Scene.Title = title
	NewLabel(d, "title").SetText(title).
		SetType(LabelHeadlineSmall).Style(func(s *styles.Style) {
		s.Align.X = styles.AlignCenter
		s.Align.Y = styles.AlignStart
	})
	return d
}

// Prompt adds the given prompt to the dialog
func (d *Dialog) Prompt(prompt string) *Dialog {
	NewLabel(d, "prompt").SetText(prompt).
		SetType(LabelBodyMedium).Style(func(s *styles.Style) {
		s.Min.X.Ch(30)
		s.Text.Align = styles.AlignStart
		s.Align.Y = styles.AlignStart
		s.Color = colors.Scheme.OnSurfaceVariant
	})
	return d
}

// Buttons returns the layout for holding the buttons at bottom
// of the dialog, creating it if it does not already exist.
func (d *Dialog) Buttons() *Layout {
	if d.Btns != nil {
		return d.Btns
	}
	bb := NewLayout(d, "buttons")
	bb.Style(func(s *styles.Style) {
		s.Gap.Set(units.Dp(8))
	})
	bb.OnWidgetAdded(func(w Widget) {
		// new window and full window dialogs don't need text buttons
		if bt := AsButton(w); bt != nil && !d.MainStage().FullWindow && !d.MainStage().NewWindow {
			bt.Type = ButtonText
		}
	})
	d.Btns = bb
	NewStretch(bb)
	return bb
}

// Ok adds an OK button to the Buttons at bottom of dialog,
// connecting to Accept method the Ctrl+Enter keychord event.
// Also sends a Change event to the dialog for listeners there.
// If text is passed, that text is used for the text of the button
// instead of the standard "OK".
func (d *Dialog) Ok(text ...string) *Dialog {
	bb := d.Buttons()
	txt := "OK"
	if len(text) > 0 {
		txt = text[0]
	}
	NewButton(bb, "ok").SetText(txt).OnClick(func(e events.Event) {
		e.SetHandled() // otherwise propagates to dead elements
		d.AcceptDialog()
	})
	d.OnKeyChord(func(e events.Event) {
		kf := keyfun.Of(e.KeyChord())
		if kf == keyfun.Accept {
			e.SetHandled()
			d.AcceptDialog()
		}
	})
	return d
}

// Cancel adds Cancel button to the Buttons at bottom of dialog,
// connecting to Cancel method and the Esc keychord event.
// Also sends a Change event to the dialog scene for listeners there.
// If text is passed, that text is used for the text of the button
// instead of the standard "Cancel".
func (d *Dialog) Cancel(text ...string) *Dialog {
	bb := d.Buttons()
	txt := "Cancel"
	if len(text) > 0 {
		txt = text[0]
	}
	bt := NewButton(bb, "cancel").SetText(txt)
	if d.MainStage().FullWindow || d.MainStage().NewWindow {
		bt.SetType(ButtonOutlined)
	}
	bt.OnClick(func(e events.Event) {
		e.SetHandled() // otherwise propagates to dead elements
		d.CancelDialog()
	})
	d.OnKeyChord(func(e events.Event) {
		kf := keyfun.Of(e.KeyChord())
		if kf == keyfun.Abort {
			e.SetHandled()
			d.CancelDialog()
		}
	})
	return d
}

// Modal sets whether the dialog is modal
func (d *Dialog) Modal(modal bool) *Dialog {
	d.Stage.SetModal(modal)
	return d
}

// NewWindow sets whether the dialog takes up the full window
func (d *Dialog) NewWindow(newWindow bool) *Dialog {
	d.Stage.SetNewWindow(newWindow)
	return d
}

// FullWindow sets whether the dialog takes up the full window
func (d *Dialog) FullWindow(fullWindow bool) *Dialog {
	d.Stage.SetFullWindow(fullWindow)
	return d
}

// Run runs (shows) the dialog.
func (d *Dialog) Run() *Dialog {
	d.Stage.AsMain().Run()
	return d
}

// AcceptDialog accepts the dialog, activated by the default Ok button
func (d *Dialog) AcceptDialog() {
	d.Accepted = true
	d.Send(events.Change)
	d.Close()
}

// CancelDialog cancels the dialog, activated by the default Cancel button
func (d *Dialog) CancelDialog() {
	d.Accepted = false
	d.Send(events.Change)
	d.Close()
}

// OnAccept adds an event listener for when the dialog is accepted
// (closed in a positive or neutral way)
func (d *Dialog) OnAccept(fun func(e events.Event)) *Dialog {
	d.OnChange(func(e events.Event) {
		if d.Accepted {
			fun(e)
		}
	})
	return d
}

// OnCancel adds an event listener for when the dialog is canceled
// (closed in a negative way)
func (d *Dialog) OnCancel(fun func(e events.Event)) *Dialog {
	d.OnChange(func(e events.Event) {
		if !d.Accepted {
			fun(e)
		}
	})
	return d
}

// Close closes the stage associated with this dialog
func (d *Dialog) Close() {
	mm := d.Stage.AsMain().StageMgr
	if mm == nil {
		slog.Error("dialog has no MainMgr")
		return
	}
	if d.Stage.AsBase().NewWindow {
		mm.RenderWin.CloseReq()
		return
	}
	mm.PopDeleteType(DialogStage)
}

// DefaultStyle sets default style functions for dialog Scene
func (d *Dialog) DialogStyles() {
	d.Style(func(s *styles.Style) {
		// s.Border.Radius = styles.BorderRadiusExtraLarge
		s.Direction = styles.Col
		s.Color = colors.Scheme.OnSurface
		if !d.Stage.AsBase().NewWindow && !d.Stage.AsBase().FullWindow {
			s.Padding.Set(units.Dp(24))
			s.Align.Set(styles.AlignCenter)
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
