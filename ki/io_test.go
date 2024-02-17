// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki_test

import (
	"bytes"
	"github.com/stretchr/testify/assert"
	"testing"

	"cogentcore.org/core/grows/jsons"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/ki/testdata"
)

func TestNodeJSON(t *testing.T) {
	parent := testdata.NodeEmbed{}
	parent.InitName(&parent, "par1")
	typ := parent.KiType()
	parent.Mbr1 = "bloop"
	parent.Mbr2 = 32
	// child1 :=
	parent.NewChild(typ, "child1")
	var child2 = parent.NewChild(typ, "child2").(*testdata.NodeEmbed)
	// child3 :=
	parent.NewChild(typ, "child3")
	child2.NewChild(typ, "subchild1")

	var buf bytes.Buffer
	err := jsons.Write(&parent, &buf)
	if err != nil {
		t.Error(err)
	} else {
		// jsons.SaveIndent(&parent, "json_test.json")
		// fmt.Printf("json output:\n%v\n", string(buf.Bytes()))
	}
	b := buf.Bytes()

	tstload := testdata.NodeEmbed{}
	tstload.InitName(&tstload, "")
	err = jsons.Read(&tstload, bytes.NewReader(b))
	if err != nil {
		t.Error(err)
	} else {
		var buf2 bytes.Buffer
		err = jsons.Write(tstload, &buf2)
		if err != nil {
			t.Error(err)
		}
		tstb := buf2.Bytes()
		// fmt.Printf("test loaded json output: %v\n", string(tstb))
		if !bytes.Equal(tstb, b) {
			t.Error("original and unmarshal'd json rep are not equivalent")
		}
	}

	var bufn bytes.Buffer
	err = ki.WriteNewJSON(parent.This(), &bufn)
	b = bufn.Bytes()
	nwnd, err := ki.ReadNewJSON(bytes.NewReader(b))
	if err != nil {
		t.Error(err)
	} else {
		var buf2 bytes.Buffer
		err = ki.WriteNewJSON(nwnd, &buf2)
		if err != nil {
			t.Error(err)
		}
		tstb := buf2.Bytes()
		// fmt.Printf("test loaded json output: %v\n", string(tstb))
		if !bytes.Equal(tstb, b) {
			t.Error("original and unmarshal'd json rep are not equivalent")
		}
	}
}

func TestNodeXML(t *testing.T) {
	parent := testdata.NodeEmbed{}
	parent.InitName(&parent, "par1")
	typ := parent.KiType()
	parent.Mbr1 = "bloop"
	parent.Mbr2 = 32

	parent.NewChild(typ, "child1") // child1 :=
	child2 := parent.NewChild(typ, "child1").(*testdata.NodeEmbed)

	parent.NewChild(typ, "child1") // child3 :=
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
	// fmt.Printf("test loaded json output:\n%v\n", string(tstb))
}
