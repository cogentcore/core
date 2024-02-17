// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

import cmap "github.com/orcaman/concurrent-map/v2"

// Props is the type used for holding generic properties -- the actual Go type
// is a mouthful and not very gui-friendly, and we need some special json methods
type Props struct {
	cmap.ConcurrentMap[string, any]
}

func NewProps() *Props { return &Props{ConcurrentMap: cmap.New[any]()} }

// PropStruct is a struct of Name and Value, for use in a PropSlice to hold
// properties that require order information (maps do not retain any order)
type PropStruct struct {
	Name  string
	Value any
}

// PropSlice is a slice of PropStruct, for when order is important within a
// subset of properties (maps do not retain order) -- can set the value of a
// property to a PropSlice to create an ordered list of property values.
type PropSlice []PropStruct

// ElemLabel satisfies the gi.SliceLabeler interface to provide labels for slice elements
func (ps *PropSlice) ElemLabel(idx int) string {
	return (*ps)[idx].Name
}
