// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package metadata

import (
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
