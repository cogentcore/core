// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensorcore

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/stats/simat"
	"cogentcore.org/core/tensor/table"
)

func init() {
	core.AddValueType[table.Table, TableButton]()
	core.AddValueType[tensor.Float32, TensorButton]()
	core.AddValueType[tensor.Float64, TensorButton]()
	core.AddValueType[tensor.Int, TensorButton]()
	core.AddValueType[tensor.Int32, TensorButton]()
	core.AddValueType[tensor.Byte, TensorButton]()
	core.AddValueType[tensor.String, TensorButton]()
	core.AddValueType[tensor.Bits, TensorButton]()
	core.AddValueType[simat.SimMat, SimMatButton]()
}

// TensorButton represents a Tensor with a button for making a [TensorGrid]
// viewer for an [tensor.Tensor].
type TensorButton struct {
	core.Button
	Tensor tensor.Tensor
}

func (tb *TensorButton) WidgetValue() any { return &tb.Tensor }

func (tb *TensorButton) Init() {
	tb.Button.Init()
	tb.SetType(core.ButtonTonal).SetIcon(icons.Edit)
	tb.Updater(func() {
		text := "None"
		if tb.Tensor != nil {
			text = "Tensor"
		}
		tb.SetText(text)
	})
	core.InitValueButton(tb, true, func(d *core.Body) {
		NewTensorGrid(d).SetTensor(tb.Tensor)
	})
}

// TableButton presents a button that pulls up the [Table] viewer for a [table.Table].
type TableButton struct {
	core.Button
	Table *table.Table
}

func (tb *TableButton) WidgetValue() any { return &tb.Table }

func (tb *TableButton) Init() {
	tb.Button.Init()
	tb.SetType(core.ButtonTonal).SetIcon(icons.Edit)
	tb.Updater(func() {
		text := "None"
		if tb.Table != nil {
			if nm, has := tb.Table.MetaData["name"]; has {
				text = nm
			} else {
				text = "Table"
			}
		}
		tb.SetText(text)
	})
	core.InitValueButton(tb, true, func(d *core.Body) {
		NewTable(d).SetTable(tb.Table)
	})
}

// SimMatValue presents a button that pulls up the [SimMatGrid] viewer for a [table.Table].
type SimMatButton struct {
	core.Button
	SimMat *simat.SimMat
}

func (tb *SimMatButton) WidgetValue() any { return &tb.SimMat }

func (tb *SimMatButton) Init() {
	tb.Button.Init()
	tb.SetType(core.ButtonTonal).SetIcon(icons.Edit)
	tb.Updater(func() {
		text := "None"
		if tb.SimMat != nil && tb.SimMat.Mat != nil {
			if nm, has := tb.SimMat.Mat.MetaData("name"); has {
				text = nm
			} else {
				text = "SimMat"
			}
		}
		tb.SetText(text)
	})
	core.InitValueButton(tb, true, func(d *core.Body) {
		NewSimMatGrid(d).SetSimMat(tb.SimMat)
	})
}
