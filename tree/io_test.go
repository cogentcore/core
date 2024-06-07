// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tree_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"cogentcore.org/core/base/iox/jsonx"
	. "cogentcore.org/core/tree"
	"cogentcore.org/core/tree/testdata"
)

func testNodeTree() *testdata.NodeEmbed {
	parent := testdata.NewNodeEmbed()
	parent.Mbr1 = "bloop"
	parent.Mbr2 = 32
	child1 := testdata.NewNodeEmbed(parent)
	child1.SetName("child1")
	child2 := testdata.NewNodeEmbed(parent)
	child2.SetName("child2")
	child3 := testdata.NewNodeEmbed(parent)
	child3.SetName("child3")
	schild1 := testdata.NewNodeEmbed(child2)
	schild1.SetName("subchild1")
	return parent
}

func TestNodeJSON(t *testing.T) {
	parent := testNodeTree()

	var buf bytes.Buffer
	assert.NoError(t, jsonx.Write(&parent, &buf))
	b := buf.Bytes()

	tstload := testdata.NewNodeEmbed()
	if assert.NoError(t, jsonx.Read(&tstload, bytes.NewReader(b))) {
		assert.Equal(t, parent.Mbr1, tstload.Mbr1)
		assert.Equal(t, parent.Mbr2, tstload.Mbr2)
		var buf2 bytes.Buffer
		assert.NoError(t, jsonx.Write(tstload, &buf2))
		tstb := buf2.Bytes()
		assert.Equal(t, string(b), string(tstb))
	}

	var bufn bytes.Buffer
	assert.NoError(t, WriteNewJSON(parent.This(), &bufn))
	b = bufn.Bytes()
	nwnd, err := ReadNewJSON(bytes.NewReader(b))
	if assert.NoError(t, err) {
		var buf2 bytes.Buffer
		err = WriteNewJSON(nwnd, &buf2)
		if err != nil {
			t.Error(err)
		}
		tstb := buf2.Bytes()
		assert.Equal(t, string(b), string(tstb))
	}
}

func BenchmarkNodeMarshalJSON(b *testing.B) {
	parent := testNodeTree()
	for range b.N {
		var buf bytes.Buffer
		assert.NoError(b, jsonx.Write(&parent, &buf))
	}
}

/* TODO(config): add back tree xml support
func TestNodeXML(t *testing.T) {
	parent := testNodeTree()

	var buf bytes.Buffer
	assert.NoError(t, parent.WriteXML(&buf, true))
	b := buf.Bytes()

	tstload := testdata.NewNodeEmbed()
	if assert.NoError(t, tstload.ReadXML(bytes.NewReader(b))) {
		var buf2 bytes.Buffer
		assert.NoError(t, tstload.WriteXML(&buf2, true))
		tstb := buf2.Bytes()
		assert.Equal(t, string(b), string(tstb))
	}
}
*/
