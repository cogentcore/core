// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"image/color"
	"log"
	"math"
	"reflect"
	"sort"
	"strconv"
	"unicode"
	"unicode/utf8"

	"github.com/chewxy/math32"
	"github.com/rcoreilly/goki/gi/oswin"
	"github.com/rcoreilly/goki/gi/oswin/key"
	"github.com/rcoreilly/goki/gi/oswin/mouse"
	"github.com/rcoreilly/goki/gi/units"
	"github.com/rcoreilly/goki/ki"
	"github.com/rcoreilly/goki/ki/bitflag"
	"github.com/rcoreilly/goki/ki/kit"
)

////////////////////////////////////////////////////////////////////////////////////////
// Labeler Interface and ToLabel method

// the labeler interface provides a GUI-appropriate label (todo: rich text html tags!?) for an item -- use ToLabel converter to attempt to use this interface and then fall back on Stringer via kit.ToString conversion function
type Labeler interface {
	Label() string
}

// ToLabel returns the gui-appropriate label for an item, using the Labeler interface if it is defined, and falling back on kit.ToString converter otherwise -- also contains label impls for basic interface types for which we cannot easily define the Labeler interface
func ToLabel(it interface{}) string {
	lbler, ok := it.(Labeler)
	if !ok {
		// typ := reflect.TypeOf(it)
		// if kit.EmbeddedTypeImplements(typ, reflect.TypeOf((*reflect.Type)(nil)).Elem()) {
		// 	to, ok :=
		// }
		switch v := it.(type) {
		case reflect.Type:
			return v.Name()
		}
		return kit.ToString(it)
	}
	return lbler.Label()
}

// Labeler interface for some key types
// note: this doesn't work b/c reflect.Type is an interface..
// func (ty reflect.Type) Label() string {
// 	return ty.Name() //  stringer adds the prefix -- we drop that..
// }

////////////////////////////////////////////////////////////////////////////////////////
// Label

// Label is a widget for rendering text labels -- supports full widget model
// including box rendering
type Label struct {
	WidgetBase
	Text string `xml:"text" desc:"label to display"`
}

var KiT_Label = kit.Types.AddType(&Label{}, LabelProps)

var LabelProps = ki.Props{
	"padding":        units.NewValue(2, units.Px),
	"margin":         units.NewValue(2, units.Px),
	"vertical-align": AlignTop,
}

func (g *Label) Style2D() {
	g.Style2DWidget(LabelProps)
}

func (g *Label) Size2D() {
	g.InitLayout2D()
	g.Size2DFromText(g.Text)
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

// mutually-exclusive textfield states -- determines appearance
type TextFieldStates int32

const (
	// normal state -- there but not being interacted with
	TextFieldActive TextFieldStates = iota
	// textfield is the focus -- will respond to keyboard input
	TextFieldFocus
	// read only / disabled -- not editable
	TextFieldReadOnly
	TextFieldStatesN
)

//go:generate stringer -type=TextFieldStates

// Style selector names for the different states
var TextFieldSelectors = []string{":active", ":focus", ":read-only"}

// TextField is a widget for editing a line of text
type TextField struct {
	WidgetBase
	Text          string                  `json:"-" xml:"text" desc:"the last saved value of the text string being edited"`
	EditText      string                  `json:"-" xml:"-" desc:"the live text string being edited, with latest modifications"`
	StartPos      int                     `xml:"start-pos" desc:"starting display position in the string"`
	EndPos        int                     `xml:"end-pos" desc:"ending display position in the string"`
	CursorPos     int                     `xml:"cursor-pos" desc:"current cursor position"`
	CharWidth     int                     `xml:"char-width" desc:"approximate number of chars that can be displayed at any time -- computed from font size etc"`
	SelectMode    bool                    `xml:"select-mode" desc:"if true, select text as cursor moves"`
	TextFieldSig  ki.Signal               `json:"-" xml:"-" desc:"signal for line edit -- see TextFieldSignals for the types"`
	StateStyles   [TextFieldStatesN]Style `json:"-" xml:"-" desc:"normal style and focus style"`
	CharPos       []float32               `json:"-" xml:"-" desc:"character positions, for point just AFTER the given character -- todo there are likely issues with runes here -- need to test.."`
	lastSizedText string                  `json:"-" xml:"-" the last text string we got charpos for"`
}

var KiT_TextField = kit.Types.AddType(&TextField{}, TextFieldProps)

var TextFieldProps = ki.Props{
	TextFieldSelectors[TextFieldActive]: ki.Props{
		"border-width":     units.NewValue(1, units.Px),
		"border-color":     color.Black,
		"border-style":     "solid",
		"padding":          units.NewValue(4, units.Px),
		"margin":           units.NewValue(1, units.Px),
		"text-align":       AlignLeft,
		"vertical-align":   AlignTop,
		"color":            "black",
		"background-color": "#EEE",
	},
	TextFieldSelectors[TextFieldFocus]: ki.Props{
		"background-color": color.White,
	},
	TextFieldSelectors[TextFieldReadOnly]: ki.Props{
		"background-color": "#CCC",
	},
}

func (g *TextField) SetText(txt string) {
	if g.Text == txt && g.EditText == txt {
		return
	}
	g.Text = txt
	g.RevertEdit()
}

// done editing: return key pressed or out of focus
func (g *TextField) EditDone() {
	g.Text = g.EditText
	g.TextFieldSig.Emit(g.This, int64(TextFieldDone), g.Text)
}

// abort editing -- revert to last saved text
func (g *TextField) RevertEdit() {
	updt := g.UpdateStart()
	g.EditText = g.Text
	g.StartPos = 0
	g.EndPos = g.CharWidth
	g.UpdateEnd(updt)
}

func (g *TextField) CursorForward(steps int) {
	updt := g.UpdateStart()
	g.CursorPos += steps
	if g.CursorPos > len(g.EditText) {
		g.CursorPos = len(g.EditText)
	}
	if g.CursorPos > g.EndPos {
		inc := g.CursorPos - g.EndPos
		g.EndPos += inc
	}
	g.UpdateEnd(updt)
}

func (g *TextField) CursorBackward(steps int) {
	updt := g.UpdateStart()
	// todo: select mode
	g.CursorPos -= steps
	if g.CursorPos < 0 {
		g.CursorPos = 0
	}
	if g.CursorPos <= g.StartPos {
		dec := kit.MinInt(g.StartPos, 8)
		g.StartPos -= dec
	}
	g.UpdateEnd(updt)
}

func (g *TextField) CursorStart() {
	updt := g.UpdateStart()
	// todo: select mode
	g.CursorPos = 0
	g.StartPos = 0
	g.EndPos = kit.MinInt(len(g.EditText), g.StartPos+g.CharWidth)
	g.UpdateEnd(updt)
}

func (g *TextField) CursorEnd() {
	updt := g.UpdateStart()
	g.CursorPos = len(g.EditText)
	g.EndPos = len(g.EditText) // try -- display will adjust
	g.StartPos = kit.MaxInt(0, g.EndPos-g.CharWidth)
	g.UpdateEnd(updt)
}

func (g *TextField) CursorBackspace(steps int) {
	if g.CursorPos < steps {
		steps = g.CursorPos
	}
	if steps <= 0 {
		return
	}
	updt := g.UpdateStart()
	g.EditText = g.EditText[:g.CursorPos-steps] + g.EditText[g.CursorPos:]
	g.CursorBackward(steps)
	g.UpdateEnd(updt)
}

func (g *TextField) CursorDelete(steps int) {
	if g.CursorPos+steps > len(g.EditText) {
		steps = len(g.EditText) - g.CursorPos
	}
	if steps <= 0 {
		return
	}
	updt := g.UpdateStart()
	g.EditText = g.EditText[:g.CursorPos] + g.EditText[g.CursorPos+steps:]
	g.UpdateEnd(updt)
}

func (g *TextField) CursorKill() {
	steps := len(g.EditText) - g.CursorPos
	g.CursorDelete(steps)
}

func (g *TextField) InsertAtCursor(str string) {
	updt := g.UpdateStart()
	g.EditText = g.EditText[:g.CursorPos] + str + g.EditText[g.CursorPos:]
	g.EndPos += len(str)
	g.CursorForward(len(str))
	g.UpdateEnd(updt)
}

func (g *TextField) KeyInput(kt *key.ChordEvent) {
	kf := KeyFun(kt.ChordString())
	switch kf {
	case KeyFunSelectItem:
		g.EditDone() // not processed, others could consume
	case KeyFunAccept:
		g.EditDone() // not processed, others could consume
	case KeyFunAbort:
		g.RevertEdit() // not processed, others could consume
	case KeyFunMoveRight:
		g.CursorForward(1)
		kt.SetProcessed()
	case KeyFunMoveLeft:
		g.CursorBackward(1)
		kt.SetProcessed()
	case KeyFunHome:
		g.CursorStart()
		kt.SetProcessed()
	case KeyFunEnd:
		g.CursorEnd()
		kt.SetProcessed()
	case KeyFunBackspace:
		g.CursorBackspace(1)
		kt.SetProcessed()
	case KeyFunKill:
		g.CursorKill()
		kt.SetProcessed()
	case KeyFunDelete:
		g.CursorDelete(1)
		kt.SetProcessed()
	case KeyFunNil:
		if unicode.IsPrint(kt.Rune) {
			g.InsertAtCursor(string(kt.Rune))
		}
	}
}

// PixelToCursor finds the cursor position that corresponds to the given pixel location
func (g *TextField) PixelToCursor(pixOff float32) int {
	st := &g.Style

	spc := st.BoxSpace()
	px := pixOff - spc

	if px <= 0 {
		return g.StartPos
	}

	// todo: we're getting the wrong sizes here -- not sure why..
	// fmt.Printf("em %v ex %v ch %v\n", st.UnContext.ToDotsFactor(units.Em), st.UnContext.ToDotsFactor(units.Ex), st.UnContext.ToDotsFactor(units.Ch))

	sz := len(g.EditText)
	c := g.StartPos + int(math.Round(float64(px/st.UnContext.ToDotsFactor(units.Ch))))
	c = kit.MinInt(c, sz)

	lastbig := false
	lastsm := false
	for {
		w := g.TextWidth(g.StartPos, c)
		if w > px {
			if lastsm { // last was smaller, break
				break
			}
			c--
			if c <= g.StartPos {
				c = g.StartPos
				break
			}
			lastbig = true
			// fmt.Printf("dec c: %v, w: %v, px: %v\n", c, w, px)
		} else if w < px { // last was bigger, brea
			if lastbig {
				break
			}
			c++
			// fmt.Printf("inc c: %v, w: %v, px: %v\n", c, w, px)
			if c > sz {
				c = sz
				break
			}
			lastsm = true
		}
	}
	return c
}

func (g *TextField) SetCursorFromPixel(pixOff float32) {
	updt := g.UpdateStart()
	g.CursorPos = g.PixelToCursor(pixOff)
	g.UpdateEnd(updt)
}

////////////////////////////////////////////////////
//  Node2D Interface

func (g *TextField) Init2D() {
	g.Init2DWidget()
	g.EditText = g.Text
	// if g.IsReadOnly() {
	// 	return
	// }
	g.ReceiveEventType(oswin.MouseEvent, func(recv, send ki.Ki, sig int64, d interface{}) {
		tf := recv.(*TextField)
		if tf.IsReadOnly() { // todo: need more subtle read-only behavior here -- can select but not edit
			return
		}
		me := d.(*mouse.Event)
		me.SetProcessed()
		if !tf.HasFocus() {
			tf.GrabFocus()
		}
		if me.Action == mouse.Press {
			pt := tf.PointToRelPos(me.Pos())
			tf.SetCursorFromPixel(float32(pt.X))
		}
	})
	g.ReceiveEventType(oswin.KeyChordEvent, func(recv, send ki.Ki, sig int64, d interface{}) {
		tf := recv.(*TextField)
		if tf.IsReadOnly() {
			return
		}
		kt := d.(*key.ChordEvent)
		tf.KeyInput(kt)
	})
}

func (g *TextField) Style2D() {
	if g.IsReadOnly() {
		bitflag.Clear(&g.Flag, int(CanFocus))
	} else {
		bitflag.Set(&g.Flag, int(CanFocus))
	}
	g.Style2DWidget(g.StyleProps(TextFieldSelectors[TextFieldActive]))
	for i := 0; i < int(TextFieldStatesN); i++ {
		g.StateStyles[i] = g.Style
		if i > 0 {
			g.StateStyles[i].SetStyle(nil, g.StyleProps(TextFieldSelectors[i]))
		}
		g.StateStyles[i].SetUnitContext(g.Viewport, Vec2DZero)
	}
}

func (g *TextField) UpdateCharPos() bool {
	if g.EditText == g.lastSizedText && len(g.EditText) == len(g.CharPos) {
		return false
	}
	g.CharPos = g.Paint.MeasureChars(g.EditText)
	g.lastSizedText = g.EditText
	return true
}

func (g *TextField) Size2D() {
	g.EditText = g.Text
	g.StartPos = 0
	g.EndPos = len(g.EditText)
	g.UpdateCharPos()
	h := g.Paint.FontHeight()
	w := float32(10.0)
	sz := len(g.CharPos)
	if sz > 0 {
		w = g.CharPos[sz-1]
	}
	g.Size2DFromWH(w, h)
}

func (g *TextField) Layout2D(parBBox image.Rectangle) {
	g.Layout2DWidget(parBBox)
	for i := 0; i < int(TextFieldStatesN); i++ {
		g.StateStyles[i].CopyUnitContext(&g.Style.UnContext)
	}
	g.Layout2DChildren()
}

// StartCharPos returns the starting position of the given character -- CharPos contains the ending positions
func (g *TextField) StartCharPos(idx int) float32 {
	if idx <= 0 {
		return 0.0
	}
	sz := len(g.CharPos)
	if sz == 0 {
		return 0.0
	}
	if idx > sz {
		return g.CharPos[sz-1]
	}
	return g.CharPos[idx-1]
}

// TextWidth returns the text width in dots between the two text string
// positions (ed is exclusive -- +1 beyond actual char)
func (g *TextField) TextWidth(st, ed int) float32 {
	return g.StartCharPos(ed) - g.StartCharPos(st)
}

func (g *TextField) RenderCursor() {
	pc := &g.Paint
	rs := &g.Viewport.Render
	st := &g.Style
	pc.FontStyle = st.Font
	pc.TextStyle = st.Text
	spc := st.BoxSpace()
	pos := g.LayData.AllocPos.AddVal(spc)

	cpos := g.TextWidth(g.StartPos, g.CursorPos)

	h := pc.FontHeight()
	pc.DrawLine(rs, pos.X+cpos, pos.Y, pos.X+cpos, pos.Y+h)
	pc.Stroke(rs)
}

// AutoScroll scrolls the starting position to keep the cursor visible
func (g *TextField) AutoScroll() {
	st := &g.Style

	g.UpdateCharPos()

	sz := len(g.EditText)

	if sz == 0 {
		g.CursorPos = 0
		g.EndPos = 0
		g.StartPos = 0
		return
	}
	spc := st.BoxSpace()
	maxw := g.LayData.AllocSize.X - 2.0*spc
	g.CharWidth = int(maxw / st.UnContext.ToDotsFactor(units.Ch)) // rough guess in chars

	// first rationalize all the values
	if g.EndPos == 0 || g.EndPos > sz { // not init
		g.EndPos = sz
	}
	if g.StartPos >= g.EndPos {
		g.StartPos = kit.MaxInt(0, g.EndPos-g.CharWidth)
	}
	g.CursorPos = kit.MinInt(g.CursorPos, sz)
	g.CursorPos = kit.MaxInt(g.CursorPos, 0)

	inc := int(math32.Ceil(.1 * float32(g.CharWidth)))
	inc = kit.MaxInt(4, inc)

	// keep cursor in view with buffer
	startIsAnchor := true
	if g.CursorPos < (g.StartPos + inc) {
		g.StartPos -= inc
		g.StartPos = kit.MaxInt(g.StartPos, 0)
		g.EndPos = g.StartPos + g.CharWidth
		g.EndPos = kit.MinInt(sz, g.EndPos)
	} else if g.CursorPos > (g.EndPos - inc) {
		g.EndPos += inc
		g.EndPos = kit.MinInt(g.EndPos, sz)
		g.StartPos = g.EndPos - g.CharWidth
		g.StartPos = kit.MaxInt(0, g.StartPos)
		startIsAnchor = false
	}

	if startIsAnchor {
		gotWidth := false
		spos := g.StartCharPos(g.StartPos)
		for {
			w := g.StartCharPos(g.EndPos) - spos
			if w < maxw {
				if g.EndPos == sz {
					break
				}
				nw := g.StartCharPos(g.EndPos+1) - spos
				if nw >= maxw {
					gotWidth = true
					break
				}
				g.EndPos++
			} else {
				g.EndPos--
			}
		}
		if gotWidth || g.StartPos == 0 {
			return
		}
		// otherwise, try getting some more chars by moving up start..
	}

	// end is now anchor
	epos := g.StartCharPos(g.EndPos)
	for {
		w := epos - g.StartCharPos(g.StartPos)
		if w < maxw {
			if g.StartPos == 0 {
				break
			}
			nw := epos - g.StartCharPos(g.StartPos-1)
			if nw >= maxw {
				break
			}
			g.StartPos--
		} else {
			g.StartPos++
		}
	}
}

func (g *TextField) Render2D() {
	if g.PushBounds() {
		g.AutoScroll()
		if g.IsReadOnly() {
			g.Style = g.StateStyles[TextFieldReadOnly]
		} else if g.HasFocus() {
			g.Style = g.StateStyles[TextFieldFocus]
		} else {
			g.Style = g.StateStyles[TextFieldActive]
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

func (g *TextField) FocusChanged2D(gotFocus bool) {
	if !gotFocus && !g.IsReadOnly() {
		g.EditDone() // lose focus
	}
	g.UpdateSig()
}

// check for interface implementation
var _ Node2D = &TextField{}

////////////////////////////////////////////////////////////////////////////////////////
// SpinBox

//go:generate stringer -type=TextFieldSignals

// SpinBox combines a TextField with up / down buttons for incrementing / decrementing values -- all configured within the Parts of the widget
type SpinBox struct {
	WidgetBase
	Value      float32   `xml:"value" desc:"current value"`
	HasMin     bool      `xml:"has-min" desc:"is there a minimum value to enforce"`
	Min        float32   `xml:"min" desc:"minimum value in range"`
	HasMax     bool      `xml:"has-max" desc:"is there a maximumvalue to enforce"`
	Max        float32   `xml:"max" desc:"maximum value in range"`
	Step       float32   `xml:"step" desc:"smallest step size to increment"`
	PageStep   float32   `xml:"pagestep" desc:"larger PageUp / Dn step size"`
	Prec       int       `desc:"specifies the precision of decimal places (total, not after the decimal point) to use in representing the number -- this helps to truncate small weird floating point values in the nether regions"`
	UpIcon     *Icon     `json:"-" xml:"-" desc:"icon to use for up button -- defaults to widget-wedge-up"`
	DownIcon   *Icon     `json:"-" xml:"-" desc:"icon to use for down button -- defaults to widget-wedge-down"`
	SpinBoxSig ki.Signal `json:"-" xml:"-" desc:"signal for spin box -- has no signal types, just emitted when the value changes"`
}

var KiT_SpinBox = kit.Types.AddType(&SpinBox{}, SpinBoxProps)

var SpinBoxProps = ki.Props{
	":active": ki.Props{ // todo: could add other states
		"#buttons": ki.Props{
			"vert-align": AlignMiddle,
		},
		"#up": ki.Props{
			"max-width":  units.NewValue(1.5, units.Ex),
			"max-height": units.NewValue(1.5, units.Ex),
			"margin":     units.NewValue(1, units.Px),
			"padding":    units.NewValue(0, units.Px),
		},
		"#down": ki.Props{
			"max-width":  units.NewValue(1.5, units.Ex),
			"max-height": units.NewValue(1.5, units.Ex),
			"margin":     units.NewValue(1, units.Px),
			"padding":    units.NewValue(0, units.Px),
		},
		"#space": ki.Props{
			"width": units.NewValue(.1, units.Ex),
		},
		"#text-field": ki.Props{
			"min-width": units.NewValue(4, units.Ex),
			"width":     units.NewValue(8, units.Ex),
			"margin":    units.NewValue(2, units.Px),
			"padding":   units.NewValue(2, units.Px),
		},
	},
}

func (g *SpinBox) Defaults() { // todo: should just get these from props
	g.Step = 0.1
	g.PageStep = 0.2
	g.Max = 1.0
	g.Prec = 9
}

// SetMin sets the min limits on the value
func (g *SpinBox) SetMin(min float32) {
	g.HasMin = true
	g.Min = min
}

// SetMax sets the max limits on the value
func (g *SpinBox) SetMax(max float32) {
	g.HasMax = true
	g.Max = max
}

// SetMinMax sets the min and max limits on the value
func (g *SpinBox) SetMinMax(hasMin bool, min float32, hasMax bool, max float32) {
	g.HasMin = hasMin
	g.Min = min
	g.HasMax = hasMax
	g.Max = max
	if g.Max < g.Min {
		log.Printf("gi.SpinBox SetMinMax: max was less than min -- disabling limits\n")
		g.HasMax = false
		g.HasMin = false
	}
}

// SetValue sets the value, enforcing any limits, and updates the display
func (g *SpinBox) SetValue(val float32) {
	updt := g.UpdateStart()
	g.Value = val
	if g.HasMax {
		g.Value = Min32(g.Value, g.Max)
	}
	if g.HasMin {
		g.Value = Max32(g.Value, g.Min)
	}
	g.Value = Truncate32(g.Value, g.Prec)
	g.UpdateEnd(updt)
}

// SetValueAction calls SetValue and also emits the signal
func (g *SpinBox) SetValueAction(val float32) {
	g.SetValue(val)
	g.SpinBoxSig.Emit(g.This, 0, g.Value)
}

// IncrValue increments the value by given number of steps (+ or -), and enforces it to be an even multiple of the step size (snap-to-value), and emits the signal
func (g *SpinBox) IncrValue(steps float32) {
	val := g.Value + steps*g.Step
	val = FloatMod32(val, g.Step)
	g.SetValueAction(val)
}

// internal indexes for accessing elements of the widget
const (
	sbTextFieldIdx = iota
	sbSpaceIdx
	sbButtonsIdx
)

func (g *SpinBox) ConfigParts() {
	if g.UpIcon == nil {
		g.UpIcon = IconByName("widget-wedge-up")
	}
	if g.DownIcon == nil {
		g.DownIcon = IconByName("widget-wedge-down")
	}
	g.Parts.Lay = LayoutRow
	g.Parts.SetProp("vert-align", AlignMiddle) // todo: style..
	props := g.StyleProps(":active")
	config := kit.TypeAndNameList{}
	config.Add(KiT_TextField, "text-field")
	config.Add(KiT_Space, "space")
	config.Add(KiT_Layout, "buttons")
	mods, updt := g.Parts.ConfigChildren(config, false) // not unique names
	if mods {
		buts := g.Parts.Child(sbButtonsIdx).(*Layout)
		buts.Lay = LayoutCol
		g.PartStyleProps(buts, props)
		buts.SetNChildren(2, KiT_Action, "but")
		// up
		up := buts.Child(0).(*Action)
		up.SetName("up")
		bitflag.SetState(up.Flags(), g.IsReadOnly(), int(ReadOnly))
		up.Icon = g.UpIcon
		g.PartStyleProps(up.This, props)
		if !g.IsReadOnly() {
			up.ActionSig.ConnectOnly(g.This, func(recv, send ki.Ki, sig int64, data interface{}) {
				sb := recv.(*SpinBox)
				sb.IncrValue(1.0)
			})
		}
		// dn
		dn := buts.Child(1).(*Action)
		bitflag.SetState(dn.Flags(), g.IsReadOnly(), int(ReadOnly))
		dn.SetName("down")
		dn.Icon = g.DownIcon
		g.PartStyleProps(dn.This, props)
		if !g.IsReadOnly() {
			dn.ActionSig.ConnectOnly(g.This, func(recv, send ki.Ki, sig int64, data interface{}) {
				sb := recv.(*SpinBox)
				sb.IncrValue(-1.0)
			})
		}
		// space
		g.PartStyleProps(g.Parts.Child(sbSpaceIdx), props) // also get the space
		// text-field
		tf := g.Parts.Child(sbTextFieldIdx).(*TextField)
		bitflag.SetState(tf.Flags(), g.IsReadOnly(), int(ReadOnly))
		g.PartStyleProps(tf.This, props)
		tf.Text = fmt.Sprintf("%g", g.Value)
		if !g.IsReadOnly() {
			tf.TextFieldSig.ConnectOnly(g.This, func(recv, send ki.Ki, sig int64, data interface{}) {
				sb := recv.(*SpinBox)
				tf := send.(*TextField)
				vl, err := strconv.ParseFloat(tf.Text, 32)
				if err == nil {
					sb.SetValueAction(float32(vl))
				}
			})
		}
		g.UpdateEnd(updt)
	}
}

func (g *SpinBox) ConfigPartsIfNeeded() {
	if !g.Parts.HasChildren() {
		g.ConfigParts()
	}
	tf := g.Parts.Child(sbTextFieldIdx).(*TextField)
	txt := fmt.Sprintf("%g", g.Value)
	if tf.Text != txt {
		tf.SetText(txt)
	}
}

func (g *SpinBox) Init2D() {
	g.Init2DWidget()
	g.ConfigParts()
}

func (g *SpinBox) Style2D() {
	if g.Step == 0 {
		g.Defaults()
	}
	g.Style2DWidget(g.StyleProps(":active"))
	g.ConfigParts()
}

func (g *SpinBox) Size2D() {
	g.Size2DWidget()
	g.ConfigParts()
}

func (g *SpinBox) Layout2D(parBBox image.Rectangle) {
	g.ConfigPartsIfNeeded()
	g.Layout2DWidget(parBBox)
	g.Layout2DChildren()
}

func (g *SpinBox) Render2D() {
	if g.PushBounds() {
		// g.Style = g.StateStyles[g.State] // get current styles
		g.ConfigPartsIfNeeded()
		g.Render2DChildren()
		g.Render2DParts()
		g.PopBounds()
	}
}

////////////////////////////////////////////////////////////////////////////////////////
// ComboBox for selecting items from a list

type ComboBox struct {
	ButtonBase
	Editable  bool          `desc:"provide a text field for editing the value, or just a button for selecting items?"`
	CurVal    interface{}   `json:"-" xml:"-" desc:"current selected value"`
	CurIndex  int           `json:"-" xml:"-" desc:"current index in list of possible items"`
	Items     []interface{} `json:"-" xml:"-" desc:"items available for selection"`
	ItemsMenu Menu          `json:"-" xml:"-" desc:"the menu of actions for selecting items -- automatically generated from Items"`
	ComboSig  ki.Signal     `json:"-" xml:"-" desc:"signal for combo box, when a new value has been selected -- the signal type is the index of the selected item, and the data is the value"`
	MaxLength int           `desc:"maximum label length (in runes)"`
}

var KiT_ComboBox = kit.Types.AddType(&ComboBox{}, ComboBoxProps)

var ComboBoxProps = ki.Props{
	ButtonSelectors[ButtonActive]: ki.Props{
		"border-width":     units.NewValue(1, units.Px),
		"border-radius":    units.NewValue(4, units.Px),
		"border-color":     color.Black,
		"border-style":     BorderSolid,
		"padding":          units.NewValue(4, units.Px),
		"margin":           units.NewValue(4, units.Px),
		"text-align":       AlignCenter,
		"vertical-align":   AlignMiddle,
		"color":            color.Black,
		"background-color": "#EEF",
		"#icon": ki.Props{
			"width":   units.NewValue(1, units.Em),
			"height":  units.NewValue(1, units.Em),
			"margin":  units.NewValue(0, units.Px),
			"padding": units.NewValue(0, units.Px),
		},
		"#label": ki.Props{
			"margin":           units.NewValue(0, units.Px),
			"padding":          units.NewValue(0, units.Px),
			"background-color": "none",
		},
		"#indicator": ki.Props{
			"width":          units.NewValue(1.5, units.Ex),
			"height":         units.NewValue(1.5, units.Ex),
			"margin":         units.NewValue(0, units.Px),
			"padding":        units.NewValue(0, units.Px),
			"vertical-align": AlignBottom,
		},
	},
	ButtonSelectors[ButtonDisabled]: ki.Props{
		"border-color":     "#BBB",
		"color":            "#AAA",
		"background-color": "#DDD",
	},
	ButtonSelectors[ButtonHover]: ki.Props{
		"background-color": "#CCF", // todo "darker"
	},
	ButtonSelectors[ButtonFocus]: ki.Props{
		"border-color":     "#EEF",
		"box-shadow.color": "#BBF",
	},
	ButtonSelectors[ButtonDown]: ki.Props{
		"border-color":     "#DDF",
		"color":            "white",
		"background-color": "#008",
	},
	ButtonSelectors[ButtonSelected]: ki.Props{
		"border-color":     "#DDF",
		"color":            "white",
		"background-color": "#00F",
	},
}

// ButtonWidget interface

func (g *ComboBox) ButtonAsBase() *ButtonBase {
	return &(g.ButtonBase)
}

func (g *ComboBox) ButtonRelease() {
	if g.IsReadOnly() {
		g.SetButtonState(ButtonActive)
		return
	}
	win := g.Viewport.Win
	wasPressed := (g.State == ButtonDown)
	updt := g.UpdateStart()
	g.MakeItemsMenu()
	g.SetButtonState(ButtonActive)
	g.ButtonSig.Emit(g.This, int64(ButtonReleased), nil)
	if wasPressed {
		g.ButtonSig.Emit(g.This, int64(ButtonClicked), nil)
	}
	g.UpdateEnd(updt)
	pos := g.WinBBox.Max
	_, indic := KiToNode2D(g.Parts.ChildByName("indicator", 3))
	if indic != nil {
		pos = indic.WinBBox.Min
	} else {
		pos.Y -= 10
		pos.X -= 10
	}
	PopupMenu(g.ItemsMenu, pos.X, pos.Y, win, g.Text)
}

// MakeItems makes sure the Items list is made, and if not, or reset is true, creates one with the given capacity
func (g *ComboBox) MakeItems(reset bool, capacity int) {
	if g.Items == nil || reset {
		g.Items = make([]interface{}, 0, capacity)
	}
}

// SortItems sorts the items according to their labels
func (g *ComboBox) SortItems(ascending bool) {
	sort.Slice(g.Items, func(i, j int) bool {
		if ascending {
			return ToLabel(g.Items[i]) < ToLabel(g.Items[j])
		} else {
			return ToLabel(g.Items[i]) > ToLabel(g.Items[j])
		}
	})
}

// SetToMaxLength gets the maximum label length so that the width of the button label is automatically set according to the max length of all items in the list -- if maxLen > 0 then it is used as an upper do-not-exceed length
func (g *ComboBox) SetToMaxLength(maxLen int) {
	ml := 0
	for _, it := range g.Items {
		ml = kit.MaxInt(ml, utf8.RuneCountInString(ToLabel(it)))
	}
	if maxLen > 0 {
		ml = kit.MinInt(ml, maxLen)
	}
	g.MaxLength = ml
}

// ItemsFromTypes sets the Items list from a list of types -- see e.g., AllImplementersOf or AllEmbedsOf in kit.TypeRegistry -- if setFirst then set current item to the first item in the list, sort sorts the list in ascending order, and maxLen if > 0 auto-sets the width of the button to the contents, with the given upper limit
func (g *ComboBox) ItemsFromTypes(tl []reflect.Type, setFirst, sort bool, maxLen int) {
	sz := len(tl)
	g.Items = make([]interface{}, sz)
	for i, typ := range tl {
		g.Items[i] = typ
	}
	if sort {
		g.SortItems(true)
	}
	if maxLen > 0 {
		g.SetToMaxLength(maxLen)
	}
	if setFirst {
		g.SetCurIndex(0)
	}
}

// ItemsFromEnumList sets the Items list from a list of enum values (see kit.EnumRegistry) -- if setFirst then set current item to the first item in the list, and maxLen if > 0 auto-sets the width of the button to the contents, with the given upper limit
func (g *ComboBox) ItemsFromEnumList(el []kit.EnumValue, setFirst bool, maxLen int) {
	sz := len(el)
	g.Items = make([]interface{}, sz)
	for i, enum := range el {
		g.Items[i] = enum
	}
	if maxLen > 0 {
		g.SetToMaxLength(maxLen)
	}
	if setFirst {
		g.SetCurIndex(0)
	}
}

// ItemsFromEnum sets the Items list from an enum type, which must be registered on kit.EnumRegistry -- if setFirst then set current item to the first item in the list, and maxLen if > 0 auto-sets the width of the button to the contents, with the given upper limit -- see kit.EnumRegistry, and maxLen if > 0 auto-sets the width of the button to the contents, with the given upper limit
func (g *ComboBox) ItemsFromEnum(enumtyp reflect.Type, setFirst bool, maxLen int) {
	g.ItemsFromEnumList(kit.Enums.TypeValues(enumtyp, true), setFirst, maxLen)
}

// FindItem finds an item on list of items and returns its index
func (g *ComboBox) FindItem(it interface{}) int {
	if g.Items == nil {
		return -1
	}
	for i, v := range g.Items {
		if v == it {
			return i
		}
	}
	return -1
}

// SetCurVal sets the current value (CurVal) and the corresponding CurIndex for that item on the current Items list (adds to items list if not found) -- returns that index -- and sets the text to the string value of that value (using standard Stringer string conversion)
func (g *ComboBox) SetCurVal(it interface{}) int {
	g.CurVal = it
	g.CurIndex = g.FindItem(it)
	if g.CurIndex < 0 { // add to list if not found..
		g.CurIndex = len(g.Items)
		g.Items = append(g.Items, it)
	}
	g.SetText(ToLabel(it))
	return g.CurIndex
}

// SetCurIndex sets the current index (CurIndex) and the corresponding CurVal for that item on the current Items list (-1 if not found) -- returns value -- and sets the text to the string value of that value (using standard Stringer string conversion)
func (g *ComboBox) SetCurIndex(idx int) interface{} {
	g.CurIndex = idx
	if idx < 0 || idx >= len(g.Items) {
		g.CurVal = nil
		g.SetText(fmt.Sprintf("idx %v > len", idx))
	} else {
		g.CurVal = g.Items[idx]
		g.SetText(ToLabel(g.CurVal))
	}
	return g.CurVal
}

// SelectItem selects a given item and emits the index as the ComboSig signal and the selected item as the data
func (g *ComboBox) SelectItem(idx int) {
	g.SetCurIndex(idx)
	g.ComboSig.Emit(g.This, int64(g.CurIndex), g.CurVal)
}

// set the text and update button -- does NOT change the currently-selected value or index
func (g *ComboBox) SetText(txt string) {
	SetButtonText(g, txt)
}

// set the Icon (could be nil) and update button
func (g *ComboBox) SetIcon(ic *Icon) {
	SetButtonIcon(g, ic)
}

// make menu of all the items
func (g *ComboBox) MakeItemsMenu() {
	if g.ItemsMenu == nil {
		g.ItemsMenu = make(Menu, 0, len(g.Items))
	}
	sz := len(g.ItemsMenu)
	for i, it := range g.Items {
		var ac *Action
		if sz > i {
			ac = g.ItemsMenu[i].(*Action)
		} else {
			ac = &Action{}
			ac.Init(ac)
			g.ItemsMenu = append(g.ItemsMenu, ac.This.(Node2D))
		}
		txt := ToLabel(it)
		nm := fmt.Sprintf("Item_%v", i)
		ac.SetName(nm)
		ac.Text = txt
		ac.Data = i // index is the data
		ac.SetSelected(i == g.CurIndex)
		ac.SetAsMenu()
		ac.ActionSig.ConnectOnly(g.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			idx := data.(int)
			cb := recv.(*ComboBox)
			cb.SelectItem(idx)
		})
	}
}

func (g *ComboBox) Init2D() {
	g.Init2DWidget()
	g.ConfigParts()
	Init2DButtonEvents(g)
}

func (g *ComboBox) ConfigParts() {
	config, icIdx, lbIdx := g.ConfigPartsIconLabel(g.Icon, g.Text)
	wrIdx := -1
	icnm := kit.ToString(g.Prop("indicator", false, false))
	if icnm == "" || icnm == "nil" {
		icnm = "widget-wedge-down"
	}
	if icnm != "none" {
		config.Add(KiT_Stretch, "indic-stretch")
		wrIdx = len(config)
		config.Add(KiT_Icon, "indicator")
	}
	mods, updt := g.Parts.ConfigChildren(config, false) // not unique names
	props := g.StyleProps(ButtonSelectors[ButtonActive])
	g.ConfigPartsSetIconLabel(g.Icon, g.Text, icIdx, lbIdx, props)
	if g.MaxLength > 0 && lbIdx >= 0 {
		lbl := g.Parts.Child(lbIdx).(*Label)
		lbl.SetMinPrefWidth(units.NewValue(float32(g.MaxLength), units.Ex))
	}
	if wrIdx >= 0 {
		ic := g.Parts.Child(wrIdx).(*Icon)
		if !ic.HasChildren() || ic.UniqueNm != icnm {
			ic.CopyFrom(IconByName(icnm))
			ic.UniqueNm = icnm
			g.PartStyleProps(ic.This, props)
		}
	}
	if mods {
		g.UpdateEnd(updt)
	}
}

func (g *ComboBox) ConfigPartsIfNeeded() {
	if !g.PartsNeedUpdateIconLabel(g.Icon, g.Text) {
		return
	}
	g.ConfigParts()
}

func (g *ComboBox) Style2D() {
	bitflag.Set(&g.Flag, int(CanFocus))
	props := g.StyleProps(ButtonSelectors[ButtonActive])
	g.Style2DWidget(props)
	for i := 0; i < int(ButtonStatesN); i++ {
		g.StateStyles[i] = g.Style
		if i > 0 {
			g.StateStyles[i].SetStyle(nil, g.StyleProps(ButtonSelectors[i]))
		}
		g.StateStyles[i].SetUnitContext(g.Viewport, Vec2DZero)
	}
	g.ConfigParts()
}

func (g *ComboBox) Size2D() {
	g.Size2DWidget()
}

func (g *ComboBox) Layout2D(parBBox image.Rectangle) {
	g.ConfigParts()
	g.Layout2DWidget(parBBox)
	for i := 0; i < int(ButtonStatesN); i++ {
		g.StateStyles[i].CopyUnitContext(&g.Style.UnContext)
	}
	g.Layout2DChildren()
}

// todo: need color brigher / darker functions

func (g *ComboBox) Render2D() {
	if g.PushBounds() {
		if !g.HasChildren() {
			g.Render2DDefaultStyle()
		} else {
			g.Render2DChildren()
		}
		g.PopBounds()
	}
}

// render using a default style if not otherwise styled
func (g *ComboBox) Render2DDefaultStyle() {
	st := &g.Style
	g.RenderStdBox(st)
	g.Render2DParts()
}

func (g *ComboBox) FocusChanged2D(gotFocus bool) {
	if gotFocus {
		g.SetButtonState(ButtonFocus)
	} else {
		g.SetButtonState(ButtonActive) // lose any hover state but whatever..
	}
	g.UpdateSig()
}

// check for interface implementation
var _ Node2D = &ComboBox{}
