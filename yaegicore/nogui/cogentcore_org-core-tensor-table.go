// Code generated by 'yaegi extract cogentcore.org/core/tensor/table'. DO NOT EDIT.

package nogui

import (
	"cogentcore.org/core/tensor/table"
	"reflect"
)

func init() {
	Symbols["cogentcore.org/core/tensor/table/table"] = map[string]reflect.Value{
		// function, constant and variable definitions
		"CleanCatTSV":            reflect.ValueOf(table.CleanCatTSV),
		"ConfigFromDataValues":   reflect.ValueOf(table.ConfigFromDataValues),
		"ConfigFromHeaders":      reflect.ValueOf(table.ConfigFromHeaders),
		"ConfigFromTableHeaders": reflect.ValueOf(table.ConfigFromTableHeaders),
		"DetectTableHeaders":     reflect.ValueOf(table.DetectTableHeaders),
		"Headers":                reflect.ValueOf(table.Headers),
		"InferDataType":          reflect.ValueOf(table.InferDataType),
		"New":                    reflect.ValueOf(table.New),
		"NewCols":                reflect.ValueOf(table.NewCols),
		"NewSliceTable":          reflect.ValueOf(table.NewSliceTable),
		"NewView":                reflect.ValueOf(table.NewView),
		"NoHeaders":              reflect.ValueOf(table.NoHeaders),
		"ShapeFromString":        reflect.ValueOf(table.ShapeFromString),
		"TableColumnType":        reflect.ValueOf(table.TableColumnType),
		"TableHeaderChar":        reflect.ValueOf(table.TableHeaderChar),
		"TableHeaderToType":      reflect.ValueOf(&table.TableHeaderToType).Elem(),
		"UpdateSliceTable":       reflect.ValueOf(table.UpdateSliceTable),

		// type definitions
		"Cols":       reflect.ValueOf((*table.Cols)(nil)),
		"FilterFunc": reflect.ValueOf((*table.FilterFunc)(nil)),
		"Table":      reflect.ValueOf((*table.Table)(nil)),
	}
}
