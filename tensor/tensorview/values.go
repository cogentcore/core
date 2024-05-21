// Copyright (c) 2019, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tensorview

import (
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/tensor"
	"cogentcore.org/core/tensor/stats/simat"
	"cogentcore.org/core/tensor/table"
	"cogentcore.org/core/views"
)

func init() {
	// new:
	core.AddValueType[tensor.Float32, *TensorButton]()
	core.AddValueType[table.Table, *TableButton]()

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
	*core.Button
	Tensor tensor.Tensor
}

func (v *TensorButton) WidgetValue() any { return &v.Tensor }

func (v *TensorButton) OnInit() {
	v.Button.OnInit()
	v.SetType(core.ButtonTonal).SetIcon(icons.Edit)
	views.ConfigDialogValue(v, true)
}

func (v *TensorButton) Make(p *core.Plan) {
	txt := "nil"
	if v.Tensor != nil {
		txt = "Tensor"
	}
	v.SetText(txt)
	v.Button.Make(p)
}

func (v *TensorButton) ConfigDialog(d *core.Body) (bool, func()) {
	if v.Tensor == nil {
		return false, nil
	}
	NewTensorGrid(d).SetTensor(v.Tensor)
	return true, nil
}

// TableButton presents a button that pulls up the [TableView] viewer for a [table.Table].
type TableButton struct {
	*core.Button
	Table *table.Table
}

func (v *TableButton) WidgetValue() any { return &v.Table }

func (v *TableButton) OnInit() {
	v.Button.OnInit()
	v.SetType(core.ButtonTonal).SetIcon(icons.Edit)
	views.ConfigDialogValue(v, true)
}

func (v *TableButton) Make(p *core.Plan) {
	txt := "nil"
	if v.Table != nil {
		if nm, has := v.Table.MetaData["name"]; has {
			txt = nm
		} else {
			txt = "Table"
		}
	}
	v.SetText(txt)
	v.Button.Make(p)
}

func (v *TableButton) ConfigDialog(d *core.Body) (bool, func()) {
	if v.Table == nil {
		return false, nil
	}
	NewTableView(d).SetTable(v.Table)
	return true, nil
}

// SimMatValue presents a button that pulls up the [SimMatGridView] viewer for a [table.Table].
type SimMatValue struct {
	views.ValueBase[*core.Button]
}

func (v *SimMatValue) Config() {
	v.Widget.SetType(core.ButtonTonal).SetIcon(icons.Edit)
	views.ConfigDialogWidget(v, false)
}

func (v *SimMatValue) Update() {
	npv := reflectx.NonPointerValue(v.Value)
	if !v.Value.IsValid() || v.Value.IsZero() || !npv.IsValid() || npv.IsZero() {
		v.Widget.SetText("nil")
	} else {
		opv := reflectx.UnderlyingPointer(v.Value)
		smat := opv.Interface().(*simat.SimMat)
		if smat != nil && smat.Mat != nil {
			if nm, has := smat.Mat.MetaData("name"); has {
				v.Widget.SetText(nm)
			} else {
				v.Widget.SetText("simat.SimMat")
			}
		} else {
			v.Widget.SetText("simat.SimMat")
		}
	}
}

func (v *SimMatValue) ConfigDialog(d *core.Body) (bool, func()) {
	opv := reflectx.UnderlyingPointer(v.Value)
	smat := opv.Interface().(*simat.SimMat)
	if smat == nil || smat.Mat == nil {
		return false, nil
	}
	NewSimMatGrid(d).SetSimMat(smat)
	return true, nil
}
