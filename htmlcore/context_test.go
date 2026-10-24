// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"strings"
	"testing"

	"cogentcore.org/core/core"
	"github.com/aymerick/douceur/parser"
	selcss "github.com/ericchiang/css"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/html"
)

func TestInlineContainer(t *testing.T) {
	b := core.NewBody()
	assert.NoError(t, ReadHTMLString(NewContext(), b, `<button>Test</button>`))
	if tag := b.Child(0).AsTree().Property("tag"); tag != "body" {
		t.Errorf("expected first child to be body but got %v", tag)
	}
	if !strings.Contains(b.Child(0).AsTree().Child(0).AsTree().Name, "inline") {
		t.Errorf("expected inline container for h1 but got %v", b.Child(0))
	}
}

func TestNoInlineContainer(t *testing.T) {
	b := core.NewBody()
	assert.NoError(t, ReadHTMLString(NewContext(), b, `<h1>Test</h1>`))
	if tag := b.Child(0).AsTree().Property("tag"); tag != "body" {
		t.Errorf("expected first child to be body but got %v", tag)
	}
	if strings.Contains(b.Child(0).AsTree().Child(0).AsTree().Name, "inline") {
		t.Errorf("expected no inline container for h1 but got %v", b.Child(0))
	}
}

// TestSelectRule tests that the rules of the user agent styles select the
// same nodes as the selector library selects.
func TestSelectRule(t *testing.T) {
	doc, err := html.Parse(strings.NewReader(`<html><body><h1>Title</h1><p>Text <b>bold</b></p><ul><li>one</li><li>two</li></ul><div><table><tr><td>cell</td><th>head</th></tr></table></div><ruby>x<rt>y</rt><rp>z</rp></ruby><svg><p>svg p</p></svg><center>c</center></body></html>`))
	assert.NoError(t, err)
	ss, err := parser.Parse(userAgentStyles + "\nH1, p { color: red; }\ndiv, *, p { color: blue; }\np, p { color: green; }")
	assert.NoError(t, err)
	simple := 0
	for _, rule := range ss.Rules {
		want := []*html.Node{}
		if s, err := selcss.Parse(strings.Join(rule.Selectors, ",")); err == nil {
			want = s.Select(doc)
		}
		got := selectRule(doc, rule)
		if selectorAtoms(rule.Selectors) != nil {
			simple++
		}
		count := func(nodes []*html.Node) map[*html.Node]int {
			m := map[*html.Node]int{}
			for _, n := range nodes {
				m[n]++
			}
			return m
		}
		assert.Equal(t, count(want), count(got), rule.Selectors)
	}
	assert.Greater(t, simple, 5)
}
