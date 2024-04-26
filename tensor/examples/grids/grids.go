// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"embed"

	"cogentcore.org/core/core"
	"cogentcore.org/core/errors"
	"cogentcore.org/core/tensor/table"
	"cogentcore.org/core/tensor/tensorview"
)

//go:embed *.tsv
var tsv embed.FS

func main() {
	pats := table.NewTable(0, "pats")
	pats.SetMetaData("name", "TrainPats")
	pats.SetMetaData("desc", "Training patterns")
	// todo: meta data for grid size
	errors.Log(pats.OpenFS(tsv, "random_5x5_25.tsv", table.Tab))

	b := core.NewBody("grids")

	tv := core.NewTabs(b)

	// nt := tv.NewTab("First")
	nt := tv.NewTab("Patterns")
	etv := tensorview.NewTableView(nt).SetTable(pats)
	b.AddAppBar(etv.ConfigToolbar)

	b.RunMainWindow()
}
