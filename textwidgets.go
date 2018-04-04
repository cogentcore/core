// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"
	"math"

	"github.com/rcoreilly/goki/gi/oswin"
	"github.com/rcoreilly/goki/gi/units"
	"github.com/rcoreilly/goki/ki"
	"github.com/rcoreilly/goki/ki/bitflag"
	"github.com/rcoreilly/goki/ki/kit"
)

////////////////////////////////////////////////////////////////////////////////////////
// Label

// Label is a widget for rendering text labels -- supports full widget model
// including box rendering
type Label struct {
	WidgetBase
	Text string `xml:"text" desc:"label to display"`
}

var KiT_Label = kit.Types.AddType(&Label{}, nil)

func (g *Label) AsNode2D() *Node2DBase {
	return &g.Node2DBase
}

func (g *Label) AsViewport2D() *Viewport2D {
	return nil
}

func (g *Label) AsLayout2D() *Layout {
	return nil
}

func (g *Label) Init2D() {
	g.Init2DBase()
}

var LabelProps = map[string]interface{}{
	"padding":        units.NewValue(2, units.Px),
	"margin":         units.NewValue(2, units.Px),
	"font-size":      units.NewValue(24, units.Pt),
	"vertical-align": AlignTop,
}

func (g *Label) Style2D() {
	g.Style2DWidget(LabelProps)
}

func (g *Label) Size2D() {
	g.InitLayout2D()
	g.Size2DFromText(g.Text)
}

func (g *Label) Layout2D(parBBox image.Rectangle) {
	g.Layout2DBase(parBBox, true) // init style
	g.Layout2DChildren()
}

func (g *Label) BBox2D() image.Rectangle {
	return g.BBoxFromAlloc()
}

func (g *Label) ComputeBBox2D(parBBox image.Rectangle) Vec2D {
	return g.ComputeBBox2DBase(parBBox)
}

func (g *Label) ChildrenBBox2D() image.Rectangle {
	return g.ChildrenBBox2DWidget()
}

func (g *Label) Render2D() {
	if g.PushBounds() {
		st := &g.Style
		g.RenderStdBox(st)
		g.Render2DText(g.Text)
		g.Render2DChildren()
		g.PopBounds()
	}
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
	Text         string    `xml:"text" desc:"the last saved value of the text string being edited"`
	EditText     string    `xml:"-" desc:"the live text string being edited, with latest modifications"`
	StartPos     int       `xml:"start-pos" desc:"starting display position in the string"`
	EndPos       int       `xml:"end-pos" desc:"ending display position in the string"`
	CursorPos    int       `xml:"cursor-pos" desc:"current cursor position"`
	CharWidth    int       `xml:"char-width" desc:"approximate number of chars that can be displayed at any time -- computed from font size etc"`
	SelectMode   bool      `xml:"select-mode" desc:"if true, select text as cursor moves"`
	TextFieldSig ki.Signal `json:"-" desc:"signal for line edit -- see TextFieldSignals for the types"`
	StateStyles  [2]Style  `desc:"normal style and focus style"`
}

var KiT_TextField = kit.Types.AddType(&TextField{}, nil)

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
		dec := kit.MinInt(g.StartPos, 8)
		g.StartPos -= dec
	}
	g.UpdateEnd()
}

func (g *TextField) CursorStart() {
	g.UpdateStart()
	// todo: select mode
	g.CursorPos = 0
	g.StartPos = 0
	g.EndPos = kit.MinInt(len(g.EditText), g.StartPos+g.CharWidth)
	g.UpdateEnd()
}

func (g *TextField) CursorEnd() {
	g.UpdateStart()
	g.CursorPos = len(g.EditText)
	g.EndPos = len(g.EditText) // try -- display will adjust
	g.StartPos = kit.MaxInt(0, g.EndPos-g.CharWidth)
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

func (g *TextField) CursorKill() {
	steps := len(g.EditText) - g.CursorPos
	g.CursorDelete(steps)
}

func (g *TextField) InsertAtCursor(str string) {
	g.UpdateStart()
	g.EditText = g.EditText[:g.CursorPos] + str + g.EditText[g.CursorPos:]
	g.EndPos += len(str)
	g.CursorForward(len(str))
	g.UpdateEnd()
}

func (g *TextField) KeyInput(kt oswin.KeyTypedEvent) {
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
	case KeyFunKill:
		g.CursorKill()
	case KeyFunDelete:
		g.CursorDelete(1)
	case KeyFunAbort:
		g.RevertEdit()
	case KeyFunNil:
		k := oswin.KeyToLetter(kt.Key, kt.Chord)
		if k != "" {
			g.InsertAtCursor(k)
		}
	}
}

func (g *TextField) PixelToCursor(pixOff float64) int {
	pc := &g.Paint
	st := &g.Style

	spc := st.BoxSpace()
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

func (g *TextField) Init2D() {
	g.Init2DBase()
	g.EditText = g.Text
	g.ReceiveEventType(oswin.MouseDownEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		tf := recv.(*TextField)
		md := d.(oswin.MouseDownEvent)
		if !tf.HasFocus() {
			tf.GrabFocus()
		}
		pt := tf.PointToRelPos(md.EventPos())
		tf.SetCursorFromPixel(float64(pt.X))
	})
	g.ReceiveEventType(oswin.KeyTypedEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		tf := recv.(*TextField)
		kt := d.(oswin.KeyTypedEvent)
		tf.KeyInput(kt)
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
		"vertical-align":   "top",
		"color":            "black",
		"background-color": "#EEE",
	}, { // focus
		"background-color": "#FFF",
	},
}

func (g *TextField) Style2D() {
	bitflag.Set(&g.NodeFlags, int(CanFocus))
	g.Style2DWidget(TextFieldProps[0])
	g.StateStyles[0] = g.Style
	g.StateStyles[1] = g.Style
	g.StateStyles[1].SetStyle(nil, &StyleDefault, TextFieldProps[1])
}

func (g *TextField) Size2D() {
	g.EditText = g.Text
	g.EndPos = len(g.EditText)
	g.Size2DFromText(g.EditText)
}

func (g *TextField) Layout2D(parBBox image.Rectangle) {
	g.Layout2DBase(parBBox, true) // init style
	for i := 0; i < 2; i++ {
		g.StateStyles[i].CopyUnitContext(&g.Style.UnContext)
	}
	g.Layout2DChildren()
}

func (g *TextField) BBox2D() image.Rectangle {
	return g.BBoxFromAlloc()
}

func (g *TextField) ComputeBBox2D(parBBox image.Rectangle) Vec2D {
	return g.ComputeBBox2DBase(parBBox)
}

func (g *TextField) ChildrenBBox2D() image.Rectangle {
	return g.ChildrenBBox2DWidget()
}

func (g *TextField) RenderCursor() {
	pc := &g.Paint
	rs := &g.Viewport.Render
	st := &g.Style
	pc.FontStyle = st.Font
	pc.TextStyle = st.Text

	spc := st.BoxSpace()

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
	spc := st.BoxSpace()
	maxw := g.LayData.AllocSize.X - 2.0*spc
	g.CharWidth = int(maxw / st.UnContext.ToDotsFactor(units.Ch))

	if g.EndPos == 0 || g.EndPos > sz { // not init
		g.EndPos = sz
	}
	if g.StartPos >= g.EndPos {
		g.StartPos = kit.MaxInt(0, g.EndPos-g.CharWidth)
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
	g.EndPos = kit.MinInt(len(g.EditText), g.EndPos)
	if g.StartPos > g.EndPos {
		g.StartPos = kit.MaxInt(0, g.EndPos-g.CharWidth)
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
	g.EndPos = kit.MinInt(len(g.EditText), g.EndPos)
	if g.StartPos > g.EndPos {
		g.StartPos = kit.MaxInt(0, g.EndPos-g.CharWidth)
	}
}

func (g *TextField) Render2D() {
	if g.PushBounds() {
		g.AutoScroll()
		if g.HasFocus() {
			g.Style = g.StateStyles[1]
		} else {
			g.Style = g.StateStyles[0]
		}
		g.RenderStdBox(&g.Style)
		cur := g.EditText[g.StartPos:g.EndPos]
		g.Render2DText(cur)
		if g.HasFocus() {
			g.RenderCursor()
		}
		g.Render2DChildren()
		g.PopBounds()
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
