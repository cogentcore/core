// Code generated by 'yaegi extract cogentcore.org/core/base/labels'. DO NOT EDIT.

package basesymbols

import (
	"cogentcore.org/core/base/labels"
	"reflect"
)

func init() {
	Symbols["cogentcore.org/core/base/labels/labels"] = map[string]reflect.Value{
		// function, constant and variable definitions
		"FriendlyMapLabel":    reflect.ValueOf(labels.FriendlyMapLabel),
		"FriendlySliceLabel":  reflect.ValueOf(labels.FriendlySliceLabel),
		"FriendlyStructLabel": reflect.ValueOf(labels.FriendlyStructLabel),
		"FriendlyTypeName":    reflect.ValueOf(labels.FriendlyTypeName),
		"ToLabel":             reflect.ValueOf(labels.ToLabel),
		"ToLabeler":           reflect.ValueOf(labels.ToLabeler),

		// type definitions
		"Labeler":      reflect.ValueOf((*labels.Labeler)(nil)),
		"SliceLabeler": reflect.ValueOf((*labels.SliceLabeler)(nil)),

		// interface wrapper definitions
		"_Labeler":      reflect.ValueOf((*_cogentcore_org_core_base_labels_Labeler)(nil)),
		"_SliceLabeler": reflect.ValueOf((*_cogentcore_org_core_base_labels_SliceLabeler)(nil)),
	}
}

// _cogentcore_org_core_base_labels_Labeler is an interface wrapper for Labeler type
type _cogentcore_org_core_base_labels_Labeler struct {
	IValue interface{}
	WLabel func() string
}

func (W _cogentcore_org_core_base_labels_Labeler) Label() string { return W.WLabel() }

// _cogentcore_org_core_base_labels_SliceLabeler is an interface wrapper for SliceLabeler type
type _cogentcore_org_core_base_labels_SliceLabeler struct {
	IValue     interface{}
	WElemLabel func(idx int) string
}

func (W _cogentcore_org_core_base_labels_SliceLabeler) ElemLabel(idx int) string {
	return W.WElemLabel(idx)
}