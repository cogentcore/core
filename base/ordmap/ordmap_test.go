// Copyright (c) 2022, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ordmap

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMap(t *testing.T) {
	om := New[string, int]()
	om.Add("key0", 0)
	om.Add("key1", 1)
	om.Add("key2", 2)

	assert.Equal(t, 1, om.ValueByKey("key1"))
	assert.Equal(t, 2, om.IndexByKey("key2"))

	assert.Equal(t, "key0", om.KeyByIndex(0))
	assert.Equal(t, 1, om.ValueByIndex(1))

	assert.Equal(t, 3, om.Len())

	om.DeleteIndex(1, 2)
	assert.Equal(t, 2, om.ValueByIndex(1))
	assert.Equal(t, 1, om.IndexByKey("key2"))

	om.InsertAtIndex(0, "new0", 3)
	assert.Equal(t, 3, om.ValueByIndex(0))
	assert.Equal(t, 0, om.ValueByIndex(1))
	assert.Equal(t, 2, om.IndexByKey("key2"))

	nm := Make([]KeyValue[string, int]{{"one", 1}, {"two", 2}, {"three", 3}})

	assert.Equal(t, 3, nm.ValueByIndex(2))
	assert.Equal(t, 2, nm.ValueByIndex(1))
	assert.Equal(t, 3, nm.ValueByKey("three"))
}
