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
}

// ExecTmpl executes the given template with the given data and
// writes the result to [Generator.Buf]. It logs any error in
// addition to returning it, so callers do not have to handle tbe error.
func (g *Generator) ExecTmpl(t *template.Template, data *TmplData) error {
	err := t.Execute(&g.Buf, data)
	if err != nil {
		werr := fmt.Errorf("error executing template: %w", err)
		log.Println(werr)
		return werr
	}
	return nil
}

// SetMethod sets [TmplData.MethodName] and [TmplData.MethodComment]
// based on whether the type is a bit flag type. It is assumed
// that [TmplData.TypeName] is already set.
func (td *TmplData) SetMethod(isBitFlag bool) {
	if isBitFlag {
		td.MethodName = BitIndexStringMethodName
		td.MethodComment = fmt.Sprintf(BitIndexStringMethodComment, td.TypeName)
	} else {
		td.MethodName = StringMethodName
		td.MethodComment = fmt.Sprintf(StringMethodComment, td.TypeName)
	}
}
