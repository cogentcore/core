// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package metadata

//core:js-test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

type mytest struct {
	Meta Data
}

func (mt *mytest) Metadata() *Data {
	return &mt.Meta
}

func TestMetadata(t *testing.T) {
	mt := &mytest{}

	SetName(mt, "test")
	assert.Equal(t, "test", Name(mt))

	SetDoc(mt, "this is good")
	assert.Equal(t, "this is good", Doc(mt))

	SetFilename(mt, "path/me.go")
	assert.Equal(t, "path/me.go", Filename(mt))
}

func TestMetadataOps(t *testing.T) {
	mt := &mytest{}
	assert.Nil(t, File(mt))
	f, err := os.CreateTemp(t.TempDir(), "meta")
	assert.NoError(t, err)
	defer f.Close()
	SetFile(mt, f)
	assert.Same(t, f, File(mt))

	assert.NoError(t, Set(mt, "count", 3))
	n, err := Get[int](mt, "count")
	assert.NoError(t, err)
	assert.Equal(t, 3, n)
	_, err = Get[string](mt, "count")
	assert.Error(t, err, "a different type")
	_, err = Get[int](mt, "missing")
	assert.Error(t, err)
	_, err = Get[int](3, "count")
	assert.Error(t, err, "no metadata for the type")
	assert.Error(t, Set(3, "count", 1))
	assert.Nil(t, GetData(3))

	other := &mytest{}
	other.Meta.Set("count", 7)
	other.Meta.Set("extra", "x")
	cp := Copy(mt, other)
	assert.Same(t, &mt.Meta, cp)
	n, _ = Get[int](mt, "count")
	assert.Equal(t, 7, n, "copied over")
	assert.Equal(t, "x", mt.Meta["extra"])
	assert.Nil(t, Copy(3, other))
	assert.Same(t, &mt.Meta, Copy(mt, 3), "nothing to copy from")

	var d Data
	d.Copy(nil)
	assert.Nil(t, d)
	d.Copy(Data{"a": 1})
	assert.Equal(t, 1, d["a"])
	v, err := GetFromData[int](d, "a")
	assert.NoError(t, err)
	assert.Equal(t, 1, v)
}
