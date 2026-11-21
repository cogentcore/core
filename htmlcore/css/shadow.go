// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package css

import (
	"regexp"
	"strings"
)

// ScopeKind is the kind of the elements that a rule of a style sheet of a
// shadow tree selects (see [Context.inScope]).
type ScopeKind int8

const (
	// ScopeAll is the elements of its tree (or, for a rule of the
	// document, the elements of the document).
	ScopeAll ScopeKind = iota

	// ScopeHost is the host of the shadow root (:host).
	ScopeHost

	// ScopeSlotted is the children of the host in the slots of the shadow
	// tree (::slotted()).
	ScopeSlotted

	// ScopeAny is all elements, of the document and of shadow trees, for
	// the rules of the user agent.
	ScopeAny
)

// hostFunc matches :host and :host() at the start of a selector, with its
// argument.
var hostFunc = regexp.MustCompile(`^:host(?:\(([^()]*(?:\([^()]*\))?[^()]*)\))?`)

// slottedFunc matches ::slotted() in a selector, with the compound before it
// and its argument.
var slottedFunc = regexp.MustCompile(`([^\s>+~]*)::slotted\(([^()]*(?:\([^()]*\))?[^()]*)\)`)

// ScopedSelector returns the given selector of a style sheet of a shadow
// tree as one of the elements of the composed tree, and the kind of the
// elements that it selects, or false if it can not be: :host (or :host(X))
// alone selects the host (as X); :host before other compounds is the host
// (as X, or any element, which the host is of the elements of its tree);
// ::slotted(X) selects the children X of the slots (as slot > X);
// :host-context() and ::part() are not supported.
func ScopedSelector(sel string) (string, ScopeKind, bool) {
	if strings.Contains(sel, ":host-context") || strings.Contains(sel, "::part") {
		return "", 0, false
	}
	kind := ScopeAll
	if m := hostFunc.FindStringSubmatchIndex(sel); m != nil {
		// the compound of the host: the argument of :host, and what is
		// after it before a combinator (as :host(.open):hover)
		host := ""
		if m[2] >= 0 {
			host = strings.TrimSpace(sel[m[2]:m[3]])
		}
		rest := sel[m[1]:]
		end := strings.IndexAny(rest, " >+~")
		if end < 0 {
			end = len(rest)
		}
		host += rest[:end]
		if host == "" {
			host = "*"
		}
		rest = rest[end:]
		if strings.TrimSpace(rest) == "" {
			return host, ScopeHost, true
		}
		sel = host + rest
	}
	if strings.Contains(sel, "::slotted(") {
		sel = slottedFunc.ReplaceAllStringFunc(sel, func(s string) string {
			m := slottedFunc.FindStringSubmatch(s)
			pre := m[1]
			if pre == "" {
				pre = "slot"
			}
			return pre + " > " + m[2]
		})
		kind = ScopeSlotted
	}
	if strings.Contains(sel, ":host") || strings.Contains(sel, "::") && !strings.Contains(sel, "::before") && !strings.Contains(sel, "::after") {
		return "", 0, false
	}
	return sel, kind, true
}
