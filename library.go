// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xyz

import (
	"fmt"

	"goki.dev/ki/v2"
)

// AddToLibrary adds given Group to library, using group's name as unique key
// in Library map.
func (sc *Scene) AddToLibrary(gp *Group) {
	if sc.Library == nil {
		sc.Library = make(map[string]*Group)
	}
	sc.Library[gp.Name()] = gp
}

// NewInLibrary makes a new Group in library, using given name as unique key
// in Library map.
func (sc *Scene) NewInLibrary(nm string) *Group {
	gp := &Group{}
	gp.InitName(gp, nm)
	sc.AddToLibrary(gp)
	return gp
}

// AddFmLibrary adds a Clone of named item in the Library under given parent
// in the scenegraph.  Returns an error if item not found.
func (sc *Scene) AddFmLibrary(nm string, parent ki.Ki) (*Group, error) {
	gp, ok := sc.Library[nm]
	if !ok {
		return nil, fmt.Errorf("Scene AddFmLibrary: Library item: %s not found", nm)
	}
	updt := sc.UpdateStart()
	nwgp := gp.Clone().(*Group)
	parent.AddChild(nwgp)

	parent.WalkPre(func(k ki.Ki) bool {
		ni, nb := AsNode(k)
		if ni == nil {
			return ki.Break
		}
		nb.Sc = sc
		return ki.Continue
	})

	sc.UpdateEnd(updt)
	return nwgp, nil
}
