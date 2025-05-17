// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package search

import (
	"testing"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/core"
	"github.com/stretchr/testify/assert"
)

func TestSearchPaths(t *testing.T) {
	core.SystemSettings.BigFileSize = 10000000
	res, err := Paths([]string{"./"}, "package search", false, false, []fileinfo.Known{fileinfo.Go})
	assert.NoError(t, err)
	// for _, r := range res {
	// 	fmt.Println(r.String())
	// }
	assert.Equal(t, 4, len(res))

	res, err = Paths([]string{"./"}, "package search", false, false, []fileinfo.Known{fileinfo.C})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(res))

	res, err = Paths([]string{"./"}, "package .*", false, true, nil)
	assert.NoError(t, err)
	assert.Equal(t, 4, len(res))

	res, err = Paths([]string{"./"}, "package search", false, false, []fileinfo.Known{fileinfo.Go}, "*.go")
	assert.NoError(t, err)
	assert.Equal(t, 0, len(res))

	res, err = Paths([]string{"./"}, "package search", false, false, []fileinfo.Known{fileinfo.Go}, "all.go")
	assert.NoError(t, err)
	assert.Equal(t, 3, len(res))
}

func TestSearchAll(t *testing.T) {
	core.SystemSettings.BigFileSize = 10000000
	res, err := All("./", "package search", false, false, []fileinfo.Known{fileinfo.Go})
	assert.NoError(t, err)
	// for _, r := range res {
	// 	fmt.Println(r.String())
	// }
	assert.Equal(t, 4, len(res))

	res, err = All("./", "package search", false, false, []fileinfo.Known{fileinfo.C})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(res))

	res, err = All("./", "package .*", false, true, nil)
	assert.NoError(t, err)
	assert.Equal(t, 4, len(res))

	res, err = All("./", "package search", false, false, []fileinfo.Known{fileinfo.Go}, "*.go")
	assert.NoError(t, err)
	assert.Equal(t, 0, len(res))

	res, err = All("./", "package search", false, false, []fileinfo.Known{fileinfo.Go}, "all.go")
	assert.NoError(t, err)
	assert.Equal(t, 3, len(res))
}
