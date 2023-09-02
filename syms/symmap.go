// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package syms

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"

	"goki.dev/ki/v2/kit"
	"goki.dev/pi/v2/lex"
	"goki.dev/pi/v2/token"
)

// SymMap is a map between symbol names and their full information.
// A given project will have a top-level SymMap and perhaps local
// maps for individual files, etc.  Namespaces / packages can be
// created and elements added to them to create appropriate
// scoping structure etc.  Note that we have to use pointers
// for symbols b/c otherwise it is very expensive to re-assign
// values all the time -- https://github.com/golang/go/issues/3117
type SymMap map[string]*Symbol

// Alloc ensures that map is made
func (sm *SymMap) Alloc() {
	if *sm == nil {
		*sm = make(SymMap)
	}
}

// Add adds symbol to map
func (sm *SymMap) Add(sy *Symbol) {
	sm.Alloc()
	(*sm)[sy.Name] = sy
}

// AddNew adds a new symbol to the map with the basic info
func (sm *SymMap) AddNew(name string, kind token.Tokens, fname string, reg lex.Reg) *Symbol {
	sy := NewSymbol(name, kind, fname, reg)
	sm.Alloc()
	(*sm)[name] = sy
	return sy
}

// Reset resets the symbol map
func (sm *SymMap) Reset() {
	*sm = make(SymMap)
}

// CopyFrom copies all the symbols from given source map into this one,
// including merging everything from common elements.
// Symbols with Type resolved are retained when there are duplicates.
// srcIsNewer means that the src map has the newer information to grab for
// updating the symbol region info during the merge.
func (sm *SymMap) CopyFrom(src SymMap, srcIsNewer bool) {
	sm.Alloc()
	for nm, ssy := range src {
		dsy, has := (*sm)[nm]
		if !has {
			(*sm)[nm] = ssy
			continue
		}
		if srcIsNewer {
			dsy.CopyFromSrc(ssy)
		} else {
			ssy.CopyFromSrc(dsy)
		}
		if dsy.Type != "" {
			// fmt.Printf("dupe sym: %v, using existing with type: %v\n", nm, dsy.Type)
			// fmt.Printf("\texisting region: %v  new source region: %v\n", dsy.Region, ssy.Region)
			dsy.Children.CopyFrom(ssy.Children, srcIsNewer)
		} else if ssy.Type != "" {
			// fmt.Printf("dupe sym: %v, using new with type: %v\n", nm, ssy.Type)
			// fmt.Printf("\texisting region: %v  new source region: %v\n", dsy.Region, ssy.Region)
			ssy.Children.CopyFrom(dsy.Children, !srcIsNewer)
			(*sm)[nm] = ssy
		} else {
			dsy.Children.CopyFrom(ssy.Children, srcIsNewer)
		}
	}
}

// First returns the first symbol in the map -- only sensible when there
// is just one such element
func (sm *SymMap) First() *Symbol {
	for _, sy := range *sm {
		return sy
	}
	return nil
}

// Slice returns a slice of the elements in the map, optionally sorted by name
func (sm *SymMap) Slice(sorted bool) []*Symbol {
	sys := make([]*Symbol, len(*sm))
	idx := 0
	for _, sy := range *sm {
		sys[idx] = sy
		idx++
	}
	if sorted {
		sort.Slice(sys, func(i, j int) bool {
			return sys[i].Name < sys[j].Name
		})
	}
	return sys
}

// FindName looks for given symbol name within this map and any children on the map
func (sm *SymMap) FindName(nm string) (*Symbol, bool) {
	if *sm == nil {
		return nil, false
	}
	sy, has := (*sm)[nm]
	if has {
		return sy, has
	}
	for _, ss := range *sm {
		if sy, has = ss.Children.FindName(nm); has {
			return sy, has
		}
	}
	return nil, false
}

// FindNameScoped looks for given symbol name within this map and any children on the map
// that are of subcategory token.NameScope (i.e., namespace, module, package, library)
func (sm *SymMap) FindNameScoped(nm string) (*Symbol, bool) {
	if *sm == nil {
		return nil, false
	}
	sy, has := (*sm)[nm]
	if has {
		return sy, has
	}
	for _, ss := range *sm {
		if ss.Kind.SubCat() == token.NameScope {
			if sy, has = ss.Children.FindNameScoped(nm); has {
				return sy, has
			}
		}
	}
	return nil, false
}

// FindKind looks for given symbol kind within this map and any children on the map
// Returns all instances found.  Uses cat / subcat based token matching -- if you
// specify a category-level or subcategory level token, it will match everything in that group
func (sm *SymMap) FindKind(kind token.Tokens, matches *SymMap) {
	if *sm == nil {
		return
	}
	for _, sy := range *sm {
		if kind.Match(sy.Kind) {
			if *matches == nil {
				*matches = make(SymMap)
			}
			(*matches)[sy.Name] = sy
		}
	}
	for _, ss := range *sm {
		ss.Children.FindKind(kind, matches)
	}
}

// FindKindScoped looks for given symbol kind within this map and any children on the map
// that are of subcategory token.NameScope (i.e., namespace, module, package, library).
// Returns all instances found.  Uses cat / subcat based token matching -- if you
// specify a category-level or subcategory level token, it will match everything in that group
func (sm *SymMap) FindKindScoped(kind token.Tokens, matches *SymMap) {
	if *sm == nil {
		return
	}
	for _, sy := range *sm {
		if kind.Match(sy.Kind) {
			if *matches == nil {
				*matches = make(SymMap)
			}
			(*matches)[sy.Name] = sy
		}
	}
	for _, ss := range *sm {
		if ss.Kind.SubCat() == token.NameScope {
			ss.Children.FindKindScoped(kind, matches)
		}
	}
}

// FindContainsRegion looks for given symbol kind that contains the given
// source file path (must be filepath.Abs file path) and position.
// Returns all instances found.  Uses cat / subcat based token matching -- if you
// specify a category-level or subcategory level token, it will match everything
// in that group.  if you specify kind = token.None then all tokens that contain
// region will be returned.  extraLns are extra lines added to the symbol region
// for purposes of matching.
func (sm *SymMap) FindContainsRegion(fpath string, pos lex.Pos, extraLns int, kind token.Tokens, matches *SymMap) {
	if *sm == nil {
		return
	}
	for _, sy := range *sm {
		fp, _ := filepath.Abs(sy.Filename)
		if fp != fpath {
			continue
		}
		reg := sy.Region
		if extraLns > 0 {
			reg.Ed.Ln += extraLns
		}
		if !reg.Contains(pos) {
			continue
		}
		if kind == token.None || kind.Match(sy.Kind) {
			if *matches == nil {
				*matches = make(SymMap)
			}
			(*matches)[sy.Name] = sy
		}
	}
	for _, ss := range *sm {
		ss.Children.FindContainsRegion(fpath, pos, extraLns, kind, matches)
	}
}

// OpenJSON opens from a JSON-formatted file.
func (sm *SymMap) OpenJSON(filename string) error {
	b, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, sm)
}

// SaveJSON saves to a JSON-formatted file.
func (sm *SymMap) SaveJSON(filename string) error {
	b, err := json.MarshalIndent(sm, "", "  ")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filename, b, 0644)
	return err
}

// Names returns a slice of the names in this map, optionally sorted
func (sm *SymMap) Names(sorted bool) []string {
	nms := make([]string, len(*sm))
	idx := 0
	for _, sy := range *sm {
		nms[idx] = sy.Name
		idx++
	}
	if sorted {
		sort.StringSlice(nms).Sort()
	}
	return nms
}

// KindNames returns a slice of the kind:names in this map, optionally sorted
func (sm *SymMap) KindNames(sorted bool) []string {
	nms := make([]string, len(*sm))
	idx := 0
	for _, sy := range *sm {
		nms[idx] = sy.Kind.String() + ":" + sy.Name
		idx++
	}
	if sorted {
		sort.StringSlice(nms).Sort()
	}
	return nms
}

// WriteDoc writes basic doc info, sorted by kind and name
func (sm *SymMap) WriteDoc(out io.Writer, depth int) {
	nms := sm.KindNames(true)
	for _, nm := range nms {
		ci := strings.Index(nm, ":")
		sy := (*sm)[nm[ci+1:]]
		sy.WriteDoc(out, depth)
	}
}

//////////////////////////////////////////////////////////////////////
// Partial lookups

// FindNamePrefix looks for given symbol name prefix within this map
// adds to given matches map (which can be nil), for more efficient recursive use
func (sm *SymMap) FindNamePrefix(seed string, matches *SymMap) {
	noCase := true
	if kit.HasUpperCase(seed) {
		noCase = false
	}
	for _, sy := range *sm {
		nm := sy.Name
		if noCase {
			nm = strings.ToLower(nm)
		}
		if strings.HasPrefix(nm, seed) {
			if *matches == nil {
				*matches = make(SymMap)
			}
			(*matches)[sy.Name] = sy
		}
	}
}

// FindNamePrefixRecursive looks for given symbol name prefix within this map
// and any children on the map.
// adds to given matches map (which can be nil), for more efficient recursive use
func (sm *SymMap) FindNamePrefixRecursive(seed string, matches *SymMap) {
	noCase := true
	if kit.HasUpperCase(seed) {
		noCase = false
	}
	for _, sy := range *sm {
		nm := sy.Name
		if noCase {
			nm = strings.ToLower(nm)
		}
		if strings.HasPrefix(nm, seed) {
			if *matches == nil {
				*matches = make(SymMap)
			}
			(*matches)[sy.Name] = sy
		}
	}
	for _, ss := range *sm {
		ss.Children.FindNamePrefixRecursive(seed, matches)
	}
}

// FindNamePrefixScoped looks for given symbol name prefix within this map
// and any children on the map that are of subcategory
// token.NameScope (i.e., namespace, module, package, library)
// adds to given matches map (which can be nil), for more efficient recursive use
func (sm *SymMap) FindNamePrefixScoped(seed string, matches *SymMap) {
	noCase := true
	if kit.HasUpperCase(seed) {
		noCase = false
	}
	for _, sy := range *sm {
		nm := sy.Name
		if nm[0] == '"' {
			nm = strings.Trim(nm, `"`) // path names may be quoted
			nm = filepath.Base(nm)     // sorry, this is a bit of a Go-specific hack to look at package names only
			sy = sy.Clone()
			sy.Name = nm
		}
		if noCase {
			nm = strings.ToLower(nm)
		}
		if strings.HasPrefix(nm, seed) {
			if *matches == nil {
				*matches = make(SymMap)
			}
			(*matches)[nm] = sy
		}
	}
	for _, ss := range *sm {
		if ss.Kind.SubCat() == token.NameScope {
			ss.Children.FindNamePrefixScoped(seed, matches)
		}
	}
}
