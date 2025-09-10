// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package xyz

import (
	"fmt"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/tree"
)

// AddToLibrary adds given Group to library, using group's name as unique key
// in Library map.
func (sc *Scene) AddToLibrary(gp *Group) {
	if sc.Library == nil {
		sc.Library = make(map[string]*Group)
	}
	sc.Library[gp.Name] = gp
	gp.Scene = sc
}

// NewInLibrary makes a new Group in library, using given name as unique key
// in Library map.
func (sc *Scene) NewInLibrary(nm string) *Group {
	gp := NewGroup()
	gp.SetName(nm)
	gp.Scene = sc
	sc.AddToLibrary(gp)
	return gp
}

// AddFromLibrary adds a Clone of named item in the Library under given parent
// in the scenegraph.  Returns an error if item not found.
func (sc *Scene) AddFromLibrary(nm string, parent tree.Node) (*Group, error) {
	gp, ok := sc.Library[nm]
	if !ok {
		return nil, fmt.Errorf("Scene AddFromLibrary: Library item: %s not found", nm)
	}
	nwgp := gp.Clone().(*Group)
	parent.AsTree().AddChild(nwgp)
	tree.SetUniqueName(nwgp)
	sc.SetScene(parent)
	return nwgp, nil
}

// Object is an object cloned from a Library.
type Object struct {
	Group

	// ObjectName is the name of the object to load from the Library.
	ObjectName string

	// currentName is the name of the object currently loaded.
	// if ObjectName is different than this, it is loaded on Update.
	currentName string
}

func (ob *Object) Update() {
	if ob.currentName == ob.ObjectName {
		return
	}
	if ob.Scene == nil {
		return
	}
	gp, ok := ob.Scene.Library[ob.ObjectName]
	if !ok {
		err := fmt.Errorf("Object Update: Library item: %s not found", ob.ObjectName)
		errors.Log(err)
	}
	ob.currentName = ob.ObjectName
	nwgp := gp.Clone().(*Group)
	ob.DeleteChildren()
	ob.AddChild(nwgp)
	ob.Scene.SetScene(ob)
}
