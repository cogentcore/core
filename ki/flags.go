// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

import "github.com/goki/ki/kit"

// Flags are bit flags for efficient core state of nodes -- see bitflag
// package for using these ordinal values to manipulate bit flag field.
type Flags int32

//go:generate stringer -type=Flags

var KiT_Flags = kit.Enums.AddEnum(FlagsN, kit.BitFlag, nil)

const (
	// IsField indicates a node is a field in its parent node, not a child in children.
	IsField Flags = iota

	// HasKiFields indicates a node has Ki Node fields that will be processed in recursive descent.
	// Use the HasFields() method to check as it will establish validity of flags on first call.
	// If neither HasFields nor HasNoFields are set, then it knows to update flags.
	HasKiFields

	// HasNoKiFields indicates a node has NO Ki Node fields that will be processed in recursive descent.
	// Use the HasFields() method to check as it will establish validity of flags on first call.
	// If neither HasFields nor HasNoFields are set, then it knows to update flags.
	HasNoKiFields

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

	// FieldUpdated means a field was updated.
	FieldUpdated

	// PropUpdated means a property was set.
	PropUpdated

	// FlagsN is total number of flags used by base Ki Node -- can extend from
	// here up to 64 bits.
	FlagsN

	// ChildUpdateFlagsMask is a mask for all child updates.
	ChildUpdateFlagsMask = (1 << uint32(ChildAdded)) | (1 << uint32(ChildDeleted)) | (1 << uint32(ChildrenDeleted))

	// StruUpdateFlagsMask is a mask for all structural changes update flags.
	StruUpdateFlagsMask = ChildUpdateFlagsMask | (1 << uint32(NodeDeleted))

	// ValUpdateFlagsMask is a mask for all non-structural, value-only changes update flags.
	ValUpdateFlagsMask = (1 << uint32(FieldUpdated)) | (1 << uint32(PropUpdated))

	// UpdateFlagsMask is a Mask for all the update flags -- destroyed is
	// excluded b/c otherwise it would get cleared.
	UpdateFlagsMask = StruUpdateFlagsMask | ValUpdateFlagsMask
)
