// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mvmcore

import (
	"bytes"
	"testing"

	"cogentcore.org/core/core"
)

// run runs src with b a new frame, and returns the frame and the output.
func run(t *testing.T, src string) (*core.Frame, string) {
	t.Helper()
	var out bytes.Buffer
	in, err := NewGo(&out)
	if err != nil {
		t.Fatal(err)
	}
	b := core.NewBody()
	fr := core.NewFrame(b)
	*interpreterParent = fr
	if _, err := in.Eval(src); err != nil {
		t.Fatal(err)
	}
	return fr, out.String()
}

func TestButton(t *testing.T) {
	fr, _ := run(t, `core.NewButton(b).SetText("Hello")`)
	if fr.NumChildren() != 1 {
		t.Fatalf("got %d children", fr.NumChildren())
	}
	if bt, ok := fr.Child(0).(*core.Button); !ok || bt.Text != "Hello" {
		t.Errorf("got %T %v", fr.Child(0), fr.Child(0))
	}
}

func TestPrint(t *testing.T) {
	_, out := run(t, `fmt.Println("hi", 1+2)`)
	if out != "hi 3\n" {
		t.Errorf("got %q", out)
	}
}

func TestEvents(t *testing.T) {
	fr, _ := run(t, `
count := 0
bt := core.NewButton(b).SetText("Click")
bt.OnClick(func(e events.Event) {
	count++
	bt.SetText(strconv.Itoa(count))
})`)
	if fr.NumChildren() != 1 {
		t.Fatalf("got %d children", fr.NumChildren())
	}
}

func TestMakerAndBind(t *testing.T) {
	fr, _ := run(t, `
name := "Gopher"
fr := core.NewFrame(b)
fr.Maker(func(p *tree.Plan) {
	tree.Add(p, func(w *core.Text) { w.SetText("x") })
	tree.AddAt(p, "b", func(w *core.Button) { w.SetText("y") })
})
fr.Update()
core.Bind(&name, core.NewTextField(b))`)
	if fr.NumChildren() != 2 {
		t.Fatalf("got %d children", fr.NumChildren())
	}
	inner := fr.Child(0).(*core.Frame)
	if inner.NumChildren() != 2 {
		t.Errorf("inner frame has %d children", inner.NumChildren())
	}
	if tf := fr.Child(1).(*core.TextField); tf.Text() != "Gopher" {
		t.Errorf("text field has %q", tf.Text())
	}
}

func TestTypes(t *testing.T) {
	_, out := run(t, `
type person struct {
	Name string
	Age  int
}
p := &person{Name: "Go", Age: 16}
core.NewForm(b).SetStruct(p)
fmt.Println(p.Name)`)
	if out != "Go\n" {
		t.Errorf("got %q", out)
	}
}
