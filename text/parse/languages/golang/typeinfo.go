// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package golang

import (
	"cogentcore.org/core/text/parse/syms"
	"cogentcore.org/core/text/token"
	"slices"
)

// FuncParams returns the parameters of given function / method symbol,
// in proper order, name type for each param space separated in string
func (gl *GoLang) FuncParams(fsym *syms.Symbol) []string {
	// the params are in the order of their positions in the source, and
	// in the order of their indexes for the same position (the names
	// of one type, as in w, h float64), which is the order of the parser.
	var params []*syms.Symbol
	for _, cs := range fsym.Children {
		if cs.Kind == token.NameVarParam {
			params = append(params, cs)
		}
	}
	slices.SortFunc(params, func(a, b *syms.Symbol) int {
		switch {
		case a.Region.Start.IsLess(b.Region.Start):
			return -1
		case b.Region.Start.IsLess(a.Region.Start):
			return 1
		}
		return a.Index - b.Index
	})
	ps := make([]string, len(params))
	for i, cs := range params {
		ps[i] = cs.Name + " " + cs.Type
	}
	return ps
}
