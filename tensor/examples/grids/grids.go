// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"embed"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/metadata"
	"cogentcore.org/core/core"
	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/table"
	"cogentcore.org/core/tensor/tensorcore"
)

//go:embed *.tsv
var tsv embed.FS

func main() {
	pats := table.New("TrainPats")
	metadata.SetDoc(pats, "Training patterns")
	// todo: meta data for grid size
	errors.Log(pats.OpenFS(tsv, "random_5x5_25.tsv", tensor.Tab))

	b := core.NewBody("grids")
	tv := core.NewTabs(b)
	nt, _ := tv.NewTab("Patterns")
	etv := tensorcore.NewTable(nt)
	tensorcore.AddGridStylerTo(pats, func(s *tensorcore.GridStyle) {
		s.TotalSize = 200
	})
	etv.SetTable(pats)
	b.AddTopBar(func(bar *core.Frame) {
		core.NewToolbar(bar).Maker(etv.MakeToolbar)
	})

	b.RunMainWindow()
}
