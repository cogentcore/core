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
	"log"
	"text/template"
)

// ExecTmpl executes the given template with the given type and
// writes the result to [Generator.Buf]. It fatally logs any error.
// All enumgen templates take a [Type] as their data.
func (g *Generator) ExecTmpl(t *template.Template, typ *Type) {
	err := t.Execute(&g.Buf, typ)
	if err != nil {
		log.Fatalf("programmer error: internal error: error executing template: %v", err)
	}
}
