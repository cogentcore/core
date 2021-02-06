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

	"github.com/goki/gi/gist"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/gi/units"
	"github.com/goki/ki/ints"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// DialogsSepWindow determines if dialog windows open in a separate OS-level
// window, or do they open within the same parent window.  If only within
// parent window, then they are always effectively modal.
var DialogsSepWindow = true

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

//go:generate stringer -type=DialogState

// standard vertical space between elements in a dialog, in Ex units
var StdDialogVSpace = float32(1)
var StdDialogVSpaceUnits = units.Value{Val: StdDialogVSpace, Un: units.Ex, Dots: 0}

// Dialog supports dialog functionality -- based on a viewport that can either
// be rendered in a separate window or on top of an existing one.
type Dialog struct {
	Viewport2D
	Title     string      `desc:"title text displayed as the window title for the dialog"`
	Prompt    string      `desc:"a prompt string displayed below the title"`
	Modal     bool        `desc:"open the dialog in a modal state, blocking all other input"`
	DefSize   image.Point `desc:"default size -- if non-zero, then this is used instead of doing an initial size computation -- can save a lot of time for complex dialogs -- sizes are remembered and used after first use anyway"`
	State     DialogState `desc:"state of the dialog"`
	SigVal    int64       `desc:"signal value that will be sent, if >= 0 (by default, DialogAccepted or DialogCanceled will be sent for standard Ok / Cancel buttons)"`
	DialogSig ki.Signal   `json:"-" xml:"-" view:"-" desc:"signal for dialog -- sends a signal when opened, accepted, or canceled"`
	Data      interface{} `json:"-" xml:"-" view:"-" desc:"the main data element represented by this window -- used for Recycle* methods for windows that represent a given data element -- prevents redundant windows"`
}

var KiT_Dialog = kit.Types.AddType(&Dialog{}, DialogProps)

func (dlg *Dialog) Disconnect() {
	dlg.Viewport2D.Disconnect()
	dlg.DialogSig.DisconnectAll()
}

// ValidViewport finds a non-nil viewport, either using the provided one, or
// using the first main window's viewport
func ValidViewport(avp *Viewport2D) *Viewport2D {
	if avp != nil {
		return avp
	}
	if fwin, _ := AllWindows.Focused(); fwin != nil {
		return fwin.Viewport
	}
	if fwin := AllWindows.Win(0); fwin != nil {
		return fwin.Viewport
	}
	log.Printf("gi.ValidViewport: No gi.AllWindows to get viewport from!\n")
	return nil
}

// Open this dialog, in given location (0 = middle of window), finding window
// from given viewport -- returns false if it fails for any reason.  optional
// cvgFunc can perform additional configuration after the dialog window has
// been created and dialog added to it -- some configs require the window.
func (dlg *Dialog) Open(x, y int, avp *Viewport2D, cfgFunc func()) bool {
	avp = ValidViewport(avp)
	if avp == nil {
		return false
	}
	win := avp.Win
	if win == nil {
		return false
	}

	updt := dlg.UpdateStart()
	if dlg.Modal {
		dlg.State = DialogOpenModal
	} else {
		dlg.State = DialogOpenModeless
	}

	if DialogsSepWindow {
		win = NewDialogWin(dlg.Nm, dlg.Title, 100, 100, dlg.Modal)
		win.Data = dlg.Data
		win.AddChild(dlg)
		win.Viewport = &dlg.Viewport2D
		win.MasterVLay = dlg.Frame().Embed(KiT_Layout).(*Layout)
		// fmt.Printf("new win dpi: %v\n", win.LogicalDPI())
	}

	dlg.Win = win

	if cfgFunc != nil {
		cfgFunc()
	}

	vpsz := dlg.DefSize
	if dlg.DefSize == image.ZP {
		vpsz = dlg.PrefSize(win.OSWin.Screen().PixSize)
		if !DialogsSepWindow {
			vpsz = dlg.LayState.Size.Pref.Min(win.Viewport.LayState.Alloc.Size.MulScalar(.9)).ToPoint()
		}
	}
	dlg.Win = nil

	// note: LowPri allows all other events to be processed before dialog
	win.EventMgr.ConnectEvent(dlg.This(), oswin.KeyChordEvent, LowPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		kt := d.(*key.ChordEvent)
		ddlg, _ := recv.Embed(KiT_Dialog).(*Dialog)
		if KeyEventTrace {
			fmt.Printf("gi.Dialog LowPri KeyInput: %v\n", ddlg.Path())
		}
		kf := KeyFun(kt.Chord())
		switch kf {
		case KeyFunAbort:
			ddlg.Cancel()
			kt.SetProcessed()
		}
	})
	win.EventMgr.ConnectEvent(dlg.This(), oswin.KeyChordEvent, LowRawPri, func(recv, send ki.Ki, sig int64, d interface{}) {
		kt := d.(*key.ChordEvent)
		ddlg, _ := recv.Embed(KiT_Dialog).(*Dialog)
		if KeyEventTrace {
			fmt.Printf("gi.Dialog LowPriRaw KeyInput: %v\n", ddlg.Path())
		}
		kf := KeyFun(kt.Chord())
		switch kf {
		case KeyFunAccept:
			ddlg.Accept()
			kt.SetProcessed()
		}
	})
	// this is not a good idea
	// win.ConnectEvent(dlg.This(), oswin.MouseEvent, LowRawPri, func(recv, send ki.Ki, sig int64, d interface{}) {
	// 	me := d.(*mouse.Event)
	// 	ddlg, _ := recv.Embed(KiT_Dialog).(*Dialog)
	// 	if me.Button == mouse.Left && me.Action == mouse.DoubleClick {
	// 		ddlg.Accept()
	// 		me.SetProcessed()
	// 	}
	// })

	if DialogsSepWindow {
		dlg.UpdateEndNoSig(updt)
		if !win.HasGeomPrefs() {
			// fmt.Printf("setsz: %v\n", vpsz)
			win.SetPixSize(vpsz)
		}
		win.GoStartEventLoop()
	} else {
		x = ints.MaxInt(0, x)
		y = ints.MaxInt(0, y)
		if x == 0 && y == 0 {
			x = win.Viewport.Geom.Size.X / 3
			y = win.Viewport.Geom.Size.Y / 3
		}
		x = ints.MinInt(x, win.Viewport.Geom.Size.X-vpsz.X) // fit
		y = ints.MinInt(y, win.Viewport.Geom.Size.Y-vpsz.Y) // fit
		frame := dlg.Child(0).(*Frame)
		dlg.StylePart(Node2D(frame)) // use special styles
		dlg.SetFlag(int(VpFlagPopup))
		dlg.Resize(vpsz)
		dlg.Geom.Pos = image.Point{x, y}
		dlg.UpdateEndNoSig(updt)
		win.SetNextPopup(dlg.This(), nil)
	}
	return true
}

// Close requests that the dialog be closed -- it does not alter any state or send any signals
func (dlg *Dialog) Close() {
	if dlg == nil || dlg.This() == nil || dlg.IsDestroyed() || dlg.IsDeleted() {
		return
	}
	win := dlg.Win
	if win != nil {
		if DialogsSepWindow {
			win.Close()
		} else {
			win.ClosePopup(dlg.This())
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
		dlg.DialogSig.Emit(dlg.This(), dlg.SigVal, nil)
	} else {
		dlg.DialogSig.Emit(dlg.This(), int64(dlg.State), nil)
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
		dlg.DialogSig.Emit(dlg.This(), dlg.SigVal, nil)
	} else {
		dlg.DialogSig.Emit(dlg.This(), int64(dlg.State), nil)
	}
	dlg.Close()
}

////////////////////////////////////////////////////////////////////////////////////////
//  Configuration functions construct standard types of dialogs but anything can be done

var DialogProps = ki.Props{
	"EnumType:Flag": KiT_VpFlags,
	"color":         &Prefs.Colors.Font,
	"#frame": ki.Props{
		"border-width":        units.NewPx(2),
		"margin":              units.NewPx(8),
		"padding":             units.NewPx(4),
		"box-shadow.h-offset": units.NewPx(4),
		"box-shadow.v-offset": units.NewPx(4),
		"box-shadow.blur":     units.NewPx(4),
		"box-shadow.color":    &Prefs.Colors.Shadow,
	},
	"#title": ki.Props{
		// todo: add "bigger" font
		"max-width":        units.NewPx(-1),
		"horizontal-align": gist.AlignCenter,
		"vertical-align":   gist.AlignTop,
		"background-color": "none",
		"font-size":        "large",
	},
	"#prompt": ki.Props{
		"white-space":      gist.WhiteSpaceNormal, // wrap etc
		"max-width":        -1,
		"width":            units.NewCh(30),
		"text-align":       gist.AlignLeft,
		"vertical-align":   gist.AlignTop,
		"background-color": "none",
	},
}

// SetFrame creates a standard vertical column frame layout as first element of the dialog, named "frame"
func (dlg *Dialog) SetFrame() *Frame {
	dlg.SetProp("color", &Prefs.Colors.Font)
	frame := AddNewFrame(dlg, "frame", LayoutVert)
	frame.SetProp("spacing", StdDialogVSpaceUnits)
	return frame
}

// Frame returns the main frame for the dialog, assumed to be the first element in the dialog
func (dlg *Dialog) Frame() *Frame {
	return dlg.Child(0).(*Frame)
}

// SetTitle sets the title and adds a Label named "title" to the given frame layout if passed
func (dlg *Dialog) SetTitle(title string, frame *Frame) *Label {
	dlg.Title = title
	if frame != nil {
		lab := AddNewLabel(frame, "title", title)
		dlg.StylePart(Node2D(lab))
		return lab
	}
	return nil
}

// Title returns the title label widget, and its index, within frame -- nil, -1 if not found
func (dlg *Dialog) TitleWidget(frame *Frame) (*Label, int) {
	idx, ok := frame.Children().IndexByName("title", 0)
	if !ok {
		return nil, -1
	}
	return frame.Child(idx).(*Label), idx
}

// SetPrompt sets the prompt and adds a Label named "prompt" to the given
// frame layout if passed
func (dlg *Dialog) SetPrompt(prompt string, frame *Frame) *Label {
	dlg.Prompt = prompt
	if frame != nil {
		lab := AddNewLabel(frame, "prompt", prompt)
		dlg.StylePart(Node2D(lab))
		return lab
	}
	return nil
}

// PromptWidget returns the prompt label widget, and its index, within frame -- if
// nil returns the title widget (flexible if prompt is nil)
func (dlg *Dialog) PromptWidget(frame *Frame) (*Label, int) {
	idx, ok := frame.Children().IndexByName("prompt", 0)
	if !ok {
		return dlg.TitleWidget(frame)
	}
	return frame.Child(idx).(*Label), idx
}

// PromptWidgetIdx returns the prompt label widget index only
// for use in Python with only one return value.
func (dlg *Dialog) PromptWidgetIdx(frame *Frame) int {
	_, idx := dlg.PromptWidget(frame)
	return idx
}

// AddButtonBox adds a button box (Row Layout) named "buttons" to given frame,
// with an extra space above it
func (dlg *Dialog) AddButtonBox(frame *Frame) *Layout {
	if frame == nil {
		return nil
	}
	AddNewSpace(frame, "button-space")
	bb := AddNewLayout(frame, "buttons", LayoutHoriz)
	bb.SetProp("max-width", -1)
	return bb
}

// ButtonBox returns the ButtonBox layout widget, and its index, within frame -- nil, -1 if not found
func (dlg *Dialog) ButtonBox(frame *Frame) (*Layout, int) {
	idx, ok := frame.Children().IndexByName("buttons", 0)
	if !ok {
		return nil, -1
	}
	return frame.Child(idx).(*Layout), idx
}

// Dialog Ok, Cancel options
const (
	AddOk     = true
	NoOk      = false
	AddCancel = true
	NoCancel  = false
)

// StdButtonConfig returns a kit.TypeAndNameList for calling on ConfigChildren
// of a button box, to create standard Ok, Cancel buttons (if true),
// optionally starting with a Stretch element that will cause the buttons to
// be arranged on the right -- a space element is added between buttons if
// more than one
func (dlg *Dialog) StdButtonConfig(stretch, ok, cancel bool) kit.TypeAndNameList {
	config := kit.TypeAndNameList{}
	if stretch {
		config.Add(KiT_Stretch, "stretch")
	}
	if ok {
		config.Add(KiT_Button, "ok")
	}
	if cancel {
		if ok {
			config.Add(KiT_Space, "space")
		}
		config.Add(KiT_Button, "cancel")
	}
	return config
}

// StdButtonConnect connects standard buttons in given button box layout to
// Accept / Cancel actions
func (dlg *Dialog) StdButtonConnect(ok, cancel bool, bb *Layout) {
	if ok {
		okb := bb.ChildByName("ok", 0).Embed(KiT_Button).(*Button)
		okb.SetText("Ok")
		okb.ButtonSig.Connect(dlg.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			if sig == int64(ButtonClicked) {
				dlg := recv.Embed(KiT_Dialog).(*Dialog)
				dlg.Accept()
			}
		})
	}
	if cancel {
		canb := bb.ChildByName("cancel", 0).Embed(KiT_Button).(*Button)
		canb.SetText("Cancel")
		canb.ButtonSig.Connect(dlg.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			if sig == int64(ButtonClicked) {
				dlg := recv.Embed(KiT_Dialog).(*Dialog)
				dlg.Cancel()
			}
		})
	}
}

// StdDialog configures a basic standard dialog with a title, prompt, and ok /
// cancel buttons -- any empty text will not be added
func (dlg *Dialog) StdDialog(title, prompt string, ok, cancel bool) {
	dlg.SigVal = -1
	frame := dlg.SetFrame()
	if title != "" {
		dlg.SetTitle(title, nil) // frame) // don't set title element
	}
	if prompt != "" {
		dlg.SetPrompt(prompt, frame)
	}
	if ok || cancel {
		bb := dlg.AddButtonBox(frame)
		bbc := dlg.StdButtonConfig(false, ok, cancel) // no stretch -- left better
		mods, updt := bb.ConfigChildren(bbc)
		dlg.StdButtonConnect(ok, cancel, bb)
		if mods {
			bb.UpdateEnd(updt)
		}
	}
	dlg.SetFlag(int(VpFlagPopupDestroyAll)) // std is disposable
}

// DlgOpts are the basic dialog options accepted by all dialog methods --
// provides a named, optional way to specify these args
type DlgOpts struct {
	Title  string   `desc:"generally should be provided -- will also be used for setting name of dialog and associated window"`
	Prompt string   `desc:"optional more detailed description of what is being requested and how it will be used -- is word-wrapped and can contain full html formatting etc."`
	CSS    ki.Props `desc:"optional style properties applied to dialog -- can be used to customize any aspect of existing dialogs"`
}

// NewStdDialog returns a basic standard dialog with given options (title,
// prompt, CSS styling) and whether ok, cancel buttons should be shown -- any
// empty text will not be added -- returns with UpdateStart started but NOT
// ended -- must call UpdateEnd(true) once done configuring!
// Use AddOk / NoOk, AddCancel / NoCancel for bool args.
func NewStdDialog(opts DlgOpts, ok, cancel bool) *Dialog {
	title := opts.Title
	nm := strcase.ToKebab(title)
	if title == "" {
		nm = "unnamed-dialog"
	}
	dlg := Dialog{}
	dlg.InitName(&dlg, nm)
	dlg.UpdateStart() // guaranteed to be true
	dlg.CSS = opts.CSS
	dlg.StdDialog(opts.Title, opts.Prompt, ok, cancel)
	return &dlg
}

// RecycleStdDialog looks for existing dialog window with same Data --
// if found brings that to the front, returns it, and true bool.
// else (and if data is nil) calls NewStdDialog, returns false.
func RecycleStdDialog(data interface{}, opts DlgOpts, ok, cancel bool) (*Dialog, bool) {
	if data == nil {
		return NewStdDialog(opts, ok, cancel), false
	}
	ew, has := DialogWindows.FindData(data)
	if has && ew.NumChildren() > 0 {
		ew.OSWin.Raise()
		dlg := ew.Child(0).Embed(KiT_Dialog).(*Dialog)
		return dlg, true
	}
	dlg := NewStdDialog(opts, ok, cancel)
	dlg.Data = data
	return dlg, false
}

//////////////////////////////////////////////////////////////////////////
// Node2D interface

func (dlg *Dialog) Init2D() {
	dlg.Viewport2D.Init2D()
}

func (dlg *Dialog) HasFocus2D() bool {
	return true // dialog ALWAYS gets all the events!
}

//////////////////////////////////////////////////////////////////////////
//     Specific Dialogs

// PromptDialog opens a basic standard dialog with a title, prompt, and ok /
// cancel buttons -- any empty text will not be added -- optionally connects
// to given signal receiving object and function for dialog signals (nil to
// ignore).  Viewport is optional to properly contextualize dialog to given
// master window.
func PromptDialog(avp *Viewport2D, opts DlgOpts, ok, cancel bool, recv ki.Ki, fun ki.RecvFunc) {
	dlg := NewStdDialog(opts, ok, cancel)
	dlg.Modal = true
	if recv != nil && fun != nil {
		dlg.DialogSig.Connect(recv, fun)
	}
	dlg.UpdateEndNoSig(true) // going to be shown
	dlg.Open(0, 0, avp, nil)
}

// ChoiceDialog presents any number of buttons with labels as given, for the
// user to choose among -- the clicked button number (starting at 0) will be
// sent to the receiving object and function for dialog signals.  Viewport is
// optional to properly contextualize dialog to given master window.
func ChoiceDialog(avp *Viewport2D, opts DlgOpts, choices []string, recv ki.Ki, fun ki.RecvFunc) {
	dlg := NewStdDialog(opts, NoOk, NoCancel) // no buttons
	dlg.Modal = true
	if recv != nil && fun != nil {
		dlg.DialogSig.Connect(recv, fun)
	}

	frame := dlg.Frame()
	bb := dlg.AddButtonBox(frame) // not otherwise made because no buttons above
	for i, ch := range choices {
		chnm := strcase.ToKebab(ch)
		b := AddNewButton(bb, chnm)
		b.SetProp("__cdSigVal", int64(i))
		b.SetText(ch)
		if chnm == "cancel" {
			b.ButtonSig.Connect(dlg.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
				if sig == int64(ButtonClicked) {
					tb := send.Embed(KiT_Button).(*Button)
					dlg := recv.Embed(KiT_Dialog).(*Dialog)
					dlg.SigVal = tb.Prop("__cdSigVal").(int64)
					dlg.Cancel()
				}
			})
		} else {
			b.ButtonSig.Connect(dlg.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
				if sig == int64(ButtonClicked) {
					tb := send.Embed(KiT_Button).(*Button)
					dlg := recv.Embed(KiT_Dialog).(*Dialog)
					dlg.SigVal = tb.Prop("__cdSigVal").(int64)
					dlg.Accept()
				}
			})
		}
	}

	dlg.UpdateEndNoSig(true) // going to be shown
	dlg.Open(0, 0, avp, nil)
}

// NewKiDialog prompts for creating new item(s) of a given type, showing types
// that implement given interface.
// Use construct of form: reflect.TypeOf((*gi.Node2D)(nil)).Elem()
// Optionally connects to given signal receiving object and function for
// dialog signals (nil to ignore).
func NewKiDialog(avp *Viewport2D, iface reflect.Type, opts DlgOpts, recv ki.Ki, fun ki.RecvFunc) *Dialog {
	dlg := NewStdDialog(opts, AddOk, AddCancel)
	dlg.Modal = true

	frame := dlg.Frame()
	_, prIdx := dlg.PromptWidget(frame)

	nrow := frame.InsertNewChild(KiT_Layout, prIdx+2, "n-row").(*Layout)
	nrow.Lay = LayoutHoriz

	AddNewLabel(nrow, "n-label", "Number:  ")

	nsb := AddNewSpinBox(nrow, "n-field")
	nsb.Defaults()
	nsb.SetMin(1)
	nsb.Value = 1
	nsb.Step = 1

	tspc := frame.InsertNewChild(KiT_Space, prIdx+3, "type-space").(*Space)
	tspc.SetFixedHeight(units.NewEm(0.5))

	trow := frame.InsertNewChild(KiT_Layout, prIdx+4, "t-row").(*Layout)
	trow.Lay = LayoutHoriz

	AddNewLabel(trow, "t-label", "Type:    ")

	typs := AddNewComboBox(trow, "types")
	typs.ItemsFromTypes(kit.Types.AllImplementersOf(iface, false), true, true, 50)

	if recv != nil && fun != nil {
		dlg.DialogSig.Connect(recv, fun)
	}
	dlg.UpdateEndNoSig(true)
	dlg.Open(0, 0, avp, nil)
	return dlg
}

// NewKiDialogValues gets the user-set values from a NewKiDialog.
func NewKiDialogValues(dlg *Dialog) (int, reflect.Type) {
	frame := dlg.Frame()
	nrow := frame.ChildByName("n-row", 0).(*Layout)
	ntf := nrow.ChildByName("n-field", 0).(*SpinBox)
	n := int(ntf.Value)
	trow := frame.ChildByName("t-row", 0).(*Layout)
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
// (nil to ignore).  Viewport is optional to properly contextualize dialog to
// given master window.
func StringPromptDialog(avp *Viewport2D, strval, placeholder string, opts DlgOpts, recv ki.Ki, fun ki.RecvFunc) *Dialog {
	dlg := NewStdDialog(opts, AddOk, AddCancel)
	dlg.Modal = true

	frame := dlg.Frame()
	_, prIdx := dlg.PromptWidget(frame)
	tf := frame.InsertNewChild(KiT_TextField, prIdx+1, "str-field").(*TextField)
	tf.Placeholder = placeholder
	tf.SetText(strval)
	tf.SetStretchMaxWidth()
	tf.SetMinPrefWidth(units.NewCh(40))

	if recv != nil && fun != nil {
		dlg.DialogSig.Connect(recv, fun)
	}
	dlg.UpdateEndNoSig(true)
	dlg.Open(0, 0, avp, nil)
	return dlg
}

// StringPromptDialogValue gets the string value the user set.
func StringPromptDialogValue(dlg *Dialog) string {
	frame := dlg.Frame()
	tf := frame.ChildByName("str-field", 0).(*TextField)
	return tf.Text()
}
