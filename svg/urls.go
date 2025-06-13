// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"fmt"
	"log"
	"math/rand"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/base/slicesx"
	"cogentcore.org/core/tree"
)

////////  Naming elements with unique id's

// SplitNameIDDig splits name into numerical end part and preceding name,
// based on string of digits from end of name.
// If Id == 0 then it was not specified or didn't parse.
// SVG object names are element names + numerical id
func SplitNameIDDig(nm string) (string, int) {
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

// SplitNameID splits name after the element name (e.g., 'rect')
// returning true if it starts with element name,
// and numerical id part after that element.
// if numerical id part is 0, then it didn't parse.
// SVG object names are element names + numerical id
func SplitNameID(elnm, nm string) (bool, int) {
	if !strings.HasPrefix(nm, elnm) {
		// fmt.Printf("not elnm: %s  %s\n", nm, elnm)
		return false, 0
	}
	idstr := nm[len(elnm):]
	id, _ := strconv.Atoi(idstr)
	return true, id
}

// NameID returns the name with given unique id.
// returns plain name if id == 0
func NameID(nm string, id int) string {
	if id == 0 {
		return nm
	}
	return fmt.Sprintf("%s%d", nm, id)
}

// GatherIDs gathers all the numeric id suffixes currently in use.
// It automatically renames any that are not unique or empty.
func (sv *SVG) GatherIDs() {
	sv.UniqueIDs = make(map[int]struct{})
	sv.Root.WalkDown(func(n tree.Node) bool {
		sv.NodeEnsureUniqueID(n.(Node))
		return tree.Continue
	})
}

// NodeEnsureUniqueID ensures that the given node has a unique ID.
// Call this on any newly created nodes.
func (sv *SVG) NodeEnsureUniqueID(n Node) {
	elnm := n.SVGName()
	if elnm == "" {
		return
	}
	nb := n.AsNodeBase()
	elpfx, id := SplitNameID(elnm, nb.Name)
	if !elpfx {
		if !n.EnforceSVGName() { // if we end in a number, just register it anyway
			_, id = SplitNameIDDig(nb.Name)
			if id > 0 {
				sv.UniqueIDs[id] = struct{}{}
			}
			return
		}
		_, id = SplitNameIDDig(nb.Name)
		if id > 0 {
			nb.SetName(NameID(elnm, id))
		}
	}
	_, exists := sv.UniqueIDs[id]
	if id <= 0 || exists {
		id = sv.NewUniqueID() // automatically registers it
		nb.SetName(NameID(elnm, id))
	} else {
		sv.UniqueIDs[id] = struct{}{}
	}
}

// NewUniqueID returns a new unique numerical id number, for naming an object
func (sv *SVG) NewUniqueID() int {
	if sv.UniqueIDs == nil {
		sv.GatherIDs()
	}
	sz := len(sv.UniqueIDs)
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
		if _, has := sv.UniqueIDs[nid]; has {
			continue
		}
		break
	}
	sv.UniqueIDs[nid] = struct{}{}
	return nid
}

// FindDefByName finds Defs item by name, using cached indexes for speed.
func (sv *SVG) FindDefByName(defnm string) Node {
	if sv.DefIndexes == nil {
		sv.DefIndexes = make(map[string]int)
	}
	idx, has := sv.DefIndexes[defnm]
	if !has {
		idx = len(sv.Defs.Children) / 2
	}
	dn := sv.Defs.ChildByName(defnm, idx)
	if dn != nil {
		sv.DefIndexes[defnm] = dn.AsTree().IndexInParent()
		return dn.(Node)
	}
	delete(sv.DefIndexes, defnm) // not found, so delete from map
	return nil
}

func (sv *SVG) FindNamedElement(name string) Node {
	name = strings.TrimPrefix(name, "#")
	def := sv.FindDefByName(name)
	if def != nil {
		return def
	}
	sv.Root.WalkDown(func(n tree.Node) bool {
		if n.AsTree().Name == name {
			def = n.(Node)
			return tree.Break
		}
		return tree.Continue
	})
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
func (sv *SVG) NodeFindURL(n Node, url string) Node {
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
		log.Printf("svg.NodeFindURL could not find element named: %s for element: %s\n", url, n.AsTree().Path())
	}
	return rv
}

// NodePropURL returns a url(#name) url from given prop name on node,
// or empty string if none.  Returned value is just the 'name' part
// of the url, not the full string.
func NodePropURL(n Node, prop string) string {
	fp := n.AsTree().Property(prop)
	fs, iss := fp.(string)
	if !iss {
		return ""
	}
	return NameFromURL(fs)
}

const SVGRefCountKey = "SVGRefCount"

func IncRefCount(k tree.Node) {
	rc := k.AsTree().Property(SVGRefCountKey).(int)
	rc++
	k.AsTree().SetProperty(SVGRefCountKey, rc)
}

// RemoveOrphanedDefs removes any items from Defs that are not actually referred to
// by anything in the current SVG tree. Returns true if items were removed.
// Does not remove gradients with StopsName = "" with extant stops -- these
// should be removed manually, as they are not automatically generated.
// Also ensures that referenced gradients are before their referencers,
// so that they work correctly on loading.
func (sv *SVG) RemoveOrphanedDefs() bool {
	refkey := SVGRefCountKey
	for _, k := range sv.Defs.Children {
		k.AsTree().SetProperty(refkey, 0)
	}
	sv.Root.WalkDown(func(n tree.Node) bool {
		nb := n.(Node).AsNodeBase()
		pr := nb.Properties
		for _, v := range pr {
			nm := NameFromURL(reflectx.ToString(v))
			if nm == "" {
				continue
			}
			el := sv.FindDefByName(nm)
			if el != nil {
				IncRefCount(el)
			}
		}
		return tree.Continue
	})
	refGr := map[Node]bool{} // ref'd gradients -- move to top!
	sv.Defs.WalkDown(func(n tree.Node) bool {
		gr, isgr := n.(*Gradient)
		if !isgr {
			return tree.Continue
		}
		if gr.StopsName != "" {
			el := sv.FindDefByName(gr.StopsName)
			if el != nil {
				IncRefCount(el)
				refGr[el] = true
			} else {
				fmt.Println("stopsname not found:", gr.StopsName)
			}
		} else {
			if gr.Grad != nil && len(gr.Grad.AsBase().Stops) > 0 {
				IncRefCount(n) // keep us around
			}
		}
		return tree.Continue
	})
	ng := len(refGr)
	for gn := range refGr {
		idx := gn.AsNodeBase().IndexInParent()
		if idx >= ng { // not in first elements
			sv.Defs.Children = slicesx.Move(sv.Defs.Children, idx, 0) // move to top
		}
	}
	sz := len(sv.Defs.Children)
	del := false
	for i := sz - 1; i >= 0; i-- {
		n := sv.Defs.Children[i]
		rc := n.AsTree().Property(refkey).(int)
		if rc == 0 {
			// fmt.Printf("Deleting unused item: %s\n", n.AsTree().Name)
			sv.Defs.Children = slices.Delete(sv.Defs.Children, i, i+1)
			del = true
		} else {
			n.AsTree().DeleteProperty(refkey)
		}
	}
	return del
}
