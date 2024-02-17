// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki_test

import (
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"

	"cogentcore.org/core/grows/jsons"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/ki/testdata"
)

func TestNodeJSON(t *testing.T) {
	parent := testdata.NodeEmbed{
		Node: ki.Node{
			Nm:              "",
			Flags:           0,
			Props:           ki.NewProps(),
			Par:             nil,
			Kids:            make(ki.Slice, 0),
			Ths:             nil,
			NumLifetimeKids: 0,
		},
		Mbr1: "",
		Mbr2: 0,
	}
	parent.InitName(&parent, "par1")
	typ := parent.KiType()
	parent.Mbr1 = "bloop"
	parent.Mbr2 = 32

	parent.NewChild(typ, "child1") // child1 :=
	child2 := parent.NewChild(typ, "child2").(*testdata.NodeEmbed)

	parent.NewChild(typ, "child3") // child3 :=
	child2.NewChild(typ, "subchild1")

	var buf bytes.Buffer
	assert.NoError(t, jsons.Write(&parent, &buf))

	jsons.SaveIndent(&parent, "json_test.json")
	fmt.Printf("json output:\n%v\n", buf.String())

	b := buf.Bytes()

	nodeEmbed := testdata.NodeEmbed{
		Node: ki.Node{
			Nm:              "",
			Flags:           0,
			Props:           ki.NewProps(),
			Par:             nil,
			Kids:            make(ki.Slice, 0),
			Ths:             nil,
			NumLifetimeKids: 0,
		},
		Mbr1: "",
		Mbr2: 0,
	}
	nodeEmbed.InitName(&nodeEmbed, "")

	assert.NoError(t, jsons.Read(&nodeEmbed, bytes.NewReader(b)))

	var buf2 bytes.Buffer
	assert.NoError(t, jsons.Write(nodeEmbed, &buf2))
	tstb := buf2.Bytes()
	fmt.Printf("test loaded json output: %v\n", buf2.String())
	assert.Equal(t, tstb, b)
	//if !bytes.Equal(tstb, b) {
	//	t.Error("original and unmarshal'd json rep are not equivalent")
	//}

	var bufn bytes.Buffer
	assert.NoError(t, ki.WriteNewJSON(parent.This(), &bufn))
	b = bufn.Bytes()
	readNewJSON, err := ki.ReadNewJSON(bytes.NewReader(b))
	assert.NoError(t, err)
	var buf3 bytes.Buffer
	assert.NoError(t, ki.WriteNewJSON(readNewJSON, &buf3))

	bb := buf3.Bytes()
	fmt.Printf("test loaded json output: %v\n", buf3.String())
	assert.Equal(t, bb, b)
	//if !bytes.Equal(bb, b) {
	//	t.Error("original and unmarshal'd json rep are not equivalent")
	//}
}

func TestNodeXML(t *testing.T) {
	parent := testdata.NodeEmbed{}
	parent.InitName(&parent, "par1")
	typ := parent.KiType()
	parent.Mbr1 = "bloop"
	parent.Mbr2 = 32

	parent.NewChild(typ, "child1") // child1 :=
	child2 := parent.NewChild(typ, "child1").(*testdata.NodeEmbed)

	parent.NewChild(typ, "child3") // child3 :=
	child2.NewChild(typ, "subchild1")

	var buf bytes.Buffer
	assert.NoError(t, parent.WriteXML(&buf, true))

	b := buf.Bytes()
	nodeEmbed := testdata.NodeEmbed{}
	nodeEmbed.InitName(&nodeEmbed, "")
	assert.NoError(t, nodeEmbed.ReadXML(&buf))

	var buf2 bytes.Buffer
	assert.NoError(t, nodeEmbed.WriteXML(&buf2, true))
	assert.Equal(t, buf2.Bytes(), b)
	//fmt.Printf("test loaded json output:\n%v\n", buf2.String())
}
