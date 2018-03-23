// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	// "fmt"
	"github.com/rcoreilly/goki/gi/units"
	"github.com/rcoreilly/goki/ki"
	"image"
	"math"
	// "reflect"
)

// Widget base type
type WidgetBase struct {
	Node2DBase
	Controls Layout `desc:"a separate tree of sub-widgets that implement discrete subcomponents of a widget -- positions are always relative to the parent widget"`
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KiT_WidgetBase = ki.Types.AddType(&WidgetBase{}, nil)

// Styling notes:
// simple elemental widgets (buttons etc) have a DefaultRender method that renders based on
// Style, with full css styling support -- code has built-in initial defaults for a default
// style based on fusion style parameters on QML Qt Quick Controls

// Alternatively they support custom svg code for rendering each state as appropriate in a Stack
// more complex widgets such as a TreeView automatically render and don't support custom svg

// WidgetBase supports full Box rendering model, so Button just calls these methods to render
// -- base function needs to take a Style arg.

func (g *WidgetBase) DrawBoxImpl(pos Vec2D, sz Vec2D, rad float64) {
	pc := &g.Paint
	rs := &g.Viewport.Render
	if rad == 0.0 {
		pc.DrawRectangle(rs, pos.X, pos.Y, sz.X, sz.Y)
	} else {
		pc.DrawRoundedRectangle(rs, pos.X, pos.Y, sz.X, sz.Y, rad)
	}
	pc.FillStrokeClear(rs)
}

// draw standard box using current style
func (g *WidgetBase) DrawStdBox() {
	pc := &g.Paint
	// rs := &g.Viewport.Render
	st := &g.Style

	pos := g.LayData.AllocPos.AddVal(st.Layout.Margin.Dots)
	sz := g.LayData.AllocSize.AddVal(-2.0 * st.Layout.Margin.Dots)

	// first do any shadow
	if st.BoxShadow.HasShadow() {
		spos := pos.Add(Vec2D{st.BoxShadow.HOffset.Dots, st.BoxShadow.VOffset.Dots})
		pc.StrokeStyle.SetColor(nil)
		pc.FillStyle.SetColor(&st.BoxShadow.Color)
		g.DrawBoxImpl(spos, sz, st.Border.Radius.Dots)
	}
	// then draw the box over top of that -- note: won't work well for transparent! need to set clipping to box first..
	pc.StrokeStyle.SetColor(&st.Border.Color)
	pc.StrokeStyle.Width = st.Border.Width
	pc.FillStyle.SetColor(&st.Background.Color)
	g.DrawBoxImpl(pos, sz, st.Border.Radius.Dots)
}

////////////////////////////////////////////////////////////////////////////////////////
// Label

// Label is a widget for rendering text labels -- supports full widget model
// including box rendering
type Label struct {
	WidgetBase
	Text string `xml:"text",desc:"label to display"`
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KiT_Label = ki.Types.AddType(&Label{}, nil)

func (g *Label) AsNode2D() *Node2DBase {
	return &g.Node2DBase
}

func (g *Label) AsViewport2D() *Viewport2D {
	return nil
}

func (g *Label) AsLayout2D() *Layout {
	return nil
}

func (g *Label) InitNode2D() {

}

var LabelProps = map[string]interface{}{
	"padding":    "2px",
	"margin":     "2px",
	"font-size":  "24pt",
	"text-align": "left",
	"color":      "black",
}

func (g *Label) Style2D() {
	// first do our normal default styles
	g.Style.SetStyle(nil, &StyleDefault, LabelProps)
	// then style with user props
	g.Style2DWidget()
}

func (g *Label) Layout2D(iter int) {
	if iter == 0 {
		g.InitLayout2D()
		st := &g.Style
		pc := &g.Paint
		var w, h float64
		w, h = pc.MeasureString(g.Text)
		if st.Layout.Width.Dots > 0 {
			w = math.Max(st.Layout.Width.Dots, w)
		}
		if st.Layout.Height.Dots > 0 {
			h = math.Max(st.Layout.Height.Dots, h)
		}
		w += 2.0*st.Padding.Dots + 2.0*st.Layout.Margin.Dots
		h += 2.0*st.Padding.Dots + 2.0*st.Layout.Margin.Dots
		g.LayData.AllocSize = Vec2D{w, h}
	} else {
		g.GeomFromLayout() // get our geom from layout -- always do this for widgets  iter > 0
	}
	g.Style.SetUnitContext(&g.Viewport.Render, 0)
}

func (g *Label) Node2DBBox() image.Rectangle {
	return g.WinBBoxFromAlloc()
}

func (g *Label) Render2D() {
	pc := &g.Paint
	rs := &g.Viewport.Render
	st := &g.Style
	pc.FontStyle = st.Font
	pc.TextStyle = st.Text
	g.DrawStdBox()
	pc.StrokeStyle.SetColor(&st.Color) // ink color

	pos := g.LayData.AllocPos.AddVal(st.Layout.Margin.Dots + st.Padding.Dots)
	// sz := g.LayData.AllocSize.AddVal(-2.0 * (st.Layout.Margin.Dots + st.Padding.Dots))

	pc.DrawStringAnchored(rs, g.Text, pos.X, pos.Y, 0.0, 0.9)
}

func (g *Label) CanReRender2D() bool {
	return true
}

func (g *Label) FocusChanged2D(gotFocus bool) {
}

// check for interface implementation
var _ Node2D = &Label{}

////////////////////////////////////////////////////////////////////////////////////////
// LineEdit

// signals that buttons can send
type LineEditSignals int64

const (
	// main signal -- return was pressed and an edit was completed -- data is the text
	LineEditDone LineEditSignals = iota
	LineEditSignalsN
)

//go:generate stringer -type=LineEditSignals

// LineEdit is a widget for editing a line of text
type LineEdit struct {
	WidgetBase
	Text        string    `xml:"text",desc:"the last saved value of the text string being edited"`
	EditText    string    `xml:"-",desc:"the live text string being edited, with latest modifications"`
	StartPos    int       `xml:"start-pos",desc:"starting display position in the string"`
	EndPos      int       `xml:"end-pos",desc:"ending display position in the string"`
	CursorPos   int       `xml:"cursor-pos",desc:"current cursor position"`
	CharWidth   int       `xml:"char-width",desc:"approximate number of chars that can be displayed at any time -- computed from font size etc"`
	SelectMode  bool      `xml:"select-mode",desc:"if true, select text as cursor moves"`
	LineEditSig ki.Signal `json:"-",desc:"signal for line edit -- see LineEditSignals for the types"`
	StateStyles [2]Style  `desc:"normal style and focus style"`
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KiT_LineEdit = ki.Types.AddType(&LineEdit{}, nil)

func (g *LineEdit) EditDone() {
	g.Text = g.EditText
	g.LineEditSig.Emit(g.This, int64(LineEditDone), g.Text)
}

func (g *LineEdit) CursorForward(steps int) {
	g.UpdateStart()
	g.CursorPos += steps
	if g.CursorPos > len(g.EditText) {
		g.CursorPos = len(g.EditText)
	}
	if g.CursorPos > g.EndPos {
		inc := g.CursorPos - g.EndPos
		g.EndPos += inc
	}
	g.UpdateEnd()
}

func (g *LineEdit) CursorBackward(steps int) {
	g.UpdateStart()
	// todo: select mode
	g.CursorPos -= steps
	if g.CursorPos < 0 {
		g.CursorPos = 0
	}
	if g.CursorPos <= g.StartPos {
		dec := ki.MinInt(g.StartPos, 8)
		g.StartPos -= dec
	}
	g.UpdateEnd()
}

func (g *LineEdit) CursorStart() {
	g.UpdateStart()
	// todo: select mode
	g.CursorPos = 0
	g.StartPos = 0
	g.EndPos = ki.MinInt(len(g.EditText), g.StartPos+g.CharWidth)
	g.UpdateEnd()
}

func (g *LineEdit) CursorEnd() {
	g.UpdateStart()
	g.CursorPos = len(g.EditText)
	g.EndPos = len(g.EditText) // try -- display will adjust
	g.StartPos = ki.MaxInt(0, g.EndPos-g.CharWidth)
	g.UpdateEnd()
}

func (g *LineEdit) CursorBackspace(steps int) {
	if g.CursorPos < steps {
		steps = g.CursorPos
	}
	if steps <= 0 {
		return
	}
	g.UpdateStart()
	g.EditText = g.EditText[:g.CursorPos-steps] + g.EditText[g.CursorPos:]
	g.CursorBackward(steps)
	g.UpdateEnd()
}

func (g *LineEdit) CursorDelete(steps int) {
	if g.CursorPos+steps > len(g.EditText) {
		steps = len(g.EditText) - g.CursorPos
	}
	if steps <= 0 {
		return
	}
	g.UpdateStart()
	g.EditText = g.EditText[:g.CursorPos] + g.EditText[g.CursorPos+steps:]
	g.UpdateEnd()
}

func (g *LineEdit) InsertAtCursor(str string) {
	g.UpdateStart()
	g.EditText = g.EditText[:g.CursorPos] + str + g.EditText[g.CursorPos:]
	g.CursorForward(len(str))
	g.UpdateEnd()
}

func (g *LineEdit) AsNode2D() *Node2DBase {
	return &g.Node2DBase
}

func (g *LineEdit) AsViewport2D() *Viewport2D {
	return nil
}

func (g *LineEdit) AsLayout2D() *Layout {
	return nil
}

func (g *LineEdit) InitNode2D() {
	g.EditText = g.Text
	g.ReceiveEventType(KeyTypedEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		ab, ok := recv.(*LineEdit)
		if !ok {
			return
		}
		kt, ok := d.(KeyTypedEvent)
		if ok {
			// todo: shift key not causing caps!
			kf := KeyFun(kt.Key, kt.Chord)
			switch kf {
			case KeyFunSelectItem:
				ab.EditDone()
			case KeyFunMoveRight:
				ab.CursorForward(1)
			case KeyFunMoveLeft:
				ab.CursorBackward(1)
			case KeyFunHome:
				ab.CursorStart()
			case KeyFunEnd:
				ab.CursorEnd()
			case KeyFunBackspace:
				ab.CursorBackspace(1)
			case KeyFunDelete:
				ab.CursorDelete(1)
			case KeyFunNil:
				switch kt.Key {
				case "space":
					ab.InsertAtCursor(" ")
				default:
					if len(kt.Key) < 3 { // avoid long names = control keys
						ab.InsertAtCursor(kt.Key)
					}
				}
			}
		}
	})
}

var LineEditProps = [2]map[string]interface{}{
	{ // normal
		"border-width":     "1px",
		"border-color":     "black",
		"border-style":     "solid",
		"padding":          "2px",
		"margin":           "2px",
		"font-size":        "24pt",
		"text-align":       "left",
		"color":            "black",
		"background-color": "#EEE",
	}, { // focus
		"background-color": "#FFF",
	},
}

func (g *LineEdit) Style2D() {
	ki.SetBitFlag(&g.NodeFlags, int(CanFocus))
	// first do our normal default styles
	g.Style.SetStyle(nil, &StyleDefault, LineEditProps[0])
	// then style with user props
	g.Style2DWidget()
	g.StateStyles[0] = g.Style
	g.StateStyles[1] = g.Style
	g.StateStyles[1].SetStyle(nil, &StyleDefault, LineEditProps[1])
}

func (g *LineEdit) Layout2D(iter int) {
	if iter == 0 {
		g.EditText = g.Text
		g.EndPos = len(g.EditText)
		g.InitLayout2D()
		st := &g.Style
		pc := &g.Paint
		var w, h float64
		w, h = pc.MeasureString(g.Text)
		if st.Layout.Width.Dots > 0 {
			w = math.Max(st.Layout.Width.Dots, w)
		}
		if st.Layout.Height.Dots > 0 {
			h = math.Max(st.Layout.Height.Dots, h)
		}
		w += 2.0*st.Padding.Dots + 2.0*st.Layout.Margin.Dots
		h += 2.0*st.Padding.Dots + 2.0*st.Layout.Margin.Dots
		g.LayData.AllocSize = Vec2D{w, h}
	} else {
		g.GeomFromLayout() // get our geom from layout -- always do this for widgets  iter > 0
	}
	g.Style.SetUnitContext(&g.Viewport.Render, 0)
}

func (g *LineEdit) Node2DBBox() image.Rectangle {
	return g.WinBBoxFromAlloc()
}

func (g *LineEdit) RenderCursor() {
	pc := &g.Paint
	rs := &g.Viewport.Render
	st := &g.Style
	pc.FontStyle = st.Font
	pc.TextStyle = st.Text
	tocur := g.EditText[g.StartPos:g.CursorPos]
	w, h := pc.MeasureString(tocur)

	pos := g.LayData.AllocPos.AddVal(st.Layout.Margin.Dots + st.Padding.Dots)

	pc.DrawLine(rs, pos.X+w, pos.Y, pos.X+w, pos.Y+h)
	pc.Stroke(rs)
}

// scroll the starting position to keep the cursor visible
func (g *LineEdit) AutoScroll() {
	pc := &g.Paint
	st := &g.Style

	spc := (st.Layout.Margin.Dots + st.Padding.Dots)

	maxw := g.LayData.AllocSize.X - spc
	g.CharWidth = int(maxw / st.UnContext.ToDotsFactor(units.Ch))

	if g.EndPos == 0 { // not init
		g.EndPos = len(g.EditText)
	}

	tocur := g.EditText[g.StartPos:g.CursorPos]
	w, _ := pc.MeasureString(tocur)

	// scroll to keep cursor in view
	if w >= maxw {
		inc := 8 // todo: scroll amount
		g.StartPos += inc
		g.EndPos += inc
	}
	// keep sane
	g.EndPos = ki.MinInt(len(g.EditText), g.EndPos)
	if g.StartPos > g.EndPos {
		g.StartPos = ki.MaxInt(0, g.EndPos-g.CharWidth)
	}

	// now make sure text fits -- iteratively for 10 tries..
	for i := 0; i < 10; i++ {
		cur := g.EditText[g.StartPos:g.EndPos]
		w, _ = pc.MeasureString(cur)

		// scroll endpos to keep cursor in view
		if w >= maxw {
			if g.EndPos > g.CursorPos {
				g.EndPos--
			} else {
				g.StartPos++
			}
		} else {
			break
		}
	}
	// keep sane
	g.EndPos = ki.MinInt(len(g.EditText), g.EndPos)
	if g.StartPos > g.EndPos {
		g.StartPos = ki.MaxInt(0, g.EndPos-g.CharWidth)
	}
}

func (g *LineEdit) Render2D() {
	pc := &g.Paint
	rs := &g.Viewport.Render
	if g.HasFocus() {
		g.Style = g.StateStyles[1]
	} else {
		g.Style = g.StateStyles[0]
	}
	st := &g.Style
	pc.FontStyle = st.Font
	pc.TextStyle = st.Text
	g.DrawStdBox()
	pc.StrokeStyle.SetColor(&st.Color) // ink color

	// keep everything in range
	g.AutoScroll()

	pos := g.LayData.AllocPos.AddVal(st.Layout.Margin.Dots + st.Padding.Dots)
	// sz := g.LayData.AllocSize.AddVal(-2.0 * (st.Layout.Margin.Dots + st.Padding.Dots))

	cur := g.EditText[g.StartPos:g.EndPos]

	pc.DrawStringAnchored(rs, cur, pos.X, pos.Y, 0.0, 0.9)
	if g.HasFocus() {
		g.RenderCursor()
	}
}

func (g *LineEdit) CanReRender2D() bool {
	return true
}

func (g *LineEdit) FocusChanged2D(gotFocus bool) {
	g.UpdateStart()
	if !gotFocus {
		g.EditDone() // lose focus
	}
	g.UpdateEnd()
}

// check for interface implementation
var _ Node2D = &LineEdit{}

////////////////////////////////////////////////////////////////////////////////////////
// Buttons

// these extend NodeBase NodeFlags to hold button state
const (
	// button is selected
	ButtonFlagSelected NodeFlags = NodeFlagsN + iota
	// button is checkable -- enables display of check control
	ButtonFlagCheckable
	// button is checked
	ButtonFlagChecked
)

// signals that buttons can send
type ButtonSignals int64

const (
	// main signal -- button pressed down and up
	ButtonClicked ButtonSignals = iota
	// button pushed down but not yet up
	ButtonPressed
	ButtonReleased
	// toggled is for checked / unchecked state
	ButtonToggled
	ButtonSignalsN
)

//go:generate stringer -type=ButtonSignals

// https://ux.stackexchange.com/questions/84872/what-is-the-buttons-unpressed-and-unhovered-state-called

// mutually-exclusive button states -- determines appearance
type ButtonStates int32

const (
	// normal state -- there but not being interacted with
	ButtonNormal ButtonStates = iota
	// disabled -- not pressable
	ButtonDisabled
	// mouse is hovering over the button
	ButtonHover
	// button is the focus -- will respond to keyboard input
	ButtonFocus
	// button is currently being pressed down
	ButtonDown
	// button has been selected -- maintains selected state
	ButtonSelected
	// total number of button states
	ButtonStatesN
)

//go:generate stringer -type=ButtonStates

// ButtonBase has common button functionality -- properties: checkable, checked, autoRepeat, autoRepeatInterval, autoRepeatDelay
type ButtonBase struct {
	WidgetBase
	Text        string               `xml:"text",desc:"label for the button"`
	Shortcut    string               `xml:"shortcut",desc:"keyboard shortcut -- todo: need to figure out ctrl, alt etc"`
	StateStyles [ButtonStatesN]Style `desc:"styles for different states of the button, one for each state -- everything inherits from the base Style which is styled first according to the user-set styles, and then subsequent style settings can override that"`
	State       ButtonStates
	ButtonSig   ki.Signal `json:"-",desc:"signal for button -- see ButtonSignals for the types"`
	// todo: icon -- should be an xml
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KiT_ButtonBase = ki.Types.AddType(&ButtonBase{}, nil)

// is this button selected?
func (g *ButtonBase) IsSelected() bool {
	return ki.HasBitFlag(g.NodeFlags, int(ButtonFlagSelected))
}

// is this button checkable
func (g *ButtonBase) IsCheckable() bool {
	return ki.HasBitFlag(g.NodeFlags, int(ButtonFlagCheckable))
}

// is this button checked
func (g *ButtonBase) IsChecked() bool {
	return ki.HasBitFlag(g.NodeFlags, int(ButtonFlagChecked))
}

// set the selected state of this button
func (g *ButtonBase) SetSelected(sel bool) {
	ki.SetBitFlagState(&g.NodeFlags, int(ButtonFlagSelected), sel)
	g.SetButtonState(ButtonNormal) // update state
}

// set the checked state of this button
func (g *ButtonBase) SetChecked(chk bool) {
	ki.SetBitFlagState(&g.NodeFlags, int(ButtonFlagChecked), chk)
}

// set the button state to target
func (g *ButtonBase) SetButtonState(state ButtonStates) {
	// todo: process disabled state -- probably just deal with the property directly?
	// it overrides any choice here and just sets state to disabled..
	if state == ButtonNormal && g.IsSelected() {
		state = ButtonSelected
	} else if state == ButtonNormal && g.HasFocus() {
		state = ButtonFocus
	}
	g.State = state
	g.Style = g.StateStyles[state] // get relevant styles
}

// set the button in the down state -- mouse clicked down but not yet up --
// emits ButtonPressed signal -- ButtonClicked is down and up
func (g *ButtonBase) ButtonPressed() {
	g.UpdateStart()
	g.SetButtonState(ButtonDown)
	g.ButtonSig.Emit(g.This, int64(ButtonPressed), nil)
	g.UpdateEnd()
}

// the button has just been released -- sends a released signal and returns
// state to normal, and emits clicked signal if if it was previously in pressed state
func (g *ButtonBase) ButtonReleased() {
	wasPressed := (g.State == ButtonDown)
	g.UpdateStart()
	g.SetButtonState(ButtonNormal)
	g.ButtonSig.Emit(g.This, int64(ButtonReleased), nil)
	if wasPressed {
		g.ButtonSig.Emit(g.This, int64(ButtonClicked), nil)
	}
	g.UpdateEnd()
}

// button starting hover-- todo: keep track of time and popup a tooltip -- signal?
func (g *ButtonBase) ButtonEnterHover() {
	if g.State != ButtonHover {
		g.UpdateStart()
		g.SetButtonState(ButtonHover)
		g.UpdateEnd()
	}
}

// button exiting hover
func (g *ButtonBase) ButtonExitHover() {
	if g.State == ButtonHover {
		g.UpdateStart()
		g.SetButtonState(ButtonNormal)
		g.UpdateEnd()
	}
}

///////////////////////////////////////////////////////////

// Button is a standard command button -- PushButton in Qt Widgets, and Button in Qt Quick
type Button struct {
	ButtonBase
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KiT_Button = ki.Types.AddType(&Button{}, nil)

func (g *Button) AsNode2D() *Node2DBase {
	return &g.Node2DBase
}

func (g *Button) AsViewport2D() *Viewport2D {
	return nil
}

func (g *Button) AsLayout2D() *Layout {
	return nil
}

func (g *Button) InitNode2D() {
	g.ReceiveEventType(MouseDownEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		ab, ok := recv.(*Button)
		if !ok {
			return
		}
		ab.ButtonPressed()
	})
	g.ReceiveEventType(MouseUpEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		ab, ok := recv.(*Button)
		if !ok {
			return
		}
		ab.ButtonReleased()
	})
	g.ReceiveEventType(MouseEnteredEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		ab, ok := recv.(*Button)
		if !ok {
			return
		}
		ab.ButtonEnterHover()
	})
	g.ReceiveEventType(MouseExitedEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		ab, ok := recv.(*Button)
		if !ok {
			return
		}
		ab.ButtonExitHover()
	})
	g.ReceiveEventType(KeyTypedEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		ab, ok := recv.(*Button)
		if !ok {
			return
		}
		kt, ok := d.(KeyTypedEvent)
		if ok {
			// todo: register shortcuts with window, and generalize these keybindings
			kf := KeyFun(kt.Key, kt.Chord)
			if kf == KeyFunSelectItem || kt.Key == "space" {
				ab.ButtonPressed()
				// todo: brief delay??
				ab.ButtonReleased()
			}
		}
	})
}

var ButtonProps = []map[string]interface{}{
	{
		"border-width":        "1px",
		"border-radius":       "4px",
		"border-color":        "black",
		"border-style":        "solid",
		"padding":             "8px",
		"margin":              "4px",
		"box-shadow.h-offset": "4px",
		"box-shadow.v-offset": "4px",
		"box-shadow.blur":     "4px",
		"box-shadow.color":    "#CCC",
		// "font-family":         "Arial", // this is crashing
		"font-size":        "24pt",
		"text-align":       "center",
		"color":            "black",
		"background-color": "#EEF",
	}, { // disabled
		"border-color":     "#BBB",
		"color":            "#AAA",
		"background-color": "#DDD",
	}, { // hover
		"background-color": "#CCF", // todo "darker"
	}, { // focus
		"border-color":     "#EEF",
		"box-shadow.color": "#BBF",
	}, { // press
		"border-color":     "#DDF",
		"color":            "white",
		"background-color": "#008",
	}, { // selected
		"border-color":     "#DDF",
		"color":            "white",
		"background-color": "#00F",
	},
}

func (g *Button) Style2D() {
	// we can focus by default
	ki.SetBitFlag(&g.NodeFlags, int(CanFocus))
	// first do our normal default styles
	g.Style.SetStyle(nil, &StyleDefault, ButtonProps[ButtonNormal])
	// then style with user props
	g.Style2DWidget()
	// now get styles for the different states
	for i := 0; i < int(ButtonStatesN); i++ {
		g.StateStyles[i] = g.Style
		if i > 0 {
			g.StateStyles[i].SetStyle(nil, &StyleDefault, ButtonProps[i])
		}
		g.StateStyles[i].SetUnitContext(&g.Viewport.Render, 0)
	}
	// todo: how to get state-specific user prefs?  need an extra prefix..
}

func (g *Button) Layout2D(iter int) {
	if iter == 0 {
		g.InitLayout2D()
		st := &g.Style
		pc := &g.Paint
		var w, h float64
		w, h = pc.MeasureString(g.Text)
		if st.Layout.Width.Dots > 0 {
			w = math.Max(st.Layout.Width.Dots, w)
		}
		if st.Layout.Height.Dots > 0 {
			h = math.Max(st.Layout.Height.Dots, h)
		}
		w += 2.0*st.Padding.Dots + 2.0*st.Layout.Margin.Dots
		h += 2.0*st.Padding.Dots + 2.0*st.Layout.Margin.Dots
		g.LayData.AllocSize = Vec2D{w, h}
	} else {
		g.GeomFromLayout() // get our geom from layout -- always do this for widgets  iter > 0
	}

	// todo: test for use of parent-el relative units -- indicates whether multiple loops
	// are required
	g.Style.SetUnitContext(&g.Viewport.Render, 0)
	// now get styles for the different states
	for i := 0; i < int(ButtonStatesN); i++ {
		g.StateStyles[i].SetUnitContext(&g.Viewport.Render, 0)
	}

}

func (g *Button) Node2DBBox() image.Rectangle {
	// fmt.Printf("button win box: %v\n", g.WinBBox)
	return g.WinBBoxFromAlloc()
}

// todo: need color brigher / darker functions

func (g *Button) Render2D() {
	if g.IsLeaf() {
		g.Render2DDefaultStyle()
	} else {
		// todo: manage stacked layout to select appropriate image based on state
		return
	}
}

// render using a default style if not otherwise styled
func (g *Button) Render2DDefaultStyle() {
	pc := &g.Paint
	rs := &g.Viewport.Render
	st := &g.Style
	pc.FontStyle = st.Font
	pc.TextStyle = st.Text
	g.DrawStdBox()
	pc.StrokeStyle.SetColor(&st.Color) // ink color

	pos := g.LayData.AllocPos.AddVal(st.Layout.Margin.Dots + st.Padding.Dots)
	// sz := g.LayData.AllocSize.AddVal(-2.0 * (st.Layout.Margin.Dots + st.Padding.Dots))

	pc.DrawStringAnchored(rs, g.Text, pos.X, pos.Y, 0.0, 0.9)
}

func (g *Button) CanReRender2D() bool {
	return true
}

func (g *Button) FocusChanged2D(gotFocus bool) {
	// fmt.Printf("focus changed %v\n", gotFocus)
	g.UpdateStart()
	if gotFocus {
		g.SetButtonState(ButtonFocus)
	} else {
		g.SetButtonState(ButtonNormal) // lose any hover state but whatever..
	}
	g.UpdateEnd()
}

// check for interface implementation
var _ Node2D = &Button{}
