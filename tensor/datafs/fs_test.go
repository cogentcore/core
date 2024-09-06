// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package datafs

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func makeTestData(t *testing.T) *Data {
	dfs := NewDir("/")
	net, err := dfs.Mkdir("network")
	assert.NoError(t, err)
	NewTensor[float32]("units", net, []int{50, 50})
	log, err := dfs.Mkdir("log")
	assert.NoError(t, err)
	NewTable("Trial", log)
	return dfs
}

func TestFS(t *testing.T) {
	dfs := makeTestData(t)
	dirs, err := dfs.ReadDir(".")
	assert.NoError(t, err)
	for _, d := range dirs {
		fmt.Println(d.Name())
	}
}
