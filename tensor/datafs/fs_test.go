// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package datafs

import (
	"fmt"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
)

func makeTestData(t *testing.T) *Data {
	dfs, err := NewDir("root")
	assert.NoError(t, err)
	net, err := dfs.Mkdir("network")
	assert.NoError(t, err)
	NewTensor[float32](net, "units", []int{50, 50})
	log, err := dfs.Mkdir("log")
	assert.NoError(t, err)
	_, err = NewTable(log, "Trial")
	assert.NoError(t, err)
	return dfs
}

func TestFS(t *testing.T) {
	dfs := makeTestData(t)
	dirs, err := dfs.ReadDir(".")
	assert.NoError(t, err)
	for _, d := range dirs {
		fmt.Println(d.Name())
	}
	sd, err := dfs.DirAtPath("root")
	assert.NoError(t, err)
	sd, err = sd.DirAtPath("network")
	assert.NoError(t, err)
	sd, err = dfs.DirAtPath("root/network")
	assert.NoError(t, err)

	if err := fstest.TestFS(dfs, "network/units"); err != nil {
		t.Fatal(err)
	}
}
