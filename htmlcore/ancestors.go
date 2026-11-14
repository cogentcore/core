// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package htmlcore

import (
	"hash/maphash"
	"strings"

	"golang.org/x/net/html"
)

// The ancestor filter of selectors, as browsers have (a Bloom filter of
// the keys of the elements that an element is in): a selector whose
// compound selectors before its last one, that are of the elements that
// the element that it selects is in (as .menu in .menu .item), have keys
// (ids, classes, and names of elements) that no element that an element
// is in has, does not select it, which is found without matching it (most
// of the time of matching the elements of npr.org again, as its scripts
// changed them, was of the ancestors of each element).

// keyMask is a set of keys of elements (see [ancestorKeys]), as bits of a
// Bloom filter: a set that has a key has its bit, and a set may have the
// bit of a key that it does not have.
type keyMask [2]uint64

// keySeed is the seed of the hashes of the bits of keys.
var keySeed = maphash.MakeSeed()

// keyBit returns the mask of the given key ("#id", ".class", or a name).
func keyBit(key string) keyMask {
	h := maphash.String(keySeed, key)
	var m keyMask
	m[(h>>6)&1] = 1 << (h & 63)
	return m
}

// or returns the union of the masks.
func (m keyMask) or(o keyMask) keyMask {
	return keyMask{m[0] | o[0], m[1] | o[1]}
}

// has returns whether the mask may have all of the keys of the given one.
func (m keyMask) has(o keyMask) bool {
	return m[0]&o[0] == o[0] && m[1]&o[1] == o[1]
}

// ownKeys returns the mask of the keys of the given element: its name,
// id, and classes.
func ownKeys(n *html.Node) keyMask {
	m := keyBit(strings.ToLower(n.Data))
	for _, a := range n.Attr {
		if a.Namespace != "" {
			continue
		}
		switch a.Key {
		case "id":
			if a.Val != "" {
				m = m.or(keyBit("#" + a.Val))
			}
		case "class":
			for _, cl := range strings.Fields(a.Val) {
				m = m.or(keyBit("." + cl))
			}
		}
	}
	return m
}

// ancestorMask returns the mask of the keys of the elements that the given
// element is in, from c.ancestorMasks while it is set (for the elements
// matched at once, whose ancestors do not change meanwhile).
func (c *Context) ancestorMask(n *html.Node) keyMask {
	p := n.Parent
	if p == nil || p.Type != html.ElementNode {
		return keyMask{}
	}
	if c.ancestorMasks != nil {
		if m, ok := c.ancestorMasks[n]; ok {
			return m
		}
	}
	m := c.ancestorMask(p).or(ownKeys(p))
	if c.ancestorMasks != nil {
		c.ancestorMasks[n] = m
	}
	return m
}

// ancestorKeys returns the mask of the keys that the elements that an
// element that the given selector selects is in must have: those of its
// compound selectors that are followed by a combinator of descendants or
// children (whose elements are ones that it is in, also before a
// combinator of siblings, as the siblings of an element are in the same
// elements), outside of parentheses (as of :not), or none for a selector
// with escapes or namespaces.
func ancestorKeys(sel string) keyMask {
	var m keyMask
	if strings.ContainsAny(sel, "\\|") {
		return m
	}
	// the compound selectors and the combinators after them, at the top
	// level of the selector
	depth, start := 0, -1
	var compounds []string
	var combs []byte
	comb := byte(0)
	flush := func(end int) {
		if start >= 0 {
			if len(compounds) > len(combs) { // (two compounds without a combinator)
				combs = append(combs, ' ')
			}
			compounds = append(compounds, sel[start:end])
			start = -1
		}
	}
	for i := 0; i < len(sel); i++ {
		ch := sel[i]
		if depth == 0 && (ch == ' ' || ch == '\t' || ch == '\n' || ch == '>' || ch == '+' || ch == '~') {
			flush(i)
			if ch != ' ' && ch != '\t' && ch != '\n' {
				comb = ch
			} else if comb == 0 {
				comb = ' '
			}
			continue
		}
		switch ch {
		case '(', '[':
			depth++
		case ')', ']':
			depth--
		}
		if start < 0 {
			if len(compounds) > 0 {
				combs = append(combs, comb)
			}
			comb = 0
			start = i
		}
	}
	flush(len(sel))
	for i := 0; i < len(compounds)-1 && i < len(combs); i++ {
		if combs[i] != ' ' && combs[i] != '>' {
			continue // a sibling of the element of the next one
		}
		m = m.or(compoundKeys(compounds[i]))
	}
	return m
}

// compoundKeys returns the mask of the keys that an element that the given
// compound selector selects has: its name, and its ids and classes outside
// of parentheses and brackets.
func compoundKeys(cs string) keyMask {
	var m keyMask
	isName := func(ch byte) bool {
		return ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || ch == '-' || ch == '_' || ch >= 0x80
	}
	name := func(i int) string {
		j := i
		for j < len(cs) && isName(cs[j]) {
			j++
		}
		return cs[i:j]
	}
	if t := name(0); t != "" {
		m = m.or(keyBit(strings.ToLower(t)))
	}
	depth := 0
	for i := 0; i < len(cs); i++ {
		switch ch := cs[i]; ch {
		case '(', '[':
			depth++
		case ')', ']':
			depth--
		case '.', '#':
			if depth == 0 {
				if k := name(i + 1); k != "" {
					m = m.or(keyBit(string(ch) + k))
				}
			}
		}
	}
	return m
}

// selMatch returns whether the given selector selects the given element,
// without matching it if it can not for its ancestors (see
// [ancestorKeys]).
func (c *Context) selMatch(cs *compiledSelector, n *html.Node) bool {
	if cs.anc != (keyMask{}) && !c.ancestorMask(n).has(cs.anc) {
		return false
	}
	return cs.sel.Match(n)
}

// keepAncestorMasks keeps the masks of the ancestors of the elements as
// they are matched (see [Context.ancestorMask]), until the returned
// function is called, for elements that do not change meanwhile.
func (c *Context) keepAncestorMasks() func() {
	if c.ancestorMasks != nil {
		return func() {}
	}
	c.ancestorMasks = map[*html.Node]keyMask{}
	return func() { c.ancestorMasks = nil }
}
