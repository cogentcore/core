// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gear

import (
	"reflect"
	"strconv"
	"strings"

	"github.com/iancoleman/strcase"
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/giv"
	"goki.dev/goosi/events"
	"goki.dev/grr"
	"goki.dev/ki/v2"
	"goki.dev/xe"
)

// App is a GUI view of a gear command.
type App struct {
	gi.Frame

	// Cmd is the root command associated with this app.
	Cmd *Cmd
}

var _ ki.Ki = (*App)(nil)

func (a *App) TopAppBar(tb *gi.TopAppBar) {
	gi.DefaultTopAppBarStd(tb)
	for _, cmd := range a.Cmd.Cmds {
		cmd := cmd
		gi.NewButton(tb).SetText(cmd.Name).SetTooltip(cmd.Doc).
			OnClick(func(e events.Event) {
				fields := strings.Fields(cmd.Cmd)
				grr.Log0(xe.Verbose().Run(fields[0], fields[1:]...))
			})
	}
}

func (a *App) ConfigWidget(sc *gi.Scene) {
	if a.HasChildren() {
		return
	}

	updt := a.UpdateStart()

	sfs := make([]reflect.StructField, len(a.Cmd.Flags))

	used := map[string]bool{}
	for i, flag := range a.Cmd.Flags {
		sf := reflect.StructField{
			Name: strcase.ToCamel(flag),
			// TODO(kai/gear): support type determination
			Type: reflect.TypeOf(""),
		}
		if used[sf.Name] {
			// TODO(kai/gear): consider better approach to unique names
			nm := sf.Name + "1"
			for i := 2; used[nm]; i++ {
				nm = sf.Name + strconv.Itoa(i)
			}
			sf.Name = nm
		}
		used[sf.Name] = true
		sfs[i] = sf
	}
	stt := reflect.StructOf(sfs)
	st := reflect.New(stt)

	giv.NewStructView(a).SetStruct(st.Interface())

	a.UpdateEnd(updt)
}
