// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tree

import (
	"slices"

	"cogentcore.org/core/base/findfast"
	"cogentcore.org/core/types"
)

// Slice is just a slice of tree nodes: []Node, providing methods for accessing
// elements in the slice, and JSON marshal / unmarshal with encoding of
// underlying types
type Slice []Node

// IndexOf returns index of element in list, false if not there.  startIndex arg
// allows for optimized bidirectional find if you have an idea where it might
// be, which can be key speedup for large lists. If no value is specified for
// startIndex, it starts in the middle, which is a good default.
func (sl *Slice) IndexOf(kid Node, startIndex ...int) int {
	return findfast.FindFunc(*sl, func(ch Node) bool { return ch == kid }, startIndex...)
}

// IndexByName returns index of first element that has given name, false if
// not found. See [Slice.IndexOf] for info on startIndex.
func (sl *Slice) IndexByName(name string, startIndex ...int) int {
	return findfast.FindFunc(*sl, func(ch Node) bool { return ch.Name() == name }, startIndex...)
}

// IndexByType returns index of element that either is that type or embeds
// that type, false if not found. See [Slice.IndexOf] for info on startIndex.
func (sl *Slice) IndexByType(t *types.Type, embeds bool, startIndex ...int) int {
	if embeds {
		return findfast.FindFunc(*sl, func(ch Node) bool { return ch.NodeType().HasEmbed(t) }, startIndex...)
	}
	return findfast.FindFunc(*sl, func(ch Node) bool { return ch.NodeType() == t }, startIndex...)
}

// Move moves the element in the given slice at the given
// old position to the given new position and returns the
// resulting slice.
func Move[E any](s []E, from, to int) []E {
	temp := s[from]
	s = slices.Delete(s, from, from+1)
	s = slices.Insert(s, to, temp)
	return s
}

// Swap swaps the elements at the given two indices in the given slice.
func Swap[E any](s []E, i, j int) {
	s[i], s[j] = s[j], s[i]
}
