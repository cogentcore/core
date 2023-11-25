// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gear

import (
	"io"
	"reflect"
	"strconv"
	"strings"

	"github.com/iancoleman/strcase"
	"goki.dev/colors"
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/giv"
	"goki.dev/gi/v2/keyfun"
	"goki.dev/gi/v2/texteditor"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/glop/sentencecase"
	"goki.dev/goosi/events"
	"goki.dev/grr"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
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
		fields := strings.Fields(cmd.Cmd)
		text := sentencecase.Of(strcase.ToCamel(strings.Join(fields[1:], " ")))
		bt := gi.NewButton(tb).SetText(text).SetTooltip(cmd.Doc)
		bt.OnClick(func(e events.Event) {
			d := gi.NewBody().AddTitle(text).AddText(cmd.Doc)
			st := StructForFlags(cmd.Flags)
			giv.NewStructView(d).SetStruct(st)
			d.AddBottomBar(func(pw gi.Widget) {
				d.AddCancel(pw)
				d.AddOk(pw).SetText(text).OnClick(func(e events.Event) {
					grr.Log0(xe.Verbose().Run(fields[0], fields[1:]...))
				})
			})
			d.NewFullDialog(bt).Run()
		})
	}
}

func (a *App) ConfigWidget() {
	if a.HasChildren() {
		return
	}

	updt := a.UpdateStart()

	// st := StructForFlags(a.Cmd.Flags)
	// giv.NewStructView(a).SetStruct(st)

	sp := gi.NewSplits(a, "splits").SetDim(mat32.Y)

	cmds := gi.NewFrame(sp, "commands")
	cmds.Style(func(s *styles.Style) {
		s.Wrap = true
		s.Align.Content = styles.End
	})

	tb := texteditor.NewBuf()
	tb.NewBuf(0)
	tb.Hi.Lang = "sh"
	grr.Log0(tb.Stat())
	te := texteditor.NewEditor(sp, "editor").SetBuf(tb)
	te.OnKeyChord(func(e events.Event) {
		kf := keyfun.Of(e.KeyChord())
		if !(kf == keyfun.Enter && e.Modifiers() == 0) {
			return
		}
		e.SetHandled()
		cmd := string(tb.Text())
		tb.NewBuf(0)

		grr.Log0(a.RunCmd(cmd, cmds))
	})

	sp.SetSplits(0.8, 0.2)

	a.UpdateEnd(updt)
}

// RunCmd runs the given command in the context of the given commands frame.
func (a *App) RunCmd(cmd string, cmds *gi.Frame) error {
	updt := cmds.UpdateStart()

	cfr := gi.NewFrame(cmds).Style(func(s *styles.Style) {
		s.Grow.Set(1, 0)
		s.Direction = styles.Column
		s.Border.Radius = styles.BorderRadiusLarge
		s.BackgroundColor.SetSolid(colors.Scheme.SurfaceContainer)
	})
	gi.NewLabel(cfr, "cmd").SetText("$ " + cmd)

	r, w := io.Pipe()
	buf := texteditor.NewBuf()
	buf.NewBuf(0)
	buf.Opts.LineNos = false

	te := texteditor.NewEditor(cfr).SetBuf(buf)
	te.Style(func(s *styles.Style) {
		s.Min.Set(units.Em(30), units.Em(10))
	})

	ob := &texteditor.OutBuf{}
	ob.Init(r, buf, 0, func(line []byte) []byte { return line })
	go func() {
		ob.MonOut()
	}()

	cmds.Update()
	cmds.UpdateEnd(updt)

	xc := xe.Major().SetStdout(w).SetStderr(w).SetErrors(w)

	return xc.Run("bash", "-c", cmd)
}

// StructForFlags returns a new struct object for the given flags.
func StructForFlags(flags []*Flag) any {
	sfs := make([]reflect.StructField, len(flags))

	used := map[string]bool{}
	for i, flag := range flags {
		sf := reflect.StructField{}
		sf.Name = strings.Trim(flag.Name, "-[]")
		sf.Name = strcase.ToCamel(sf.Name)

		// TODO(kai/gear): better type determination
		if flag.Type == "bool" {
			sf.Type = reflect.TypeOf(false)
		} else if flag.Type == "int" {
			sf.Type = reflect.TypeOf(0)
		} else if flag.Type == "float" || flag.Type == "float32" || flag.Type == "float64" || flag.Type == "number" {
			sf.Type = reflect.TypeOf(0.0)
		} else {
			sf.Type = reflect.TypeOf("")
		}

		sf.Tag = reflect.StructTag(`desc:"` + flag.Doc + `"`)

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
	return st.Interface()
}
