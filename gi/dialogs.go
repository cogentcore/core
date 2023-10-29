// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"log/slog"

	"github.com/iancoleman/strcase"
	"goki.dev/colors"
	"goki.dev/gi/v2/keyfun"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi/events"
	"goki.dev/gti"
)

var (
	// standard vertical space between elements in a dialog, in Ex units
	StdDialogVSpace = float32(1)

	StdDialogVSpaceUnits = units.Ex(StdDialogVSpace)
)

// Dialog is a scene with methods for configuring a dialog
type Dialog struct {
	// Scene is the scene associated with dialog
	Scene *Scene

	// Stage is the main stage associated with the dialog
	Stage *MainStage

	// Data has arbitrary data for this dialog
	Data any

	// RdOnly is whether the dialog is read only
	RdOnly bool

	// a record of parent View names that have led up to this dialog,
	// which is displayed as extra contextual information in view dialog windows
	VwPath string

	// Accepted means that the dialog was accepted -- else canceled
	Accepted bool

	// ButtonBox goes here when added
	ButtonBox *Layout
}

// NewDialog returns a new [Dialog] in the context of the given widget,
// optionally with the given name.
func NewDialog(ctx Widget, name ...string) *Dialog {
	dlg := &Dialog{}
	nm := ""
	if len(name) > 0 {
		nm = name[0]
	} else {
		nm = ctx.Name() + "-dialog"
	}
	dlg.Scene = NewScene(nm)
	dlg.Stage = NewMainStage(DialogStage, dlg.Scene, ctx)
	dlg.Modal(true)
	return dlg
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

// Title adds the given title to the dialog
func (dlg *Dialog) Title(title string) *Dialog {
	dlg.Scene.Title = title
	NewLabel(dlg.Scene, "title").SetText(title).
		SetType(LabelHeadlineSmall).Style(func(s *styles.Style) {
		s.SetStretchMaxWidth()
		s.AlignH = styles.AlignCenter
		s.AlignV = styles.AlignTop
	})
	return dlg
}

// Prompt adds the given prompt to the dialog
func (dlg *Dialog) Prompt(prompt string) *Dialog {
	NewLabel(dlg.Scene, "prompt").SetText(prompt).
		SetType(LabelBodyMedium).Style(func(s *styles.Style) {
		s.Text.WhiteSpace = styles.WhiteSpaceNormal
		s.SetStretchMaxWidth()
		s.Width.Ch(30)
		s.Text.Align = styles.AlignLeft
		s.AlignV = styles.AlignTop
		s.Color = colors.Scheme.OnSurfaceVariant
	})
	return dlg
}

// ConfigButtonBox adds layout for holding buttons at bottom of dialog
// and saves as ButtonBox field, if not already done.
func (dlg *Dialog) ConfigButtonBox() *Layout {
	if dlg.ButtonBox != nil {
		return dlg.ButtonBox
	}
	bb := NewLayout(dlg.Scene, "buttons").
		SetLayout(LayoutHoriz)
	bb.Style(func(s *styles.Style) {
		bb.Spacing.Dp(8)
		s.SetStretchMaxWidth()
	})
	dlg.ButtonBox = bb
	NewStretch(bb)
	return bb
}

// Ok adds an OK button to the ButtonBox at bottom of dialog,
// connecting to Accept method the Ctrl+Enter keychord event.
// Also sends a Change event to the dialog for listeners there.
// If text is passed, that text is used for the text of the button
// instead of the standard "OK".
func (dlg *Dialog) Ok(text ...string) *Dialog {
	bb := dlg.ConfigButtonBox()
	txt := "OK"
	if len(text) > 0 {
		txt = text[0]
	}
	NewButton(bb, "ok").SetType(ButtonText).SetText(txt).OnClick(func(e events.Event) {
		e.SetHandled() // otherwise propagates to dead elements
		dlg.AcceptDialog()
	})
	dlg.Scene.OnKeyChord(func(e events.Event) {
		kf := keyfun.Of(e.KeyChord())
		if kf == keyfun.Accept {
			e.SetHandled()
			dlg.AcceptDialog()
		}
	})
	return dlg
}

// Cancel adds Cancel button to the ButtonBox at bottom of dialog,
// connecting to Cancel method and the Esc keychord event.
// Also sends a Change event to the dialog scene for listeners there.
// If text is passed, that text is used for the text of the button
// instead of the standard "Cancel".
func (dlg *Dialog) Cancel(text ...string) *Dialog {
	bb := dlg.ConfigButtonBox()
	txt := "Cancel"
	if len(text) > 0 {
		txt = text[0]
	}
	NewButton(bb, "cancel").SetType(ButtonText).SetText(txt).OnClick(func(e events.Event) {
		e.SetHandled() // otherwise propagates to dead elements
		dlg.CancelDialog()
	})
	dlg.Scene.OnKeyChord(func(e events.Event) {
		kf := keyfun.Of(e.KeyChord())
		if kf == keyfun.Abort {
			e.SetHandled()
			dlg.CancelDialog()
		}
	})
	return dlg
}

func (dlg *Dialog) ReadOnly(readOnly bool) *Dialog {
	dlg.RdOnly = readOnly
	return dlg
}

func (dlg *Dialog) ViewPath(viewPath string) *Dialog {
	dlg.VwPath = viewPath
	return dlg
}

func (dlg *Dialog) Modal(modal bool) *Dialog {
	dlg.Stage.Modal = modal
	return dlg
}

func (dlg *Dialog) NewWindow(newWindow bool) *Dialog {
	dlg.Stage.NewWindow = newWindow
	return dlg
}

func (dlg *Dialog) FullWindow(fullWindow bool) *Dialog {
	dlg.Stage.FullWindow = fullWindow
	return dlg
}

// Run runs (shows) the dialog.
func (dlg *Dialog) Run() {
	dlg.Stage.Run()
}

// StringPrompt adds to the dialog a prompt for a string value.
// The string is set as the Data field in the Dialog.
func (dlg *Dialog) StringPrompt(strval, placeholder string) *Dialog {
	tf := NewTextField(dlg.Scene).SetPlaceholder(placeholder).
		SetText(strval)
	tf.SetStretchMaxWidth().
		SetMinPrefWidth(units.Ch(40))
	tf.OnChange(func(e events.Event) {
		dlg.Data = tf.Text()
	})
	return dlg
}

/*
// NewDialog returns a new DialogStage stage with given scene contents,
// in connection with given widget (which provides key context).
// Make further configuration choices using Set* methods, which
// can be chained directly after the New call.
// Use an appropriate Run call at the end to start the Stage running.
func NewDialog(sc *Scene, ctx Widget) *Dialog {
	dlg := &Dialog{}
	dlg.Stage = NewMainStage(DialogStage, sc, ctx)
	sc.Geom.Pos = ctx.ContextMenuPos(nil)
	if dlg.Stage.Title != "" {
		dlg.Title(dlg.Stage.Title)
	}
	dlg.DefaultStyle()
	return dlg
}

func (dlg *Dialog) Run() *Dialog {
	dlg.Stage.Run()
	return dlg
}

// Title adds title to dialog.  If title string is passed
// then a new title is set -- otherwise the existing Title is used.
func (dlg *Dialog) Title(title ...string) *Dialog {
	if len(title) > 0 {
		dlg.Stage.Title = title[0]
	}
	NewLabel(dlg.Stage.Scene, "title").SetText(dlg.Stage.Title).
		SetType(LabelHeadlineSmall).Style(func(s *styles.Style) {
		s.MaxWidth.Dp(-1)
		s.AlignH = styles.AlignCenter
		s.AlignV = styles.AlignTop
		s.BackgroundColor.SetSolid(colors.Transparent)
	})
	return dlg
}

// Prompt adds given prompt to dialog.
func (dlg *Dialog) Prompt(prompt string) *Dialog {
	NewLabel(dlg.Stage.Scene, "prompt").SetText(prompt).
		SetType(LabelBodyMedium).Style(func(s *styles.Style) {
		s.Text.WhiteSpace = styles.WhiteSpaceNormal
		s.MaxWidth.Dp(-1)
		s.Width.Ch(30)
		s.Text.Align = styles.AlignLeft
		s.AlignV = styles.AlignTop
		s.Color = colors.Scheme.OnSurfaceVariant
		s.BackgroundColor.SetSolid(colors.Transparent)
	})
	return dlg
}
*/
// PromptWidgetIdx returns the prompt label widget index,
// for adding additional elements below the prompt. If it
// is not found, it returns the title label widget index.
// If neither are found, it returns -1.
func (dlg *Dialog) PromptWidgetIdx() int {
	idx, ok := dlg.Scene.Children().IndexByName("prompt", 1)
	if !ok {
		idx, ok := dlg.Scene.Children().IndexByName("title", 0)
		if !ok {
			return -1
		}
		return idx
	}
	return idx
}

// // Modal sets the modal behavior of the dialog:
// // true = blocks all other input, false = allows other input
// func (dlg *Dialog) Modal(modal bool) *Dialog {
// 	dlg.Stage.Modal = modal
// 	return dlg
// }

// // NewWindow sets whether dialog opens in a new window
// // or on top of the existing window.
// func (dlg *Dialog) NewWindow(newWindow bool) *Dialog {
// 	dlg.Stage.NewWindow = newWindow
// 	return dlg
// }

/*
// ConfigButtonBox adds layout for holding buttons at bottom of dialog
// and saves as ButtonBox field, if not already done.
func (dlg *Dialog) ConfigButtonBox() *Layout {
	if dlg.ButtonBox != nil {
		return dlg.ButtonBox
	}
	bb := NewLayout(dlg.Stage.Scene, "buttons").
		SetLayout(LayoutHoriz)
	bb.Style(func(s *styles.Style) {
		bb.Spacing.Dp(8)
		s.SetStretchMaxWidth()
	})
	dlg.ButtonBox = bb
	return bb
}

// Ok adds Ok button to the ButtonBox at bottom of dialog,
// connecting to Accept method the Ctrl+Enter keychord event.
// Also sends a Change event to the dialog scene for listeners there.
func (dlg *Dialog) Ok() *Dialog {
	bb := dlg.ConfigButtonBox()
	sc := dlg.Stage.Scene
	NewButton(bb, "ok").SetType(ButtonText).SetText("OK").OnClick(func(e events.Event) {
		e.SetHandled() // otherwise propagates to dead elements
		dlg.AcceptDialog()
	})
	sc.OnKeyChord(func(e events.Event) {
		kf := keyfun.Of(e.KeyChord())
		if kf == keyfun.Accept {
			e.SetHandled()
			dlg.AcceptDialog()
		}
	})
	return dlg
}

// Cancel adds Cancel button to the ButtonBox at bottom of dialog,
// connecting to Cancel method and the Esc keychord event.
// Also sends a Change event to the dialog scene for listeners there
func (dlg *Dialog) Cancel() *Dialog {
	bb := dlg.ConfigButtonBox()
	sc := dlg.Stage.Scene
	NewButton(bb, "cancel").SetType(ButtonText).SetText("Cancel").OnClick(func(e events.Event) {
		e.SetHandled() // otherwise propagates to dead elements
		dlg.CancelDialog()
	})
	sc.OnKeyChord(func(e events.Event) {
		kf := keyfun.Of(e.KeyChord())
		if kf == keyfun.Abort {
			e.SetHandled()
			dlg.CancelDialog()
		}
	})
	return dlg
}

// OkCancel adds Ok, Cancel buttons,
// and standard Esc = Cancel, Ctrl+Enter keyboard action
func (dlg *Dialog) OkCancel() *Dialog {
	dlg.Cancel()
	dlg.Ok()
	return dlg
}
*/

// AcceptDialog accepts the dialog, activated by the default Ok button
func (dlg *Dialog) AcceptDialog() {
	dlg.Accepted = true
	dlg.Scene.Send(events.Change)
	dlg.Close()
}

// CancelDialog cancels the dialog, activated by the default Cancel button
func (dlg *Dialog) CancelDialog() {
	dlg.Accepted = false
	dlg.Scene.Send(events.Change)
	dlg.Close()
}

// OnAccept adds an event listener for when the dialog is accepted
// (closed in a positive or neutral way)
func (dlg *Dialog) OnAccept(fun func(e events.Event)) *Dialog {
	dlg.Scene.OnChange(func(e events.Event) {
		if dlg.Accepted {
			fun(e)
		}
	})
	return dlg
}

// OnCancel adds an event listener for when the dialog is canceled
// (closed in a negative way)
func (dlg *Dialog) OnCancel(fun func(e events.Event)) *Dialog {
	dlg.Scene.OnChange(func(e events.Event) {
		if !dlg.Accepted {
			fun(e)
		}
	})
	return dlg
}

// Close closes the stage associated with this dialog
func (dlg *Dialog) Close() {
	mm := dlg.Stage.StageMgr
	if mm == nil {
		slog.Error("dialog has no MainMgr")
		return
	}
	if dlg.Stage.NewWindow {
		mm.RenderWin.CloseReq()
		return
	}
	mm.PopDeleteType(DialogStage)
}

// DefaultStyle sets default style functions for dialog Scene
func (dlg *Dialog) DialogStyles() {
	dlg.Scene.Style(func(s *styles.Style) {
		// s.Border.Radius = styles.BorderRadiusExtraLarge
		s.Color = colors.Scheme.OnSurface
		dlg.Scene.Spacing = StdDialogVSpaceUnits
		s.Padding.Left.Dp(8)
		if !dlg.Stage.NewWindow && !dlg.Stage.FullWindow {
			s.Padding.Set(units.Dp(24))
			s.Border.Radius = styles.BorderRadiusLarge
			s.BoxShadow = styles.BoxShadow3()
			// material likes SurfaceContainerHigh here, but that seems like too much; STYTODO: maybe figure out a better background color setup for dialogs?
			s.BackgroundColor.SetSolid(colors.Scheme.SurfaceContainerLow)
		}
	})
}

/*

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

// NewStdDialog configures a standard DialogStage per the options provided.
// Call Run() to run the returned dialog (can be further configed).
// Context provides the relevant source context opening the dialog,
// for positioning and constructing the dialog.
func NewStdDialog(ctx Widget, opts DlgOpts, fun func(dlg *Dialog)) *Dialog {
	// TOOD(kai/stage): need to have a unique name, so we use title, but that
	// is user-facing (has spaces and special characters), so ideally we could
	// use something else
	dlg := NewDialog(NewScene(opts.Title), ctx)
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
	dlg.Stage.ClickOff = true // by default
	if fun != nil {
		dlg.Stage.Scene.OnChange(func(e events.Event) {
			fun(dlg)
		})
	}
	return dlg
}

// RecycleStdDialog looks for existing dialog window with same Data.
// if found brings that to the front, returns it, and true bool.
// else (and if data is nil) calls StdDialog, returns false.
func RecycleStdDialog(ctx Widget, opts DlgOpts, data any, fun func(dlg *Dialog)) (*Dialog, bool) {
	if data == nil {
		return NewStdDialog(ctx, opts, fun), false
	}
	ew, has := DialogRenderWins.FindData(data) // todo: this needs to save DialogStage not renderwin
	_ = ew
	if has {
		// ew.RenderWin.Raise()
		// dlg := ew.Child(0).Embed(TypeDialog).(*Dialog)
		// return dlg, true
	}
	dlg := NewStdDialog(ctx, opts, fun)
	dlg.Data = data
	return dlg, false
}

//////////////////////////////////////////////////////////////////////////
//     Specific Dialogs

// TODO: this doesn't do anything beyond NewStdDialog?

// PromptDialog opens a standard dialog configured via options.
// The given closure will be called with the dialog when it returns,
// and the Accepted flag indicates if Ok or Cancel was pressed.
// Call Run() to run the returned dialog (can be further configed).
// Context provides the relevant source context opening the dialog,
// for positioning and constructing the dialog.
func PromptDialog(ctx Widget, opts DlgOpts, fun func(dlg *Dialog)) *Dialog {
	dlg := NewStdDialog(ctx, opts, fun)
	return dlg
}

*/

// Choice adds to the dialog any number of buttons with the given labels
// for the user to choose among. The clicked button index (starting at 0)
// is the [Dialog.Data].
func (dlg *Dialog) Choice(choices ...string) *Dialog {
	sc := dlg.Stage.Scene
	bb := dlg.ConfigButtonBox()
	NewStretch(bb, "stretch")
	for i, ch := range choices {
		chnm := strcase.ToKebab(ch)
		chidx := i

		b := NewButton(bb, chnm).SetType(ButtonText).SetText(ch)
		b.OnClick(func(e events.Event) {
			e.SetHandled() // otherwise propagates to dead elements
			dlg.Data = chidx
			if chnm == "cancel" {
				dlg.CancelDialog()
			} else {
				dlg.AcceptDialog()
			}
			sc.Send(events.Change, e)
		})
		b.OnKeyChord(func(e events.Event) {
			dlg.Data = chidx
			kf := keyfun.Of(e.KeyChord())
			if chnm == "cancel" {
				if kf == keyfun.Abort {
					e.SetHandled()
					dlg.CancelDialog()
				}
			} else {
				if kf == keyfun.Accept {
					e.SetHandled()
					dlg.AcceptDialog()
				}
			}
		})
	}
	return dlg
}

// NewItems adds to the dialog a prompt for creating new item(s) of the given type,
// showing registered gti types that embed given type.
func (dlg *Dialog) NewItems(typ *gti.Type) *Dialog {
	prIdx := dlg.PromptWidgetIdx()

	sc := dlg.Stage.Scene
	nrow := sc.InsertNewChild(LayoutType, prIdx+1, "n-row").(*Layout)
	nrow.Lay = LayoutHoriz

	NewLabel(nrow, "n-label").SetText("Number:  ")

	nsb := NewSpinner(nrow, "n-field")
	nsb.SetMin(1)
	nsb.Value = 1
	nsb.Step = 1

	tspc := sc.InsertNewChild(SpaceType, prIdx+2, "type-space").(*Space)
	tspc.SetFixedHeight(units.Em(0.5))

	trow := sc.InsertNewChild(LayoutType, prIdx+3, "t-row").(*Layout)
	trow.Lay = LayoutHoriz

	NewLabel(trow, "t-label").SetText("Type:    ")

	typs := NewChooser(trow, "types")
	typs.ItemsFromTypes(gti.AllEmbeddersOf(typ), true, true, 50)

	typs.OnChange(func(e events.Event) {
		dlg.Data = typs.CurVal
	})
	return dlg
}

/*

/////////////////////////////////////////////
//  	Proposed new model

/*
type Dialog struct {
	Scene

	Stage *Stage

	ButtonBox *Layout
}

func Do() {
	dlg := gi.NewDialog().Title("hello").Prompt("Enter your name").
		StringPrompt("", "Enter name..").Ok().Cancel()
	dlg.OnChange(func(e events.Event) { // OnChange is only emitted when accepted
		fmt.Println("Hello,", dlg.Data.(string))
	})
	dlg.Modal(true).Run(ctx)
}

*/
