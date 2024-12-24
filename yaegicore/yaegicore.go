// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package yaegicore provides functions connecting
// https://github.com/cogentcore/yaegi to Cogent Core.
package yaegicore

import (
	"fmt"
	"reflect"
	"strings"
	"sync/atomic"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/htmlcore"
	"cogentcore.org/core/texteditor"
	"cogentcore.org/core/yaegicore/basesymbols"
	"cogentcore.org/core/yaegicore/coresymbols"
	"github.com/cogentcore/yaegi/interp"
)

var autoPlanNameCounter uint64

func init() {
	htmlcore.BindTextEditor = BindTextEditor
	coresymbols.Symbols["."] = map[string]reflect.Value{} // make "." available for use
	basesymbols.Symbols["."] = map[string]reflect.Value{} // make "." available for use
}

// BindTextEditor binds the given text editor to a yaegi interpreter
// such that the contents of the text editor are interpreted as Go
// code, which is run in the context of the given parent widget.
// It is used as the default value of [htmlcore.BindTextEditor].
func BindTextEditor(ed *texteditor.Editor, parent core.Widget) {
	oc := func() {
		in := interp.New(interp.Options{})
		core.ExternalParent = parent
		coresymbols.Symbols["."]["b"] = reflect.ValueOf(parent)
		// the normal AutoPlanName cannot be used because the stack trace in yaegi is not helpful
		coresymbols.Symbols["cogentcore.org/core/tree/tree"]["AutoPlanName"] = reflect.ValueOf(func(int) string {
			return fmt.Sprintf("yaegi-%v", atomic.AddUint64(&autoPlanNameCounter, 1))
		})
		fmt.Println("will use")
		errors.Log(in.Use(basesymbols.Symbols))
		fmt.Println("did base")
		errors.Log(in.Use(coresymbols.Symbols))
		fmt.Println("did core")
		in.ImportUsed()
		fmt.Println("imported")

		parent.AsTree().DeleteChildren()
		str := ed.Buffer.String()
		// all code must be in a function for declarations to be handled correctly
		if !strings.Contains(str, "func main()") {
			str = "func main() {\n" + str + "\n}"
		}
		fmt.Println("will eval")
		_, err := in.Eval(str)
		fmt.Println("did eval")
		if err != nil {
			core.ErrorSnackbar(ed, err, "Error interpreting Go code")
			return
		}
		fmt.Println("will update")
		parent.AsWidget().Update()
		fmt.Println("did update")
	}
	ed.OnChange(func(e events.Event) { oc() })
	oc()
}
