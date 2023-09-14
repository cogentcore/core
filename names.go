// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

// Named consts for bool args
const (
	// Continue = true can be returned from tree iteration functions to continue
	// processing down the tree, as compared to Break = false which stops this branch.
	Continue = true

	// Break = false can be returned from tree iteration functions to stop processing
	// this branch of the tree.
	Break = false

	// Embeds is used for methods that look for children or parents of different types.
	// Passing this argument means to look for embedded types for matches.
	Embeds = true

	// NoEmbeds is used for methods that look for children or parents of different types.
	// Passing this argument means to NOT look for embedded types for matches.
	NoEmbeds = false

	// DestroyKids is used for Delete methods to indicate that deleted children
	// should be destroyed (else can be re-used somewhere else).
	DestroyKids = true

	// NoDestroyKids is used for Delete methods to indicate that deleted children
	// should NOT be destroyed, so they can be re-used somewhere else.
	NoDestroyKids = false

	// ShallowCopy is used for Props CopyFrom functions to indicate a shallow copy of
	// Props or PropSlice within Props (points to source props)
	ShallowCopy = true

	// DeepCopy is used for Props CopyFrom functions to indicate a deep copy of
	// Props or PropSlice within Props
	DeepCopy = true

	// Inherit is used for PropInherit to indicate that inherited properties
	// from parent objects should be checked as well.  Otherwise not.
	Inherit = true

	// NoInherit is used for PropInherit to indicate that inherited properties
	// from parent objects should NOT be checked.
	NoInherit = false

	// Indent is used for Write methods to indicate that indenting should be done.
	Indent = true

	// NoIndent is used for Write methods to indicate that indenting should NOT be done.
	NoIndent = false
)
