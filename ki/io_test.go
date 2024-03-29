// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

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
	assert.NoError(t, jsons.Write(&parent, &buf))
	b := buf.Bytes()

	tstload := testdata.NodeEmbed{}
	tstload.InitName(&tstload, "")
	if assert.NoError(t, jsons.Read(&tstload, bytes.NewReader(b))) {
		var buf2 bytes.Buffer
		assert.NoError(t, jsons.Write(tstload, &buf2))
		tstb := buf2.Bytes()
		if !bytes.Equal(tstb, b) {
			t.Error("original and unmarshal'd json rep are not equivalent")
		}
	}

	var bufn bytes.Buffer
	assert.NoError(t, ki.WriteNewJSON(parent.This(), &bufn))
	b = bufn.Bytes()
	nwnd, err := ki.ReadNewJSON(bytes.NewReader(b))
	if assert.NoError(t, err) {
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
	// child1 :=
	parent.NewChild(typ, "child1")
	var child2 = parent.NewChild(typ, "child1").(*testdata.NodeEmbed)
	// child3 :=
	parent.NewChild(typ, "child1")
	child2.NewChild(typ, "subchild1")

	var buf bytes.Buffer
	assert.NoError(t, parent.WriteXML(&buf, true))
	b := buf.Bytes()

	tstload := testdata.NodeEmbed{}
	tstload.InitName(&tstload, "")
	if assert.NoError(t, tstload.ReadXML(bytes.NewReader(b))) {
		var buf2 bytes.Buffer
		assert.NoError(t, tstload.WriteXML(&buf2, true))
		tstb := buf2.Bytes()
		if !bytes.Equal(tstb, b) {
			t.Error("original and unmarshal'd XML rep are not equivalent")
		}
	}
}
