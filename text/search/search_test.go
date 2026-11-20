// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package search

import (
	"strings"
	"testing"

	"cogentcore.org/core/base/fileinfo"
	"cogentcore.org/core/core"
	"cogentcore.org/core/text/textpos"
	"github.com/stretchr/testify/assert"
)

func TestSearchPaths(t *testing.T) {
	core.SystemSettings.BigFileSize = 10000000
	res, err := Paths([]string{"./"}, "package search", textpos.UseCase, textpos.Literal, []fileinfo.Known{fileinfo.Go})
	assert.NoError(t, err)
	// for _, r := range res {
	// 	fmt.Println(r.String())
	// }
	assert.Equal(t, 4, len(res))

	res, err = Paths([]string{"./"}, "package search", textpos.UseCase, textpos.Literal, []fileinfo.Known{fileinfo.C})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(res))

	res, err = Paths([]string{"./"}, "package .*", textpos.UseCase, textpos.RegExp, nil)
	assert.NoError(t, err)
	assert.Equal(t, 4, len(res))

	res, err = Paths([]string{"./"}, "package search", textpos.UseCase, textpos.Literal, []fileinfo.Known{fileinfo.Go}, "*.go")
	assert.NoError(t, err)
	assert.Equal(t, 0, len(res))

	res, err = Paths([]string{"./"}, "package search", textpos.UseCase, textpos.Literal, []fileinfo.Known{fileinfo.Go}, "all.go")
	assert.NoError(t, err)
	assert.Equal(t, 3, len(res))
}

func TestSearchAll(t *testing.T) {
	core.SystemSettings.BigFileSize = 10000000
	res, err := All("./", "package search", textpos.UseCase, textpos.Literal, []fileinfo.Known{fileinfo.Go})
	assert.NoError(t, err)
	// for _, r := range res {
	// 	fmt.Println(r.String())
	// }
	assert.Equal(t, 4, len(res))

	res, err = All("./", "package search", textpos.UseCase, textpos.Literal, []fileinfo.Known{fileinfo.C})
	assert.NoError(t, err)
	assert.Equal(t, 0, len(res))

	res, err = All("./", "package .*", textpos.UseCase, textpos.RegExp, nil)
	assert.NoError(t, err)
	assert.Equal(t, 4, len(res))

	res, err = All("./", "package search", textpos.UseCase, textpos.Literal, []fileinfo.Known{fileinfo.Go}, "*.go")
	assert.NoError(t, err)
	assert.Equal(t, 0, len(res))

	res, err = All("./", "package search", textpos.UseCase, textpos.Literal, []fileinfo.Known{fileinfo.Go}, "all.go")
	assert.NoError(t, err)
	assert.Equal(t, 3, len(res))
}

// TestSearchCase tests case-sensitive and case-insensitive searches.
func TestSearchCase(t *testing.T) {
	src := "Apple apple\nAPPLE pear"
	n, _ := Reader(strings.NewReader(src), []byte("apple"), textpos.UseCase)
	assert.Equal(t, 1, n)
	n, ms := Reader(strings.NewReader(src), []byte("apple"), textpos.IgnoreCase)
	assert.Equal(t, 3, n)
	assert.Equal(t, 1, ms[2].Region.Start.Line)

	lns := [][]rune{[]rune("Apple apple"), []rune("APPLE pear")}
	n, _ = RuneLines(lns, []byte("APPLE"), textpos.UseCase)
	assert.Equal(t, 1, n)
	n, _ = RuneLines(lns, []byte("APPLE"), textpos.IgnoreCase)
	assert.Equal(t, 3, n)

	// only this file has the mixed case, while all of them have the lower case
	res, err := Paths([]string{"./"}, "PaCkAgE sEaRcH", textpos.UseCase, textpos.Literal, []fileinfo.Known{fileinfo.Go})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(res))
	res, err = Paths([]string{"./"}, "PaCkAgE sEaRcH", textpos.IgnoreCase, textpos.Literal, []fileinfo.Known{fileinfo.Go})
	assert.NoError(t, err)
	assert.Equal(t, 4, len(res))
}
