// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"log"
	"reflect"

	"github.com/iancoleman/strcase"

	"goki.dev/colors"
	"goki.dev/girl/gist"
	"goki.dev/girl/units"
	"goki.dev/goosi"
	"goki.dev/goosi/key"
	"goki.dev/ki/v2"
)

// DialogsSepOSWin determines if dialog windows open in a separate OS-level
// window, or do they open within the same parent window.  If only within
// parent window, then they are always effectively modal.
var DialogsSepOSWin = true

// DialogState indicates the state of the dialog.
type DialogState int64

const (
	// DialogExists is the existential state -- struct exists and is likely
	// being constructed.
	DialogExists DialogState = iota

	// DialogOpenModal means dialog is open in a modal state, blocking all other input.
	DialogOpenModal

	// DialogOpenModeless means dialog is open in a modeless state, allowing other input.
	DialogOpenModeless

	// DialogAccepted means Ok was pressed -- dialog accepted.
	DialogAccepted

	// DialogCanceled means Cancel was pressed -- button canceled.
	DialogCanceled

	DialogStateN
)

// standard vertical space between elements in a dialog, in Ex units
var StdDialogVSpace = float32(1)
var StdDialogVSpaceUnits = units.Ex(StdDialogVSpace)

// Dialog supports dialog functionality -- based on a scene that can either
// be rendered in a separate window or on top of an existing one.
type Dialog struct {
	Scene

	// title text displayed as the window title for the dialog
	Title string `desc:"title text displayed as the window title for the dialog"`

	// a prompt string displayed below the title
	Prompt string `desc:"a prompt string displayed below the title"`

	// open the dialog in a modal state, blocking all other input
	Modal bool `desc:"open the dialog in a modal state, blocking all other input"`

	// default size -- if non-zero, then this is used instead of doing an initial size computation -- can save a lot of time for complex dialogs -- sizes are remembered and used after first use anyway
	DefSize image.Point `desc:"default size -- if non-zero, then this is used instead of doing an initial size computation -- can save a lot of time for complex dialogs -- sizes are remembered and used after first use anyway"`

	// state of the dialog
	State DialogState `desc:"state of the dialog"`

	// signal value that will be sent, if >= 0 (by default, DialogAccepted or DialogCanceled will be sent for standard Ok / Cancel buttons)
	SigVal int64 `desc:"signal value that will be sent, if >= 0 (by default, DialogAccepted or DialogCanceled will be sent for standard Ok / Cancel buttons)"`

	// [view: -] signal for dialog -- sends a signal when opened, accepted, or canceled
	DialogSig ki.Signal `json:"-" xml:"-" view:"-" desc:"signal for dialog -- sends a signal when opened, accepted, or canceled"`

	// [view: -] the main data element represented by this window -- used for Recycle* methods for windows that represent a given data element -- prevents redundant windows
	Data any `json:"-" xml:"-" view:"-" desc:"the main data element represented by this window -- used for Recycle* methods for windows that represent a given data element -- prevents redundant windows"`
}

func (dlg *Dialog) StyleFrame() {
	dlg.Frame.AddStyler(func(w *WidgetBase, s *gist.Style) {
		// material likes SurfaceContainerHigh here, but that seems like too much; STYTODO: maybe figure out a better background color setup for dialogs?
		s.BackgroundColor.SetSolid(ColorScheme.SurfaceContainer)
		s.Color = ColorScheme.OnSurface
		s.Border.Radius = gist.BorderRadiusExtraLarge

		dlg.Frame.Spacing = StdDialogVSpaceUnits
		s.Border.Style.Set(gist.BorderNone)
		s.Padding.Set(units.Px(24 * Prefs.DensityMul()))
		s.BackgroundColor.SetSolid(dlg.Frame.Style.BackgroundColor.Color)
		if !DialogsSepOSWin {
			s.BoxShadow = BoxShadow3
		}

	})
}

// todo: need to do this on frame

func (dlg *Dialog) OnChildAdded(child ki.Ki) {
	if _, wb := AsWidget(child); wb != nil {
		switch wb.Name() {
		case "title":
			title := child.(*Label)
			title.Type = LabelHeadlineSmall
			title.AddStyler(func(w *WidgetBase, s *gist.Style) {
				s.MaxWidth.SetPx(-1)
				s.AlignH = gist.AlignCenter
				s.AlignV = gist.AlignTop
				s.BackgroundColor.SetSolid(colors.Transparent)
			})
		case "prompt":
			prompt := child.(*Label)
			prompt.Type = LabelBodyMedium
			prompt.AddStyler(func(w *WidgetBase, s *gist.Style) {
				s.Text.WhiteSpace = gist.WhiteSpaceNormal
				s.MaxWidth.SetPx(-1)
				s.Width.SetCh(30)
				s.Text.Align = gist.AlignLeft
				s.AlignV = gist.AlignTop
				s.Color = ColorScheme.OnSurfaceVariant
				s.BackgroundColor.SetSolid(colors.Transparent)
			})
		case "buttons":
			bts := child.(*Layout)
			bts.AddStyler(func(w *WidgetBase, s *gist.Style) {
				bts.Spacing.SetPx(8 * Prefs.DensityMul())
				s.SetStretchMaxWidth()
			})
		}
		if button, ok := child.(*Button); ok {
			button.Type = ButtonText
		}
	}
}

// ValidScene finds a non-nil scene, either using the provided one, or
// using the first main window's scene
func ValidScene(avp *Scene) *Scene {
	if avp != nil {
		return avp
	}
	if fwin, _ := AllOSWins.Focused(); fwin != nil {
		return fwin.Scene
	}
	if fwin := AllOSWins.Win(0); fwin != nil {
		return fwin.Scene
	}
	log.Printf("gi.ValidScene: No gi.AllOSWins to get scene from!\n")
	return nil
}

// Open this dialog, in given location (0 = middle of window), finding window
// from given scene -- returns false if it fails for any reason.  optional
// cvgFunc can perform additional configuration after the dialog window has
// been created and dialog added to it -- some configs require the window.
func (dlg *Dialog) Open(x, y int, avp *Scene, cfgFunc func()) bool {
	avp = ValidScene(avp)
	if avp == nil {
		return false
	}
	win := avp.Win
	if win == nil {
		return false
	}

	if dlg.Modal {
		dlg.State = DialogOpenModal
	} else {
		dlg.State = DialogOpenModeless
	}
	dlg.Frame.Lay = LayoutVert

	if DialogsSepOSWin {
		win = NewDialogWin(dlg.Name, dlg.Title, 100, 100, dlg.Modal)
		win.Data = dlg.Data
		// todo: win.Scene
		// win.AddChild(dlg)
		// win.Scene = &dlg.Scene
		// win.MasterVLay = dlg.Frame.Embed(LayoutType).(*Layout)
		// fmt.Printf("new win dpi: %v\n", win.LogicalDPI())
	}

	dlg.Win = win

	if cfgFunc != nil {
		cfgFunc()
	}

	vpsz := dlg.DefSize
	if dlg.DefSize == (image.Point{}) {
		vpsz = dlg.PrefSize(win.OSWin.Screen().PixSize)
		if !DialogsSepOSWin {
			// vpsz = dlg.Frame.LayState.Size.Pref.Min(win.Scene.LayState.Alloc.Size.MulScalar(.9)).ToPoint()
		}
	}
	dlg.Win = nil

	// note: LowPri allows all other events to be processed before dialog
	win.EventMgr.ConnectEvent(dlg.Frame.This(), goosi.KeyChordEvent, LowPri, func(recv, send ki.Ki, sig int64, d any) {
		kt := d.(*key.ChordEvent)
		if KeyEventTrace {
			fmt.Printf("gi.Dialog LowPri KeyInput: %v\n", dlg.Name)
		}
		kf := KeyFun(kt.Chord())
		switch kf {
		case KeyFunAbort:
			dlg.Cancel()
			kt.SetProcessed()
		}
	})
	win.EventMgr.ConnectEvent(dlg.Frame.This(), goosi.KeyChordEvent, LowRawPri, func(recv, send ki.Ki, sig int64, d any) {
		kt := d.(*key.ChordEvent)
		if KeyEventTrace {
			fmt.Printf("gi.Dialog LowPriRaw KeyInput: %v\n", dlg.Name)
		}
		kf := KeyFun(kt.Chord())
		switch kf {
		case KeyFunAccept:
			dlg.Accept()
			kt.SetProcessed()
		}
	})
	// this is not a good idea
	// win.ConnectEvent(dlg.Frame.This(), goosi.MouseEvent, LowRawPri, func(recv, send ki.Ki, sig int64, d any) {
	// 	me := d.(*mouse.Event)
	// 	ddlg, _ := recv.Embed(TypeDialog).(*Dialog)
	// 	if me.Button == mouse.Left && me.Action == mouse.DoubleClick {
	// 		ddlg.Accept()
	// 		me.SetProcessed()
	// 	}
	// })

	if DialogsSepOSWin {
		if !win.HasGeomPrefs() {
			// fmt.Printf("setsz: %v\n", vpsz)
			win.SetSize(vpsz)
		}
		win.GoStartEventLoop()
	} else {
		vpsz.X = 800
		vpsz.Y = 800
		x = max(0, x)
		y = max(0, y)
		if x == 0 && y == 0 {
			x = win.Scene.Geom.Size.X / 3
			y = win.Scene.Geom.Size.Y / 3
		}
		x = min(x, win.Scene.Geom.Size.X-vpsz.X) // fit
		y = min(y, win.Scene.Geom.Size.Y-vpsz.Y) // fit
		dlg.Type = ScDialog                      // ScPopup
		dlg.Resize(vpsz)
		dlg.Geom.Pos = image.Point{x, y}
		win.SetNextPopup(&dlg.Scene, nil)
	}
	return true
}

// Close requests that the dialog be closed -- it does not alter any state or send any signals
func (dlg *Dialog) Close() {
	if dlg == nil || dlg.Frame.This() == nil || dlg.Frame.IsDestroyed() || dlg.Frame.IsDeleted() {
		return
	}
	win := dlg.Win
	if win != nil {
		if DialogsSepOSWin {
			win.Close()
		} else {
			win.ClosePopup(&dlg.Scene)
		}
	}
}

// Accept accepts the dialog, activated by the default Ok button
func (dlg *Dialog) Accept() {
	if dlg == nil {
		return
	}
	dlg.State = DialogAccepted
	if dlg.SigVal >= 0 {
		dlg.DialogSig.Emit(dlg.Frame.This(), dlg.SigVal, nil)
	} else {
		dlg.DialogSig.Emit(dlg.Frame.This(), int64(dlg.State), nil)
	}
	dlg.Close()
}

// Cancel cancels the dialog, activated by the default Cancel button
func (dlg *Dialog) Cancel() {
	if dlg == nil {
		return
	}
	dlg.State = DialogCanceled
	if dlg.SigVal >= 0 {
		dlg.DialogSig.Emit(dlg.Frame.This(), dlg.SigVal, nil)
	} else {
		dlg.DialogSig.Emit(dlg.Frame.This(), int64(dlg.State), nil)
	}
	dlg.Close()
}

////////////////////////////////////////////////////////////////////////////////////////
//  Configuration functions construct standard types of dialogs but anything can be done

// SetTitle sets the title and adds a Label named "title" to the given frame layout if passed
func (dlg *Dialog) SetTitle(title string) *Label {
	dlg.Title = title
	lab := NewLabel(dlg.Frame.This(), "title")
	lab.Text = title
	return lab
}

// Title returns the title label widget, and its index, within frame -- nil, -1 if not found
func (dlg *Dialog) TitleWidget() (*Label, int) {
	idx, ok := dlg.Frame.Children().IndexByName("title", 0)
	if !ok {
		return nil, -1
	}
	return dlg.Frame.Child(idx).(*Label), idx
}

// SetPrompt sets the prompt and adds a Label named "prompt" to the given
// frame layout if passed
func (dlg *Dialog) SetPrompt(prompt string) *Label {
	dlg.Prompt = prompt
	lab := NewLabel(dlg.Frame.This(), "prompt")
	lab.Text = prompt
	return lab
}

// PromptWidget returns the prompt label widget, and its index, within frame -- if
// nil returns the title widget (flexible if prompt is nil)
func (dlg *Dialog) PromptWidget() (*Label, int) {
	idx, ok := dlg.Frame.Children().IndexByName("prompt", 0)
	if !ok {
		return dlg.TitleWidget()
	}
	return dlg.Frame.Child(idx).(*Label), idx
}

// PromptWidgetIdx returns the prompt label widget index only
// for use in Python with only one return value.
func (dlg *Dialog) PromptWidgetIdx() int {
	_, idx := dlg.PromptWidget()
	return idx
}

// AddButtonBox adds a button box (Row Layout) named "buttons" to given frame,
// with an extra space above it
func (dlg *Dialog) AddButtonBox() *Layout {
	NewSpace(&dlg.Frame, "button-space")
	bb := NewLayout(&dlg.Frame, "buttons")
	bb.Lay = LayoutHoriz
	return bb
}

// ButtonBox returns the ButtonBox layout widget, and its index, within frame -- nil, -1 if not found
func (dlg *Dialog) ButtonBox() (*Layout, int) {
	idx, ok := dlg.Frame.Children().IndexByName("buttons", 0)
	if !ok {
		return nil, -1
	}
	return dlg.Frame.Child(idx).(*Layout), idx
}

// Dialog Ok, Cancel options
const (
	AddOk     = true
	NoOk      = false
	AddCancel = true
	NoCancel  = false
)

// StdButtonConfig returns a ki.TypeAndNameList for calling on ConfigChildren
// of a button box, to create standard Ok, Cancel buttons (if true),
// optionally starting with a Stretch element that will cause the buttons to
// be arranged on the right -- a space element is added between buttons if
// more than one
func (dlg *Dialog) StdButtonConfig(stretch, ok, cancel bool) ki.TypeAndNameList {
	config := ki.TypeAndNameList{}
	if stretch {
		config.Add(StretchType, "stretch")
	}
	if cancel {
		config.Add(ButtonType, "cancel")
	}
	if ok {
		config.Add(ButtonType, "ok")
	}
	return config
}

// StdButtonConnect connects standard buttons in given button box layout to
// Accept / Cancel actions
func (dlg *Dialog) StdButtonConnect(ok, cancel bool, bb *Layout) {
	if ok {
		okb := AsButton(bb.ChildByName("ok", 0))
		okb.SetText("Ok")
		okb.ButtonSig.Connect(dlg.Frame.This(), func(recv, send ki.Ki, sig int64, data any) {
			if sig == int64(ButtonClicked) {
				dlg.Accept()
			}
		})
	}
	if cancel {
		canb := AsButton(bb.ChildByName("cancel", 0))
		canb.SetText("Cancel")
		canb.ButtonSig.Connect(dlg.Frame.This(), func(recv, send ki.Ki, sig int64, data any) {
			if sig == int64(ButtonClicked) {
				dlg.Cancel()
			}
		})
	}
}

// StdDialog configures a basic standard dialog with a title, prompt, and ok /
// cancel buttons -- any empty text will not be added
func (dlg *Dialog) StdDialog(title, prompt string, ok, cancel bool) {
	dlg.SigVal = -1
	if title != "" {
		dlg.SetTitle(title)
	}
	if prompt != "" {
		dlg.SetPrompt(prompt)
	}
	if ok || cancel {
		bb := dlg.AddButtonBox()
		bbc := dlg.StdButtonConfig(true, ok, cancel)
		mods, updt := bb.ConfigChildren(bbc)
		dlg.StdButtonConnect(ok, cancel, bb)
		if mods {
			bb.UpdateEnd(updt)
		}
	}
	dlg.SetFlag(true, ScPopupDestroyAll) // std is disposable
}

// DlgOpts are the basic dialog options accepted by all dialog methods --
// provides a named, optional way to specify these args
type DlgOpts struct {

	// generally should be provided -- will also be used for setting name of dialog and associated window
	Title string `desc:"generally should be provided -- will also be used for setting name of dialog and associated window"`

	// optional more detailed description of what is being requested and how it will be used -- is word-wrapped and can contain full html formatting etc.
	Prompt string `desc:"optional more detailed description of what is being requested and how it will be used -- is word-wrapped and can contain full html formatting etc."`

	// optional style properties applied to dialog -- can be used to customize any aspect of existing dialogs
	CSS ki.Props `desc:"optional style properties applied to dialog -- can be used to customize any aspect of existing dialogs"`
}

// NewStdDialog returns a basic standard dialog with given options (title,
// prompt, CSS styling) and whether ok, cancel buttons should be shown -- any
// empty text will not be added.
// Use AddOk / NoOk, AddCancel / NoCancel for bool args.
func NewStdDialog(opts DlgOpts, ok, cancel bool) *Dialog {
	title := opts.Title
	nm := strcase.ToKebab(title)
	if title == "" {
		nm = "unnamed-dialog"
	}
	dlg := Dialog{}
	dlg.Name = nm
	dlg.Frame.CSS = opts.CSS
	dlg.StdDialog(opts.Title, opts.Prompt, ok, cancel)
	return &dlg
}

// RecycleStdDialog looks for existing dialog window with same Data --
// if found brings that to the front, returns it, and true bool.
// else (and if data is nil) calls NewStdDialog, returns false.
func RecycleStdDialog(data any, opts DlgOpts, ok, cancel bool) (*Dialog, bool) {
	if data == nil {
		return NewStdDialog(opts, ok, cancel), false
	}
	ew, has := DialogOSWins.FindData(data)
	if has && ew.Scene.Frame.NumChildren() > 0 {
		ew.OSWin.Raise()
		// dlg := ew.Child(0).Embed(TypeDialog).(*Dialog)
		// return dlg, true
	}
	dlg := NewStdDialog(opts, ok, cancel)
	dlg.Data = data
	return dlg, false
}

//////////////////////////////////////////////////////////////////////////
// Node2D interface

func (dlg *Dialog) ConfigWidget(sc *Scene) {
	dlg.Scene.Config()
}

func (dlg *Dialog) HasFocus() bool {
	return true // dialog ALWAYS gets all the events!
}

//////////////////////////////////////////////////////////////////////////
//     Specific Dialogs

// PromptDialog opens a basic standard dialog with a title, prompt, and ok /
// cancel buttons -- any empty text will not be added -- optionally connects
// to given signal receiving object and function for dialog signals (nil to
// ignore).  Scene is optional to properly contextualize dialog to given
// master window.
func PromptDialog(avp *Scene, opts DlgOpts, ok, cancel bool, recv ki.Ki, fun ki.RecvFunc) {
	dlg := NewStdDialog(opts, ok, cancel)
	dlg.Modal = true
	if recv != nil && fun != nil {
		dlg.DialogSig.Connect(recv, fun)
	}
	dlg.Open(0, 0, avp, nil)
}

// ChoiceDialog presents any number of buttons with labels as given, for the
// user to choose among -- the clicked button number (starting at 0) will be
// sent to the receiving object and function for dialog signals.  Scene is
// optional to properly contextualize dialog to given master window.
func ChoiceDialog(avp *Scene, opts DlgOpts, choices []string, recv ki.Ki, fun ki.RecvFunc) {
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
// Use construct of form: reflect.TypeOf((*gi.Node2D)(nil)).Elem()
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
