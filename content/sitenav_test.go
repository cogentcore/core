// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package content

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cogentcore.org/core/base/iox/imagex"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/styles/states"
	"cogentcore.org/core/styles/units"
	"cogentcore.org/core/tree"
	"github.com/stretchr/testify/assert"
)

// navStep is one step of navigation in [TestNavigateKeepsSidebar].
type navStep func(ct *Content)

// navOpen opens the given URL.
func navOpen(url string) navStep {
	return func(ct *Content) { ct.Open(url) }
}

// navClick selects the node of the categories tree with the given text,
// as a click on it does.
func navClick(text string) navStep {
	return func(ct *Content) {
		ct.leftFrame.WalkDown(func(n tree.Node) bool {
			if tr, ok := n.(*core.Tree); ok && tr.Text == text {
				tr.SelectEvent(events.SelectOne)
				return tree.Break
			}
			return tree.Continue
		})
	}
}

func navBack(ct *Content)    { ct.historyBack() }
func navForward(ct *Content) { ct.historyForward() }

// TestNavigateKeepsSidebar checks that the left trees and the rendering after
// navigating between pages, which keeps the categories tree, are the same as
// those of the same navigation making a new categories tree for each page.
func TestNavigateKeepsSidebar(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	tests := []struct {
		name  string
		steps []navStep
	}{
		{"other-category", []navStep{navOpen("button"), navOpen("widget"), navOpen("value-binding")}},
		{"same-category", []navStep{navOpen("button"), navOpen("func-button")}},
		{"click", []navStep{navClick("Button"), navClick("Widget")}},
		{"click-home", []navStep{navClick("Button"), navClick("<b>Categories</b>")}},
		{"back", []navStep{navOpen("button"), navOpen("widget"), navBack}},
		{"forward", []navStep{navOpen("button"), navOpen("widget"), navBack, navBack, navForward, navForward}},
		{"heading", []navStep{navOpen("widget"), navOpen("button#types")}},
	}
	// The first window can have another size, from the saved window geometry.
	imagex.UpdateTestImages = true
	renderNav(t, filepath.Join("nav", "first"), nil)
	imagex.UpdateTestImages = false
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fname := filepath.Join("nav", tt.name)
			imagex.UpdateTestImages = true
			keepCategories = false
			want := renderNav(t, fname, tt.steps)
			keepCategories = true
			imagex.UpdateTestImages = false
			got := renderNav(t, fname, tt.steps)
			assert.Equal(t, want, got)
		})
	}
}

// renderNav runs the given steps in a new content of the example pages and
// asserts its rendering against the given file, returning [dumpTreeStyles].
func renderNav(t *testing.T, fname string, steps []navStep) string {
	// A name of its own keeps the saved window geometry of the other tests.
	b := core.NewBody("navigate")
	b.Styler(func(s *styles.Style) {
		s.Min.Set(units.Dp(1280), units.Dp(800))
	})
	ct := NewContent(b).SetSource(exampleContent)
	var funs []func()
	for _, step := range steps {
		funs = append(funs, func() {
			ct.AsyncLock()
			step(ct)
			ct.AsyncUnlock()
		})
	}
	var dump string
	// A new tree node is positioned from the scene position of its last layout,
	// so the sizes of new nodes are only right from the second layout.
	funs = append(funs, func() {
		ct.AsyncLock()
		ct.leftFrame.NeedsLayout()
		ct.AsyncUnlock()
	}, func() {
		time.Sleep(200 * time.Millisecond) // to give the image time to load
		ct.AsyncLock()
		dump = dumpTreeStyles(ct)
		ct.AsyncUnlock()
	})
	b.AssertRender(t, fname, funs...)
	return dump
}

// dumpTreeStyles returns the shown nodes of the trees in the left frame
// with their state, colors, and geometry.
func dumpTreeStyles(ct *Content) string {
	var sb strings.Builder
	ct.leftFrame.WalkDown(func(n tree.Node) bool {
		tr, ok := n.(*core.Tree)
		if !ok {
			return tree.Continue
		}
		st := &tr.Styles
		fmt.Fprintf(&sb, "%s sel=%v closed=%v color=%v bg=%v pos=%v size=%v", tr.Text, tr.StateIs(states.Selected), tr.Closed, st.Color, st.Background, tr.Geom.Pos.Total, tr.Geom.Size.Actual.Total)
		tr.Parts.WidgetWalkDown(func(cw core.Widget, cwb *core.WidgetBase) bool {
			fmt.Fprintf(&sb, " | %s color=%v", cwb.Name, cwb.Styles.Color)
			return tree.Continue
		})
		sb.WriteString("\n")
		if tr.Closed {
			return tree.Break
		}
		return tree.Continue
	})
	return sb.String()
}
