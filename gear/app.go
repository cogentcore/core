// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gear

import (
	"reflect"

	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/giv"
	"goki.dev/ki/v2"
)

// App is a GUI view of a gear command.
type App struct {
	gi.Frame

	// Cmd is the root command associated with this app.
	Cmd *Cmd
}

var _ ki.Ki = (*App)(nil)

func (a *App) TopAppBar(tb *gi.TopAppBar) {
	for _, cmd := range a.Cmd.Cmds {
		gi.NewButton(tb).SetText(cmd.Name)
	}
}

func (a *App) ConfigWidget(sc *gi.Scene) {
	if a.HasChildren() {
		return
	}

	updt := a.UpdateStart()

	sfs := make([]reflect.StructField, len(a.Cmd.Flags))

	for i, flag := range a.Cmd.Flags {
		sf := reflect.StructField{
			Name: flag,
			// TODO(kai/gear): support type determination
			Type: reflect.TypeOf(""),
		}
		sfs[i] = sf
	}
	stt := reflect.StructOf(sfs)
	st := reflect.New(stt)

	giv.NewStructView(a).SetStruct(st)

	a.UpdateEnd(updt)
}
