// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"fmt"
	"maps"
	"strings"
)

// This file has the custom properties of CSS (--name) of SVG nodes, and
// their uses by var(--name, fallback) in the values of the properties of
// nodes and of CSS rules (as the icons of bicycling.com, of
// style="fill:var(--primary, #000)", with --primary set for :root in a
// style element of some, which failed to parse as colors).

// customVars returns the custom properties of the node: those of its
// parent, with those that it sets (for :root, in a CSS rule of its type,
// class or name, or in its properties) over them.
func (g *NodeBase) customVars(parent map[string]string) map[string]string {
	type custom struct{ name, value string }
	var own []custom
	add := func(props map[string]any) {
		for k, v := range props {
			if strings.HasPrefix(k, "--") {
				own = append(own, custom{k, fmt.Sprint(v)})
			}
		}
	}
	for _, key := range []string{":root", strings.ToLower(g.NodeType().Name), "." + strings.ToLower(g.Class), "#" + strings.ToLower(g.Name)} {
		if pm, ok := g.CSSAgg[key].(map[string]any); ok {
			add(pm)
		}
	}
	add(g.Properties)
	if len(own) == 0 {
		return parent
	}
	vars := maps.Clone(parent)
	if vars == nil {
		vars = map[string]string{}
	}
	for _, c := range own {
		vars[c.name] = strings.TrimSpace(resolveVars(c.value, vars))
	}
	return vars
}

// resolveVarsIn returns the given properties with the uses of custom
// properties in their string values (var(--name, fallback)) replaced by
// their values in the given custom properties, or their fallbacks; it is
// the given map if none has any.
func resolveVarsIn(props map[string]any, vars map[string]string) map[string]any {
	var out map[string]any
	for k, v := range props {
		s, ok := v.(string)
		if !ok || !strings.Contains(s, "var(") {
			continue
		}
		if out == nil {
			out = maps.Clone(props)
		}
		out[k] = resolveVars(s, vars)
	}
	if out == nil {
		return props
	}
	return out
}

// resolveVars returns the given value with its uses of custom properties
// (var(--name, fallback)) replaced by their values in the given custom
// properties, or their fallbacks (or nothing), as in CSS.
func resolveVars(s string, vars map[string]string) string {
	for range 16 { // (for the uses in values and fallbacks)
		i := strings.Index(s, "var(")
		if i < 0 {
			return s
		}
		depth, end, comma := 0, -1, -1
		for j := i + 3; j < len(s); j++ {
			switch s[j] {
			case '(':
				depth++
			case ')':
				depth--
			case ',':
				if depth == 1 && comma < 0 {
					comma = j
				}
			}
			if depth == 0 {
				end = j
				break
			}
		}
		if end < 0 {
			return s
		}
		name, fallback := s[i+4:end], ""
		if comma > 0 {
			name, fallback = s[i+4:comma], s[comma+1:end]
		}
		v, ok := vars[strings.TrimSpace(name)]
		if !ok {
			v = strings.TrimSpace(fallback)
		}
		s = s[:i] + v + s[end+1:]
	}
	return s
}
