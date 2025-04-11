// Code generated by 'yaegi extract cogentcore.org/core/text/lines'. DO NOT EDIT.

package coresymbols

import (
	"cogentcore.org/core/text/lines"
	"reflect"
)

func init() {
	Symbols["cogentcore.org/core/text/lines/lines"] = map[string]reflect.Value{
		// function, constant and variable definitions
		"ApplyOneDiff":           reflect.ValueOf(lines.ApplyOneDiff),
		"BytesToLineStrings":     reflect.ValueOf(lines.BytesToLineStrings),
		"CountWordsLines":        reflect.ValueOf(lines.CountWordsLines),
		"CountWordsLinesRegion":  reflect.ValueOf(lines.CountWordsLinesRegion),
		"DiffLines":              reflect.ValueOf(lines.DiffLines),
		"DiffLinesUnified":       reflect.ValueOf(lines.DiffLinesUnified),
		"DiffOpReverse":          reflect.ValueOf(lines.DiffOpReverse),
		"DiffOpString":           reflect.ValueOf(lines.DiffOpString),
		"FileBytes":              reflect.ValueOf(lines.FileBytes),
		"FileRegionBytes":        reflect.ValueOf(lines.FileRegionBytes),
		"KnownComments":          reflect.ValueOf(lines.KnownComments),
		"NewDiffSelected":        reflect.ValueOf(lines.NewDiffSelected),
		"NewLines":               reflect.ValueOf(lines.NewLines),
		"NewLinesFromBytes":      reflect.ValueOf(lines.NewLinesFromBytes),
		"NextSpace":              reflect.ValueOf(lines.NextSpace),
		"PreCommentStart":        reflect.ValueOf(lines.PreCommentStart),
		"ReplaceMatchCase":       reflect.ValueOf(lines.ReplaceMatchCase),
		"ReplaceNoMatchCase":     reflect.ValueOf(lines.ReplaceNoMatchCase),
		"StringLinesToByteLines": reflect.ValueOf(lines.StringLinesToByteLines),
		"UndoGroupDelay":         reflect.ValueOf(&lines.UndoGroupDelay).Elem(),
		"UndoTrace":              reflect.ValueOf(&lines.UndoTrace).Elem(),

		// type definitions
		"DiffSelectData": reflect.ValueOf((*lines.DiffSelectData)(nil)),
		"DiffSelected":   reflect.ValueOf((*lines.DiffSelected)(nil)),
		"Diffs":          reflect.ValueOf((*lines.Diffs)(nil)),
		"Lines":          reflect.ValueOf((*lines.Lines)(nil)),
		"Patch":          reflect.ValueOf((*lines.Patch)(nil)),
		"PatchRec":       reflect.ValueOf((*lines.PatchRec)(nil)),
		"Settings":       reflect.ValueOf((*lines.Settings)(nil)),
		"Undo":           reflect.ValueOf((*lines.Undo)(nil)),
	}
}
