// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensorview

import (
	"cogentcore.org/core/core"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/stats/simat"
	"cogentcore.org/core/tensor/table"
	"cogentcore.org/core/views"
)

func init() {
	// new:
	core.AddValueType[table.Table, *TableButton]()
	core.AddValueType[tensor.Float32, *TensorButton]()
	core.AddValueType[tensor.Float64, *TensorButton]()
	core.AddValueType[tensor.String, *TensorButton]()
	core.AddValueType[tensor.Int, *TensorButton]()
	core.AddValueType[simat.SimMat, *SimMatButton]()

	// old:
	// views.AddValue(tensor.Float32{}, func() views.Value { return &TensorValue{} })
	// views.AddValue(tensor.Float64{}, func() views.Value { return &TensorValue{} })
	// views.AddValue(tensor.Int{}, func() views.Value { return &TensorValue{} })
	// views.AddValue(tensor.Int32{}, func() views.Value { return &TensorValue{} })
	// views.AddValue(tensor.Byte{}, func() views.Value { return &TensorValue{} })
	// views.AddValue(tensor.String{}, func() views.Value { return &TensorValue{} })
	// views.AddValue(tensor.Bits{}, func() views.Value { return &TensorValue{} })
	// views.AddValue(table.Table{}, func() views.Value { return &TableValue{} })
	// views.AddValue(simat.SimMat{}, func() views.Value { return &SimMatValue{} })
}

// TensorGridValue manages a [TensorGrid] view of an [tensor.Tensor].
type TensorGridValue struct {
	views.ValueBase[*TensorGrid]
}

func (v *TensorGridValue) Config() {
	tsr := v.Value.Interface().(tensor.Tensor)
	v.Widget.SetTensor(tsr)
}

func (v *TensorGridValue) Update() {
	tsr := v.Value.Interface().(tensor.Tensor)
	v.Widget.SetTensor(tsr)
}

// TensorButton represents a Tensor with a button for making a [TensorView]
// viewer for an [tensor.Tensor].
type TensorButton struct {
	core.Button
	Tensor tensor.Tensor
}

func (tb *TensorButton) WidgetValue() any { return &tb.Tensor }

func (tb *TensorButton) OnInit() {
	tb.Button.OnInit()
	tb.SetType(core.ButtonTonal).SetIcon(icons.Edit)
	views.ConfigDialogValue(tb, true)
}

func (tb *TensorButton) Make(p *core.Plan) {
	txt := "nil"
	if tb.Tensor != nil {
		txt = "Tensor"
	}
	tb.SetText(txt)
	tb.Button.Make(p)
}

func (tb *TensorButton) ConfigDialog(d *core.Body) (bool, func()) {
	if tb.Tensor == nil {
		return false, nil
	}
	NewTensorGrid(d).SetTensor(tb.Tensor)
	return true, nil
}

// TableButton presents a button that pulls up the [TableView] viewer for a [table.Table].
type TableButton struct {
	core.Button
	Table *table.Table
}

func (tb *TableButton) WidgetValue() any { return &tb.Table }

func (tb *TableButton) OnInit() {
	tb.Button.OnInit()
	tb.SetType(core.ButtonTonal).SetIcon(icons.Edit)
	views.ConfigDialogValue(tb, true)
}

func (tb *TableButton) Make(p *core.Plan) {
	txt := "nil"
	if tb.Table != nil {
		if nm, has := tb.Table.MetaData["name"]; has {
			txt = nm
		} else {
			txt = "aTable"
		}
	}
	tb.SetText(txt)
	tb.Button.Make(p)
}

func (tb *TableButton) ConfigDialog(d *core.Body) (bool, func()) {
	if tb.Table == nil {
		return false, nil
	}
	NewTableView(d).SetTable(tb.Table)
	return true, nil
}

// SimMatValue presents a button that pulls up the [SimMatGridView] viewer for a [table.Table].
type SimMatButton struct {
	core.Button
	SimMat *simat.SimMat
}

func (tb *SimMatButton) WidgetValue() any { return &tb.SimMat }

func (tb *SimMatButton) OnInit() {
	tb.Button.OnInit()
	tb.SetType(core.ButtonTonal).SetIcon(icons.Edit)
	views.ConfigDialogValue(tb, true)
}

func (tb *SimMatButton) Make(p *core.Plan) {
	txt := "nil"
	if tb.SimMat != nil && tb.SimMat.Mat != nil {
		if nm, has := tb.SimMat.Mat.MetaData("name"); has {
			txt = nm
		} else {
			txt = "SimMat"
		}
	}
	tb.SetText(txt)
	tb.Button.Make(p)
}

func (tb *SimMatButton) ConfigDialog(d *core.Body) (bool, func()) {
	if tb.SimMat == nil || tb.SimMat.Mat == nil {
		return false, nil
	}
	NewSimMatGrid(d).SetSimMat(tb.SimMat)
	return true, nil
}
