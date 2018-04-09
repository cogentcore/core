// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"reflect"
	"sort"
	"unicode/utf8"

	"github.com/rcoreilly/goki/gi/oswin"
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

var KiT_Label = kit.Types.AddType(&Label{}, nil)

var LabelProps = map[string]interface{}{
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

func (g *TextField) KeyInput(kt *oswin.KeyTypedEvent) {
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

func (g *TextField) Init2D() {
	g.Init2DWidget()
	g.EditText = g.Text
	g.ReceiveEventType(oswin.MouseDownEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		tf := recv.(*TextField)
		md := d.(*oswin.MouseDownEvent)
		if !tf.HasFocus() {
			tf.GrabFocus()
		}
		pt := tf.PointToRelPos(md.EventPos())
		tf.SetCursorFromPixel(float64(pt.X))
		md.SetProcessed()
	})
	g.ReceiveEventType(oswin.KeyTypedEventType, func(recv, send ki.Ki, sig int64, d interface{}) {
		tf := recv.(*TextField)
		kt := d.(*oswin.KeyTypedEvent)
		tf.KeyInput(kt)
		kt.SetProcessed()
	})
}

var TextFieldProps = [2]map[string]interface{}{
	{ // normal
		"border-width":     units.NewValue(1, units.Px),
		"border-color":     color.Black,
		"border-style":     "solid",
		"padding":          units.NewValue(4, units.Px),
		"margin":           units.NewValue(1, units.Px),
		"text-align":       AlignLeft,
		"vertical-align":   AlignTop,
		"color":            "black",
		"background-color": "#EEE",
	}, { // focus
		"background-color": color.White,
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
	g.Layout2DWidget(parBBox)
	for i := 0; i < 2; i++ {
		g.StateStyles[i].CopyUnitContext(&g.Style.UnContext)
	}
	g.Layout2DChildren()
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

	if sz == 0 {
		g.CursorPos = 0
		g.EndPos = 0
		g.StartPos = 0
		return
	}

	spc := st.BoxSpace()
	maxw := g.LayData.AllocSize.X - 2.0*spc
	g.CharWidth = int(maxw / st.UnContext.ToDotsFactor(units.Ch))

	g.CursorPos = kit.MinInt(g.CursorPos, sz)
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
// ComboBox for selecting items from a list

type ComboBox struct {
	ButtonBase
	Editable  bool          `desc:"provide a text field for editing the value, or just a button for selecting items?"`
	CurVal    interface{}   `desc:"current selected value"`
	CurIndex  int           `desc:"current index in list of possible items"`
	Items     []interface{} `desc:"items available for selection"`
	ItemsMenu Menu          `desc:"the menu of actions for selecting items -- automatically generated from Items"`
	ComboSig  ki.Signal     `desc:"signal for combo box, when a new value has been selected -- the signal type is the index of the selected item, and the data is the value"`
	MaxLength int           `desc:"maximum label length (in runes)"`
}

var KiT_ComboBox = kit.Types.AddType(&ComboBox{}, nil)

// ButtonWidget interface

func (g *ComboBox) ButtonAsBase() *ButtonBase {
	return &(g.ButtonBase)
}

func (g *ComboBox) ButtonRelease() {
	win := g.Viewport.ParentWindow()
	wasPressed := (g.State == ButtonDown)
	g.UpdateStart()
	g.MakeItemsMenu()
	g.SetButtonState(ButtonNormal)
	g.ButtonSig.Emit(g.This, int64(ButtonReleased), nil)
	if wasPressed {
		g.ButtonSig.Emit(g.This, int64(ButtonClicked), nil)
	}
	g.UpdateEnd()
	pos := g.WinBBox.Max
	_, indic := KiToNode2D(g.Parts.ChildByName("Indicator", 3))
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
		return ToLabel(g.Items[i]) < ToLabel(g.Items[j])
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

// SetCurVal sets the current value (CurVal) and the corresponding CurIndex for that item on the current Items list (-1 if not found) -- returns that index -- and sets the text to the string value of that value (using standard Stringer string conversion)
func (g *ComboBox) SetCurVal(it interface{}) int {
	g.CurVal = it
	g.CurIndex = g.FindItem(it)
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
		ac.ActionSig.Connect(g.This, func(recv, send ki.Ki, sig int64, data interface{}) {
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

var ComboBoxProps = []map[string]interface{}{
	{
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
		"#icon": map[string]interface{}{
			"width":   units.NewValue(1, units.Em),
			"height":  units.NewValue(1, units.Em),
			"margin":  units.NewValue(0, units.Px),
			"padding": units.NewValue(0, units.Px),
		},
		"#label": map[string]interface{}{
			"margin":           units.NewValue(0, units.Px),
			"padding":          units.NewValue(0, units.Px),
			"background-color": "none",
		},
		"#indicator": map[string]interface{}{
			"width":          units.NewValue(1.5, units.Ex),
			"height":         units.NewValue(1.5, units.Ex),
			"margin":         units.NewValue(0, units.Px),
			"padding":        units.NewValue(0, units.Px),
			"vertical-align": AlignBottom,
		},
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

func (g *ComboBox) ConfigParts() {
	config, icIdx, lbIdx := g.ConfigPartsIconLabel(g.Icon, g.Text)
	wrIdx := -1
	icnm := kit.ToString(g.Prop("indicator", false, false))
	if icnm == "" || icnm == "nil" {
		icnm = "widget-down-wedge"
	}
	if icnm != "none" {
		config.Add(KiT_Stretch, "InStretch")
		wrIdx = len(config)
		config.Add(KiT_Icon, "Indicator")
	}
	g.Parts.ConfigChildren(config, false) // not unique names
	g.ConfigPartsSetIconLabel(g.Icon, g.Text, icIdx, lbIdx, ComboBoxProps[ButtonNormal])
	if g.MaxLength > 0 && lbIdx >= 0 {
		lbl := g.Parts.Child(lbIdx).(*Label)
		lbl.SetMinPrefWidth(units.NewValue(float64(g.MaxLength), units.Ex))
	}
	if wrIdx >= 0 {
		ic := g.Parts.Child(wrIdx).(*Icon)
		if !ic.HasChildren() || ic.UniqueNm != icnm {
			ic.CopyFrom(IconByName(icnm))
			ic.UniqueNm = icnm
			g.PartStyleProps(ic.This, ComboBoxProps[ButtonNormal])
		}
	}
}

func (g *ComboBox) ConfigPartsIfNeeded() {
	if !g.PartsNeedUpdateIconLabel(g.Icon, g.Text) {
		return
	}
	g.ConfigParts()
}

func (g *ComboBox) Style2D() {
	bitflag.Set(&g.NodeFlags, int(CanFocus))
	g.Style2DWidget(ComboBoxProps[ButtonNormal])
	for i := 0; i < int(ButtonStatesN); i++ {
		g.StateStyles[i] = g.Style
		if i > 0 {
			g.StateStyles[i].SetStyle(nil, &StyleDefault, ComboBoxProps[i])
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
	g.UpdateStart()
	if gotFocus {
		g.SetButtonState(ButtonFocus)
	} else {
		g.SetButtonState(ButtonNormal) // lose any hover state but whatever..
	}
	g.UpdateEnd()
}

// check for interface implementation
var _ Node2D = &ComboBox{}
