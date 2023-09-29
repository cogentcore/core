// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

// Flags are bit flags for efficient core state of nodes -- see bitflag
// package for using these ordinal values to manipulate bit flag field.
type Flags int64 //enums:bitflag

const (
	// Field indicates a node is a field in its parent node, not a child in children.
	Field Flags = iota

	// Updating flag is set at UpdateStart and cleared if we were the first
	// updater at UpdateEnd.
	Updating

	// Deleted means this node has been deleted (removed from Parent)
	// Set just prior to calling OnDelete()
	Deleted

	// Destroyed means this node has been destroyed.
	// It should be skipped in all further processing, if there
	// are remaining pointers to it.
	Destroyed

	// ChildAdded means one or more new children were added to the node.
	ChildAdded

	// ChildDeleted means one or more children were deleted from the node.
	ChildDeleted

	// ChildrenDeleted means all children were deleted.
	ChildrenDeleted
)
