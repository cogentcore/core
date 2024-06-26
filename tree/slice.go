// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tree

import (
	"cogentcore.org/core/base/slicesx"
)

// IndexOf returns the index of the given node in the given slice,
// or -1 if it is not found. The optional startIndex argument
// allows for optimized bidirectional searching if you have a guess
// at where the node might be, which can be a key speedup for large
// slices. If no value is specified for startIndex, it starts in the
// middle, which is a good default.
func IndexOf(slice []Node, child Node, startIndex ...int) int {
	return slicesx.Search(slice, func(e Node) bool { return e == child }, startIndex...)
}

// IndexByName returns the index of the first element in the given slice that
// has the given name, or -1 if none is found. See [IndexOf] for info on startIndex.
func IndexByName(slice []Node, name string, startIndex ...int) int {
	return slicesx.Search(slice, func(ch Node) bool { return ch.AsTree().Name == name }, startIndex...)
}
