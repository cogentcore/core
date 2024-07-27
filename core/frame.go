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

	// StackTop, for a [styles.Stacked] frame, is the index of the node to use
	// as the top of the stack. Only the node at this index is rendered; if it is
	// not a valid index, nothing is rendered.
	StackTop int

	// LayoutStackTopOnly is whether to only layout the top widget
	// (specified by [Frame.StackTop]) for a [styles.Stacked] frame.
	// This is appropriate for widgets such as [Tabs], which do a full
	// redraw on stack changes, but not for widgets such as [Switch]es
	// which don't.
	LayoutStackTopOnly bool

	// layout contains implementation state info for doing layout
	layout layoutState

	// HasScroll is whether scrollbars exist for each dimension.
	HasScroll [2]bool `edit:"-" copier:"-" json:"-" xml:"-" set:"-"`

	// scrolls are the scroll bars, which are fully managed as needed.
	scrolls [2]*Slider

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
			if fr.Scene.Stage.closePopupAndBelow() {
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
				if fr.focusNextChild(false) {
					e.SetHandled()
				}
				return
			case keymap.MoveLeft:
				if fr.focusPreviousChild(false) {
					e.SetHandled()
				}
				return
			}
		}
		if fr.Styles.Direction == styles.Column || grid {
			switch kf {
			case keymap.MoveDown:
				if fr.focusNextChild(true) {
					e.SetHandled()
				}
				return
			case keymap.MoveUp:
				if fr.focusPreviousChild(true) {
					e.SetHandled()
				}
				return
			case keymap.PageDown:
				proc := false
				for st := 0; st < SystemSettings.LayoutPageSteps; st++ {
					if !fr.focusNextChild(true) {
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
					if !fr.focusPreviousChild(true) {
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
		fr.focusOnName(e)
	})
	fr.On(events.Scroll, func(e events.Event) {
		fr.scrollDelta(e)
	})
	// We treat slide events on frames as scroll events.
	fr.On(events.SlideMove, func(e events.Event) {
		// We must negate the delta for "natural" scrolling behavior.
		del := math32.Vector2FromPoint(e.PrevDelta()).MulScalar(-0.034)
		fr.scrollDelta(events.NewScroll(e.WindowPos(), del, e.Modifiers()))
	})
	fr.On(events.SlideStop, func(e events.Event) {
		// If we have enough velocity, we continue scrolling over the
		// next second in a goroutine while slowly decelerating for a
		// smoother experience.
		vel := math32.Vector2FromPoint(e.StartDelta()).DivScalar(1.5 * float32(e.SinceStart().Milliseconds())).Negate()
		if math32.Abs(vel.X) < 1 && math32.Abs(vel.Y) < 1 {
			return
		}
		go func() {
			i := 0
			tick := time.NewTicker(time.Second / 60)
			for range tick.C {
				fr.AsyncLock()
				fr.scrollDelta(events.NewScroll(e.WindowPos(), vel, e.Modifiers()))
				fr.AsyncUnlock()
				vel.SetMulScalar(0.95)
				i++
				if i > 120 {
					tick.Stop()
					break
				}
			}
		}()
	})
}

func (fr *Frame) Style() {
	fr.WidgetBase.Style()
	for d := math32.X; d <= math32.Y; d++ {
		if fr.HasScroll[d] && fr.scrolls[d] != nil {
			fr.scrolls[d].Style()
		}
	}
}

func (fr *Frame) Destroy() {
	for d := math32.X; d <= math32.Y; d++ {
		fr.deleteScroll(d)
	}
	fr.WidgetBase.Destroy()
}

// deleteScroll deletes scrollbar along given dimesion.
func (fr *Frame) deleteScroll(d math32.Dims) {
	if fr.scrolls[d] == nil {
		return
	}
	sb := fr.scrolls[d]
	sb.This.Destroy()
	fr.scrolls[d] = nil
}

func (fr *Frame) RenderChildren() {
	if fr.Styles.Display == styles.Stacked {
		wb := fr.StackTopWidget()
		if wb != nil {
			wb.This.(Widget).RenderWidget()
		}
		return
	}
	fr.ForWidgetChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		cw.RenderWidget()
		return tree.Continue
	})
}

func (fr *Frame) RenderWidget() {
	if fr.PushBounds() {
		fr.This.(Widget).Render()
		fr.renderParts()
		fr.RenderChildren()
		fr.RenderScrolls()
		fr.PopBounds()
	}
}

// childWithFocus returns a direct child of this layout that either is the
// current window focus item, or contains that focus item (along with its
// index) -- nil, -1 if none.
func (fr *Frame) childWithFocus() (Widget, int) {
	em := fr.Events()
	if em == nil {
		return nil, -1
	}
	var foc Widget
	focIndex := -1
	fr.ForWidgetChildren(func(i int, cw Widget, cwb *WidgetBase) bool {
		if cwb.ContainsFocus() {
			foc = cw
			focIndex = i
			return tree.Break
		}
		return tree.Continue
	})
	return foc, focIndex
}

// focusNextChild attempts to move the focus into the next layout child
// (with wraparound to start); returns true if successful.
// if updn is true, then for Grid layouts, it moves down to next row
// instead of just the sequentially next item.
func (fr *Frame) focusNextChild(updn bool) bool {
	sz := len(fr.Children)
	if sz <= 1 {
		return false
	}
	foc, idx := fr.childWithFocus()
	if foc == nil {
		// fmt.Println("no child foc")
		return false
	}
	em := fr.Events()
	if em == nil {
		return false
	}
	cur := em.focus
	nxti := idx + 1
	if fr.Styles.Display == styles.Grid && updn {
		nxti = idx + fr.Styles.Columns
	}
	did := false
	if nxti < sz {
		nx := fr.Child(nxti).(Widget)
		did = em.focusOnOrNext(nx)
	} else {
		nx := fr.Child(0).(Widget)
		did = em.focusOnOrNext(nx)
	}
	if !did || em.focus == cur {
		return false
	}
	return true
}

// focusPreviousChild attempts to move the focus into the previous layout child
// (with wraparound to end); returns true if successful.
// If updn is true, then for Grid layouts, it moves up to next row
// instead of just the sequentially next item.
func (fr *Frame) focusPreviousChild(updn bool) bool {
	sz := len(fr.Children)
	if sz <= 1 {
		return false
	}
	foc, idx := fr.childWithFocus()
	if foc == nil {
		return false
	}
	em := fr.Events()
	if em == nil {
		return false
	}
	cur := em.focus
	nxti := idx - 1
	if fr.Styles.Display == styles.Grid && updn {
		nxti = idx - fr.Styles.Columns
	}
	did := false
	if nxti >= 0 {
		did = em.focusOnOrPrev(fr.Child(nxti).(Widget))
	} else {
		did = em.focusOnOrPrev(fr.Child(sz - 1).(Widget))
	}
	if !did || em.focus == cur {
		return false
	}
	return true
}

// focusOnName processes key events to look for an element starting with given name
func (fr *Frame) focusOnName(e events.Event) bool {
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
	focel := childByLabelCanFocus(fr, fr.focusName, fr.focusNameLast)
	if focel != nil {
		em := fr.Events()
		if em != nil {
			em.setFocusEvent(focel.(Widget)) // this will also scroll by default!
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

// childByLabelCanFocus uses breadth-first search to find
// the first focusable element within the layout whose Label (using
// [ToLabel]) matches the given name using [complete.IsSeedMatching].
// If after is non-nil, it only finds after that element.
func childByLabelCanFocus(fr *Frame, name string, after tree.Node) tree.Node {
	gotAfter := false
	completions := []complete.Completion{}
	fr.WalkDownBreadth(func(n tree.Node) bool {
		if n == fr.This { // skip us
			return tree.Continue
		}
		wb := AsWidget(n)
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
