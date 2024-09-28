// Code generated by 'yaegi extract cogentcore.org/core/tensor/datafs'. DO NOT EDIT.

package symbols

import (
	"cogentcore.org/core/tensor/datafs"
	"reflect"
)

func init() {
	Symbols["cogentcore.org/core/tensor/datafs/datafs"] = map[string]reflect.Value{
		// function, constant and variable definitions
		"Chdir":     reflect.ValueOf(datafs.Chdir),
		"CurDir":    reflect.ValueOf(&datafs.CurDir).Elem(),
		"CurRoot":   reflect.ValueOf(&datafs.CurRoot).Elem(),
		"DirOnly":   reflect.ValueOf(datafs.DirOnly),
		"Get":       reflect.ValueOf(datafs.Get),
		"List":      reflect.ValueOf(datafs.List),
		"Long":      reflect.ValueOf(datafs.Long),
		"Mkdir":     reflect.ValueOf(datafs.Mkdir),
		"NewDir":    reflect.ValueOf(datafs.NewDir),
		"Overwrite": reflect.ValueOf(datafs.Overwrite),
		"Preserve":  reflect.ValueOf(datafs.Preserve),
		"Record":    reflect.ValueOf(datafs.Record),
		"Recursive": reflect.ValueOf(datafs.Recursive),
		"Set":       reflect.ValueOf(datafs.Set),
		"Short":     reflect.ValueOf(datafs.Short),

		// type definitions
		"Data":    reflect.ValueOf((*datafs.Data)(nil)),
		"DirFile": reflect.ValueOf((*datafs.DirFile)(nil)),
		"File":    reflect.ValueOf((*datafs.File)(nil)),
	}
}
