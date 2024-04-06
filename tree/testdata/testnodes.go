// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package testdata

//go:generate core generate

import "cogentcore.org/core/tree"

type TestNode struct {
	tree.NodeBase
}

// NodeEmbed embeds tree.Node and adds a couple of fields.
// Also has a directive processed by gti
//
//direct:value
type NodeEmbed struct {
	tree.NodeBase
	Mbr1 string
	Mbr2 int
}

type NodeField struct {
	NodeEmbed
	Field1 NodeEmbed
}

func (nf *NodeField) FieldByName(field string) (tree.Node, error) {
	if field == "Field1" {
		return &nf.Field1, nil
	}
	return nf.NodeEmbed.FieldByName(field)
}

type NodeField2 struct {
	NodeField
	Field2 NodeEmbed
}

func (nf *NodeField2) FieldByName(field string) (tree.Node, error) {
	if field == "Field2" {
		return &nf.Field2, nil
	}
	return nf.NodeField.FieldByName(field)
}
