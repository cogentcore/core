// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package keylist

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKeyList(t *testing.T) {
	kl := New[string, int]()
	kl.Add("key0", 0)
	kl.Add("key1", 1)
	kl.Add("key2", 2)

	assert.Equal(t, 1, kl.ValueByKey("key1"))
	assert.Equal(t, 2, kl.IndexByKey("key2"))

	assert.Equal(t, 1, kl.Values[1])

	assert.Equal(t, 3, kl.Len())

	kl.DeleteIndex(1, 2)
	assert.Equal(t, 2, kl.Values[1])
	assert.Equal(t, 1, kl.IndexByKey("key2"))

	kl.Insert(0, "new0", 3)
	assert.Equal(t, 3, kl.Values[0])
	assert.Equal(t, 0, kl.Values[1])
	assert.Equal(t, 2, kl.IndexByKey("key2"))

	//	nm := Make([]KeyValue[string, int]{{"one", 1}, {"two", 2}, {"three", 3}})
	// assert.Equal(t, 3, nm.Values[2])
	// assert.Equal(t, 2, nm.Values[1])
	// assert.Equal(t, 3, nm.ValueByKey("three"))
}
