// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"embed"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/tensor/table"
	"cogentcore.org/core/tensor/tensorcore"
)

//go:embed *.tsv
var tsv embed.FS

func main() {
	pats := table.NewTable("pats")
	pats.SetMetaData("name", "TrainPats")
	pats.SetMetaData("desc", "Training patterns")
	// todo: meta data for grid size
	errors.Log(pats.OpenFS(tsv, "random_5x5_25.tsv", table.Tab))

	b := core.NewBody("grids")

	tv := core.NewTabs(b)

	// nt, _ := tv.NewTab("First")
	nt, _ := tv.NewTab("Patterns")
	etv := tensorcore.NewTable(nt).SetTable(pats)
	b.AddTopBar(func(bar *core.Frame) {
		core.NewToolbar(bar).Maker(etv.MakeToolbar)
	})

	b.RunMainWindow()
}
