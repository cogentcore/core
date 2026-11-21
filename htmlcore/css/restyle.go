// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package css

import (
	"slices"
	"strings"

	"golang.org/x/net/html"
)

// KeyIn returns whether the given element has the key of the selector
// (see [compiledSelector.key]), which every element that it selects has,
// or it has none.
func (cs *CompiledSelector) KeyIn(n *html.Node) bool {
	switch cs.KeyKind {
	case '#':
		return attrOf(n, "id") == cs.Key
	case '.':
		return slices.Contains(strings.Fields(attrOf(n, "class")), cs.Key)
	case 't':
		return strings.EqualFold(n.Data, cs.Key)
	}
	return true
}

// attrOf returns the value of the given attribute of the given node,
// or "" if it has none, as [GetAttr].
func attrOf(n *html.Node, key string) string {
	res := ""
	for _, a := range n.Attr {
		if a.Key == key {
			res = a.Val
		}
	}
	return res
}
