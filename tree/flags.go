// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tree

// Flags are bit flags for efficient core state of nodes -- see bitflag
// package for using these ordinal values to manipulate bit flag field.
type Flags int64 //enums:bitflag

const (
	// Field indicates a node is a field in its parent node, not a child in children.
	Field Flags = iota
)
