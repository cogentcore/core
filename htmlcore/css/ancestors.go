// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package css

import (
	"hash/maphash"
	"strings"

	"golang.org/x/net/html"
)

// KeyMask is a set of keys of elements (see [ancestorKeys]), as bits of a
// Bloom filter: a set that has a key has its bit, and a set may have the
// bit of a key that it does not have.
type KeyMask [2]uint64

// KeySeed is the seed of the hashes of the bits of keys.
var KeySeed = maphash.MakeSeed()

// KeyBit returns the mask of the given key ("#id", ".class", or a name).
func KeyBit(key string) KeyMask {
	h := maphash.String(KeySeed, key)
	var m KeyMask
	m[(h>>6)&1] = 1 << (h & 63)
	return m
}

// Or returns the union of the masks.
func (m KeyMask) Or(o KeyMask) KeyMask {
	return KeyMask{m[0] | o[0], m[1] | o[1]}
}

// Has returns whether the mask may have all of the keys of the given one.
func (m KeyMask) Has(o KeyMask) bool {
	return m[0]&o[0] == o[0] && m[1]&o[1] == o[1]
}

// OwnKeys returns the mask of the keys of the given element: its name,
// id, and classes.
func OwnKeys(n *html.Node) KeyMask {
	m := KeyBit(strings.ToLower(n.Data))
	for _, a := range n.Attr {
		if a.Namespace != "" {
			continue
		}
		switch a.Key {
		case "id":
			if a.Val != "" {
				m = m.Or(KeyBit("#" + a.Val))
			}
		case "class":
			for cl := range strings.FieldsSeq(a.Val) {
				m = m.Or(KeyBit("." + cl))
			}
		}
	}
	return m
}

// AncestorKeys returns the mask of the keys that the elements that an
// element that the given selector selects is in must have: those of its
// compound selectors that are followed by a combinator of descendants or
// children (whose elements are ones that it is in, also before a
// combinator of siblings, as the siblings of an element are in the same
// elements), outside of parentheses (as of :not), or none for a selector
// with escapes or namespaces.
func AncestorKeys(sel string) KeyMask {
	var m KeyMask
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
		m = m.Or(CompoundKeys(compounds[i]))
	}
	return m
}

// CompoundKeys returns the mask of the keys that an element that the given
// compound selector selects has: its name, and its ids and classes outside
// of parentheses and brackets.
func CompoundKeys(cs string) KeyMask {
	var m KeyMask
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
		m = m.Or(KeyBit(strings.ToLower(t)))
	}
	depth := 0
	for i := 0; i < len(cs); i++ {
		switch ch := cs[i]; ch {
		case '[':
			if depth == 0 {
				m = m.Or(attrKeys(cs[i:]))
			}
			depth++
		case '(':
			depth++
		case ')', ']':
			depth--
		case '.', '#':
			if depth == 0 {
				if k := name(i + 1); k != "" {
					m = m.Or(KeyBit(string(ch) + k))
				}
			}
		}
	}
	return m
}

// attrKeys returns the mask of the key that an element that the given
// attribute selector (at the start of the text) selects has, as for
// [class~=name] (as .name) and [id=name] (as #name), or none.
func attrKeys(text string) KeyMask {
	end := strings.IndexByte(text, ']')
	if end < 0 {
		return KeyMask{}
	}
	in := strings.TrimSpace(text[1:end])
	var prefix, value string
	if v, ok := strings.CutPrefix(in, "class~="); ok {
		prefix, value = ".", v
	} else if v, ok := strings.CutPrefix(in, "id="); ok {
		prefix, value = "#", v
	} else {
		return KeyMask{}
	}
	value = strings.TrimSpace(value)
	if len(value) >= 2 && (value[0] == '"' || value[0] == '\'') && value[len(value)-1] == value[0] {
		value = value[1 : len(value)-1]
	}
	if value == "" || strings.ContainsAny(value, " \t\"'\\") {
		return KeyMask{} // (as with a flag, as i)
	}
	return KeyBit(prefix + value)
}
