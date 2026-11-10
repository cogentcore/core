// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"strings"
	"testing"
	"time"

	"cogentcore.org/core/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/html"
)

// TestTransitions tests that the restyles of changes of the classes of
// elements report the transitions of their properties (for transitionend,
// which Google sign-in waits for to show its next step), with their
// durations and delays, and not changes of properties without transitions.
func TestTransitions(t *testing.T) {
	const page = `<html><head><style>
.slide { transform: translateX(0); opacity: 1; transition: transform .3s ease-out, opacity 200ms 100ms }
.slide.away { transform: translateX(-100px); opacity: 0; color: red }
.long { width: 10px; transition-property: width; transition-duration: 1s }
.long.wide { width: 50px }
.none { width: 10px }
.none.wide { width: 50px }
</style></head><body><div id="a" class="slide">a</div><div id="b" class="long">b</div><div id="c" class="none">c</div></body></html>`
	ctx := NewContext()
	doc, err := ParseHTML(ctx, strings.NewReader(page))
	require.NoError(t, err)
	bd := core.NewBody()
	require.NoError(t, ReadDocument(ctx, bd, doc))
	var got []Transition
	ctx.OnTransition = func(ts []Transition) { got = append(got, ts...) }
	bd.RenderImage(func() {
		bd.AsyncLock()
		defer bd.AsyncUnlock()
		a, b, c := elementByID(doc, "a"), elementByID(doc, "b"), elementByID(doc, "c")
		setAttr(a, "class", "slide away")
		setAttr(b, "class", "long wide")
		setAttr(c, "class", "none wide")
		ctx.Restyle([]*html.Node{a, b, c})
	})
	byProp := map[string]Transition{}
	for _, tr := range got {
		byProp[GetAttr(tr.Node, "id")+" "+tr.Property] = tr
	}
	assert.Len(t, got, 3, "%v", byProp)
	assert.Equal(t, 300*time.Millisecond, byProp["a transform"].Duration)
	assert.Equal(t, 200*time.Millisecond, byProp["a opacity"].Duration)
	assert.Equal(t, 100*time.Millisecond, byProp["a opacity"].Delay)
	assert.Equal(t, time.Second, byProp["b width"].Duration)
}
