// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package events

import (
	"image"
	"testing"
	"time"

	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/events/key"
	"cogentcore.org/core/math32"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mods returns the modifiers with the given flags set.
func mods(flags ...key.Modifiers) key.Modifiers {
	var m key.Modifiers
	for _, f := range flags {
		m.SetFlag(true, f)
	}
	return m
}

func TestBasePositions(t *testing.T) {
	ev := NewMouseDrag(Left, image.Pt(30, 40), image.Pt(25, 35), image.Pt(10, 20), 0)
	ev.Init()
	assert.True(t, ev.HasPos())
	assert.Equal(t, image.Pt(30, 40), ev.WindowPos())
	assert.Equal(t, image.Pt(10, 20), ev.WindowStartPos())
	assert.Equal(t, image.Pt(25, 35), ev.WindowPrevPos())
	assert.Equal(t, image.Pt(30, 40), ev.Pos(), "no local offset after Init")

	ev.SetLocalOff(image.Pt(5, 10))
	assert.Equal(t, image.Pt(5, 10), ev.LocalOff())
	assert.Equal(t, image.Pt(25, 30), ev.Pos())
	assert.Equal(t, image.Pt(5, 10), ev.StartPos())
	assert.Equal(t, image.Pt(20, 25), ev.PrevPos())
	assert.Equal(t, image.Pt(20, 20), ev.StartDelta())
	assert.Equal(t, image.Pt(5, 5), ev.PrevDelta())
	assert.Equal(t, image.Pt(30, 40), ev.WindowPos(), "the window position is unchanged")
}

func TestBaseTimes(t *testing.T) {
	ev := NewMouse(MouseUp, Left, image.Point{}, 0)
	start := time.Now()
	ev.StTime.SetTime(start.Add(-2 * time.Second))
	ev.PrvTime.SetTime(start.Add(-time.Second))
	ev.SetTime()
	assert.False(t, ev.Time().Before(start))
	assert.GreaterOrEqual(t, ev.SinceStart(), 2*time.Second)
	assert.GreaterOrEqual(t, ev.SincePrev(), time.Second)
	assert.Less(t, ev.SincePrev(), ev.SinceStart())
	assert.Equal(t, ev.StTime.Time(), ev.StartTime())
	assert.Equal(t, ev.PrvTime.Time(), ev.PrevTime())
}

func TestBaseFlagsAndClone(t *testing.T) {
	ev := NewKey(KeyChord, 'a', key.CodeA, mods(key.Shift))
	assert.True(t, ev.IsUnique())
	assert.True(t, ev.NeedsFocus())
	assert.False(t, ev.HasPos())
	assert.True(t, ev.OnWinFocus())
	assert.Equal(t, KeyChord, ev.Type())
	assert.Same(t, &ev.Base, ev.AsBase())
	assert.Equal(t, 'a', ev.KeyRune())
	assert.Equal(t, key.CodeA, ev.KeyCode())
	assert.Equal(t, key.Chord("Shift+A"), ev.KeyChord())
	assert.Equal(t, ExtendContinuous, ev.SelectMode())
	assert.Equal(t, mods(key.Shift), ev.Modifiers())

	assert.False(t, ev.IsHandled())
	ev.SetHandled()
	assert.True(t, ev.IsHandled())
	cl := ev.Clone()
	assert.False(t, cl.IsHandled(), "a clone is not handled")
	assert.Equal(t, ev.Base.Rune, cl.KeyRune())
	ev.ClearHandled()
	assert.False(t, ev.IsHandled())

	ev.SetHandled()
	nw := ev.NewFromClone(KeyUp)
	assert.Equal(t, KeyUp, nw.Type())
	assert.False(t, nw.IsHandled())
	assert.Equal(t, KeyChord, ev.Type(), "the original is unchanged")

	ev.SetModifiers(key.Control)
	assert.True(t, ev.HasAllModifiers(key.Shift, key.Control))

	var b Base
	b.Typ = Click
	assert.False(t, b.NeedsFocus())
	assert.False(t, b.HasPos())
	assert.True(t, b.IsSame(NewMouse(Click, Left, image.Point{}, 0)))
	assert.False(t, b.IsSame(NewMouse(MouseUp, Left, image.Point{}, 0)))
	assert.Contains(t, b.String(), "Click")
	assert.Equal(t, NoButton, b.MouseButton())
}

func TestSelectModeBits(t *testing.T) {
	assert.Equal(t, SelectOne, SelectModeBits(0))
	assert.Equal(t, ExtendContinuous, SelectModeBits(mods(key.Shift)))
	assert.Equal(t, ExtendOne, SelectModeBits(mods(key.Meta)))
	assert.Equal(t, ExtendContinuous, SelectModeBits(mods(key.Shift, key.Meta)), "shift wins")
	assert.Equal(t, SelectOne, SelectModeBits(mods(key.Control)))
}

func TestTypesCategories(t *testing.T) {
	tests := []struct {
		typ                                    Types
		key, mouse, touch, drag, slide, window bool
	}{
		{KeyDown, true, false, false, false, false, false},
		{KeyChord, true, false, false, false, false, false},
		{MouseDown, false, true, false, false, false, false},
		{Click, false, true, false, false, false, false},
		{TouchStart, false, false, true, false, false, false},
		{DragStart, false, false, false, true, false, false},
		{DragLeave, false, false, false, true, false, false},
		{SlideMove, false, false, false, false, true, false},
		{Window, false, false, false, false, false, true},
		{WindowPaint, false, false, false, false, false, true},
		{Custom, false, false, false, false, false, false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.key, tt.typ.IsKey(), "%v key", tt.typ)
		assert.Equal(t, tt.mouse, tt.typ.IsMouse(), "%v mouse", tt.typ)
		assert.Equal(t, tt.touch, tt.typ.IsTouch(), "%v touch", tt.typ)
		assert.Equal(t, tt.drag, tt.typ.IsDrag(), "%v drag", tt.typ)
		assert.Equal(t, tt.slide, tt.typ.IsSlide(), "%v slide", tt.typ)
		assert.Equal(t, tt.window, tt.typ.IsWindow(), "%v window", tt.typ)
	}
}

func TestEventStrings(t *testing.T) {
	scroll := NewScroll(image.Pt(1, 2), math32.Vec2(0, 3), 0)
	tests := []struct {
		ev   Event
		want []string
	}{
		{NewKey(KeyChord, 'x', key.CodeX, mods(key.Control)), []string{"KeyChord", "Control+X"}},
		{NewKey(KeyDown, 0, key.CodeEscape, 0), []string{"KeyDown", "Escape"}},
		{NewMouse(MouseDown, Right, image.Pt(3, 4), 0), []string{"MouseDown", "Right", "(3,4)"}},
		{scroll, []string{"Scroll", "(0, 3)"}},
		{NewTouch(TouchMove, 7, image.Pt(5, 6)), []string{"TouchMove", "Sequence: 7"}},
		{NewWindow(WinClose), []string{"Window", "Close"}},
		{NewExternalDrop(Drop, Left, image.Pt(8, 9), 0, nil), []string{"Drop", "(8,9)"}},
		{NewOSEvent(OSOpenFiles), []string{"OSOpenFiles"}},
		{NewTextComposition("ka", 0, 2), []string{"Composition", `"ka"`, "End: 2"}},
		{&CustomEvent{Base: Base{Typ: Custom, Data: 42}}, []string{"Custom", "42"}},
	}
	for _, tt := range tests {
		s := tt.ev.String()
		for _, w := range tt.want {
			assert.Contains(t, s, w)
		}
	}
}

func TestEventConstructors(t *testing.T) {
	tc := NewTouch(TouchStart, 3, image.Pt(1, 1))
	assert.True(t, tc.HasPos())
	assert.True(t, tc.IsUnique())
	assert.Equal(t, Sequence(3), tc.Sequence)

	mg := NewMagnify(1.5, image.Pt(2, 2))
	assert.Equal(t, Magnify, mg.Type())
	assert.Equal(t, float32(1.5), mg.ScaleFactor)

	wr := NewWindowResize()
	assert.Equal(t, WindowResize, wr.Type())
	assert.False(t, wr.HasPos())
	assert.False(t, wr.IsUnique(), "resizes can be compressed")
	assert.Equal(t, WindowPaint, NewWindowPaint().Type())
	assert.True(t, NewWindow(WinFocus).IsUnique())

	of := NewOSFiles(OSOpenFiles, []string{"a.txt"})
	assert.Equal(t, []string{"a.txt"}, of.Files)

	co := NewTextComposition("abc", 1, 2)
	assert.False(t, co.HasPos())
	assert.True(t, co.NeedsFocus())

	ce := &CustomEvent{}
	assert.False(t, ce.HasPos())
	ce.PosAvail = true
	assert.True(t, ce.HasPos())

	md := NewMouseMove(Left, image.Pt(5, 5), image.Pt(4, 4), 0)
	assert.False(t, md.IsUnique(), "moves can be compressed")
	assert.Equal(t, image.Pt(4, 4), md.Prev)
}

func TestNewDragDrop(t *testing.T) {
	m := NewMouseDrag(Left, image.Pt(5, 5), image.Pt(4, 4), image.Pt(1, 1), mods(key.Alt))
	m.SetHandled()
	dd := NewDragDrop(Drop, m)
	assert.Equal(t, Drop, dd.Type())
	assert.False(t, dd.IsHandled())
	assert.True(t, dd.HasPos())
	assert.Equal(t, DropLink, dd.DropMod, "alt links by default")
	assert.Equal(t, image.Pt(5, 5), dd.Where)

	ed := NewExternalDrop(Drop, Left, image.Pt(1, 2), mods(key.Shift), "data")
	assert.Equal(t, "data", ed.Data)
	ed.DefaultMod()
	assert.Equal(t, DropMove, ed.DropMod)
}

func TestListeners(t *testing.T) {
	var ls Listeners
	assert.False(t, ls.HandlesEventType(Click), "nil listeners")
	var order []int
	ls.Add(Click, func(e Event) { order = append(order, 1) })
	ls.Add(Click, func(e Event) { order = append(order, 2) })
	ls.Add(Click, func(e Event) { order = append(order, 3) })
	assert.True(t, ls.HandlesEventType(Click))
	assert.False(t, ls.HandlesEventType(KeyDown))

	ls.Call(NewMouse(Click, Left, image.Point{}, 0))
	assert.Equal(t, []int{3, 2, 1}, order, "the last added is called first")

	order = nil
	ls.Add(Click, func(e Event) {
		order = append(order, 4)
		e.SetHandled()
	})
	ls.Call(NewMouse(Click, Left, image.Point{}, 0))
	assert.Equal(t, []int{4}, order, "a handled event stops the calls")

	order = nil
	ev := NewMouse(Click, Left, image.Point{}, 0)
	ev.SetHandled()
	ls.Call(ev)
	assert.Empty(t, order, "a handled event is not passed to any")

	order = nil
	n := 0
	ls.Call(NewMouse(Click, Left, image.Point{}, 0), func() bool { n++; return n < 1 })
	assert.Empty(t, order, "shouldContinue stops the calls")

	ls.Init()
	assert.Len(t, ls[Click], 4, "Init keeps existing listeners")
}

func TestListenersCopyFromExtra(t *testing.T) {
	var calls []string
	var fr Listeners
	fr.Add(Click, func(e Event) { calls = append(calls, "fr1") })
	fr.Add(Click, func(e Event) { calls = append(calls, "fr2") })
	fr.Add(KeyDown, func(e Event) {})

	var ls Listeners
	ls.Add(Click, func(e Event) { calls = append(calls, "ls1") })
	ls.Add(Scroll, func(e Event) {})
	ls.CopyFromExtra(fr)
	require.Len(t, ls[Click], 2, "the listeners beyond those present are copied")
	ls.Call(NewMouse(Click, Left, image.Point{}, 0))
	assert.Equal(t, []string{"fr2", "ls1"}, calls)
}

// TestListenersCopyFromExtraTypes tests that CopyFromExtra keeps the
// listeners of types that are only in the receiver, and copies those of
// types that are only in the source.
func TestListenersCopyFromExtraTypes(t *testing.T) {
	var fr Listeners
	fr.Add(KeyDown, func(e Event) {})
	var ls Listeners
	ls.Add(Scroll, func(e Event) {})
	ls.CopyFromExtra(fr)
	assert.Len(t, ls[Scroll], 1, "kept")
	assert.Len(t, ls[KeyDown], 1, "copied")
}

func TestDeque(t *testing.T) {
	var q Deque
	assert.Equal(t, 0, q.Len())
	_, ok := q.PollEvent()
	assert.False(t, ok)

	k1 := NewKey(KeyDown, 'a', key.CodeA, 0)
	k2 := NewKey(KeyDown, 'b', key.CodeB, 0)
	q.Send(k1)
	q.Send(k2)
	assert.Equal(t, 2, q.Len(), "unique events are not compressed")

	w1 := NewWindowPaint()
	w2 := NewWindowPaint()
	q.SendFirst(w1)
	q.SendFirst(w2)
	assert.Equal(t, 3, q.Len(), "the second paint replaces the first")
	q.SendFirst(NewWindow(WinFocus))
	assert.Equal(t, 4, q.Len())

	ev, ok := q.PollEvent()
	assert.True(t, ok)
	assert.Equal(t, Window, ev.Type(), "front events are last in, first out")
	assert.Same(t, w2, q.NextEvent())
	assert.Same(t, k1, q.NextEvent(), "back events are first in, first out")
	ev, _ = q.PollEvent()
	assert.Same(t, k2, ev)
	assert.Equal(t, 0, q.Len())
}

func TestDequeCompression(t *testing.T) {
	var q Deque
	m1 := NewMouseMove(NoButton, image.Pt(1, 1), image.Pt(0, 0), 0)
	m1.PrvTime.SetTime(time.Unix(100, 0))
	m2 := NewMouseMove(NoButton, image.Pt(2, 2), image.Pt(1, 1), 0)
	q.Send(m1)
	q.Send(m2)
	assert.Equal(t, 1, q.Len())
	ev := q.NextEvent().(*Mouse)
	assert.Same(t, m2, ev)
	assert.Equal(t, image.Pt(0, 0), ev.Prev, "the compressed move keeps the first previous position")
	assert.Equal(t, m1.PrvTime, ev.PrvTime)

	s1 := NewScroll(image.Pt(1, 1), math32.Vec2(0, 2), 0)
	s2 := NewScroll(image.Pt(1, 1), math32.Vec2(1, 3), 0)
	q.Send(s1)
	q.Send(s2)
	assert.Equal(t, 1, q.Len())
	sc := q.NextEvent().(*MouseScroll)
	assert.Equal(t, math32.Vec2(1, 5), sc.Delta, "compressed scrolls add their deltas")
}

func TestDequeWait(t *testing.T) {
	var q Deque
	got := make(chan Event)
	go func() { got <- q.NextEvent() }()
	time.Sleep(5 * time.Millisecond)
	k := NewKey(KeyDown, 'a', key.CodeA, 0)
	q.Send(k)
	select {
	case ev := <-got:
		assert.Same(t, k, ev)
	case <-time.After(2 * time.Second):
		t.Fatal("NextEvent did not wake for the sent event")
	}
}

// drain returns all of the events in the source.
func drain(es *Source) []Event {
	var evs []Event
	for {
		ev, ok := es.Deque.PollEvent()
		if !ok {
			return evs
		}
		evs = append(evs, ev)
	}
}

// types returns the types of the given events.
func types(evs []Event) []Types {
	var ts []Types
	for _, ev := range evs {
		ts = append(ts, ev.Type())
	}
	return ts
}

func TestSourceKey(t *testing.T) {
	es := &Source{}
	es.Key(KeyDown, 'a', key.CodeA, 0)
	assert.Equal(t, []Types{KeyDown}, types(drain(es)), "a mapped printable key gets its chord from the OS text")

	es.Key(KeyDown, 'a', key.CodeA, mods(key.Control))
	evs := drain(es)
	require.Equal(t, []Types{KeyDown, KeyChord}, types(evs))
	assert.Equal(t, key.Chord("Control+A"), evs[1].KeyChord())
	assert.Equal(t, key.CodeA, es.Last.Key)

	es.Key(KeyDown, 0, key.CodeTab, 0)
	assert.Equal(t, []Types{KeyDown, KeyChord}, types(drain(es)), "tab")

	es.Key(KeyDown, 0, key.CodeKeypad1, 0)
	evs = drain(es)
	require.Equal(t, []Types{KeyDown, KeyChord}, types(evs), "keypad without num lock")

	es.Key(KeyUp, 'a', key.CodeA, mods(key.Control))
	assert.Equal(t, []Types{KeyUp}, types(drain(es)))

	es.KeyChord('z', key.CodeZ, 0)
	assert.Equal(t, []Types{KeyChord}, types(drain(es)))
}

func TestSourceMouse(t *testing.T) {
	es := &Source{}
	es.MouseMove(image.Pt(1, 1))
	evs := drain(es)
	require.Equal(t, []Types{MouseMove}, types(evs))

	es.MouseButton(MouseDown, Left, image.Pt(2, 2), mods(key.Shift))
	es.MouseMove(image.Pt(5, 6))
	es.MouseButton(MouseUp, Left, image.Pt(7, 8), 0)
	evs = drain(es)
	require.Equal(t, []Types{MouseDown, MouseDrag, MouseUp}, types(evs))
	drag := evs[1].(*Mouse)
	assert.Equal(t, image.Pt(2, 2), drag.Start)
	assert.Equal(t, image.Pt(2, 2), drag.Prev)
	assert.Equal(t, mods(key.Shift), drag.Mods, "a drag has the modifiers of the press")
	up := evs[2].(*Mouse)
	assert.Equal(t, image.Pt(2, 2), up.Start, "the release starts at the press")
	assert.Equal(t, image.Pt(5, 6), up.Prev)

	es.Scroll(image.Pt(1, 1), math32.Vec2(0, 1))
	assert.Equal(t, []Types{Scroll}, types(drain(es)))
}

func TestSourceOther(t *testing.T) {
	es := &Source{}
	md := mimedata.NewText("x")
	es.DragExternal(DragEnter, image.Pt(3, 3), mods(key.Alt), md, true)
	es.DragExternal(DragMove, image.Pt(4, 4), 0, md, false)
	es.DragExternal(DragMove, image.Pt(5, 5), 0, md, false)
	es.DragExternal(DragLeave, image.Pt(9, 9), 0, md, false)
	es.DropExternal(image.Pt(6, 6), md)
	evs := drain(es)
	require.Equal(t, []Types{DragEnter, DragMove, DragLeave, Drop}, types(evs), "moves are compressed")
	assert.True(t, evs[0].(*DragDrop).FromApp)
	assert.Equal(t, image.Pt(5, 5), evs[1].WindowPos())
	assert.Equal(t, image.Pt(6, 6), es.Last.MousePos)

	es.Touch(TouchStart, 1, image.Pt(1, 1))
	es.Touch(TouchMove, 1, image.Pt(2, 2))
	es.Touch(TouchEnd, 1, image.Pt(3, 3))
	assert.Equal(t, []Types{TouchStart, MouseDown, TouchMove, MouseDrag, TouchEnd, MouseUp}, types(drain(es)), "touches are also mouse events")

	es.Magnify(2, image.Pt(1, 1))
	es.OSOpenFiles([]string{"f"})
	es.Custom("data")
	es.Composition("k", 0, 1)
	evs = drain(es)
	assert.Equal(t, []Types{Magnify, OSOpenFiles, Custom, Composition}, types(evs))
	assert.Equal(t, "data", evs[2].AsBase().Data)
	assert.Equal(t, "k", evs[3].(*TextComposition).Text)

	es.WindowResize()
	es.WindowPaint()
	es.Window(WinShow)
	assert.Equal(t, []Types{Window, WindowPaint, WindowResize}, types(drain(es)), "window events go first")
}
