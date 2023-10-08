// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"log"

	"goki.dev/colors"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi/events"
)

// standard vertical space between elements in a dialog, in Ex units
var (
	StdDialogVSpace = float32(1)

	StdDialogVSpaceUnits = units.Ex(StdDialogVSpace)
)

// DialogStage is a MainStage with methods for configuring a dialog
type DialogStage struct {
	Stage *MainStage

	// Data has arbitrary data for this dialog
	Data any

	// Accepted means that the dialog was accepted -- else canceled
	Accepted bool

	// ButtonBox goes here when added
	ButtonBox *Layout
}

// NewDialog returns a new DialogStage stage with given scene contents,
// in connection with given widget (which provides key context).
// Make further configuration choices using Set* methods, which
// can be chained directly after the New call.
// Use an appropriate Run call at the end to start the Stage running.
func NewDialog(sc *Scene, ctx Widget) *DialogStage {
	dlg := &DialogStage{}
	dlg.Stage = NewMainStage(Dialog, sc, ctx)
	sc.Geom.Pos = ctx.ContextMenuPos()
	if dlg.Stage.Title != "" {
		dlg.Title(dlg.Stage.Title)
	}
	dlg.DefaultStyle()
	return dlg
}

func (dlg *DialogStage) Run() *DialogStage {
	dlg.Stage.Run()
	return dlg
}

// Title adds title to dialog.  If title string is passed
// then a new title is set -- otherwise the existing Title is used.
func (dlg *DialogStage) Title(title ...string) *DialogStage {
	if len(title) > 0 {
		dlg.Stage.Title = title[0]
	}
	NewLabel(dlg.Stage.Scene, "title").SetText(dlg.Stage.Title).
		SetType(LabelHeadlineSmall).AddStyles(func(s *styles.Style) {
		s.MaxWidth.SetDp(-1)
		s.AlignH = styles.AlignCenter
		s.AlignV = styles.AlignTop
		s.BackgroundColor.SetSolid(colors.Transparent)
	})
	return dlg
}

// Prompt adds given prompt to dialog.
func (dlg *DialogStage) Prompt(prompt string) *DialogStage {
	NewLabel(dlg.Stage.Scene, "prompt").SetText(prompt).
		SetType(LabelBodyMedium).AddStyles(func(s *styles.Style) {
		s.Text.WhiteSpace = styles.WhiteSpaceNormal
		s.MaxWidth.SetDp(-1)
		s.Width.SetCh(30)
		s.Text.Align = styles.AlignLeft
		s.AlignV = styles.AlignTop
		s.Color = colors.Scheme.OnSurfaceVariant
		s.BackgroundColor.SetSolid(colors.Transparent)
	})
	return dlg
}

// Modal sets the modal behavior of the dialog:
// true = blocks all other input, false = allows other input
func (dlg *DialogStage) Modal(modal bool) *DialogStage {
	dlg.Stage.Modal = modal
	return dlg
}

// NewWindow sets whether dialog opens in a new window
// or on top of the existing window.
func (dlg *DialogStage) NewWindow(newWindow bool) *DialogStage {
	dlg.Stage.NewWindow = newWindow
	return dlg
}

// ConfigButtonBox adds layout for holding buttons at bottom of dialog
// and saves as ButtonBox field, if not already done.
func (dlg *DialogStage) ConfigButtonBox() *Layout {
	if dlg.ButtonBox != nil {
		return dlg.ButtonBox
	}
	bb := NewLayout(dlg.Stage.Scene, "buttons").
		SetLayout(LayoutHoriz)
	bb.AddStyles(func(s *styles.Style) {
		bb.Spacing.SetDp(8 * Prefs.DensityMul())
		s.SetStretchMaxWidth()
	})
	dlg.ButtonBox = bb
	return bb
}

// Ok adds Ok button to the ButtonBox at bottom of dialog,
// connecting to Accept method the Ctrl+Enter keychord event.
// Also sends a Change event to the dialog scene for listeners there.
func (dlg *DialogStage) Ok() *DialogStage {
	bb := dlg.ConfigButtonBox()
	sc := dlg.Stage.Scene
	NewButton(bb, "ok").SetType(ButtonText).SetText("OK").On(events.Click, func(e events.Event) {
		e.SetHandled() // otherwise propagates to dead elements
		dlg.AcceptDialog()
		sc.Send(events.Change, e)
	})
	sc.On(events.KeyChord, func(e events.Event) {
		kf := KeyFun(e.KeyChord())
		if kf == KeyFunAccept {
			dlg.AcceptDialog()
		}
	})
	return dlg
}

// Cancel adds Cancel button to the ButtonBox at bottom of dialog,
// connecting to Cancel method and the Esc keychord event.
// Also sends a Change event to the dialog scene for listeners there
func (dlg *DialogStage) Cancel() *DialogStage {
	bb := dlg.ConfigButtonBox()
	sc := dlg.Stage.Scene
	NewButton(bb, "cancel").SetType(ButtonText).SetText("Cancel").On(events.Click, func(e events.Event) {
		e.SetHandled() // otherwise propagates to dead elements
		dlg.CancelDialog()
		sc.Send(events.Change, e)
	})
	sc.On(events.KeyChord, func(e events.Event) {
		kf := KeyFun(e.KeyChord())
		if kf == KeyFunAbort {
			dlg.CancelDialog()
		}
	})
	return dlg
}

// OkCancel adds Ok, Cancel buttons,
// and standard Esc = Cancel, Ctrl+Enter keyboard action
func (dlg *DialogStage) OkCancel() *DialogStage {
	dlg.Cancel()
	dlg.Ok()
	return dlg
}

// AcceptDialog accepts the dialog, activated by the default Ok button
func (dlg *DialogStage) AcceptDialog() {
	dlg.Accepted = true
	dlg.Close()
}

// CancelDialog cancels the dialog, activated by the default Cancel button
func (dlg *DialogStage) CancelDialog() {
	dlg.Accepted = false
	dlg.Close()
}

// Close closes this stage as a popup
func (dlg *DialogStage) Close() {
	mm := dlg.Stage.StageMgr
	if mm == nil {
		log.Println("ERROR: dlg has no MainMgr")
		return
	}
	if dlg.Stage.NewWindow {
		mm.RenderWin.CloseReq()
		return
	}
	mm.PopDelete()
}

// DefaultStyle sets default style functions for dialog Scene
func (dlg *DialogStage) DefaultStyle() {
	st := dlg.Stage
	sc := st.Scene
	sc.AddStyles(func(s *styles.Style) {
		// material likes SurfaceContainerHigh here, but that seems like too much; STYTODO: maybe figure out a better background color setup for dialogs?
		s.BackgroundColor.SetSolid(colors.Scheme.SurfaceContainer)
		s.Border.Radius = styles.BorderRadiusLarge
		// s.Border.Radius = styles.BorderRadiusExtraLarge
		s.Color = colors.Scheme.OnSurface
		sc.Spacing = StdDialogVSpaceUnits
		s.Border.Style.Set(styles.BorderNone)
		s.Padding.Set(units.Dp(24 * Prefs.DensityMul()))
		if !st.NewWindow {
			s.BoxShadow = styles.BoxShadow3()
		}
	})
}

// DlgOpts are the basic dialog options for standard dialog methods.
// provides a named, optional way to specify these args
type DlgOpts struct {

	// generally should be provided -- used for setting name of dialog and associated window
	Title string

	// optional more detailed description of what is being requested and how it will be used -- is word-wrapped and can contain full html formatting etc.
	Prompt string

	// display the Ok button
	Ok bool

	// display the Cancel button.
	Cancel bool

	// Data for dialogs that return specific data
	Data any
}

// StdDialog configures a standard DialogStage per the options provided.
// Context provides the relevant source context opening the dialog.
func StdDialog(opts DlgOpts, ctx Widget) *DialogStage {
	dlg := NewDialog(StageScene("std-dialog"), ctx)
	if opts.Title != "" {
		dlg.Title(opts.Title)
	}
	if opts.Prompt != "" {
		dlg.Prompt(opts.Prompt)
	}
	if opts.Ok {
		dlg.Ok()
	}
	if opts.Cancel {
		dlg.Cancel()
	}
	dlg.Modal(true).NewWindow(false)
	return dlg
}

/*
// RecycleStdDialog looks for existing dialog window with same Data --
// if found brings that to the front, returns it, and true bool.
// else (and if data is nil) calls NewStdDialog, returns false.
func RecycleStdDialog(data any, opts DlgOpts, ok, cancel bool) (*Dialog, bool) {
	if data == nil {
		return NewStdDialog(opts, ok, cancel), false
	}
	ew, has := DialogRenderWins.FindData(data)
	if has && ew.Scene.NumChildren() > 0 {
		ew.RenderWin.Raise()
		// dlg := ew.Child(0).Embed(TypeDialog).(*Dialog)
		// return dlg, true
	}
	dlg := NewStdDialog(opts, ok, cancel)
	dlg.Data = data
	return dlg, false
}
*/

//////////////////////////////////////////////////////////////////////////
//     Specific Dialogs

// PromptDialog opens a standard dialog configured via options.
// The given closure will be called with the dialog when it returns,
// and the Accepted flag indicates if Ok or Cancel was pressed.
func PromptDialog(opts DlgOpts, ctx Widget, fun func(dlg *DialogStage)) *DialogStage {
	dlg := StdDialog(opts, ctx)
	if fun != nil {
		dlg.Stage.Scene.On(events.Change, func(e events.Event) {
			fun(dlg)
		})
	}
	return dlg
}

/*
// ChoiceDialog presents any number of buttons with labels as given, for the
// user to choose among -- the clicked button number (starting at 0) will be
// sent to the receiving object and function for dialog signals.  Scene is
// optional to properly contextualize dialog to given master window.
func ChoiceDialog(avp *Scene, opts DlgOpts, choices []string, fun func(dlg *DialogStage)) {
	dlg := NewStdDialog(opts, NoOk, NoCancel) // no buttons
	dlg.Modal = true
	if recv != nil && fun != nil {
		dlg.DialogSig.Connect(recv, fun)
	}

	bb := dlg.AddButtonBox() // not otherwise made because no buttons above
	NewStretch(bb, "stretch")
	for i, ch := range choices {
		chnm := strcase.ToKebab(ch)
		b := NewButton(bb, chnm)
		b.SetProp("__cdSigVal", int64(i))
		b.SetText(ch)
		if chnm == "cancel" {
			b.ButtonSig.Connect(dlg.Frame.This(), func(recv, send ki.Ki, sig int64, data any) {
				if sig == int64(ButtonClicked) {
					dlg.SigVal = b.Prop("__cdSigVal").(int64)
					dlg.Cancel()
				}
			})
		} else {
			b.ButtonSig.Connect(dlg.Frame.This(), func(recv, send ki.Ki, sig int64, data any) {
				if sig == int64(ButtonClicked) {
					dlg.SigVal = b.Prop("__cdSigVal").(int64)
					dlg.Accept()
				}
			})
		}
	}

	dlg.Open(0, 0, avp, nil)
}

// NewKiDialog prompts for creating new item(s) of a given type, showing types
// that implement given interface.
// Use construct of form: reflect.TypeOf((*gi.Widget)(nil)).Elem()
// Optionally connects to given signal receiving object and function for
// dialog signals (nil to ignore).
func NewKiDialog(avp *Scene, iface reflect.Type, opts DlgOpts, recv ki.Ki, fun ki.RecvFunc) *Dialog {
	dlg := NewStdDialog(opts, AddOk, AddCancel)
	dlg.Modal = true

	_, prIdx := dlg.PromptWidget()

	nrow := dlg.Frame.InsertNewChild(LayoutType, prIdx+2, "n-row").(*Layout)
	nrow.Lay = LayoutHoriz

	lbl := NewLabel(nrow, "n-label")
	lbl.Text = "Number:  "

	nsb := NewSpinBox(nrow, "n-field")
	nsb.SetMin(1)
	nsb.Value = 1
	nsb.Step = 1

	tspc := dlg.Frame.InsertNewChild(SpaceType, prIdx+3, "type-space").(*Space)
	tspc.SetFixedHeight(units.Em(0.5))

	trow := dlg.Frame.InsertNewChild(LayoutType, prIdx+4, "t-row").(*Layout)
	trow.Lay = LayoutHoriz

	lbl = NewLabel(trow, "t-label")
	lbl.Text = "Type:    "

	typs := NewComboBox(trow, "types")
	_ = typs
	// todo: fix
	// typs.ItemsFromTypes(kit.Types.AllImplementersOf(iface, false), true, true, 50)

	if recv != nil && fun != nil {
		dlg.DialogSig.Connect(recv, fun)
	}
	dlg.Open(0, 0, avp, nil)
	return dlg
}

// NewKiDialogValues gets the user-set values from a NewKiDialog.
func NewKiDialogValues(dlg *Dialog) (int, reflect.Type) {
	nrow := dlg.Frame.ChildByName("n-row", 0).(*Layout)
	ntf := nrow.ChildByName("n-field", 0).(*SpinBox)
	n := int(ntf.Value)
	trow := dlg.Frame.ChildByName("t-row", 0).(*Layout)
	typs := trow.ChildByName("types", 0).(*ComboBox)
	var typ reflect.Type
	if typs.CurVal != nil {
		typ = typs.CurVal.(reflect.Type)
	} else {
		log.Printf("gi.NewKiDialogValues: type is nil\n")
	}
	return n, typ
}

// StringPromptDialog prompts the user for a string value -- optionally
// connects to given signal receiving object and function for dialog signals
// (nil to ignore).  Scene is optional to properly contextualize dialog to
// given master window.
func StringPromptDialog(avp *Scene, strval, placeholder string, opts DlgOpts, recv ki.Ki, fun ki.RecvFunc) *Dialog {
	dlg := NewStdDialog(opts, AddOk, AddCancel)
	dlg.Modal = true

	_, prIdx := dlg.PromptWidget()
	tf := dlg.Frame.InsertNewChild(TextFieldType, prIdx+1, "str-field").(*TextField)
	tf.Placeholder = placeholder
	tf.SetText(strval)
	tf.SetStretchMaxWidth()
	tf.SetMinPrefWidth(units.Ch(40))

	if recv != nil && fun != nil {
		dlg.DialogSig.Connect(recv, fun)
	}
	dlg.Open(0, 0, avp, nil)
	return dlg
}

// StringPromptDialogValue gets the string value the user set.
func StringPromptDialogValue(dlg *Dialog) string {
	tf := dlg.Frame.ChildByName("str-field", 0).(*TextField)
	return tf.Text()
}
*/
