// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on http://github.com/dmarkham/enumer and
// golang.org/x/tools/cmd/stringer:

// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package enumgen

import (
	"fmt"
	"log"
	"text/template"
)

// TmplData contains the data passed to a generation template
type TmplData struct {
	TypeName          string // the name of the enum type
	MinValue          string // the lowest defined value for the type as a string
	MaxValueP1        string // the highest defined value for the type, plus one, as a string
	IndexElementSize  int    // the size of the index element (8 for uint8, etc.)
	LessThanZeroCheck string // less than zero check (for signed types)
	MethodName        string // method name (String or BitIndexString)
	MethodComment     string // doc comment for the method
	IfInvalid         string // the code for what to do if the value is invalid (used in String and SetString)
	ReturnSlice       string // the code for the slice to return (used in Values, Strings, and Descs)
}

// ExecTmpl executes the given template with the given data and
// writes the result to [Generator.Buf]. It fatally logs any error.
func (g *Generator) ExecTmpl(t *template.Template, data *TmplData) {
	err := t.Execute(&g.Buf, data)
	if err != nil {
		log.Fatalf("programmer error: internal error: error executing template: %v", err)
	}
}

// SetMethod sets [TmplData.MethodName] and [TmplData.MethodComment]
// based on whether the type is a bit flag type. It is assumed
// that [TmplData.TypeName] is already set.
func (td *TmplData) SetMethod(isBitFlag bool) {
	if isBitFlag {
		td.MethodName = "BitIndexString"
		td.MethodComment = fmt.Sprintf(`// BitIndexString returns the string
		// representation of this %s value
		// if it is a bit index value
		// (typically an enum constant), and
		// not an actual bit flag value.`, td.TypeName)
	} else {
		td.MethodName = "String"
		td.MethodComment = fmt.Sprintf(`// String returns the string representation
		// of this %s value.`, td.TypeName)
	}
}

// SetIfInvalidForString sets [TmplData.IfInvalid] for a "String" method
// based on what type the type extends (none if passed ""), and how
// much the values of the type are offset (not at all if passed "" or "0").
// It assumes [TmplData.MethodName] is already set.
func (td *TmplData) SetIfInvalidForString(extends string, offset string) {
	if extends == "" {
		if offset == "" || offset == "0" {
			td.IfInvalid = `return strconv.FormatInt(int64(i), 10)`
		} else {
			td.IfInvalid = fmt.Sprintf(`return strconv.FormatInt(int64(i+%s), 10)`, offset)
		}
	} else {
		td.IfInvalid = fmt.Sprintf(`return %s(i).%s()`, extends, td.MethodName)
	}
}

// SetIfInvalidForSetString sets [TmplData.IfInvalid] for a "SetString" method
// based on what type the type extends (none if passed "") and whether it is a
// bitflag. It assumes [TmplData.TypeName] is are already set.
func (td *TmplData) SetIfInvalidForSetString(extends string, isBitFlag bool) {
	if extends == "" {
		if isBitFlag {
			td.IfInvalid = fmt.Sprintf(`return errors.New(flg+" is not a valid value for type %s")`, td.TypeName)
		} else {
			td.IfInvalid = fmt.Sprintf(`return errors.New(s+" is not a valid value for type %s")`, td.TypeName)
		}
	} else {
		if isBitFlag {
			td.IfInvalid = fmt.Sprintf(
				`err := (*%s)(i).SetString(flg)
				if err != nil {
					return err
				}`, extends)
		} else {
			td.IfInvalid = fmt.Sprintf(`return (*%s)(i).SetString(s)`, extends)

		}
	}
}
