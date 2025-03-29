// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package syms

import (
	"strings"

	"cogentcore.org/core/icons"
	"cogentcore.org/core/text/parse/complete"
	"cogentcore.org/core/text/token"
)

// AddCompleteSyms adds given symbols as matches in the given match data
// Scope is e.g., type name (label only)
func AddCompleteSyms(sym SymMap, scope string, md *complete.Matches) {
	if len(sym) == 0 {
		return
	}
	sys := sym.Slice(true) // sorted
	for _, sy := range sys {
		if sy.Name[0] == '_' { // internal / import
			continue
		}
		nm := sy.Name
		lbl := sy.Label()
		if sy.Kind.SubCat() == token.NameFunction {
			nm += "()"
		}
		if scope != "" {
			lbl = lbl + " (" + scope + ".)"
		}
		c := complete.Completion{Text: nm, Label: lbl, Icon: sy.Kind.Icon(), Desc: sy.Detail}
		// fmt.Printf("nm: %v  kind: %v  icon: %v\n", nm, sy.Kind, c.Icon)
		md.Matches = append(md.Matches, c)
	}
}

// AddCompleteTypeNames adds names from given type as matches in the given match data
// Scope is e.g., type name (label only), and seed is prefix filter for names
func AddCompleteTypeNames(typ *Type, scope, seed string, md *complete.Matches) {
	md.Seed = seed
	for _, te := range typ.Els {
		nm := te.Name
		if seed != "" {
			if !strings.HasPrefix(nm, seed) {
				continue
			}
		}
		lbl := nm
		if scope != "" {
			lbl = lbl + " (" + scope + ".)"
		}
		icon := icons.Field // assume..
		c := complete.Completion{Text: nm, Label: lbl, Icon: icon}
		// fmt.Printf("nm: %v  kind: %v  icon: %v\n", nm, sy.Kind, c.Icon)
		md.Matches = append(md.Matches, c)
	}
	for _, mt := range typ.Meths {
		nm := mt.Name
		if seed != "" {
			if !strings.HasPrefix(nm, seed) {
				continue
			}
		}
		lbl := nm + "(" + mt.ArgString() + ") " + mt.ReturnString()
		if scope != "" {
			lbl = lbl + " (" + scope + ".)"
		}
		nm += "()"
		icon := icons.Method // assume..
		c := complete.Completion{Text: nm, Label: lbl, Icon: icon}
		// fmt.Printf("nm: %v  kind: %v  icon: %v\n", nm, sy.Kind, c.Icon)
		md.Matches = append(md.Matches, c)
	}
}

// AddCompleteSymsPrefix adds subset of symbols that match seed prefix to given match data
func AddCompleteSymsPrefix(sym SymMap, scope, seed string, md *complete.Matches) {
	matches := &sym
	if seed != "" {
		matches = &SymMap{}
		md.Seed = seed
		sym.FindNamePrefixRecursive(seed, matches)
	}
	AddCompleteSyms(*matches, scope, md)
}
