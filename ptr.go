// Copyright (c) 2018, Randall C. O'Reilly. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"strings"
	"sync"

	"github.com/rcoreilly/goki/ki/kit"
)

// todo: get rid of time.Time in events -- cache out to int64 -- it has a pointer!
// https://segment.com/blog/allocation-efficiency-in-high-performance-go-services/

// todo: problem with this plan here is that without actually checking for all
// the remaining pointers, there is no way to ever delete anything, so really
// it is truly replicating exactly what the GC is doing.

// Index-based Ki pointer system takes advantage of the idea that trees tend
// to be relatively stable entities, so it isn't too wasteful to just keep
// lists of pointers to elements in each tree, and use indexes into these
// lists.  By avoiding sprinkling Ki pointers throughout many different
// objects all over memory, leveraging the chunked allocation of pointers in
// one contiguous slice, we should obtain significant savings.
//
// To preserve index validity, tree-lists can only grow in size.  Any attempt
// to re-use an index will result in the outstanding refs to that index
// pointing to a new object, and that can produce very difficult bugs.  Unlike
// the GC, we are not going to go through all of memory and find out when
// every last reference is gone.
//
//
// IdxPtr is a replacement for a pointer, using two indexes into a master
// list-of-lists of pointers to Ki objects -- each list is for a different
// tree -- the idea is to consolidate all pointers in one place to make the GC
// happier and avoid any actual pointers on Ki objects -- 0 indexes here
// indicate an invalid pointer -- both start at 1 for valid indexes
type IdxPtr struct {
	Tree int32 `desc:"ID of the tree that the item lives in"`
	Item int
}

// Ptr returns the Ki element in the pointer list
func (ip IdxPtr) Ptr() Ki {
	Ptrs.Mu.RLock()
	ki := Ptrs.Ptrs[ip.Tree][ip.Item]
	Ptrs.Mu.RUnlock()
	return ki
}

// PtrList is a map of lists of Ki objects
type PtrList struct {
	// The map of pointers
	Ptrs map[int32][]Ki

	// tree index counter -- incremented each time before a new tree list is
	// created i.e., it is the number of active tree lists and the list id of
	// the last list made
	TreeCtr int32

	// RWMutex for accessing pointers
	Mu sync.RWMutex
}

// Ptrs is the the master list of Ki ptrs
var Ptrs = PtrList{}

// NewList creates a new tree list and returns its unique id
func (pl *PtrList) NewList() int32 {
	pl.Mu.Lock()
	if pl.Ptrs == nil {
		pl.Ptrs = make(map[int32][]Ki, 1000)
	}
	pl.TreeCtr++
	tree := pl.TreeCtr
	pl.Ptrs[tree] = make([]Ki, 0, 1000)
	pl.Mu.Unlock()
	return tree
}

// Add adds a new item to the given tree and returns its unique index
func (pl *PtrList) Add(tree int32, ki Ki) int {
	pl.Mu.Lock()
	lst := pl.Ptrs[tree]
	idx := len(lst)
	pl.Ptrs[tree] = append(pl.Ptrs[tree], ki)
	pl.Mu.Unlock()
	return idx
}

// Delete deletes item in tree -- i.e., sets the element to nil
func (pl *PtrList) Delete(tree int32, itm int) {
	pl.Mu.Lock()
	pl.Ptrs[tree][itm] = nil
	pl.Mu.Unlock()
}

// key fact of Go: interface such as Ki is implicitly a pointer!

// Ptr provides JSON marshal / unmarshal via saved PathUnique
type Ptr struct {
	Ptr  Ki `json:"-" xml:"-"`
	Path string
}

var KiT_Ptr = kit.Types.AddType(&Ptr{}, nil)

// reset the pointer to nil, and the path to empty
func (k *Ptr) Reset() {
	k.Ptr = nil
	k.Path = ""
}

// GetPath updates the Path field with the current path to the pointer
func (k *Ptr) GetPath() {
	if k.Ptr != nil {
		k.Path = k.Ptr.PathUnique()
	} else {
		k.Path = ""
	}
}

// PtrFromPath finds and sets the Ptr value based on the current Path string -- returns true if pointer is found and non-nil
func (k *Ptr) PtrFmPath(root Ki) bool {
	// fmt.Printf("finding path: %v\n", k.Path)
	if len(k.Path) == 0 {
		k.Ptr = nil
		return true
	}
	k.Ptr = root.FindPathUnique(k.Path)
	// fmt.Printf("found: %v\n", k.Ptr)
	return k.Ptr != nil
}

// UpdatePath replaces any occurrence of oldPath with newPath, optionally only at the start of the path (typically true)
func (k *Ptr) UpdatePath(oldPath, newPath string, startOnly bool) {
	if startOnly {
		if strings.HasPrefix(k.Path, oldPath) {
			k.Path = newPath + strings.TrimPrefix(k.Path, oldPath)
		}
	} else {
		k.Path = strings.Replace(k.Path, oldPath, newPath, 1) // only do 1 replacement
	}
}

// MarshalJSON gets the current path and saves only the Path directly as value of this struct
func (k Ptr) MarshalJSON() ([]byte, error) {
	if k.Ptr == nil {
		b := []byte("null")
		return b, nil
	}
	k.GetPath()
	b := make([]byte, 0, len(k.Path)+8)
	b = append(b, []byte("\"")...)
	b = append(b, []byte(k.Path)...)
	b = append(b, []byte("\"")...)
	return b, nil
}

// UnarshalJSON loads the Path string directly from saved value -- KiNode must call SetPtrsFmPaths to actually update the pointers, based on the root object in the tree from which trees were generated, after all the initial loading has completed and the structure is all in place
func (k *Ptr) UnmarshalJSON(b []byte) error {
	// fmt.Printf("attempt to load path: %v\n", string(b))
	if bytes.Equal(b, []byte("null")) {
		k.Ptr = nil
		k.Path = ""
		return nil
	}
	k.Path = string(b)
	// fmt.Printf("loaded path: %v\n", k.Path)
	return nil
}

// MarshalXML getes the current path and saves it
func (k Ptr) MarshalXML(e *xml.Encoder, start xml.StartElement) error {
	tokens := []xml.Token{start}
	if k.Ptr == nil {
		tokens = append(tokens, xml.CharData("null"))
	} else {
		k.GetPath()
		tokens = append(tokens, xml.CharData(k.Path))
	}
	tokens = append(tokens, xml.EndElement{start.Name})
	for _, t := range tokens {
		err := e.EncodeToken(t)
		if err != nil {
			return err
		}
	}
	err := e.Flush()
	if err != nil {
		return err
	}
	return nil
}

// UnmarshalXML loads the Path string directly from saved value -- KiNode must call SetPtrsFmPaths to actually update the pointers, based on the root object in the tree from which trees were generated, after all the initial loading has completed and the structure is all in place
func (k *Ptr) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	t, err := d.Token()
	if err != nil {
		return err
	}
	ct, ok := t.(xml.CharData)
	if ok {
		tn := string(bytes.TrimSpace([]byte(ct)))
		if tn == "null" {
			// fmt.Printf("loading path: %v\n", tn)
			k.Ptr = nil
			k.Path = ""
		} else {
			// fmt.Printf("loading path: %v\n", tn)
			k.Path = tn
		}
		t, err := d.Token()
		if err != nil {
			return err
		}
		et, ok := t.(xml.EndElement)
		if ok {
			if et.Name != start.Name {
				return fmt.Errorf("ki.Ptr UnmarshalXML: EndElement: %v does not match StartElement: %v", et.Name, start.Name)
			}
			return nil
		}
		return fmt.Errorf("ki.Ptr UnmarshalXML: Token: %+v is not expected EndElement", et)
	}
	return fmt.Errorf("ki.Ptr UnmarshalXML: Token: %+v is not expected EndElement", ct)
}
