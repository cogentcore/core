// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tree

// Named constants for bool arguments:
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
)
