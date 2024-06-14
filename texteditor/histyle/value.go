// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package histyle

/* TODO(config)
import (
	"cogentcore.org/core/base/reflectx"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
)

func init() {
	core.AddValue(core.HiStyleName(""), func() core.Value { return &Value{} })
}

// Value represents a [core.HiStyleName] with a button.
type Value struct {
	core.ValueBase[*core.Button]
}

func (v *Value) Config() {
	v.Widget.SetType(core.ButtonTonal).SetIcon(icons.Brush)
	core.ConfigDialogWidget(v, false)
}

func (v *Value) Update() {
	txt := reflectx.ToString(v.Value.Interface())
	v.Widget.SetText(txt).Update()
}

func (v *Value) ConfigDialog(d *core.Body) (bool, func()) {
	d.SetTitle("Select a syntax highlighting style")
	si := 0
	cur := reflectx.ToString(v.Value.Interface())
	core.NewList(d).SetSlice(&StyleNames).SetSelectedValue(cur).BindSelect(&si)
	return true, func() {
		if si >= 0 {
			hs := StyleNames[si]
			v.SetValue(hs)
			v.Update()
		}
	}
}

// View opens a view of highlighting styles
func View(st *Styles) {
	if core.RecycleMainWindow(st) {
		return
	}

	d := core.NewBody("hi-styles").SetData(st)
	d.AddTitle("Highlighting Styles: use ViewStd to see builtin ones -- can add and customize -- save ones from standard and load into custom to modify standards.")
	mv := core.NewKeyedList(d).SetMap(st)
	StylesChanged = false
	mv.OnChange(func(e events.Event) {
		StylesChanged = true
	})
	d.AddAppBar(func(p *core.Plan) {
		core.Add(p, func(w *core.FuncButton) {
			w.SetFunc(st.OpenJSON).SetText("Open from file").SetIcon(icons.Open)
			w.Args[0].SetTag(`ext:".histy"`)
		})
		core.Add(p, func(w *core.FuncButton) {
			w.SetFunc(st.SaveJSON).SetText("Save from file").SetIcon(icons.Save)
			w.Args[0].SetTag(`ext:".histy"`)
		})
		core.Add(p, func(w *core.Separator) {})
		mv.MakeToolbar(p)
	})
	d.RunWindow() // note: no context here so not dialog
}
*/
