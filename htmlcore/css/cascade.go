// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package css

import (
	"regexp"
	"strings"
	"sync"

	"github.com/andybalholm/cascadia"
	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// Origin is the origin of a CSS declaration, which orders the cascade.
type Origin int

const (
	// OriginUserAgent is for the user agent (default) styles.
	OriginUserAgent Origin = iota

	// OriginHint is for the presentational attributes of
	// elements (width, bgcolor etc), which come before author styles.
	OriginHint

	// OriginAuthor is for the styles of the page.
	OriginAuthor

	// OriginInline is for the style attribute of an element.
	OriginInline
)

// PropertyName returns the name of the property of a declaration with the
// given name: in lower case, and the standard one of a legacy alias of it
// (see [legacyAliases]).
func PropertyName(p string) string {
	p = strings.ToLower(strings.TrimSpace(p))
	if a, ok := legacyAliases[p]; ok {
		return a
	}
	return p
}

// legacyAliases are the old names of properties that browsers still
// support as aliases of the standard ones, by their standard names (as the
// grid-column-gap of the grids of nytimes.com, without which their columns
// had no gaps between them).
var legacyAliases = map[string]string{
	"grid-gap": "gap", "grid-row-gap": "row-gap", "grid-column-gap": "column-gap",
}

// otherVendorValue matches a keyword of the vendor prefix of another
// browser engine (as -moz-available), which is invalid in browsers of
// WebKit and Blink, as Cogent Core is.
var otherVendorValue = regexp.MustCompile(`(^|[\s,(])-(moz|ms|o)-[a-z]`)

// ValidDeclaration returns whether a declaration of the given property
// and value is valid, as in browsers, which drop the declarations that
// are not when they are parsed, so that the declarations before them of
// the same property apply (as height: 100vh before height:
// -webkit-fill-available, of the captcha frames of DataDome for Safari,
// which were of the height of no content). A size of
// -webkit-fill-available is invalid here too, as it is not laid out, and
// the declaration before it is meant as a fallback for it. Custom
// properties and values with variables are only known when they are used.
func ValidDeclaration(prop, value string) bool {
	if strings.HasPrefix(prop, "--") || strings.Contains(value, "var(") {
		return true
	}
	if otherVendorValue.MatchString(strings.ToLower(value)) {
		return false
	}
	switch prop {
	case "width", "height", "min-width", "min-height", "max-width", "max-height", "inline-size", "block-size":
		if strings.Contains(strings.ToLower(value), "-webkit-fill-available") {
			return false
		}
	}
	return true
}

// LegacyDisplay returns the display for the given value of the display
// property, which is that of the prefixed legacy values: -webkit-box (as
// with -webkit-line-clamp, in a column) is a block, and -webkit-flex flex;
// unset and initial are inline.
func LegacyDisplay(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "-webkit-box", "-moz-box":
		return "block"
	case "-webkit-inline-box", "-moz-inline-box":
		return "inline-block"
	case "-webkit-flex":
		return "flex"
	case "-webkit-inline-flex":
		return "inline-flex"
	case "unset", "initial":
		// display is not inherited, so these are its initial value (as
		// display: unset of the widgets of the masthead of nytimes.com, in
		// a flex row, which had no box of its own, so that its ticker was
		// as wide as the row, over the logo)
		return "inline"
	}
	return v
}

// CompiledSelector is a compiled selector, with its specificity.
type CompiledSelector struct {
	Sel         cascadia.Sel
	Specificity int

	// KeyKind and key are a part of the last compound selector that every
	// node that it selects has, for finding them in a [selectorIndex]:
	// an id ('#'), a class ('.'), or an element name ('t'), or none (0).
	KeyKind byte
	Key     string

	// Anc is the mask of the keys of the elements that the elements that
	// it selects are in (see [ancestorKeys]).
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
	// [Context.Hovered]), or none
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
// (see [Context.selectNodes]), so that each selector is only matched
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

// FontShorthandValid returns whether the given value of the font shorthand
// is valid: a size (as 14px or medium, after any style, variant and
// weight) and a family after it, or a CSS-wide keyword or system font.
func FontShorthandValid(v string) bool {
	fs := strings.Fields(strings.ToLower(v))
	if len(fs) == 1 {
		switch fs[0] {
		case "inherit", "initial", "unset", "revert", "revert-layer", "caption", "icon", "menu",
			"message-box", "small-caption", "status-bar":
			return true
		}
	}
	for i, f := range fs {
		if len(f) > 0 && (f[0] >= '0' && f[0] <= '9' || f[0] == '.') || FontSizes[f] > 0 ||
			strings.HasPrefix(f, "calc(") || strings.HasPrefix(f, "clamp(") || strings.HasPrefix(f, "min(") || strings.HasPrefix(f, "max(") {
			return i+1 < len(fs) // (the family after the size)
		}
	}
	return false
}

// ImportURL returns the URL and the media query list of the given
// prelude of an @import rule (url("x") media, or "x" media).
func ImportURL(prelude string) (url, media string) {
	p := strings.TrimSpace(prelude)
	if rest, ok := strings.CutPrefix(p, "url("); ok {
		before, after, ok := strings.Cut(rest, ")")
		if !ok {
			return "", ""
		}
		url, media = before, after
	} else if len(p) > 0 && (p[0] == '"' || p[0] == '\'') {
		end := strings.IndexByte(p[1:], p[0])
		if end < 0 {
			return "", ""
		}
		url, media = p[:end+2], p[end+2:]
	} else {
		return "", ""
	}
	url = strings.Trim(strings.TrimSpace(url), `"'`)
	media = strings.TrimSpace(media)
	// layer() and supports() conditions are not supported, and are ignored
	if strings.HasPrefix(media, "layer") || strings.HasPrefix(media, "supports(") {
		media = ""
	}
	return url, media
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

// URLPattern matches the url() of CSS values, with its URL quoted or not.
var URLPattern = regexp.MustCompile(`url\(\s*(?:"([^"]*)"|'([^']*)'|([^)'"\s]*))\s*\)`)

// ModalAttr is the attribute of the dialogs that are shown as modal (as
// by the showModal of scripts), which the :modal selector of CSS selects
// (and the style of browsers for them, in the top layer).
const ModalAttr = "data-htmlcore-modal"

// HoverAttr is the attribute of the elements that are hovered while a page
// is read (see [Context.Hovered]), which no element of a document has.
const HoverAttr = "data-htmlcore-hover"
