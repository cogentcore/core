// Code generated by 'yaegi extract cogentcore.org/core/text/textpos'. DO NOT EDIT.

package coresymbols

import (
	"cogentcore.org/core/text/textpos"
	"reflect"
)

func init() {
	Symbols["cogentcore.org/core/text/textpos/textpos"] = map[string]reflect.Value{
		// function, constant and variable definitions
		"AdjustPosDelEnd":    reflect.ValueOf(textpos.AdjustPosDelEnd),
		"AdjustPosDelErr":    reflect.ValueOf(textpos.AdjustPosDelErr),
		"AdjustPosDelN":      reflect.ValueOf(textpos.AdjustPosDelN),
		"AdjustPosDelStart":  reflect.ValueOf(textpos.AdjustPosDelStart),
		"AdjustPosDelValues": reflect.ValueOf(textpos.AdjustPosDelValues),
		"BackwardWord":       reflect.ValueOf(textpos.BackwardWord),
		"ForwardWord":        reflect.ValueOf(textpos.ForwardWord),
		"IgnoreCase":         reflect.ValueOf(textpos.IgnoreCase),
		"IsWordBreak":        reflect.ValueOf(textpos.IsWordBreak),
		"MatchContext":       reflect.ValueOf(&textpos.MatchContext).Elem(),
		"NewEditFromRunes":   reflect.ValueOf(textpos.NewEditFromRunes),
		"NewMatch":           reflect.ValueOf(textpos.NewMatch),
		"NewRegion":          reflect.ValueOf(textpos.NewRegion),
		"NewRegionLen":       reflect.ValueOf(textpos.NewRegionLen),
		"NewRegionPos":       reflect.ValueOf(textpos.NewRegionPos),
		"PosErr":             reflect.ValueOf(&textpos.PosErr).Elem(),
		"PosZero":            reflect.ValueOf(&textpos.PosZero).Elem(),
		"RegionZero":         reflect.ValueOf(&textpos.RegionZero).Elem(),
		"RuneIsWordBreak":    reflect.ValueOf(textpos.RuneIsWordBreak),
		"UseCase":            reflect.ValueOf(textpos.UseCase),
		"WordAt":             reflect.ValueOf(textpos.WordAt),

		// type definitions
		"AdjustPosDel": reflect.ValueOf((*textpos.AdjustPosDel)(nil)),
		"Edit":         reflect.ValueOf((*textpos.Edit)(nil)),
		"Match":        reflect.ValueOf((*textpos.Match)(nil)),
		"Pos":          reflect.ValueOf((*textpos.Pos)(nil)),
		"Pos16":        reflect.ValueOf((*textpos.Pos16)(nil)),
		"Range":        reflect.ValueOf((*textpos.Range)(nil)),
		"Region":       reflect.ValueOf((*textpos.Region)(nil)),
	}
}
