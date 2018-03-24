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

// draw standard box using given style
func (g *WidgetBase) DrawStdBox(st *Style) {
	pc := &g.Paint
	// rs := &g.Viewport.Render

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
	g.DrawStdBox(st)
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
// TextField

// signals that buttons can send
type TextFieldSignals int64

const (
	// main signal -- return was pressed and an edit was completed -- data is the text
	TextFieldDone TextFieldSignals = iota
	TextFieldSignalsN
)

//go:generate stringer -type=TextFieldSignals

// TextField is a widget for editing a line of text
type TextField struct {
	WidgetBase
	Text         string    `xml:"text",desc:"the last saved value of the text string being edited"`
	EditText     string    `xml:"-",desc:"the live text string being edited, with latest modifications"`
	StartPos     int       `xml:"start-pos",desc:"starting display position in the string"`
	EndPos       int       `xml:"end-pos",desc:"ending display position in the string"`
	CursorPos    int       `xml:"cursor-pos",desc:"current cursor position"`
	CharWidth    int       `xml:"char-width",desc:"approximate number of chars that can be displayed at any time -- computed from font size etc"`
	SelectMode   bool      `xml:"select-mode",desc:"if true, select text as cursor moves"`
	TextFieldSig ki.Signal `json:"-",desc:"signal for line edit -- see TextFieldSignals for the types"`
	StateStyles  [2]Style  `desc:"normal style and focus style"`
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KiT_TextField = ki.Types.AddType(&TextField{}, nil)

// done editing: return key pressed or out of focus
func (g *TextField) EditDone() {
	g.Text = g.EditText
	g.TextFieldSig.Emit(g.This, int64(TextFieldDone), g.Text)
}

// abort editing -- revert to last saved text
func (g *TextField) RevertEdit() {
	g.UpdateStart()
	g.EditText = g.Text
	g.StartPos = 0
	g.EndPos = g.CharWidth
	g.UpdateEnd()
}

func (g *TextField) CursorForward(steps int) {
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

func (g *TextField) CursorBackward(steps int) {
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

func (g *TextField) CursorStart() {
	g.UpdateStart()
	// todo: select mode
	g.CursorPos = 0
	g.StartPos = 0
	g.EndPos = ki.MinInt(len(g.EditText), g.StartPos+g.CharWidth)
	g.UpdateEnd()
}

func (g *TextField) CursorEnd() {
	g.UpdateStart()
	g.CursorPos = len(g.EditText)
	g.EndPos = len(g.EditText) // try -- display will adjust
	g.StartPos = ki.MaxInt(0, g.EndPos-g.CharWidth)
	g.UpdateEnd()
}

func (g *TextField) CursorBackspace(steps int) {
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

func (g *TextField) CursorDelete(steps int) {
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

func (g *TextField) InsertAtCursor(str string) {
	g.UpdateStart()
	g.EditText = g.EditText[:g.CursorPos] + str + g.EditText[g.CursorPos:]
	g.EndPos += len(str)
	g.CursorForward(len(str))
	g.UpdateEnd()
}

func (g *TextField) KeyInput(kt KeyTypedEvent) {
	kf := KeyFun(kt.Key, kt.Chord)
	switch kf {
	case KeyFunSelectItem:
		g.EditDone()
	case KeyFunMoveRight:
		g.CursorForward(1)
	case KeyFunMoveLeft:
		g.CursorBackward(1)
	case KeyFunHome:
		g.CursorStart()
	case KeyFunEnd:
		g.CursorEnd()
	case KeyFunBackspace:
		g.CursorBackspace(1)
	case KeyFunDelete:
		g.CursorDelete(1)
	case KeyFunAbort:
		g.RevertEdit()
	case KeyFunNil:
		k := KeyToLetter(kt.Key, kt.Chord)
		if k != "" {
			g.InsertAtCursor(k)
		}
	}
}

func (g *TextField) PixelToCursor(pixOff float64) int {
	pc := &g.Paint
	st := &g.Style

	spc := (st.Layout.Margin.Dots + st.Padding.Dots)
	px := pixOff - spc

	if px <= 0 {
		return g.StartPos
	}

	// todo: we're getting the wrong sizes here -- not sure why..
	// fmt.Printf("em %v ex %v ch %v\n", st.UnContext.ToDotsFactor(units.Em), st.UnContext.ToDotsFactor(units.Ex), st.UnContext.ToDotsFactor(units.Ch))

	c := int(math.Round(px / st.UnContext.ToDotsFactor(units.Ch)))
	sz := len(g.EditText)
	if g.StartPos+c > sz {
		c = sz - g.StartPos
	}
	lastbig := false
	lastsm := false
	for i := 0; i < 20; i++ {
		cur := g.EditText[g.StartPos : g.StartPos+c]
		w, _ := pc.MeasureString(cur)
		if w > px {
			if lastsm { // last was smaller, break
				break
			}
			c--
			lastbig = true
			// fmt.Printf("dec c: %v, w: %v, px: %v\n", c, w, px)
		} else if w < px { // last was bigger, brea
			if lastbig {
				break
			}
			c++
			// fmt.Printf("inc c: %v, w: %v, px: %v\n", c, w, px)
			if g.StartPos+c > sz {
				c = sz - g.StartPos
				break
			}
			lastsm = true
		}
	}
	return g.StartPos + c
}

func (g *TextField) SetCursorFromPixel(pixOff float64) {
	g.UpdateStart()
	g.CursorPos = g.PixelToCursor(pixOff)
	g.UpdateEnd()
}

////////////////////////////////////////////////////
//  Node2D Interface

func (g *TextField) AsNode2D() *Node2DBase {
	return &g.Node2DBase
}

func (g *TextField) AsViewport2D() *Viewport2D {
	return nil
}

func (g *TextField) AsLayout2D() *Layout {
	return nil
}

func (g *TextField) InitNode2D() {
	g.EditText = g.Text
	g.ReceiveEventType(MouseDownEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		le, ok := recv.(*TextField)
		if ok {
			md, ok := d.(MouseDownEvent)
			if ok {
				if !le.HasFocus() {
					le.GrabFocus()
				}
				pt := le.PointToRelPos(md.EventPos())
				le.SetCursorFromPixel(float64(pt.X))
			}
		}
	})
	g.ReceiveEventType(KeyTypedEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		le, ok := recv.(*TextField)
		if ok {
			kt, ok := d.(KeyTypedEvent)
			if ok {
				le.KeyInput(kt)
			}
		}
	})
}

var TextFieldProps = [2]map[string]interface{}{
	{ // normal
		"border-width":     "1px",
		"border-color":     "black",
		"border-style":     "solid",
		"padding":          "4px",
		"margin":           "1px",
		"font-size":        "24pt",
		"text-align":       "left",
		"color":            "black",
		"background-color": "#EEE",
	}, { // focus
		"background-color": "#FFF",
	},
}

func (g *TextField) Style2D() {
	ki.SetBitFlag(&g.NodeFlags, int(CanFocus))
	// first do our normal default styles
	g.Style.SetStyle(nil, &StyleDefault, TextFieldProps[0])
	// then style with user props
	g.Style2DWidget()
	g.StateStyles[0] = g.Style
	g.StateStyles[1] = g.Style
	g.StateStyles[1].SetStyle(nil, &StyleDefault, TextFieldProps[1])
}

func (g *TextField) Layout2D(iter int) {
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
	g.StateStyles[0].SetUnitContext(&g.Viewport.Render, 0)
	g.StateStyles[1].SetUnitContext(&g.Viewport.Render, 0)
}

func (g *TextField) Node2DBBox() image.Rectangle {
	return g.WinBBoxFromAlloc()
}

func (g *TextField) RenderCursor() {
	pc := &g.Paint
	rs := &g.Viewport.Render
	st := &g.Style
	pc.FontStyle = st.Font
	pc.TextStyle = st.Text

	spc := (st.Layout.Margin.Dots + st.Padding.Dots)

	tocur := g.EditText[g.StartPos:g.CursorPos]
	w, h := pc.MeasureString(tocur)

	pos := g.LayData.AllocPos.AddVal(spc)

	pc.DrawLine(rs, pos.X+w, pos.Y, pos.X+w, pos.Y+h)
	pc.Stroke(rs)
}

// scroll the starting position to keep the cursor visible
func (g *TextField) AutoScroll() {
	pc := &g.Paint
	st := &g.Style

	sz := len(g.EditText)
	spc := (st.Layout.Margin.Dots + st.Padding.Dots)
	maxw := g.LayData.AllocSize.X - 2.0*spc
	g.CharWidth = int(maxw / st.UnContext.ToDotsFactor(units.Ch))

	if g.EndPos == 0 || g.EndPos > sz { // not init
		g.EndPos = sz
	}
	if g.StartPos > g.EndPos {
		g.StartPos = ki.MaxInt(0, g.EndPos-g.CharWidth)
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

func (g *TextField) Render2D() {
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
	g.DrawStdBox(st)
	pc.StrokeStyle.SetColor(&st.Color) // ink color

	// keep everything in range
	g.AutoScroll()

	pos := g.LayData.AllocPos.AddVal(st.Layout.Margin.Dots + st.Padding.Dots)
	// sz := g.LayData.AllocSize.AddVal(-2.0 * (st.Layout.Margin.Dots + st.Padding.Dots))

	cur := g.EditText[g.StartPos:g.EndPos]

	// todo: find baseline etc -- need a better anchored call for top-aligned
	pc.DrawStringAnchored(rs, cur, pos.X, pos.Y, 0.0, 0.9)
	if g.HasFocus() {
		g.RenderCursor()
	}
}

func (g *TextField) CanReRender2D() bool {
	return true
}

func (g *TextField) FocusChanged2D(gotFocus bool) {
	g.UpdateStart()
	if !gotFocus {
		g.EditDone() // lose focus
	}
	g.UpdateEnd()
}

// check for interface implementation
var _ Node2D = &TextField{}

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
		if ok {
			ab.ButtonPressed()
		}
	})
	g.ReceiveEventType(MouseUpEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		ab, ok := recv.(*Button)
		if ok {
			ab.ButtonReleased()
		}
	})
	g.ReceiveEventType(MouseEnteredEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		ab, ok := recv.(*Button)
		if ok {
			ab.ButtonEnterHover()
		}
	})
	g.ReceiveEventType(MouseExitedEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		ab, ok := recv.(*Button)
		if ok {
			ab.ButtonExitHover()
		}
	})
	g.ReceiveEventType(KeyTypedEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		ab, ok := recv.(*Button)
		if ok {
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
	g.DrawStdBox(st)
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

////////////////////////////////////////////////////////////////////////////////////////
// Slider

// // these extend NodeBase NodeFlags to hold slider state
// const (
// 	// slider is dragging
// 	SliderFlagDragging NodeFlags = NodeFlagsN + iota
// )

// signals that sliders can send
type SliderSignals int64

const (
	// value has changed -- if tracking is enabled, then this tracks online changes -- otherwise only at the end
	SliderValueChanged SliderSignals = iota
	// slider pushed down but not yet up
	SliderPressed
	SliderReleased
	SliderMoved
	SliderSignalsN
)

//go:generate stringer -type=SliderSignals

// mutually-exclusive slider states -- determines appearance
type SliderStates int32

const (
	// normal state -- there but not being interacted with
	SliderNormal SliderStates = iota
	// disabled -- not pressable
	SliderDisabled
	// mouse is hovering over the slider
	SliderHover
	// slider is the focus -- will respond to keyboard input
	SliderFocus
	// slider is currently being pressed down
	SliderDown
	// use background-color here to fill in selected value of slider
	SliderValueFill
	// these styles define the overall box around slider -- typically no border and a white background -- needs a background to allow local re-rendering
	SliderBox
	// total number of slider states
	SliderStatesN
)

//go:generate stringer -type=SliderStates

// todo: Snap options: never, always, on release

// SliderBase has common slider functionality
type SliderBase struct {
	WidgetBase
	Min         float64              `xml:"min",desc:"minimum value in range"`
	Max         float64              `xml:"max",desc:"maximum value in range"`
	Step        float64              `xml:"step",desc:"smallest step size to increment"`
	PageStep    float64              `xml:"step",desc:"larger PageUp / Dn step size"`
	Value       float64              `xml:"value",desc:"current value"`
	Size        float64              `xml:"size",desc:"size of the slide box in the relevant dimension -- range of motion -- exclusive of spacing"`
	ThumbSize   float64              `xml:"thumb-size",desc:"size of the thumb -- if PropThumb then this changes over time and is subtracted from Size in computing Value"`
	PropThumb   bool                 `xml:"prop-thumb","desc:"if true, has a proportionally-sized thumb knob reflecting another value -- e.g., the amount visible in a scrollbar, and thumb is completely inside Size -- otherwise ThumbSize affects Size so that full Size range can be traversed"`
	Pos         float64              `xml:"pos",desc:"logical position of the slider relative to Size"`
	DragPos     float64              `xml:"-",desc:"underlying drag position of slider -- not subject to snapping"`
	VisPos      float64              `xml:"vispos",desc:"visual position of the slider -- can be different from pos in a RTL environment"`
	Horiz       bool                 `xml:"horiz",desc:"true if horizontal, else vertical"`
	Tracking    bool                 `xml:"tracking",desc:"if true, will send continuous updates of value changes as user moves the slider -- otherwise only at the end"`
	Snap        bool                 `xml:"snap",desc:"snap the values to Step size increments"`
	StateStyles [SliderStatesN]Style `desc:"styles for different states of the slider, one for each state -- everything inherits from the base Style which is styled first according to the user-set styles, and then subsequent style settings can override that"`
	State       SliderStates
	SliderSig   ki.Signal `json:"-",desc:"signal for slider -- see SliderSignals for the types"`
	// todo: icon -- should be an xml
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KiT_SliderBase = ki.Types.AddType(&SliderBase{}, nil)

// if snap is set, then snap the value to step sizes
func (g *SliderBase) SnapValue() {
	if g.Snap {
		g.Value = float64(int(math.Round(g.Value/g.Step))) * g.Step
	}
}

// set the slider state to target
func (g *SliderBase) SetSliderState(state SliderStates) {
	// todo: process disabled state -- probably just deal with the property directly?
	// it overrides any choice here and just sets state to disabled..
	if state == SliderNormal && g.HasFocus() {
		state = SliderFocus
	}
	g.State = state
	g.Style = g.StateStyles[state] // get relevant styles
}

// set the slider in the down state -- mouse clicked down but not yet up --
// emits SliderPressed signal
func (g *SliderBase) SliderPressed(pos float64) {
	g.UpdateStart()
	g.SetSliderState(SliderDown)
	g.SliderAtPos(pos)
	g.SliderSig.Emit(g.This, int64(SliderPressed), g.Value)
	// ki.SetBitFlag(&g.NodeFlags, int(SliderFlagDragging))
	g.UpdateEnd()
}

// the slider has just been released -- sends a released signal and returns
// state to normal, and emits clicked signal if if it was previously in pressed state
func (g *SliderBase) SliderReleased() {
	wasPressed := (g.State == SliderDown)
	g.UpdateStart()
	g.SetSliderState(SliderNormal)
	g.SliderSig.Emit(g.This, int64(SliderReleased), g.Value)
	if wasPressed {
		g.SliderSig.Emit(g.This, int64(SliderValueChanged), g.Value)
	}
	g.UpdateEnd()
}

// slider starting hover-- todo: keep track of time and popup a tooltip -- signal?
func (g *SliderBase) SliderEnterHover() {
	if g.State != SliderHover {
		g.UpdateStart()
		g.SetSliderState(SliderHover)
		g.UpdateEnd()
	}
}

// slider exiting hover
func (g *SliderBase) SliderExitHover() {
	if g.State == SliderHover {
		g.UpdateStart()
		g.SetSliderState(SliderNormal)
		g.UpdateEnd()
	}
}

// get size from allocation
func (g *SliderBase) SizeFromAlloc() {
	if g.LayData.AllocSize.IsZero() {
		return
	}
	st := &g.Style
	spc := st.Layout.Margin.Dots
	if g.Horiz {
		g.Size = g.LayData.AllocSize.X - 2.0*spc
	} else {
		g.Size = g.LayData.AllocSize.Y - 2.0*spc
	}
	if !g.PropThumb {
		g.Size -= g.ThumbSize + 2.0*st.Border.Width.Dots + 2.0
	}
	g.UpdatePosFromValue()
	g.DragPos = g.Pos
}

func (g *SliderBase) SliderAtPos(pos float64) {
	g.UpdateStart()
	g.Pos = pos
	if g.PropThumb {
		effSz := g.Size - g.ThumbSize
		if effSz <= 0.0 {
			g.Pos = 0.0
			g.DragPos = 0.0
			g.Value = g.Min
		} else {
			g.Pos = math.Min(effSz, g.Pos)
			g.Pos = math.Max(0, g.Pos)
			g.Value = g.Min + (g.Max-g.Min)*(g.Pos/effSz)
			g.DragPos = g.Pos
			if g.Snap {
				g.SnapValue()
				g.UpdatePosFromValue()
			}
		}
	} else {
		g.Pos = math.Min(g.Size, g.Pos)
		g.Pos = math.Max(0, g.Pos)
		g.Value = g.Min + (g.Max-g.Min)*(g.Pos/g.Size)
		g.DragPos = g.Pos
		if g.Snap {
			g.SnapValue()
			g.UpdatePosFromValue()
		}
	}
	if g.Tracking {
		g.SliderSig.Emit(g.This, int64(SliderValueChanged), g.Value)
	}
	g.UpdateEnd()
}

// slider moved along relevant axis
func (g *SliderBase) SliderMoved(start, end float64) {
	del := end - start
	g.SliderAtPos(g.DragPos + del)
}

func (g *SliderBase) UpdatePosFromValue() {
	if g.Size == 0.0 {
		return
	}
	if g.PropThumb {
		effSz := g.Size - g.ThumbSize
		if effSz <= 0.0 {
			g.Pos = 0.0
			g.Value = g.Min
		}
		g.Pos = effSz * (g.Value - g.Min) / (g.Max - g.Min)

	} else {
		g.Pos = g.Size * (g.Value - g.Min) / (g.Max - g.Min)
	}
}

// set a value
func (g *SliderBase) SetValue(val float64) {
	g.UpdateStart()
	g.Value = math.Min(val, g.Max)
	g.Value = math.Max(g.Value, g.Min)
	g.UpdatePosFromValue()
	g.DragPos = g.Pos
	g.SliderSig.Emit(g.This, int64(SliderValueChanged), g.Value)
	g.UpdateEnd()
}

// slider moved along relevant axis
func (g *SliderBase) SetThumbSizeByValue(val float64) {
	g.ThumbSize = ((val - g.Min) / (g.Max - g.Min))
	g.ThumbSize = math.Min(g.ThumbSize, 1.0)
	g.ThumbSize = math.Max(g.ThumbSize, 0.0)
	g.ThumbSize *= g.Size
}

///////////////////////////////////////////////////////////

// Slider is a standard value slider with a fixed-sized thumb knob
type Slider struct {
	SliderBase
}

// must register all new types so type names can be looked up by name -- e.g., for json
var KiT_Slider = ki.Types.AddType(&Slider{}, nil)

func (g *Slider) Defaults() { // todo: should just get these from props
	g.ThumbSize = 25.0
	g.Step = 0.1
	g.PageStep = 0.2
	g.Max = 1.0
}

func (g *Slider) AsNode2D() *Node2DBase {
	return &g.Node2DBase
}

func (g *Slider) AsViewport2D() *Viewport2D {
	return nil
}

func (g *Slider) AsLayout2D() *Layout {
	return nil
}

func (g *Slider) InitNode2D() {
	g.ReceiveEventType(MouseDraggedEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		sl, ok := recv.(*Slider)
		if ok {
			if sl.IsDragging() {
				me := d.(MouseDraggedEvent)
				st := sl.PointToRelPos(me.From)
				ed := sl.PointToRelPos(me.Where)
				if sl.Horiz {
					sl.SliderMoved(float64(st.X), float64(ed.X))
				} else {
					sl.SliderMoved(float64(st.Y), float64(ed.Y))
				}
			}
		}
	})
	g.ReceiveEventType(MouseDownEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		sl, ok := recv.(*Slider)
		if ok {
			me := d.(MouseDownEvent)
			ed := sl.PointToRelPos(me.Where)
			st := &sl.Style
			spc := st.Layout.Margin.Dots + 0.5*g.ThumbSize
			if sl.Horiz {
				sl.SliderPressed(float64(ed.X) - spc)
			} else {
				sl.SliderPressed(float64(ed.Y) - spc)
			}
		}
	})
	g.ReceiveEventType(MouseUpEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		sl, ok := recv.(*Slider)
		if ok {
			sl.SliderReleased()
		}
	})
	g.ReceiveEventType(MouseEnteredEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		sl, ok := recv.(*Slider)
		if ok {
			sl.SliderEnterHover()
		}
	})
	g.ReceiveEventType(MouseExitedEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		sl, ok := recv.(*Slider)
		if ok {
			sl.SliderExitHover()
		}
	})
	g.ReceiveEventType(KeyTypedEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		sl, ok := recv.(*Slider)
		if ok {
			kt, ok := d.(KeyTypedEvent)
			if ok {
				// todo: register shortcuts with window, and generalize these keybindings
				kf := KeyFun(kt.Key, kt.Chord)
				switch kf {
				case KeyFunMoveUp:
					sl.SetValue(g.Value - g.Step)
				case KeyFunMoveLeft:
					sl.SetValue(g.Value - g.Step)
				case KeyFunMoveDown:
					sl.SetValue(g.Value + g.Step)
				case KeyFunMoveRight:
					sl.SetValue(g.Value + g.Step)
				case KeyFunPageUp:
					sl.SetValue(g.Value - g.PageStep)
				case KeyFunPageLeft:
					sl.SetValue(g.Value - g.PageStep)
				case KeyFunPageDown:
					sl.SetValue(g.Value + g.PageStep)
				case KeyFunPageRight:
					sl.SetValue(g.Value + g.PageStep)
				case KeyFunHome:
					sl.SetValue(g.Min)
				case KeyFunEnd:
					sl.SetValue(g.Max)
				}
			}
		}
	})
}

var SliderProps = []map[string]interface{}{
	{
		"border-width":     "1px",
		"border-radius":    "4px",
		"border-color":     "black",
		"border-style":     "solid",
		"padding":          "8px",
		"margin":           "4px",
		"background-color": "#EEF",
	}, { // disabled
		"border-color":     "#BBB",
		"background-color": "#DDD",
	}, { // hover
		"background-color": "#CCF", // todo "darker"
	}, { // focus
		"border-color":     "#008",
		"background.color": "#CCF",
	}, { // press
		"border-color":     "#000",
		"background-color": "#DDF",
	}, { // value fill
		"border-color":     "#00F",
		"background-color": "#00F",
	}, { // overall box -- just white
		"border-color":     "#FFF",
		"background-color": "#FFF",
	},
}

func (g *Slider) Style2D() {
	// we can focus by default
	ki.SetBitFlag(&g.NodeFlags, int(CanFocus))
	// first do our normal default styles
	g.Style.SetStyle(nil, &StyleDefault, SliderProps[SliderNormal])
	// then style with user props
	g.Style2DWidget()
	// now get styles for the different states
	for i := 0; i < int(SliderStatesN); i++ {
		g.StateStyles[i] = g.Style
		if i > 0 {
			g.StateStyles[i].SetStyle(nil, &StyleDefault, SliderProps[i])
		}
		g.StateStyles[i].SetUnitContext(&g.Viewport.Render, 0)
	}
	// todo: how to get state-specific user prefs?  need an extra prefix..
}

func (g *Slider) Layout2D(iter int) {
	if iter == 0 {
		g.InitLayout2D()
		if g.ThumbSize == 0.0 {
			g.Defaults()
		}
		st := &g.Style
		// get at least thumbsize
		sz := g.ThumbSize + 2.0*(st.Layout.Margin.Dots+st.Padding.Dots)
		if g.Horiz {
			g.LayData.AllocSize.Y = sz
		} else {
			g.LayData.AllocSize.X = sz
		}
	} else {
		g.GeomFromLayout() // get our geom from layout -- always do this for widgets  iter > 0
		g.SizeFromAlloc()
	}

	// todo: test for use of parent-el relative units -- indicates whether multiple loops
	// are required
	g.Style.SetUnitContext(&g.Viewport.Render, 0)
	// now get styles for the different states
	for i := 0; i < int(SliderStatesN); i++ {
		g.StateStyles[i].SetUnitContext(&g.Viewport.Render, 0)
	}
}

func (g *Slider) Node2DBBox() image.Rectangle {
	// fmt.Printf("slider win box: %v\n", g.WinBBox)
	return g.WinBBoxFromAlloc()
}

func (g *Slider) Render2D() {
	if g.IsLeaf() {
		g.Render2DDefaultStyle()
	} else {
		// todo: manage stacked layout to select appropriate image based on state
		return
	}
}

// render using a default style if not otherwise styled
func (g *Slider) Render2DDefaultStyle() {
	pc := &g.Paint
	st := &g.Style
	rs := &g.Viewport.Render

	// overall fill box
	g.DrawStdBox(&g.StateStyles[SliderBox])

	// draw a 1/2 thumbsize box with a circular thumb
	spc := st.Layout.Margin.Dots
	pos := g.LayData.AllocPos.AddVal(spc)
	sz := g.LayData.AllocSize.AddVal(-2.0 * spc)
	fullsz := sz

	pc.StrokeStyle.SetColor(&st.Border.Color)
	pc.StrokeStyle.Width = st.Border.Width
	pc.FillStyle.SetColor(&st.Background.Color)

	ht := 0.5 * g.ThumbSize

	if g.Horiz {
		pos.X += ht
		sz.X -= g.ThumbSize
		sz.Y = g.ThumbSize - 2.0*st.Padding.Dots
		ctr := pos.Y + 0.5*fullsz.Y
		pos.Y = ctr - ht + st.Padding.Dots
		g.DrawBoxImpl(pos, sz, st.Border.Radius.Dots)
		sz.X = spc + g.Pos
		pc.FillStyle.SetColor(&g.StateStyles[SliderValueFill].Background.Color)
		g.DrawBoxImpl(pos, sz, st.Border.Radius.Dots)
		pc.FillStyle.SetColor(&st.Background.Color)
		pc.DrawCircle(rs, pos.X+sz.X, ctr, ht)
		pc.FillStrokeClear(rs)
	} else {
		pos.Y += ht
		sz.Y -= g.ThumbSize
		sz.X = g.ThumbSize - 2.0*st.Padding.Dots
		ctr := pos.X + 0.5*fullsz.X
		pos.X = ctr - ht + st.Padding.Dots
		g.DrawBoxImpl(pos, sz, st.Border.Radius.Dots)
		sz.Y = spc + g.Pos
		pc.FillStyle.SetColor(&g.StateStyles[SliderValueFill].Background.Color)
		g.DrawBoxImpl(pos, sz, st.Border.Radius.Dots)
		pc.FillStyle.SetColor(&st.Background.Color)
		pc.DrawCircle(rs, ctr, pos.Y+sz.Y, ht)
		pc.FillStrokeClear(rs)
	}
}

func (g *Slider) CanReRender2D() bool {
	return true
}

func (g *Slider) FocusChanged2D(gotFocus bool) {
	// fmt.Printf("focus changed %v\n", gotFocus)
	g.UpdateStart()
	if gotFocus {
		g.SetSliderState(SliderFocus)
	} else {
		g.SetSliderState(SliderNormal) // lose any hover state but whatever..
	}
	g.UpdateEnd()
}

// check for interface implementation
var _ Node2D = &Slider{}
