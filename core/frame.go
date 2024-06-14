// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"log/slog"
	"time"
	"unicode"

	"cogentcore.org/core/base/labels"
	"cogentcore.org/core/events"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/math32"
	"cogentcore.org/core/parse/complete"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/abilities"
	"cogentcore.org/core/tree"
)

// Frame is the primary node type responsible for organizing the sizes
// and positions of child widgets. It also renders the standard box model.
// All collections of widgets should generally be contained within a [Frame];
// otherwise, the parent widget must take over responsibility for positioning.
// Frames automatically can add scrollbars depending on the [styles.Style.Overflow].
//
// For a [styles.Grid] frame, the [styles.Style.Columns] property should
// generally be set to the desired number of columns, from which the number of rows
// is computed; otherwise, it uses the square root of number of
// elements.
type Frame struct {
	WidgetBase

	// StackTop, for a [styles.Stacked] frame, is the index of the node to use as the top of the stack.
	// Only the node at this index is rendered; if not a valid index, nothing is rendered.
	StackTop int

	// LayoutStackTopOnly is whether to only layout the top widget (specified by [Frame.StackTop])
	// for a [styles.Stacked] frame. This is appropriate for e.g., [Tabs], which do a full
	// redraw on stack changes, but not for e.g., [Switch]es which don't.
	LayoutStackTopOnly bool

	// Layout contains implementation state info for doing layout
	Layout LayoutState `edit:"-" copier:"-" json:"-" xml:"-" set:"-"`

	// HasScroll is whether scrollbar is used for given dim.
	HasScroll [2]bool `edit:"-" copier:"-" json:"-" xml:"-" set:"-"`

	// Scrolls are the scroll bars, which are fully managed as needed.
	Scrolls [2]*Slider `edit:"-" copier:"-" json:"-" xml:"-" set:"-"`

	// accumulated name to search for when keys are typed
	focusName string

	// time of last focus name event; for timeout
	focusNameTime time.Time

	// last element focused on; used as a starting point if name is the same
	focusNameLast tree.Node
}

func (fr *Frame) Init() {
	fr.WidgetBase.Init()
	fr.FinalStyler(func(s *styles.Style) {
		// we only enable, not disable, since some other widget like Slider may want to enable
		if s.Overflow.X == styles.OverflowAuto || s.Overflow.Y == styles.OverflowAuto {
			s.SetAbilities(true, abilities.Scrollable, abilities.Slideable)
		}
	})
	fr.OnFinal(events.KeyChord, func(e events.Event) {
		kf := keymap.Of(e.KeyChord())
		if DebugSettings.KeyEventTrace {
			slog.Info("Layout KeyInput", "widget", fr, "keyFunction", kf)
		}
		if kf == keymap.Abort {
			if fr.Scene.Stage.ClosePopupAndBelow() {
				e.SetHandled()
			}
			return
		}
		em := fr.Events()
		if em == nil {
			return
		}
		grid := fr.Styles.Display == styles.Grid
		if fr.Styles.Direction == styles.Row || grid {
			switch kf {
			case keymap.MoveRight:
				if fr.FocusNextChild(false) {
					e.SetHandled()
				}
				return
			case keymap.MoveLeft:
				if fr.FocusPreviousChild(false) {
					e.SetHandled()
				}
				return
			}
		}
		if fr.Styles.Direction == styles.Column || grid {
			switch kf {
			case keymap.MoveDown:
				if fr.FocusNextChild(true) {
					e.SetHandled()
				}
				return
			case keymap.MoveUp:
				if fr.FocusPreviousChild(true) {
					e.SetHandled()
				}
				return
			case keymap.PageDown:
				proc := false
				for st := 0; st < SystemSettings.LayoutPageSteps; st++ {
					if !fr.FocusNextChild(true) {
						break
					}
					proc = true
				}
				if proc {
					e.SetHandled()
				}
				return
			case keymap.PageUp:
				proc := false
				for st := 0; st < SystemSettings.LayoutPageSteps; st++ {
					if !fr.FocusPreviousChild(true) {
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
		fr.FocusOnName(e)
	})
	fr.On(events.Scroll, func(e events.Event) {
		fr.ScrollDelta(e)
	})
	// we treat slide events on layouts as scroll events
	// we must reverse the delta for "natural" scrolling behavior
	fr.On(events.SlideMove, func(e events.Event) {
		del := math32.Vector2FromPoint(e.PrevDelta()).MulScalar(-0.1)
		fr.ScrollDelta(events.NewScroll(e.WindowPos(), del, e.Modifiers()))
	})
}

func (fr *Frame) Style() {
	fr.WidgetBase.Style()
	for d := math32.X; d <= math32.Y; d++ {
		if fr.HasScroll[d] && fr.Scrolls[d] != nil {
			fr.Scrolls[d].Style()
		}
	}
}

func (fr *Frame) Destroy() {
	for d := math32.X; d <= math32.Y; d++ {
		fr.DeleteScroll(d)
	}
	fr.WidgetBase.Destroy()
}

// DeleteScroll deletes scrollbar along given dimesion.
func (fr *Frame) DeleteScroll(d math32.Dims) {
	if fr.Scrolls[d] == nil {
		return
	}
	sb := fr.Scrolls[d]
	sb.This.Destroy()
	fr.Scrolls[d] = nil
}

func (fr *Frame) RenderChildren() {
	if fr.Styles.Display == styles.Stacked {
		kwi, _ := fr.StackTopWidget()
		if kwi != nil {
			kwi.RenderWidget()
		}
		return
	}
	fr.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		kwi.RenderWidget()
		return tree.Continue
	})
}

func (fr *Frame) RenderWidget() {
	if fr.PushBounds() {
		fr.This.(Widget).Render()
		fr.RenderParts()
		fr.RenderChildren()
		fr.RenderScrolls()
		fr.PopBounds()
	}
}

// ChildWithFocus returns a direct child of this layout that either is the
// current window focus item, or contains that focus item (along with its
// index) -- nil, -1 if none.
func (fr *Frame) ChildWithFocus() (Widget, int) {
	em := fr.Events()
	if em == nil {
		return nil, -1
	}
	var foc Widget
	focIndex := -1
	fr.WidgetKidsIter(func(i int, kwi Widget, kwb *WidgetBase) bool {
		if kwb.ContainsFocus() {
			foc = kwi
			focIndex = i
			return tree.Break
		}
		return tree.Continue
	})
	return foc, focIndex
}

// FocusNextChild attempts to move the focus into the next layout child
// (with wraparound to start); returns true if successful.
// if updn is true, then for Grid layouts, it moves down to next row
// instead of just the sequentially next item.
func (fr *Frame) FocusNextChild(updn bool) bool {
	sz := len(fr.Children)
	if sz <= 1 {
		return false
	}
	foc, idx := fr.ChildWithFocus()
	if foc == nil {
		// fmt.Println("no child foc")
		return false
	}
	em := fr.Events()
	if em == nil {
		return false
	}
	cur := em.Focus
	nxti := idx + 1
	if fr.Styles.Display == styles.Grid && updn {
		nxti = idx + fr.Styles.Columns
	}
	did := false
	if nxti < sz {
		nx := fr.Child(nxti).(Widget)
		did = em.FocusOnOrNext(nx)
	} else {
		nx := fr.Child(0).(Widget)
		did = em.FocusOnOrNext(nx)
	}
	if !did || em.Focus == cur {
		return false
	}
	return true
}

// FocusPreviousChild attempts to move the focus into the previous layout child
// (with wraparound to end); returns true if successful.
// If updn is true, then for Grid layouts, it moves up to next row
// instead of just the sequentially next item.
func (fr *Frame) FocusPreviousChild(updn bool) bool {
	sz := len(fr.Children)
	if sz <= 1 {
		return false
	}
	foc, idx := fr.ChildWithFocus()
	if foc == nil {
		return false
	}
	em := fr.Events()
	if em == nil {
		return false
	}
	cur := em.Focus
	nxti := idx - 1
	if fr.Styles.Display == styles.Grid && updn {
		nxti = idx - fr.Styles.Columns
	}
	did := false
	if nxti >= 0 {
		did = em.FocusOnOrPrev(fr.Child(nxti).(Widget))
	} else {
		did = em.FocusOnOrPrev(fr.Child(sz - 1).(Widget))
	}
	if !did || em.Focus == cur {
		return false
	}
	return true
}

// FocusOnName processes key events to look for an element starting with given name
func (fr *Frame) FocusOnName(e events.Event) bool {
	kf := keymap.Of(e.KeyChord())
	if DebugSettings.KeyEventTrace {
		slog.Info("Layout FocusOnName", "widget", fr, "keyFunction", kf)
	}
	delay := e.Time().Sub(fr.focusNameTime)
	fr.focusNameTime = e.Time()
	if kf == keymap.FocusNext { // tab means go to next match -- don't worry about time
		if fr.focusName == "" || delay > SystemSettings.LayoutFocusNameTabTime {
			fr.focusName = ""
			fr.focusNameLast = nil
			return false
		}
	} else {
		if delay > SystemSettings.LayoutFocusNameTimeout {
			fr.focusName = ""
		}
		if !unicode.IsPrint(e.KeyRune()) || e.Modifiers() != 0 {
			return false
		}
		sr := string(e.KeyRune())
		if fr.focusName == sr {
			// re-search same letter
		} else {
			fr.focusName += sr
			fr.focusNameLast = nil // only use last if tabbing
		}
	}
	// e.SetHandled()
	// fmt.Printf("searching for: %v  last: %v\n", ly.FocusName, ly.FocusNameLast)
	focel := ChildByLabelCanFocus(fr, fr.focusName, fr.focusNameLast)
	if focel != nil {
		em := fr.Events()
		if em != nil {
			em.SetFocusEvent(focel.(Widget)) // this will also scroll by default!
		}
		fr.focusNameLast = focel
		return true
	} else {
		if fr.focusNameLast == nil {
			fr.focusName = "" // nothing being found
		}
		fr.focusNameLast = nil // start over
	}
	return false
}

// ChildByLabelCanFocus uses breadth-first search to find
// the first focusable element within the layout whose Label (using
// [ToLabel]) matches the given name using [complete.IsSeedMatching].
// If after is non-nil, it only finds after that element.
func ChildByLabelCanFocus(fr *Frame, name string, after tree.Node) tree.Node {
	gotAfter := false
	completions := []complete.Completion{}
	fr.WalkDownBreadth(func(n tree.Node) bool {
		if n == fr.This { // skip us
			return tree.Continue
		}
		_, wb := AsWidget(n)
		if wb == nil || !wb.CanFocus() { // don't go any further
			return tree.Continue
		}
		if after != nil && !gotAfter {
			if n == after {
				gotAfter = true
			}
			return tree.Continue // skip to next
		}
		completions = append(completions, complete.Completion{
			Text: labels.ToLabel(n),
			Desc: n.AsTree().PathFrom(fr),
		})
		return tree.Continue
	})
	matches := complete.MatchSeedCompletion(completions, name)
	if len(matches) > 0 {
		return fr.FindPath(matches[0].Desc)
	}
	return nil
}

// Stretch and Space: spacing elements

// Stretch adds a stretchy element that grows to fill all
// available space. You can set [styles.Style.Grow] to change
// how much it grows relative to other growing elements.
// It does not render anything.
type Stretch struct {
	WidgetBase
}

func (st *Stretch) Init() {
	st.WidgetBase.Init()
	st.Styler(func(s *styles.Style) {
		s.RenderBox = false
		s.Min.X.Ch(1)
		s.Min.Y.Em(1)
		s.Grow.Set(1, 1)
	})
}

// Space is a fixed size blank space, with
// a default width of 1ch and a height of 1em.
// You can set [styles.Style.Min] to change its size.
// It does not render anything.
type Space struct {
	WidgetBase
}

func (sp *Space) Init() {
	sp.WidgetBase.Init()
	sp.Styler(func(s *styles.Style) {
		s.RenderBox = false
		s.Min.X.Ch(1)
		s.Min.Y.Em(1)
		s.Padding.Zero()
		s.Margin.Zero()
		s.MaxBorder.Width.Zero()
		s.Border.Width.Zero()
	})
}
