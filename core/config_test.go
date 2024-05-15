// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"runtime"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	var c Config
	c.Add("parts", nil, nil)
	c.Add("parts/icon", nil, nil)
	c.Add("parts/text", nil, nil)
	c.Add("tree", nil, nil)
	c.Add("tree/child1", nil, nil)
	c.Add("parts/icon/parts", nil, nil) // still works out of order
	c.Add("tree/child2", nil, nil)
	expected := "parts\nparts/icon\nparts/icon/parts\nparts/text\ntree\ntree/child1\ntree/child2\n"
	assert.Equal(t, expected, c.String())
}

func BenchmarkCaller(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, file, line, _ := runtime.Caller(1)
		name := file + ":" + strconv.Itoa(line)
		_ = name
	}
}
