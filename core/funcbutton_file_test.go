package core

import (
	"image"
	"path/filepath"
	"testing"
	"time"

	"cogentcore.org/core/events"
	"cogentcore.org/core/tree"
)

// A function with a single Filename argument opens the file picker in its own
// window; accepting it with the OK button calls the function with the file.
func TestCallFuncFilenameOK(t *testing.T) {
	skipOnWeb(t, "there is one window")
	testCallFuncFilename(t, false)
}

// The same, with the function button in a menu, as in the Save menus of
// plots and network views: the menu is closed (and deleted) when the
// button is clicked, before the file picker is accepted.
func TestCallFuncFilenameMenu(t *testing.T) {
	skipOnWeb(t, "there is one window")
	testCallFuncFilename(t, true)
}

func testCallFuncFilename(t *testing.T, menu bool) {
	a := NewBody()
	want := filepath.Join(t.TempDir(), "weights.wts.gz")
	got := make(chan string, 1)
	save := func(filename Filename) { got <- string(filename) }
	var fb *FuncButton
	var mb *Button
	if menu {
		mb = NewButton(a).SetText("Weights").SetMenu(func(m *Scene, pos image.Point) {
			NewFuncButton(m).SetFunc(save)
		})
	} else {
		fb = NewFuncButton(a).SetFunc(save)
	}
	a.runAndShowNewWindow()
	rwa := a.Scene.RenderWindow()

	a.AsyncLock()
	if menu {
		mb.Send(events.Click)
	} else {
		fb.CallFunc()
	}
	a.AsyncUnlock()
	a.waitNoEvents(rwa)
	if menu {
		a.AsyncLock()
		ms := rwa.mains.top().popups.top().Scene
		var mfb *FuncButton
		ms.WidgetWalkDown(func(cw Widget, cwb *WidgetBase) bool {
			if f, ok := cw.(*FuncButton); ok {
				mfb = f
			}
			return tree.Continue
		})
		if mfb == nil {
			a.AsyncUnlock()
			t.Fatal("no func button in the menu")
		}
		mfb.Send(events.Click)
		a.AsyncUnlock()
		a.waitNoEvents(rwa)
	}
	time.Sleep(200 * time.Millisecond)

	var dw *renderWindow
	for _, w := range AllRenderWindows.windows() {
		if w != rwa {
			dw = w
		}
	}
	if dw == nil {
		t.Fatal("no file picker window")
	}
	d := dw.mains.top().Scene
	rc := d.renderContext()
	rc.Lock()
	var fp *FilePicker
	var ok *Button
	d.WidgetWalkDown(func(cw Widget, cwb *WidgetBase) bool {
		if f, isfp := cw.(*FilePicker); isfp {
			fp = f
		}
		if b, isb := cw.(*Button); isb && b.Text == "OK" {
			ok = b
		}
		return tree.Continue
	})
	if fp == nil || ok == nil {
		rc.Unlock()
		t.Fatalf("file picker %v, OK %v", fp != nil, ok != nil)
	}
	fp.SetFilename(want)
	fp.selectField.SetText(filepath.Base(want))
	ok.Send(events.Click)
	rc.Unlock()

	select {
	case g := <-got:
		if g != want {
			t.Errorf("called with %q, want %q", g, want)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the function was not called")
	}
}
