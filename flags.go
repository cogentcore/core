// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

//go:generate enumgen

// Flags are bit flags for efficient core state of nodes -- see bitflag
// package for using these ordinal values to manipulate bit flag field.
type Flags int64 //enums:bitflag

const (
	// IsField indicates a node is a field in its parent node, not a child in children.
	IsField Flags = iota

	// Updating flag is set at UpdateStart and cleared if we were the first
	// updater at UpdateEnd.
	Updating

	// OnlySelfUpdate means that the UpdateStart / End logic only applies to
	// this node in isolation, not to its children -- useful for a parent node
	// that has a different functional role than its children.
	OnlySelfUpdate

	// following flags record what happened to a given node since the last
	// Update signal -- they are cleared at first UpdateStart and valid after
	// UpdateEnd

	// NodeDeleted means this node has been deleted.
	NodeDeleted

	// NodeDestroyed means this node has been destroyed -- do not trigger any
	// more update signals on it.
	NodeDestroyed

	// ChildAdded means one or more new children were added to the node.
	ChildAdded

	// ChildDeleted means one or more children were deleted from the node.
	ChildDeleted

	// ChildrenDeleted means all children were deleted.
	ChildrenDeleted

	// ValUpdated means a value was updated (Field, Prop, any kind of value)
	ValUpdated
)

/*
const (
	// ChildUpdateFlagsMask is a mask for all child updates.
	ChildUpdateFlagsMask = (1 << int64(ChildAdded)) | (1 << int64(ChildDeleted)) | (1 << int64(ChildrenDeleted))

	// StruUpdateFlagsMask is a mask for all structural changes update flags.
	StruUpdateFlagsMask = ChildUpdateFlagsMask | (1 << int64(NodeDeleted))

	// ValUpdateFlagsMask is a mask for all non-structural, value-only changes update flags.
	ValUpdateFlagsMask = (1 << int64(ValUpdated))

	// UpdateFlagsMask is a Mask for all the update flags -- destroyed is
	// excluded b/c otherwise it would get cleared.
	UpdateFlagsMask = StruUpdateFlagsMask | ValUpdateFlagsMask
)
*/
