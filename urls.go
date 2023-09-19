// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"strings"
	"unicode"

	"goki.dev/ki/v2"
	"goki.dev/laser"
)

/////////////////////////////////////////////////////////////////////////////
//   Naming elements with unique id's

// SplitNameIdDig splits name into numerical end part and preceding name,
// based on string of digits from end of name.
// If Id == 0 then it was not specified or didn't parse.
// SVG object names are element names + numerical id
func SplitNameIdDig(nm string) (string, int) {
	sz := len(nm)

	for i := sz - 1; i >= 0; i-- {
		c := rune(nm[i])
		if !unicode.IsDigit(c) {
			if i == sz-1 {
				return nm, 0
			}
			n := nm[:i+1]
			id, _ := strconv.Atoi(nm[i+1:])
			return n, id
		}
	}
	return nm, 0
}

// SplitNameId splits name after the element name (e.g., 'rect')
// returning true if it starts with element name,
// and numerical id part after that element.
// if numerical id part is 0, then it didn't parse.
// SVG object names are element names + numerical id
func SplitNameId(elnm, nm string) (bool, int) {
	if !strings.HasPrefix(nm, elnm) {
		// fmt.Printf("not elnm: %s  %s\n", nm, elnm)
		return false, 0
	}
	idstr := nm[len(elnm):]
	id, _ := strconv.Atoi(idstr)
	return true, id
}

// NameId returns the name with given unique id.
// returns plain name if id == 0
func NameId(nm string, id int) string {
	if id == 0 {
		return nm
	}
	return fmt.Sprintf("%s%d", nm, id)
}

// GatherIds gathers all the numeric id suffixes currently in use.
// It automatically renames any that are not unique or empty.
func (sv *SVG) GatherIds() {
	sv.UniqueIds = make(map[int]struct{})
	sv.Root.FuncDownMeFirst(0, nil, func(k ki.Ki, level int, d any) bool {
		sv.NodeEnsureUniqueId(k.(Node))
		return ki.Continue
	})
}

// NodeEnsureUniqueId ensures that the given node has a unique Id
// Call this on any newly-created nodes.
func (sv *SVG) NodeEnsureUniqueId(ni Node) {
	elnm := ni.SVGName()
	if elnm == "" {
		return
	}
	elpfx, id := SplitNameId(elnm, ni.Name())
	if !elpfx {
		if !ni.EnforceSVGName() { // if we end in a number, just register it anyway
			_, id = SplitNameIdDig(ni.Name())
			if id > 0 {
				sv.UniqueIds[id] = struct{}{}
			}
			return
		}
		_, id = SplitNameIdDig(ni.Name())
		if id > 0 {
			ni.SetName(NameId(elnm, id))
		}
	}
	_, exists := sv.UniqueIds[id]
	if id <= 0 || exists {
		id = sv.NewUniqueId() // automatically registers it
		ni.SetName(NameId(elnm, id))
	} else {
		sv.UniqueIds[id] = struct{}{}
	}
}

// NewUniqueId returns a new unique numerical id number, for naming an object
func (sv *SVG) NewUniqueId() int {
	if sv.UniqueIds == nil {
		sv.GatherIds()
	}
	sz := len(sv.UniqueIds)
	var nid int
	for {
		switch {
		case sz >= 10000:
			nid = rand.Intn(sz * 100)
		case sz >= 1000:
			nid = rand.Intn(10000)
		default:
			nid = rand.Intn(1000)
		}
		if _, has := sv.UniqueIds[nid]; has {
			continue
		}
		break
	}
	sv.UniqueIds[nid] = struct{}{}
	return nid
}

// FindDefByName finds Defs item by name, using cached indexes for speed
func (sv *SVG) FindDefByName(defnm string) Node {
	if sv.DefIdxs == nil {
		sv.DefIdxs = make(map[string]int)
	}
	idx, has := sv.DefIdxs[defnm]
	if !has {
		idx = len(sv.Defs.Kids) / 2
	}
	idx, has = sv.Defs.Kids.IndexByName(defnm, idx)
	if has {
		sv.DefIdxs[defnm] = idx
		return sv.Defs.Kids[idx].(Node)
	}
	delete(sv.DefIdxs, defnm) // not found -- delete from map
	return nil
}

func (sv *SVG) FindNamedElement(name string) Node {
	name = strings.TrimPrefix(name, "#")
	def := sv.FindDefByName(name)
	if def != nil {
		return def
	}
	log.Printf("SVG FindNamedElement: could not find name: %v\n", name)
	return nil
}

// NameFromURL returns just the name referred to in a url(#name)
// if it is not a url(#) format then returns empty string.
func NameFromURL(url string) string {
	if len(url) < 7 {
		return ""
	}
	if url[:5] != "url(#" {
		return ""
	}
	ref := url[5:]
	sz := len(ref)
	if ref[sz-1] == ')' {
		ref = ref[:sz-1]
	}
	return ref
}

// NameToURL returns url as: url(#name)
func NameToURL(nm string) string {
	return "url(#" + nm + ")"
}

// NodeFindURL finds a url element in the parent SVG of given node.
// Returns nil if not found.
// Works with full 'url(#Name)' string or plain name or "none"
func (sv *SVG) NodeFindURL(gi Node, url string) Node {
	if url == "none" {
		return nil
	}
	ref := NameFromURL(url)
	if ref == "" {
		ref = url
	}
	if ref == "" {
		return nil
	}
	rv := sv.FindNamedElement(ref)
	if rv == nil {
		log.Printf("svg.NodeFindURL could not find element named: %s for element: %s\n", url, gi.Path())
	}
	return rv
}

// NodePropURL returns a url(#name) url from given prop name on node,
// or empty string if none.  Returned value is just the 'name' part
// of the url, not the full string.
func NodePropURL(gi Node, prop string) string {
	fp, err := gi.PropTry(prop)
	if err != nil {
		return ""
	}
	fs, iss := fp.(string)
	if !iss {
		return ""
	}
	return NameFromURL(fs)
}

const SVGRefCountKey = "SVGRefCount"

func IncRefCount(k ki.Ki) {
	rc := k.Prop(SVGRefCountKey).(int)
	rc++
	k.SetProp(SVGRefCountKey, rc)
}

// RemoveOrphanedDefs removes any items from Defs that are not actually referred to
// by anything in the current SVG tree.  Returns true if items were removed.
// Does not remove gradients with StopsName = "" with extant stops -- these
// should be removed manually, as they are not automatically generated.
func (sv *SVG) RemoveOrphanedDefs() bool {
	refkey := SVGRefCountKey
	for _, k := range sv.Defs.Kids {
		k.SetProp(refkey, 0)
	}
	sv.Root.FuncDownMeFirst(0, nil, func(k ki.Ki, level int, d any) bool {
		pr := k.Properties()
		for _, v := range *pr {
			ps := laser.ToString(v)
			if !strings.HasPrefix(ps, "url(#") {
				continue
			}
			nm := NameFromURL(ps)
			el := sv.FindDefByName(nm)
			if el != nil {
				IncRefCount(el)
			}
		}
		if gr, isgr := k.(*Gradient); isgr {
			if gr.StopsName != "" {
				el := sv.FindDefByName(gr.StopsName)
				if el != nil {
					IncRefCount(el)
				}
			} else {
				if gr.Grad.Gradient != nil && len(gr.Grad.Gradient.Stops) > 0 {
					IncRefCount(k) // keep us around
				}
			}
		}
		return ki.Continue
	})
	sz := len(sv.Defs.Kids)
	del := false
	for i := sz - 1; i >= 0; i-- {
		k := sv.Defs.Kids[i]
		rc := k.Prop(refkey).(int)
		if rc == 0 {
			fmt.Printf("Deleting unused item: %s\n", k.Name())
			sv.Defs.Kids.DeleteAtIndex(i)
			del = true
		} else {
			k.DeleteProp(refkey)
		}
	}
	return del
}
