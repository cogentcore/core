// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package testdata

//go:generate core generate

import "cogentcore.org/core/tree"

// NodeEmbed embeds tree.Node and adds a couple of fields.
// It also has a directive processed by typegen.
//
//direct:value
type NodeEmbed struct {
	tree.NodeBase
	Mbr1 string
	Mbr2 int
}
