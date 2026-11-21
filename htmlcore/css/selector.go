// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package css

import (
	"slices"
	"strings"
	"sync"

	"github.com/andybalholm/cascadia"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// CompiledSelector is a compiled selector, with its specificity.
type CompiledSelector struct {
	Sel         cascadia.Sel
	Specificity int

	// KeyKind and Key are a part of the last compound selector that every
	// node that it selects has, for finding them in a [SelectorIndex]:
	// an id ('#'), a class ('.'), or an element name ('t'), or none (0).
	KeyKind byte
	Key     string

	// Anc is the mask of the keys of the elements that the elements that
	// it selects are in (see [AncestorKeys]).
	Anc KeyMask
}

// selectorCache has the compiled selectors, with nil for ones
// that are not supported (as pseudo-elements, which select no nodes).
// It is shared by the contexts that read pages, as in other goroutines.
var selectorCache = map[string]*CompiledSelector{}

// selectorCacheMu protects selectorCache.
var selectorCacheMu sync.Mutex

// CompileSelector returns the compiled selector, or nil if it is not supported.
func CompileSelector(sel string) *CompiledSelector {
	selectorCacheMu.Lock()
	defer selectorCacheMu.Unlock()
	if cs, ok := selectorCache[sel]; ok {
		return cs
	}
	var cs *CompiledSelector
	if s, err := cascadia.Parse(RewriteSelector(sel)); err == nil && s.PseudoElement() == "" {
		sp := s.Specificity()
		cs = &CompiledSelector{Sel: s, Specificity: sp[0]*10000 + sp[1]*100 + sp[2]}
		cs.KeyKind, cs.Key = SelectorKey(sel)
		cs.Anc = AncestorKeys(sel)
	}
	selectorCache[sel] = cs
	return cs
}

// RewriteSelector rewrites the parts of the given selector that the selector
// library does not support as ones that it does: :is(X) and :where(X) as
// :not(:not(X)), and :focus-within and :focus-visible as :focus (which
// selects nothing, as for :hover, in a static rendering).
func RewriteSelector(sel string) string {
	// :hover selects the elements that are marked hovered (see
	// htmlcore.Context.Hovered), or none
	sel = strings.ReplaceAll(sel, ":hover", "["+HoverAttr+"]")
	// :modal selects the dialogs that scripts show as modal (with
	// showModal), which they mark with ModalAttr
	sel = strings.ReplaceAll(sel, ":modal", "["+ModalAttr+"]")
	if !strings.Contains(sel, ":is(") && !strings.Contains(sel, ":where(") && !strings.Contains(sel, ":focus-") {
		return sel
	}
	sel = strings.ReplaceAll(sel, ":focus-within", ":focus")
	sel = strings.ReplaceAll(sel, ":focus-visible", ":focus")
	var sb strings.Builder
	var closes []bool // for each open parenthesis, whether its close is doubled
	for i := 0; i < len(sel); i++ {
		switch {
		case strings.HasPrefix(sel[i:], ":is("):
			sb.WriteString(":not(:not(")
			closes = append(closes, true)
			i += len(":is(") - 1
		case strings.HasPrefix(sel[i:], ":where("):
			sb.WriteString(":not(:not(")
			closes = append(closes, true)
			i += len(":where(") - 1
		case sel[i] == '(':
			sb.WriteByte('(')
			closes = append(closes, false)
		case sel[i] == ')':
			sb.WriteByte(')')
			if n := len(closes); n > 0 {
				if closes[n-1] {
					sb.WriteByte(')')
				}
				closes = closes[:n-1]
			}
		default:
			sb.WriteByte(sel[i])
		}
	}
	return sb.String()
}

// SelectorKey returns a part of the last compound selector of the given
// selector that every node that it selects has (see compiledSelector.key):
// its id, else its first class, else its element name, or none if it has
// none of them outside of functions and attribute selectors (or has
// escapes, which are not decoded here).
func SelectorKey(sel string) (byte, string) {
	if strings.ContainsAny(sel, "\\|") { // escapes and namespaces
		return 0, ""
	}
	// the start of the last compound selector, after the last combinator
	// at the top level
	start, depth := 0, 0
	for i := 0; i < len(sel); i++ {
		switch ch := sel[i]; ch {
		case '(', '[':
			depth++
		case ')', ']':
			depth--
		case ' ', '>', '+', '~', '\t', '\n':
			if depth == 0 {
				start = i + 1
			}
		}
	}
	last := sel[start:]
	isName := func(ch byte) bool {
		return ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || ch == '-' || ch == '_' || ch >= 0x80
	}
	name := func(i int) string {
		j := i
		for j < len(last) && isName(last[j]) {
			j++
		}
		return last[i:j]
	}
	var class string
	depth = 0
	for i := 0; i < len(last); i++ {
		switch ch := last[i]; {
		case ch == '(' || ch == '[':
			depth++
		case ch == ')' || ch == ']':
			depth--
		case depth == 0 && ch == '#':
			if id := name(i + 1); id != "" {
				return '#', id
			}
		case depth == 0 && ch == '.' && class == "":
			class = name(i + 1)
		}
	}
	if class != "" {
		return '.', class
	}
	if tag := name(0); tag != "" {
		return 't', strings.ToLower(tag)
	}
	return 0, ""
}

// SelectorIndex has the element nodes of a document by id, class, and
// element name, in document order, for selecting the nodes of style rules
// (see htmlcore.Context.selectNodes), so that each selector is only matched
// against the nodes that have its key, as in browsers.
type SelectorIndex struct {
	Root                 *html.Node
	ByID, ByClass, ByTag map[string][]*html.Node
	all                  []*html.Node
}

// NewSelectorIndex returns the index of the element nodes under root.
func NewSelectorIndex(root *html.Node) *SelectorIndex {
	ix := &SelectorIndex{Root: root, ByID: map[string][]*html.Node{}, ByClass: map[string][]*html.Node{}, ByTag: map[string][]*html.Node{}}
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if c.Type != html.ElementNode {
				continue
			}
			ix.all = append(ix.all, c)
			ix.ByTag[strings.ToLower(c.Data)] = append(ix.ByTag[strings.ToLower(c.Data)], c)
			for _, a := range c.Attr {
				if a.Namespace != "" {
					continue
				}
				switch a.Key {
				case "id":
					ix.ByID[a.Val] = append(ix.ByID[a.Val], c)
				case "class":
					seen := map[string]bool{}
					for cl := range strings.FieldsSeq(a.Val) {
						if !seen[cl] {
							seen[cl] = true
							ix.ByClass[cl] = append(ix.ByClass[cl], c)
						}
					}
				}
			}
			walk(c)
		}
	}
	walk(root)
	return ix
}

// SelectNodes returns the nodes under root that the given selector selects.
func SelectNodes(root *html.Node, sel string) []*html.Node {
	// Selectors of element names, such as the user agent styles have, select
	// the elements with those names, which one walk of the tree finds.
	if a := atom.Lookup([]byte(sel)); a != 0 && a.String() == sel {
		var matches []*html.Node
		var walk func(n *html.Node)
		walk = func(n *html.Node) {
			if n.DataAtom == a {
				matches = append(matches, n)
			}
			for c := n.FirstChild; c != nil; c = c.NextSibling {
				if c.Type == html.ElementNode {
					walk(c)
				}
			}
		}
		walk(root)
		return matches
	}
	cs := CompileSelector(sel)
	if cs == nil {
		return nil
	}
	return cascadia.QueryAll(root, cs.Sel)
}

// Specificity returns the specificity of the given selector,
// as ids*10000 + (classes, attributes, pseudo-classes)*100 + types.
func Specificity(sel string) int {
	ids, classes, types := 0, 0, 0
	start := true // at the start of a compound selector
	for i := 0; i < len(sel); i++ {
		ch := sel[i]
		switch ch {
		case '#':
			ids++
			start = false
		case '.':
			classes++
			start = false
		case '[':
			classes++
			for i < len(sel) && sel[i] != ']' {
				i++
			}
			start = false
		case ':':
			if i+1 < len(sel) && sel[i+1] == ':' {
				types++ // pseudo-element
				i++
			} else {
				classes++
			}
			start = false
		case ' ', '>', '+', '~', ',':
			start = true
			continue
		case '*':
			start = false
		default:
			if start && (ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z') {
				types++
			}
			start = false
		}
		// skip the rest of the name
		for i+1 < len(sel) && IsNameChar(sel[i+1]) {
			i++
		}
	}
	return ids*10000 + classes*100 + types
}

func IsNameChar(ch byte) bool {
	return ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || ch == '-' || ch == '_'
}

// SplitSelectors splits the given selector list at the commas that are
// not in parentheses, brackets, or quotes.
func SplitSelectors(list string) []string {
	var sels []string
	depth := 0
	var quote byte
	start := 0
	for i := 0; i < len(list); i++ {
		ch := list[i]
		switch {
		case quote != 0:
			switch ch {
			case '\\':
				i++
			case quote:
				quote = 0
			}
		case ch == '"' || ch == '\'':
			quote = ch
		case ch == '(' || ch == '[':
			depth++
		case ch == ')' || ch == ']':
			depth--
		case ch == ',' && depth == 0:
			sels = append(sels, strings.TrimSpace(list[start:i]))
			start = i + 1
		}
	}
	return append(sels, strings.TrimSpace(list[start:]))
}

// ModalAttr is the attribute of the dialogs that are shown as modal (as
// by the showModal of scripts), which the :modal selector of CSS selects
// (and the style of browsers for them, in the top layer).
const ModalAttr = "data-htmlcore-modal"

// HoverAttr is the attribute of the elements that are hovered while a page
// is read (see htmlcore.Context.Hovered), which no element of a document has.
const HoverAttr = "data-htmlcore-hover"

// PseudoSelector returns the selector of the elements of the given
// selector of a ::before or ::after pseudo-element (also :before and
// :after), and which of them it is, or false if it is not one.
func PseudoSelector(sel string) (base, kind string, ok bool) {
	lsel := strings.ToLower(sel)
	for _, k := range []string{"before", "after"} {
		for _, p := range []string{"::" + k, ":" + k} {
			if strings.HasSuffix(lsel, p) {
				base = strings.TrimSpace(sel[:len(sel)-len(p)])
				if base == "" || strings.HasSuffix(base, " ") || strings.HasSuffix(base, ">") {
					base += "*"
				}
				return base, k, true
			}
		}
	}
	return "", "", false
}

// KeyIn returns whether the given element has the key of the selector
// (see [CompiledSelector.key]), which every element that it selects has,
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
