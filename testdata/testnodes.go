// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:generate gtigen -type-var -type-method -new-method -instance

package testdata

import "goki.dev/ki/v2"

//gti:add
type TestNode struct {
	ki.Node
	Sig1 ki.Signal
	Sig2 ki.Signal
}

// func (tn *TestNode) CopyFieldsFrom(frm any) { // note nothing to copy here
// }

//gti:add
type NodeEmbed struct {
	ki.Node
	Mbr1 string
	Mbr2 int
}

// note: one _could_ do a direct literal bit copy, but children should not be overwritten etc.

// todo: gti auto-generates this method:

func (ne *NodeEmbed) CopyFieldsFrom(frm any) {
	ne.Node.CopyFieldsFrom(frm)
	fm, ok := frm.(*NodeEmbed)
	if !ok {
		return // todo: errors??
	}
	ne.Mbr1 = fm.Mbr1
	ne.Mbr2 = fm.Mbr2
}

var NodeEmbedProps = ki.Props{
	"intprop":    -17,
	"floatprop":  3.1415,
	"stringprop": "type string",
}

//gti:add
type NodeField struct {
	NodeEmbed
	Field1 NodeEmbed
}

func (nf *NodeField) CopyFieldsFrom(frm any) {
	nf.NodeEmbed.CopyFieldsFrom(frm)
	fm, ok := frm.(*NodeField)
	if !ok {
		return // todo: errors??
	}
	nf.Field1.CopyFrom(&fm.Field1) // use ki-specific method here -- hard for gti to know this..
}

func (nf *NodeField) FieldByName(field string) (ki.Ki, error) {
	if field == "Field1" {
		return &nf.Field1, nil
	}
	return nf.NodeEmbed.FieldByName(field)
}

func (nf *NodeField) Disconnect() {
	nf.Field1.DisconnectAll()
	nf.NodeEmbed.Disconnect()
}

//gti:add
type NodeField2 struct {
	NodeField
	Field2 NodeEmbed
}

func (nf *NodeField2) CopyFieldsFrom(frm any) {
	nf.NodeField.CopyFieldsFrom(frm)
	fm, ok := frm.(*NodeField2)
	if !ok {
		return // todo: errors??
	}
	nf.Field2.CopyFrom(&fm.Field2) // use ki-specific method here -- hard for gti to know this..
}

func (nf *NodeField2) FieldByName(field string) (ki.Ki, error) {
	if field == "Field2" {
		return &nf.Field2, nil
	}
	return nf.NodeField.FieldByName(field)
}

func (nf *NodeField2) Disconnect() {
	nf.Field2.DisconnectAll()
	nf.NodeField.Disconnect()
}
