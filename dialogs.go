// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"

	"github.com/rcoreilly/goki/gi/units"
	"github.com/rcoreilly/goki/ki"
	"github.com/rcoreilly/goki/ki/bitflag"
	"github.com/rcoreilly/goki/ki/kit"
)

// state of the dialog
type DialogState int64

const (
	// existential state -- struct exists and is likely being constructed
	DialogExists DialogState = iota
	// dialog is open in a modal state, blocking all other input
	DialogOpenModal
	// dialog is open in a modeless state, allowing other input
	DialogOpenModeless
	// Ok was pressed -- dialog accepted
	DialogAccepted
	// Cancel was pressed -- button canceled
	DialogCanceled
	DialogStateN
)

//go:generate stringer -type=DialogState

// Dialog supports dialog functionality -- based on a viewport that can either be rendered in a separate window or on top of an existing one
type Dialog struct {
	Viewport2D
	Title     string      `desc:"title text displayed at the top row of the dialog"`
	Prompt    string      `desc:"a prompt string displayed below the title"`
	Modal     bool        `desc:"open the dialog in a modal state, blocking all other input"`
	State     DialogState `desc:"state of the dialog"`
	DialogSig ki.Signal   `desc:"signal for dialog -- sends a signal when opened, accepted, or canceled"`
}

var KiT_Dialog = kit.Types.AddType(&Dialog{}, nil)

// Open this dialog, in given location (0 = middle of window), finding window from given viewport -- returns false if it fails for any reason
func (dlg *Dialog) Open(x, y int, avp *Viewport2D) bool {
	win := avp.ParentWindow()
	if win == nil {
		return false
	}
	if x == 0 && y == 0 {
		x = win.Viewport.ViewBox.Size.X / 3
		y = win.Viewport.ViewBox.Size.Y / 3
	}

	bitflag.Set(&dlg.NodeFlags, int(VpFlagPopup))
	// todo: deal with modeless -- need a separate window presumably -- not hard
	dlg.State = DialogOpenModal

	dlg.Init2DTree()
	dlg.Style2DTree()                                      // sufficient to get sizes
	dlg.LayData.AllocSize = win.Viewport.LayData.AllocSize // give it the whole vp initially
	dlg.Size2DTree()                                       // collect sizes

	vlay := dlg.ChildByName("VFrame", 0).(*Frame)
	vpsz := vlay.LayData.Size.Pref.Min(win.Viewport.LayData.AllocSize).ToPoint()
	x = kit.MinInt(x, win.Viewport.ViewBox.Size.X-vpsz.X) // fit
	y = kit.MinInt(y, win.Viewport.ViewBox.Size.Y-vpsz.Y) // fit

	dlg.Resize(vpsz.X, vpsz.Y)
	dlg.ViewBox.Min = image.Point{x, y}

	win.PushPopup(dlg.This)
	dlg.FullRender2DTree()
	return true
}

// Close requests that the dialog be closed -- it does not alter any state or send any signals
func (dlg *Dialog) Close() {
	win := dlg.ParentWindow()
	if win != nil {
		win.ClosePopup(dlg.This)
	}
}

// Accept accepts the dialog, activated by the default Ok button
func (dlg *Dialog) Accept() {
	dlg.State = DialogAccepted
	dlg.DialogSig.Emit(dlg.This, int64(dlg.State), nil)
	dlg.Close()
}

// Cancel cancels the dialog, activated by the default Cancel button
func (dlg *Dialog) Cancel() {
	dlg.State = DialogCanceled
	dlg.DialogSig.Emit(dlg.This, int64(dlg.State), nil)
	dlg.Close()
}

////////////////////////////////////////////////////////////////////////////////////////
//  Configuration functions construct standard types of dialogs but anything can be done

// SetVFrame creates a standard vertical column frame layout as first element of the dialog, named "VFrame"
func (dlg *Dialog) SetVFrame() *Frame {
	vlay := dlg.AddNewChildNamed(KiT_Frame, "VFrame").(*Frame)
	vlay.Lay = LayoutCol
	vlay.SetProp("border-width", units.NewValue(2, units.Px))
	vlay.SetProp("margin", units.NewValue(8, units.Px))
	vlay.SetProp("box-shadow.h-offset", units.NewValue(4, units.Px))
	vlay.SetProp("box-shadow.v-offset", units.NewValue(4, units.Px))
	vlay.SetProp("box-shadow.blur", units.NewValue(4, units.Px))
	vlay.SetProp("box-shadow.color", "#CCC")
	return vlay
}

// SetTitle sets the title and adds a Label named "Title" to the given frame layout if passed
func (dlg *Dialog) SetTitle(title string, vlay *Frame) *Label {
	dlg.Title = title
	if vlay != nil {
		lab := vlay.AddNewChildNamed(KiT_Label, "Title").(*Label)
		lab.Text = title
		lab.SetProp("max-width", -1)
		lab.SetProp("text-align", AlignCenter)
		lab.SetProp("vertical-align", AlignTop)
		return lab
	}
	return nil
}

// SetPrompt sets the prompt and adds a Label named "Prompt" to the given frame layout if passed, with the given amount of space before it, sized in "Em"'s (units of font size), if > 0
func (dlg *Dialog) SetPrompt(prompt string, spaceBefore float64, vlay *Frame) *Label {
	dlg.Prompt = prompt
	if vlay != nil {
		if spaceBefore > 0 {
			spc := vlay.AddNewChildNamed(KiT_Space, "PromptSpace").(*Space)
			spc.SetFixedHeight(units.NewValue(spaceBefore, units.Em))
		}
		lab := vlay.AddNewChildNamed(KiT_Label, "Prompt").(*Label)
		lab.Text = prompt
		lab.SetProp("max-width", -1)
		lab.SetProp("text-align", AlignLeft)
		return lab
	}
	return nil
}

// AddButtonBox adds a button box (Row Layout) named "ButtonBox" to given frame, with given amount of space before
func (dlg *Dialog) AddButtonBox(spaceBefore float64, vlay *Frame) *Layout {
	if vlay == nil {
		return nil
	}
	if spaceBefore > 0 {
		spc := vlay.AddNewChildNamed(KiT_Space, "ButtonSpace").(*Space)
		spc.SetFixedHeight(units.NewValue(spaceBefore, units.Em))
	}
	bb := vlay.AddNewChildNamed(KiT_Layout, "ButtonBox").(*Layout)
	bb.Lay = LayoutRow
	return bb
}

// StdButtonConfig returns a kit.TypeAndNameList for calling on ConfigChildren of a button box, to create standard Ok, Cancel buttons (if true), optionally starting with a Stretch element that will cause the buttons to be arranged on the right -- a space element is added between buttons if more than one
func (dlg *Dialog) StdButtonConfig(stretch, ok, cancel bool) kit.TypeAndNameList {
	config := kit.TypeAndNameList{} // note: slice is already a pointer
	if stretch {
		config.Add(KiT_Stretch, "Stretch")
	}
	if ok {
		config.Add(KiT_Button, "Ok")
	}
	if cancel {
		if ok {
			config.Add(KiT_Space, "Space")
		}
		config.Add(KiT_Button, "Cancel")
	}
	return config
}

// StdButtonConnnect connects standard buttons in given button box layout to Accept / Cancel actions
func (dlg *Dialog) StdButtonConnect(ok, cancel bool, bb *Layout) {
	if ok {
		okb := bb.ChildByName("Ok", 0).EmbeddedStruct(KiT_Button).(*Button)
		okb.SetText("Ok")
		okb.ButtonSig.Connect(dlg.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			if sig == int64(ButtonClicked) {
				dlg := recv.EmbeddedStruct(KiT_Dialog).(*Dialog)
				dlg.Accept()
			}
		})
	}
	if cancel {
		canb := bb.ChildByName("Cancel", 0).EmbeddedStruct(KiT_Button).(*Button)
		canb.SetText("Cancel")
		canb.ButtonSig.Connect(dlg.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			if sig == int64(ButtonClicked) {
				dlg := recv.EmbeddedStruct(KiT_Dialog).(*Dialog)
				dlg.Cancel()
			}
		})
	}
}

// StdDialog configures a basic standard dialog with a title, prompt, and ok / cancel buttons -- any empty text will not be added
func (dlg *Dialog) StdDialog(title, prompt string, ok, cancel bool) {
	vlay := dlg.SetVFrame()
	pspc := 0.0
	if title != "" {
		dlg.SetTitle(title, vlay)
		pspc = 2.0
	}
	if prompt != "" {
		dlg.SetPrompt(prompt, pspc, vlay)
	}
	bb := dlg.AddButtonBox(2.0, vlay)
	bbc := dlg.StdButtonConfig(true, ok, cancel)
	bb.ConfigChildren(bbc, false) // not unique names
	dlg.StdButtonConnect(ok, cancel, bb)
}

// NewStdDialog returns a basic standard dialog with a name, title, prompt, and ok / cancel buttons -- any empty text will not be added
func NewStdDialog(name, title, prompt string, ok, cancel bool) *Dialog {
	dlg := Dialog{}
	dlg.InitName(&dlg, name)
	bitflag.Set(&dlg.NodeFlags, int(VpFlagPopup))
	dlg.StdDialog(title, prompt, ok, cancel)
	return &dlg
}

// Prompt opens a basic standard dialog with a title, prompt, and ok / cancel buttons -- any empty text will not be added -- returns true if Ok was pressed, and Cancel otherwise
func PromptDialog(avp *Viewport2D, title, prompt string, ok, cancel bool) bool {
	dlg := NewStdDialog("Prompt", title, prompt, ok, cancel)

	rec := ki.Node{}          // receiver for events
	rec.InitName(&rec, "rec") // this is essential for root objects not owned by other Ki tree nodes

	dlg.DialogSig.Connect(rec.This, func(recv, send ki.Ki, sig int64, data interface{}) {
		fmt.Printf("dialog %v was: %v\n", send.Name(), DialogState(sig))
	})
	dlg.Open(0, 0, avp)
	// todo: now we need a closure for the return value..
	return true
}

////////////////////////////////////////////////////////////////////////////////////////
// Node2D interface

func (dlg *Dialog) AsNode2D() *Node2DBase {
	return &dlg.Node2DBase
}

func (dlg *Dialog) AsViewport2D() *Viewport2D {
	return &dlg.Viewport2D
}

func (dlg *Dialog) AsLayout2D() *Layout {
	return nil
}

func (dlg *Dialog) Init2D() {
	dlg.Viewport2D.Init2D()
	bitflag.Set(&dlg.NodeFlags, int(VpFlagPopup))
}

func (dlg *Dialog) Style2D() {
	dlg.Style2DWidget(nil)
}

func (dlg *Dialog) Size2D() {
	dlg.Viewport2D.Size2D()
}

func (dlg *Dialog) Layout2D(parBBox image.Rectangle) {
	dlg.Viewport2D.Layout2D(parBBox)
}

func (dlg *Dialog) BBox2D() image.Rectangle {
	return dlg.Viewport2D.BBox2D()
}

func (dlg *Dialog) ComputeBBox2D(parBBox image.Rectangle) {
	dlg.Viewport2D.ComputeBBox2D(parBBox)
}

func (dlg *Dialog) ChildrenBBox2D() image.Rectangle {
	return dlg.VpBBox // no margin, padding, etc
}

func (dlg *Dialog) Move2D(delta Vec2D, parBBox image.Rectangle) {
	dlg.Move2DBase(delta, parBBox)
	dlg.Move2DChildren(delta)
}

func (dlg *Dialog) Render2D() {
	dlg.Viewport2D.Render2D()
}

func (dlg *Dialog) ReRender2D() (node Node2D, layout bool) {
	node = dlg.This.(Node2D)
	layout = false
	return
}

func (dlg *Dialog) FocusChanged2D(gotFocus bool) {
}

// check for interface implementation
var _ Node2D = &Dialog{}
