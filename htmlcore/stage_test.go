// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"cogentcore.org/core/core"
	"cogentcore.org/core/tree"
)

// treeDump returns the types, texts and properties of the widgets under w.
func treeDump(w core.Widget) string {
	var sb strings.Builder
	w.AsWidget().WalkDown(func(n tree.Node) bool {
		fmt.Fprintf(&sb, "%d %T", len(n.AsTree().Children), n)
		if tx, ok := n.(*core.Text); ok {
			fmt.Fprintf(&sb, " %q", tx.Text)
		}
		fmt.Fprintf(&sb, " %v\n", n.AsTree().Properties)
		return tree.Continue
	})
	return sb.String()
}

// TestStaged tests that a body read in stages has the widgets of the
// body read at once, for stage sizes that stop at each element.
func TestStaged(t *testing.T) {
	for _, page := range []string{"gpu", "principles", "keymaps", "layout"} {
		md, err := os.ReadFile("../docs/content/" + page + ".md")
		if err != nil {
			t.Fatal(err)
		}
		testStaged(t, page, md, NewContext)
	}
}

// TestStagedEmbedded tests that a body read in stages with an element
// that reads more markdown when it is handled, as the <embed-page> of
// content does, has the widgets of the body read at once.
func TestStagedEmbedded(t *testing.T) {
	var embedded, md strings.Builder
	for i := range 30 {
		fmt.Fprintf(&embedded, "Embedded paragraph %d with some text in it.\n\n", i)
	}
	for i := range 20 {
		fmt.Fprintf(&md, "Paragraph %d of the page with some text in it.\n\n", i)
		if i%5 == 2 {
			md.WriteString("<embed-page></embed-page>\n\n")
		}
	}
	newContext := func() *Context {
		ctx := NewContext()
		ctx.ElementHandlers["embed-page"] = func(ctx *Context) bool {
			if err := ReadMDString(ctx, ctx.BlockParent, embedded.String()); err != nil {
				t.Error(err)
			}
			return true
		}
		return ctx
	}
	testStaged(t, "embedded", []byte(md.String()), newContext)
}

func testStaged(t *testing.T, page string, md []byte, newContext func() *Context) {
	whole := core.NewBody()
	if err := ReadMD(newContext(), whole, md); err != nil {
		t.Fatal(err)
	}
	want := treeDump(whole)
	staged := 0
	for size := 1; size < len(md)+100; size += 97 {
		b := core.NewBody()
		ctx := newContext()
		ctx.StageSize = size
		if err := ReadMD(ctx, b, md); err != nil {
			t.Fatal(err)
		}
		if ctx.HasRest() {
			staged++
			parent, start, err := ctx.ReadRest()
			if err != nil {
				t.Fatal(err)
			}
			if start == 0 || start > parent.AsTree().NumChildren() {
				t.Errorf("%s size %d: rest starts at child %d of %d", page, size, start, parent.AsTree().NumChildren())
			}
		}
		if ctx.HasRest() {
			t.Errorf("%s size %d: rest left after ReadRest", page, size)
		}
		if got := treeDump(b); got != want {
			t.Errorf("%s size %d: staged widgets differ:\n%s\nwant:\n%s", page, size, got, want)
		}
	}
	if staged == 0 {
		t.Errorf("%s: no stage size left a rest", page)
	}
}
