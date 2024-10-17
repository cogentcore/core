// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package goal

import (
	"testing"

	"cogentcore.org/core/goal/goalib"
	"github.com/stretchr/testify/assert"
)

var test = `
// # x := [3, 5, 4]
# x := zeros(3, 4)
# nd := x.ndim
# sz := x.size
# sh := x.shape

fmt.Println(x)
fmt.Println(nd)
fmt.Println(sz)
fmt.Println(sh)

type MyStru struct {
	Name string
	Doc string
}

var VarCategories = []MyStru{
	{"Act", "basic activation variables, including conductances, current, Vm, spiking"},
	{"Learn", "calcium-based learning variables and other related learning factors"},
}
`

func TestMath(t *testing.T) {
	gl := NewGoal()
	tfile := "testdata/test.goal"
	ofile := "testdata/test.go"
	goalib.WriteFile(tfile, test)
	err := gl.TranspileFile(tfile, ofile)
	assert.NoError(t, err)
}
