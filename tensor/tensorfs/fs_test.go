// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensorfs

import (
	"fmt"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/assert"
)

func makeTestNode(t *testing.T) *Node {
	dfs, err := NewDir("root")
	assert.NoError(t, err)
	net := dfs.Dir("network")
	Value[float32](net, "units", 50, 50)
	dfs.Dir("log")
	return dfs
}

func TestFS(t *testing.T) {
	dfs := makeTestNode(t)
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
