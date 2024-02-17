// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"log/slog"
	"time"
	"unicode"

	"cogentcore.org/core/abilities"
	"cogentcore.org/core/enums"
	"cogentcore.org/core/events"
	"cogentcore.org/core/keyfun"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/mat32"
	"cogentcore.org/core/pi/complete"
	"cogentcore.org/core/styles"
)

var (
	// LayoutPrefMaxRows is maximum number of rows to use in a grid layout
	// when computing the preferred size (ScPrefSizing)
	LayoutPrefMaxRows = 20

	// LayoutPrefMaxCols is maximum number of columns to use in a grid layout
	// when computing the preferred size (ScPrefSizing)
	LayoutPrefMaxCols = 20

	// AutoScrollRate determines the rate of auto-scrolling of layouts
	AutoScrollRate = float32(1.0)
)

// Layoutlags has bool flags for Layout
type LayoutFlags WidgetFlags //enums:bitflag -trim-prefix Layout

const (
	// for stacked layout, only layout the top widget.
	// this is appropriate for e.g., tab layout, which does a full
	// redraw on stack changes, but not for e.g., check boxes which don't
	LayoutStackTopOnly = LayoutFlags(WidgetFlagsN) + iota

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
	LayImpl LayImplState `edit:"-" copier:"-" json:"-" xml:"-" set:"-"`

	// whether scrollbar is used for given dim
	HasScroll [2]bool `edit:"-" copier:"-" json:"-" xml:"-" set:"-"`

	// scroll bars -- we fully manage them as needed
	Scrolls [2]*Slider `edit:"-" copier:"-" json:"-" xml:"-" set:"-"`

	// accumulated name to search for when keys are typed
	FocusName string `edit:"-" copier:"-" json:"-" xml:"-" set:"-"`

	// time of last focus name event -- for timeout
	FocusNameTime time.Time `edit:"-" copier:"-" json:"-" xml:"-" set:"-"`

	// last element focused on -- used as a starting point if name is the same
	FocusNameLast ki.Ki `edit:"-" copier:"-" json:"-" xml:"-" set:"-"`
}

func (l *Layout) FlagType() enums.BitFlagSetter {
	return (*LayoutFlags)(&l.Flags)
}

func (l *Layout) OnInit() {
	l.WidgetBase.OnInit()
	l.SetStyles()
	l.HandleEvents()
}

func (l *Layout) ConfigWidget() {
	for d := mat32.X; d <= mat32.Y; d++ {
		if l.HasScroll[d] && l.Scrolls[d] != nil {
			l.Scrolls[d].ApplyStyle()
		}
	}
}

func (l *Layout) SetStyles() {
	l.Style(func(s *styles.Style) {
		// we never want borders on layouts
		s.MaxBorder = styles.Border{}
		switch {
		case s.Display == styles.Flex:
			if s.Wrap {
				s.Grow.Set(1, 0)
			} else {
				s.Grow.SetDim(s.Direction.Dim(), 1)
				s.Grow.SetDim(s.Direction.Dim().Other(), 0)
			}
		case s.Display == styles.Stacked:
			s.Grow.Set(1, 1)
		case s.Display == styles.Grid:
			s.Grow.Set(1, 1)
		}
	})
	l.StyleFinal(func(s *styles.Style) {
		s.SetAbilities(s.Overflow.X == styles.OverflowAuto || s.Overflow.Y == styles.OverflowAuto, abilities.Scrollable, abilities.Slideable)
	})
}

func (l *Layout) Destroy() {
	for d := mat32.X; d <= mat32.Y; d++ {
		l.DeleteScroll(d)
	}
	l.WidgetBase.Destroy()
}

// DeleteScroll deletes scrollbar along given dimesion.
func (l *Layout) DeleteScroll(d mat32.Dims) {
	if l.Scrolls[d] == nil {
		return
	}
	sb := l.Scrolls[d]
	sb.This().Destroy()
	l.Scrolls[d] = nil
}

func (l *Layout) RenderChildren() {
	if l.Styles.Display == styles.Stacked {
		kwi, _ := l.StackTopWidget()
		if kwi != nil {
			kwi.Render()
		}
		return
	}
	l.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		kwi.Render()
		return ki.Continue
	})
}

func (l *Layout) Render() {
	if l.PushBounds() {
		l.RenderChildren()
		l.PopBounds()
		l.RenderScrolls()
	}
}

// ChildWithFocus returns a direct child of this layout that either is the
// current window focus item, or contains that focus item (along with its
// index) -- nil, -1 if none.
func (l *Layout) ChildWithFocus() (Widget, int) {
	em := l.EventMgr()
	if em == nil {
		return nil, -1
	}
	var foc Widget
	focIdx := -1
	l.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		if kwb.ContainsFocus() {
			foc = kwi
			focIdx = i
			return ki.Break
		}
		return ki.Continue
	})
	return foc, focIdx
}

// FocusNextChild attempts to move the focus into the next layout child
// (with wraparound to start) -- returns true if successful.
// if updn is true, then for Grid layouts, it moves down to next row
// instead of just the sequentially next item.
func (l *Layout) FocusNextChild(updn bool) bool {
	sz := len(l.Kids)
	if sz <= 1 {
		return false
	}
	foc, idx := l.ChildWithFocus()
	if foc == nil {
		fmt.Println("no child foc")
		return false
	}
	em := l.EventMgr()
	if em == nil {
		return false
	}
	cur := em.Focus
	nxti := idx + 1
	if l.Styles.Display == styles.Grid && updn {
		nxti = idx + l.Styles.Columns
	}
	did := false
	if nxti < sz {
		nx := l.Child(nxti).(Widget)
		did = em.FocusOnOrNext(nx)
	} else {
		nx := l.Child(0).(Widget)
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
func (l *Layout) FocusPrevChild(updn bool) bool {
	sz := len(l.Kids)
	if sz <= 1 {
		return false
	}
	foc, idx := l.ChildWithFocus()
	if foc == nil {
		return false
	}
	em := l.EventMgr()
	if em == nil {
		return false
	}
	cur := em.Focus
	nxti := idx - 1
	if l.Styles.Display == styles.Grid && updn {
		nxti = idx - l.Styles.Columns
	}
	did := false
	if nxti >= 0 {
		did = em.FocusOnOrPrev(l.Child(nxti).(Widget))
	} else {
		did = em.FocusOnOrPrev(l.Child(sz - 1).(Widget))
	}
	if !did || em.Focus == cur {
		return false
	}
	return true
}

// ClosePopup closes the parent Stage as a PopupStage.
// Returns false if not a popup.
func (l *Layout) ClosePopup() bool {
	ps := l.Scene.Stage
	if ps == nil {
		return false
	}
	ps.ClosePopup()
	return true
}

// ClosePopupAndBelow closes the parent Stage as a PopupStage,
// and any other popups immediately below it of the same type.
// Returns false if not a popup.
func (l *Layout) ClosePopupAndBelow() bool {
	ps := l.Scene.Stage
	if ps == nil {
		return false
	}
	ps.ClosePopupAndBelow()
	return true
}

func (l *Layout) HandleEvents() {
	l.WidgetBase.HandleEvents()
	l.HandleKeys()
	l.On(events.Scroll, func(e events.Event) {
		l.ScrollDelta(e)
	})
	// we treat slide events on layouts as scroll events
	// we must reverse the delta for "natural" scrolling behavior
	l.On(events.SlideMove, func(e events.Event) {
		del := mat32.V2FromPoint(e.PrevDelta()).MulScalar(-0.1)
		l.ScrollDelta(events.NewScroll(e.WindowPos(), del, e.Modifiers()))
	})
}

// HandleKeys handles all key events for navigating focus within a Layout.
// Typically this is done by the parent Scene level layout, but can be
// done by default if FocusWithinable Ability is set.
func (l *Layout) HandleKeys() {
	l.OnFinal(events.KeyChord, func(e events.Event) {
		if l.Is(LayoutNoKeys) {
			return
		}
		kf := keyfun.Of(e.KeyChord())
		if DebugSettings.KeyEventTrace {
			slog.Info("Layout KeyInput", "widget", l, "keyfun", kf)
		}
		if kf == keyfun.Abort {
			if l.ClosePopupAndBelow() {
				e.SetHandled()
			}
			return
		}
		em := l.EventMgr()
		if em == nil {
			return
		}
		grid := l.Styles.Display == styles.Grid
		if l.Styles.Direction == styles.Row || grid {
			switch kf {
			case keyfun.MoveRight:
				if l.FocusNextChild(false) {
					e.SetHandled()
				}
				return
			case keyfun.MoveLeft:
				if l.FocusPrevChild(false) {
					e.SetHandled()
				}
				return
			}
		}
		if l.Styles.Direction == styles.Column || grid {
			switch kf {
			case keyfun.MoveDown:
				if l.FocusNextChild(true) {
					e.SetHandled()
				}
				return
			case keyfun.MoveUp:
				if l.FocusPrevChild(true) {
					e.SetHandled()
				}
				return
			case keyfun.PageDown:
				proc := false
				for st := 0; st < SystemSettings.LayoutPageSteps; st++ {
					if !l.FocusNextChild(true) {
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
				for st := 0; st < SystemSettings.LayoutPageSteps; st++ {
					if !l.FocusPrevChild(true) {
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
		l.FocusOnName(e)
	})
}

// FocusOnName processes key events to look for an element starting with given name
func (l *Layout) FocusOnName(e events.Event) bool {
	kf := keyfun.Of(e.KeyChord())
	if DebugSettings.KeyEventTrace {
		slog.Info("Layout FocusOnName", "widget", l, "keyfun", kf)
	}
	delay := e.Time().Sub(l.FocusNameTime)
	l.FocusNameTime = e.Time()
	if kf == keyfun.FocusNext { // tab means go to next match -- don't worry about time
		if l.FocusName == "" || delay > SystemSettings.LayoutFocusNameTabTime {
			l.FocusName = ""
			l.FocusNameLast = nil
			return false
		}
	} else {
		if delay > SystemSettings.LayoutFocusNameTimeout {
			l.FocusName = ""
		}
		if !unicode.IsPrint(e.KeyRune()) || e.Modifiers() != 0 {
			return false
		}
		sr := string(e.KeyRune())
		if l.FocusName == sr {
			// re-search same letter
		} else {
			l.FocusName += sr
			l.FocusNameLast = nil // only use last if tabbing
		}
	}
	// e.SetHandled()
	// fmt.Printf("searching for: %v  last: %v\n", l.FocusName, l.FocusNameLast)
	focel, found := ChildByLabelStartsCanFocus(l, l.FocusName, l.FocusNameLast)
	if found {
		em := l.EventMgr()
		if em != nil {
			em.SetFocusEvent(focel.(Widget)) // this will also scroll by default!
		}
		l.FocusNameLast = focel
		return true
	} else {
		if l.FocusNameLast == nil {
			l.FocusName = "" // nothing being found
		}
		l.FocusNameLast = nil // start over
	}
	return false
}

// ChildByLabelStartsCanFocus uses breadth-first search to find
// the first focusable element within the layout whose Label (using
// [ToLabel]) matches the given name using [complete.IsSeedMatching].
// If after is non-nil, it only finds after that element.
func ChildByLabelStartsCanFocus(ly *Layout, name string, after ki.Ki) (ki.Ki, bool) {
	gotAfter := false
	completions := []complete.Completion{}
	ly.WalkBreadth(func(k ki.Ki) bool {
		if k == ly.This() { // skip us
			return ki.Continue
		}
		_, ni := AsWidget(k)
		if ni == nil || !ni.CanFocus() { // don't go any further
			return ki.Continue
		}
		if after != nil && !gotAfter {
			if k == after {
				gotAfter = true
			}
			return ki.Continue // skip to next
		}
		completions = append(completions, complete.Completion{
			Text: ToLabel(k),
			Desc: k.PathFrom(ly),
		})
		return ki.Continue
	})
	matches := complete.MatchSeedCompletion(completions, name)
	if len(matches) > 0 {
		if res := ly.FindPath(matches[0].Desc); res != nil {
			return res, true
		}
	}
	return nil, false
}

///////////////////////////////////////////////////////////
//    Stretch and Space: spacing elements for layouts

// Stretch adds a stretchy element that grows to fill all
// available space. You can set [styles.Style.Grow] to change
// how much it grows relative to other growing elements.
type Stretch struct {
	WidgetBase
}

func (st *Stretch) OnInit() {
	st.WidgetBase.SetStyles()
	// note: not getting base events
	st.Style(func(s *styles.Style) {
		s.Min.X.Ch(1)
		s.Min.Y.Em(1)
		s.Grow.Set(1, 1)
	})
}

// Space is a fixed size blank space, with
// a default width of 1ch and a height of 1em.
// You can set [styles.Style.Min] to change its size.
type Space struct {
	WidgetBase
}

func (sp *Space) OnInit() {
	sp.WidgetBase.SetStyles()
	// note: not getting base events
	sp.Style(func(s *styles.Style) {
		s.Min.X.Ch(1)
		s.Min.Y.Em(1)
		s.Padding.Zero()
		s.Margin.Zero()
		s.MaxBorder.Width.Zero()
		s.Border.Width.Zero()
	})
}
