// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"reflect"

	"cogentcore.org/core/base/iox/jsonx"
	"cogentcore.org/core/base/labels"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/system"
	"cogentcore.org/core/tree"
)

// Inspector represents a struct, creating a property editor of the fields --
// constructs Children widgets to show the field names and editor fields for
// each field, within an overall frame with an optional title, and a button
// box at the bottom where methods can be invoked
type Inspector struct {
	Frame

	// Root is the root of the tree being edited.
	Root tree.Node

	// CurrentNode is the currently selected node in the tree.
	CurrentNode tree.Node `set:"-"`

	// Filename is the current filename for saving / loading
	Filename Filename `set:"-"`
}

func (is *Inspector) Init() {
	is.Frame.Init()
	is.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
		s.Direction = styles.Column
	})
	is.OnWidgetAdded(func(w Widget) {
		// TODO(config)
		if tw, ok := w.(*Tree); ok {
			tw.Styler(func(s *styles.Style) {
				s.Max.X.Em(20)
			})
		}
	})

	var titleWidget *Text
	AddChildAt(is, "title", func(w *Text) {
		titleWidget = w
		is.CurrentNode = is.Root
		w.SetType(TextHeadlineSmall)
		w.Styler(func(s *styles.Style) {
			s.Grow.Set(1, 0)
			s.Align.Self = styles.Center
		})
		w.Updater(func() {
			w.SetText(fmt.Sprintf("Inspector of %s (%s)", is.CurrentNode.AsTree().Name, labels.FriendlyTypeName(reflect.TypeOf(is.CurrentNode))))
		})
	})
	renderRebuild := func() {
		sc, ok := is.Root.(*Scene)
		if !ok {
			return
		}
		sc.RenderContext().Rebuild = true // trigger full rebuild
	}
	AddChildAt(is, "splits", func(w *Splits) {
		w.SetSplits(.3, .7)
		var form *Form
		AddChildAt(w, "tree-frame", func(w *Frame) {
			w.Styler(func(s *styles.Style) {
				s.Direction = styles.Column
				s.Overflow.Set(styles.OverflowAuto)
				s.Gap.Zero()
			})
			AddChildAt(w, "tree", func(w *Tree) {
				w.OnSelect(func(e events.Event) {
					if len(w.SelectedNodes) == 0 {
						return
					}
					sn := w.SelectedNodes[0].AsCoreTree().SyncNode
					is.CurrentNode = sn
					// Note: doing Update on the entire inspector reverts all tree expansion,
					// so we only want to update the title and form
					titleWidget.Update()
					form.SetStruct(sn).Update()

					sc, ok := is.Root.(*Scene)
					if !ok {
						return
					}
					if w, wb := AsWidget(sn); w != nil {
						pselw := sc.selectedWidget
						sc.selectedWidget = w
						wb.NeedsRender()
						if pselw != nil {
							pselw.AsWidget().NeedsRender()
						}
					}
				})
				w.OnChange(func(e events.Event) {
					renderRebuild()
				})
				w.SyncTree(is.Root)
			})
		})
		AddChildAt(w, "struct", func(w *Form) {
			form = w
			w.OnChange(func(e events.Event) {
				renderRebuild()
			})
			w.OnClose(func(e events.Event) {
				sc, ok := is.Root.(*Scene)
				if !ok {
					return
				}
				if sc.renderBBoxes {
					is.ToggleSelectionMode()
				}
				pselw := sc.selectedWidget
				sc.selectedWidget = nil
				if pselw != nil {
					pselw.AsWidget().NeedsRender()
				}
			})
			w.Updater(func() {
				w.SetStruct(is.CurrentNode)
			})
		})
	})
}

// Save saves tree to current filename, in a standard JSON-formatted file
func (is *Inspector) Save() error { //types:add
	if is.Root == nil {
		return nil
	}
	if is.Filename == "" {
		return nil
	}

	err := jsonx.Save(is.Root, string(is.Filename))
	if err != nil {
		return err
	}
	return nil
}

// SaveAs saves tree to given filename, in a standard JSON-formatted file
func (is *Inspector) SaveAs(filename Filename) error { //types:add
	if is.Root == nil {
		return nil
	}
	err := jsonx.Save(is.Root, string(filename))
	if err != nil {
		return err
	}
	is.Filename = filename
	is.NeedsRender() // notify our editor
	return nil
}

// Open opens tree from given filename, in a standard JSON-formatted file
func (is *Inspector) Open(filename Filename) error { //types:add
	if is.Root == nil {
		return nil
	}
	err := jsonx.Open(is.Root, string(filename))
	if err != nil {
		return err
	}
	is.Filename = filename
	is.NeedsRender() // notify our editor
	return nil
}

// ToggleSelectionMode toggles the editor between selection mode or not.
// In selection mode, bounding boxes are rendered around each Widget,
// and clicking on a Widget pulls it up in the inspector.
func (is *Inspector) ToggleSelectionMode() { //types:add
	sc, ok := is.Root.(*Scene)
	if !ok {
		return
	}
	sc.renderBBoxes = !sc.renderBBoxes
	if sc.renderBBoxes {
		sc.selectedWidgetChan = make(chan Widget)
		go is.SelectionMonitor()
	} else {
		if sc.selectedWidgetChan != nil {
			close(sc.selectedWidgetChan)
		}
		sc.selectedWidgetChan = nil
	}
	sc.NeedsLayout()
}

// SelectionMonitor monitors for the selected widget
func (is *Inspector) SelectionMonitor() {
	sc, ok := is.Root.(*Scene)
	if !ok {
		return
	}
	sc.Stage.Raise()
	sw, ok := <-sc.selectedWidgetChan
	if !ok || sw == nil {
		return
	}
	tv := is.Tree().FindSyncNode(sw)
	if tv == nil {
		// if we can't be found, we are probably a part,
		// so we keep going up until we find somebody in
		// the tree
		sw.AsTree().WalkUpParent(func(k tree.Node) bool {
			tv = is.Tree().FindSyncNode(k)
			if tv != nil {
				return tree.Break
			}
			return tree.Continue
		})
		if tv == nil {
			MessageSnackbar(is, fmt.Sprintf("Inspector: tree node missing: %v", sw))
			return
		}
	}
	is.AsyncLock() // coming from other tree
	tv.OpenParents()
	tv.ScrollToMe()
	tv.SelectAction(events.SelectOne)
	is.NeedsLayout()
	is.AsyncUnlock()
	is.Scene.Stage.Raise()

	sc.AsyncLock()
	sc.renderBBoxes = false
	if sc.selectedWidgetChan != nil {
		close(sc.selectedWidgetChan)
	}
	sc.selectedWidgetChan = nil
	sc.NeedsRender()
	sc.AsyncUnlock()
}

// InspectApp displays the underlying operating system app
func (is *Inspector) InspectApp() { //types:add
	d := NewBody().AddTitle("Inspect app")
	NewForm(d).SetStruct(system.TheApp).SetReadOnly(true)
	d.RunFullDialog(is)
}

// Tree returns the tree widget.
func (is *Inspector) Tree() *Tree {
	return is.FindPath("splits/tree-frame/tree").(*Tree)
}

func (is *Inspector) MakeToolbar(p *Plan) {
	Add(p, func(w *FuncButton) {
		w.SetFunc(is.ToggleSelectionMode).SetText("Select element").SetIcon(icons.ArrowSelectorTool)
		w.Updater(func() {
			_, ok := is.Root.(*Scene)
			w.SetEnabled(ok)
		})
	})
	Add(p, func(w *Separator) {})
	Add(p, func(w *FuncButton) {
		w.SetFunc(is.Open).SetKey(keymap.Open)
		w.Args[0].SetValue(is.Filename).SetTag(`ext:".json"`)
	})
	Add(p, func(w *FuncButton) {
		w.SetFunc(is.Save).SetKey(keymap.Save)
		w.Updater(func() {
			w.SetEnabled(is.Filename != "")
		})
	})
	Add(p, func(w *FuncButton) {
		w.SetFunc(is.SaveAs).SetKey(keymap.SaveAs)
		w.Args[0].SetValue(is.Filename).SetTag(`ext:".json"`)
	})
	Add(p, func(w *Separator) {})
	Add(p, func(w *FuncButton) {
		w.SetFunc(is.InspectApp).SetIcon(icons.Devices)
	})
}

// InspectorWindow opens an interactive editor of the given tree
// in a new window.
func InspectorWindow(n tree.Node) {
	if RecycleMainWindow(n) {
		return
	}
	d := NewBody("Inspector")
	InspectorView(d, n)
	d.NewWindow().SetCloseOnBack(true).Run()
}

// InspectorView configures the given body to have an interactive inspector
// of the given tree.
func InspectorView(b *Body, n tree.Node) {
	b.SetTitle("Inspector").SetData(n)
	if n != nil {
		b.Name += "-" + n.AsTree().Name
		b.Title += ": " + n.AsTree().Name
	}
	is := NewInspector(b)
	is.SetRoot(n)
	b.AddAppBar(is.MakeToolbar)
}
