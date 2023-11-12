// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"log"
	"strings"
	"time"
	"unicode"

	"goki.dev/enums"
	"goki.dev/gi/v2/keyfun"
	"goki.dev/girl/abilities"
	"goki.dev/girl/states"
	"goki.dev/girl/styles"
	"goki.dev/goosi/events"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
)

var (
	// LayoutPrefMaxRows is maximum number of rows to use in a grid layout
	// when computing the preferred size (ScPrefSizing)
	LayoutPrefMaxRows = 20

	// LayoutPrefMaxCols is maximum number of columns to use in a grid layout
	// when computing the preferred size (ScPrefSizing)
	LayoutPrefMaxCols = 20

	// LayoutFocusNameTimeoutMSec is the number of milliseconds between keypresses
	// to combine characters into name to search for within layout -- starts over
	// after this delay.
	LayoutFocusNameTimeoutMSec = 500

	// LayoutFocusNameTabMSec is the number of milliseconds since last focus name
	// event to allow tab to focus on next element with same name.
	LayoutFocusNameTabMSec = 2000

	// LayoutAutoScrollDelay is amount of time to wait before trying to autoscroll again
	LayoutAutoScrollDelay = 25 * time.Millisecond

	// AutoScrollRate determines the rate of auto-scrolling of layouts
	AutoScrollRate = float32(1.0)

	// LayoutPageSteps is the number of steps to take in PageUp / Down events
	// in terms of number of items.
	LayoutPageSteps = 10
)

// Layoutlags has bool flags for Layout
type LayoutFlags WidgetFlags //enums:bitflag -trim-prefix Layout

const (
	// for stacked layout, only layout the top widget.
	// this is appropriate for e.g., tab layout, which does a full
	// redraw on stack changes, but not for e.g., check boxes which don't
	LayoutStackTopOnly LayoutFlags = LayoutFlags(WidgetFlagsN) + iota

	// true if this layout got a redo = true on previous iteration -- otherwise it just skips any re-layout on subsequent iteration
	LayoutNeedsRedo

	// LayoutNoKeys prevents processing of keyboard events for this layout.
	// By default, Layout handles focus navigation events, but if an
	// outer Widget handles these instead, then this should be set.
	LayoutNoKeys
)

///////////////////////////////////////////////////////////////////
// Layout

// Layout is the primary node type responsible for organizing the sizes
// and positions of child widgets. It does not render, only organize,
// so properties like background color will have no effect.
// All arbitrary collections of widgets should generally be contained
// within a layout -- otherwise the parent widget must take over
// responsibility for positioning.
// Layouts can automatically add scrollbars depending on the Overflow
// layout style.
// For a Grid layout, the 'columns' property should generally be set
// to the desired number of columns, from which the number of rows
// is computed -- otherwise it uses the square root of number of
// elements.
type Layout struct {
	WidgetBase

	// for Stacked layout, index of node to use as the top of the stack.
	// Only the node at this index is rendered -- if not a valid index, nothing is rendered.
	StackTop int

	// LayImpl contains implementational state info for doing layout
	LayImpl LayImplState `edit:"-" copy:"-" json:"-" xml:"-" set:"-"`

	// whether scrollbar is used for given dim
	HasScroll [2]bool `edit:"-" copy:"-" json:"-" xml:"-" set:"-"`

	// scroll bars -- we fully manage them as needed
	Scrolls [2]*Slider `edit:"-" copy:"-" json:"-" xml:"-" set:"-"`

	// accumulated name to search for when keys are typed
	FocusName string `edit:"-" copy:"-" json:"-" xml:"-" set:"-"`

	// time of last focus name event -- for timeout
	FocusNameTime time.Time `edit:"-" copy:"-" json:"-" xml:"-" set:"-"`

	// last element focused on -- used as a starting point if name is the same
	FocusNameLast ki.Ki `edit:"-" copy:"-" json:"-" xml:"-" set:"-"`
}

func (ly *Layout) FlagType() enums.BitFlag {
	return LayoutFlags(ly.Flags)
}

func (ly *Layout) CopyFieldsFrom(frm any) {
	fr, ok := frm.(*Layout)
	if !ok {
		log.Printf("GoGi node of type: %v needs a CopyFieldsFrom method defined -- currently falling back on earlier Layout one\n", ly.KiType().Name)
		return
	}
	ly.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
	ly.StackTop = fr.StackTop
}

func (ly *Layout) OnInit() {
	ly.LayoutStyles()
	ly.HandleLayoutEvents()
}

func (ly *Layout) LayoutStyles() {
	ly.Style(func(s *styles.Style) {
		s.SetAbilities(true, abilities.FocusWithinable)
		// we never want borders on layouts
		s.MaxBorder = styles.Border{}

		switch {
		case s.Display == styles.DisplayFlex:
			// if s.Wrap {
			// 	s.Grow.Set(1, 1)
			// } else {
			s.Grow.SetDim(s.MainAxis, 1)
			s.Grow.SetDim(s.MainAxis.Other(), 0)
			// }
		case s.Display == styles.DisplayStacked:
			s.Grow.Set(1, 1)
		case s.Display == styles.DisplayGrid:
			s.Grow.Set(1, 1)
		}
	})
}

func (ly *Layout) Destroy() {
	for d := mat32.X; d <= mat32.Y; d++ {
		ly.DeleteScroll(d)
	}
	ly.WidgetBase.Destroy()
}

// DeleteScroll deletes scrollbar along given dimesion.
func (ly *Layout) DeleteScroll(d mat32.Dims) {
	if ly.Scrolls[d] == nil {
		return
	}
	sb := ly.Scrolls[d]
	sb.This().Destroy()
	ly.Scrolls[d] = nil
}

func (ly *Layout) RenderChildren(sc *Scene) {
	if ly.Styles.Display == styles.DisplayStacked {
		ly.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
			kwi.SetState(i != ly.StackTop, states.Invisible)
			return ki.Continue
		})
		kwi, _ := ly.StackTopWidget()
		if kwi != nil {
			kwi.Render(sc)
		}
		return
	}
	ly.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		kwi.Render(sc)
		return ki.Continue
	})
}

func (ly *Layout) Render(sc *Scene) {
	if ly.PushBounds(sc) {
		ly.RenderChildren(sc)
		ly.PopBounds(sc)
		ly.RenderScrolls(sc)
	}
}

// ChildWithFocus returns a direct child of this layout that either is the
// current window focus item, or contains that focus item (along with its
// index) -- nil, -1 if none.
func (ly *Layout) ChildWithFocus() (ki.Ki, int) {
	em := ly.EventMgr()
	if em == nil {
		return nil, -1
	}
	for i, k := range ly.Kids {
		if k == nil {
			continue
		}
		_, ni := AsWidget(k)
		if ni == nil {
			continue
		}
		if ni.ContainsFocus() {
			return k, i
		}
	}
	return nil, -1
}

// FocusNextChild attempts to move the focus into the next layout child
// (with wraparound to start) -- returns true if successful.
// if updn is true, then for Grid layouts, it moves down to next row
// instead of just the sequentially next item.
func (ly *Layout) FocusNextChild(updn bool) bool {
	sz := len(ly.Kids)
	if sz <= 1 {
		return false
	}
	foc, idx := ly.ChildWithFocus()
	if foc == nil {
		return false
	}
	em := ly.EventMgr()
	if em == nil {
		return false
	}
	cur := em.Focus
	nxti := idx + 1
	if ly.Styles.Display == styles.DisplayGrid && updn {
		nxti = idx + ly.Styles.Columns
	}
	did := false
	if nxti < sz {
		nx := ly.Child(nxti).(Widget)
		did = em.FocusOnOrNext(nx)
	} else {
		nx := ly.Child(0).(Widget)
		did = em.FocusOnOrNext(nx)
	}
	if !did || em.Focus == cur {
		return false
	}
	return true
}

// FocusPrevChild attempts to move the focus into the previous layout child
// (with wraparound to end) -- returns true if successful.
// If updn is true, then for Grid layouts, it moves up to next row
// instead of just the sequentially next item.
func (ly *Layout) FocusPrevChild(updn bool) bool {
	sz := len(ly.Kids)
	if sz <= 1 {
		return false
	}
	foc, idx := ly.ChildWithFocus()
	if foc == nil {
		return false
	}
	em := ly.EventMgr()
	if em == nil {
		return false
	}
	cur := em.Focus
	nxti := idx - 1
	if ly.Styles.Display == styles.DisplayGrid && updn {
		nxti = idx - ly.Styles.Columns
	}
	did := false
	if nxti >= 0 {
		did = em.FocusOnOrPrev(ly.Child(nxti).(Widget))
	} else {
		did = em.FocusOnOrPrev(ly.Child(sz - 1).(Widget))
	}
	if !did || em.Focus == cur {
		return false
	}
	return true
}

// ClosePopup closes the parent Stage as a PopupStage.
// Returns false if not a popup.
func (ly *Layout) ClosePopup() bool {
	ps := ly.Sc.PopupStage()
	if ps == nil {
		return false
	}
	ps.Close()
	return true
}

func (ly *Layout) HandleLayoutEvents() {
	ly.HandleWidgetEvents()
	ly.HandleLayoutKeys()
	ly.HandleLayoutScrollEvents()
}

// HandleLayoutKeys handles all key events for navigating focus within a Layout
// Typically this is done by the parent Scene level layout, but can be
// done by default if FocusWithinable Ability is set.
func (ly *Layout) HandleLayoutKeys() {
	ly.OnKeyChord(func(e events.Event) {
		ly.LayoutKeysImpl(e)
	})
}

// LayoutKeys is key processing for layouts -- focus name and arrow keys
func (ly *Layout) LayoutKeysImpl(e events.Event) {
	if ly.Is(LayoutNoKeys) {
		return
	}
	if KeyEventTrace {
		fmt.Println("Layout KeyInput:", ly)
	}
	kf := keyfun.Of(e.KeyChord())
	if kf == keyfun.Abort {
		if ly.ClosePopup() {
			e.SetHandled()
		}
		return
	}
	em := ly.EventMgr()
	if em == nil {
		return
	}
	switch kf {
	case keyfun.FocusNext: // tab
		if em.FocusNext() {
			// fmt.Println("foc next", ly, ly.EventMgr().Focus)
			e.SetHandled()
		}
		return
	case keyfun.FocusPrev: // shift-tab
		if em.FocusPrev() {
			// fmt.Println("foc prev", ly, ly.EventMgr().Focus)
			e.SetHandled()
		}
		return
	}
	grid := ly.Styles.Display == styles.DisplayGrid
	if ly.Styles.MainAxis == mat32.X || grid {
		switch kf {
		case keyfun.MoveRight:
			if ly.FocusNextChild(false) {
				e.SetHandled()
			}
			return
		case keyfun.MoveLeft:
			if ly.FocusPrevChild(false) {
				e.SetHandled()
			}
			return
		}
	}
	if ly.Styles.MainAxis == mat32.Y || grid {
		switch kf {
		case keyfun.MoveDown:
			if ly.FocusNextChild(true) {
				e.SetHandled()
			}
			return
		case keyfun.MoveUp:
			if ly.FocusPrevChild(true) {
				e.SetHandled()
			}
			return
		case keyfun.PageDown:
			proc := false
			for st := 0; st < LayoutPageSteps; st++ {
				if !ly.FocusNextChild(true) {
					break
				}
				proc = true
			}
			if proc {
				e.SetHandled()
			}
			return
		case keyfun.PageUp:
			proc := false
			for st := 0; st < LayoutPageSteps; st++ {
				if !ly.FocusPrevChild(true) {
					break
				}
				proc = true
			}
			if proc {
				e.SetHandled()
			}
			return
		}
	}
	ly.FocusOnName(e)
}

// FocusOnName processes key events to look for an element starting with given name
func (ly *Layout) FocusOnName(e events.Event) bool {
	if KeyEventTrace {
		fmt.Printf("Layout FocusOnName: %v\n", ly.Path())
	}
	kf := keyfun.Of(e.KeyChord())
	delayMs := int(e.Time().Sub(ly.FocusNameTime) / time.Millisecond)
	ly.FocusNameTime = e.Time()
	if kf == keyfun.FocusNext { // tab means go to next match -- don't worry about time
		if ly.FocusName == "" || delayMs > LayoutFocusNameTabMSec {
			ly.FocusName = ""
			ly.FocusNameLast = nil
			return false
		}
	} else {
		if delayMs > LayoutFocusNameTimeoutMSec {
			ly.FocusName = ""
		}
		if !unicode.IsPrint(e.KeyRune()) || e.Modifiers() != 0 {
			return false
		}
		sr := string(e.KeyRune())
		if ly.FocusName == sr {
			// re-search same letter
		} else {
			ly.FocusName += sr
			ly.FocusNameLast = nil // only use last if tabbing
		}
	}
	e.SetHandled()
	// fmt.Printf("searching for: %v  last: %v\n", ly.FocusName, ly.FocusNameLast)
	focel, found := ChildByLabelStartsCanFocus(ly, ly.FocusName, ly.FocusNameLast)
	if found {
		// todo:
		// em := ly.EventMgr()
		// if em != nil {
		// 	em.SetFocus(focel.(Widget)) // this will also scroll by default!
		// }
		ly.FocusNameLast = focel
		return true
	} else {
		if ly.FocusNameLast == nil {
			ly.FocusName = "" // nothing being found
		}
		ly.FocusNameLast = nil // start over
	}
	return false
}

// ChildByLabelStartsCanFocus uses breadth-first search to find first element
// within layout whose Label (from Labeler interface) starts with given string
// (case insensitive) and can focus.  If after is non-nil, only finds after
// given element.
func ChildByLabelStartsCanFocus(ly *Layout, name string, after ki.Ki) (ki.Ki, bool) {
	lcnm := strings.ToLower(name)
	var rki ki.Ki
	gotAfter := false
	ly.WalkBreadth(func(k ki.Ki) bool {
		if k == ly.This() { // skip us
			return ki.Continue
		}
		_, ni := AsWidget(k)
		if ni != nil && !ni.CanFocus() { // don't go any further
			return ki.Break
		}
		if after != nil && !gotAfter {
			if k == after {
				gotAfter = true
			}
			return ki.Continue // skip to next
		}
		kn := strings.ToLower(ToLabel(k))
		if rki == nil && strings.HasPrefix(kn, lcnm) {
			rki = k
			return ki.Break
		}
		return rki == nil // only continue if haven't found yet
	})
	if rki != nil {
		return rki, true
	}
	return nil, false
}

// HandleLayoutScrollEvents registers scrolling-related mouse events processed by
// Layout -- most subclasses of Layout will want these..
func (ly *Layout) HandleLayoutScrollEvents() {
	ly.On(events.Scroll, func(e events.Event) {
		// fmt.Println(ly, "scroll event", e)
		ly.ScrollDelta(e)
	})
	// HiPri to do it first so others can be in view etc -- does NOT consume event!
	// we.AddFunc(events.DNDMoveEvent, HiPri, func(recv, send ki.Ki, sig int64, d any) {
	// 	me := d.(*dnd.Event)
	// 	li := AsLayout(recv)
	// 	li.AutoScroll(me.Pos())
	// })
	// we.AddFunc(events.MouseMoveEvent, HiPri, func(recv, send ki.Ki, sig int64, d any) {
	// 	me := d.(events.Event)
	// 	li := AsLayout(recv)
	// 	if li.Sc.Type == ScMenu {
	// 		li.AutoScroll(me.Pos())
	// 	}
	// })
}

///////////////////////////////////////////////////////////
//    Stretch and Space -- dummy elements for layouts

// Stretch adds an infinitely stretchy element for spacing out layouts
// (max-size = -1) set the width / height property to determine how much it
// takes relative to other stretchy elements
type Stretch struct {
	WidgetBase
}

func (st *Stretch) OnInit() {
	st.Style(func(s *styles.Style) {
		s.Min.X.Ch(1)
		s.Min.Y.Em(1)
		s.Grow.Set(1, 1)
	})
}

func (st *Stretch) CopyFieldsFrom(frm any) {
	fr := frm.(*Stretch)
	st.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
}

// Space adds a fixed sized (1 ch x 1 em by default) blank space to a layout.
// Set width / height property to change.
type Space struct {
	WidgetBase
}

func (sp *Space) OnInit() {
	sp.Style(func(s *styles.Style) {
		s.Min.X.Ch(1)
		s.Min.Y.Em(1)
		s.Padding.Zero()
		s.Margin.Zero()
		s.MaxBorder.Width.Zero()
		s.Border.Width.Zero()
	})
}

func (sp *Space) CopyFieldsFrom(frm any) {
	fr := frm.(*Space)
	sp.WidgetBase.CopyFieldsFrom(&fr.WidgetBase)
}
