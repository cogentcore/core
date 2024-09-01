// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"reflect"

	"cogentcore.org/core/base/iox/jsonx"
	"cogentcore.org/core/base/labels"
	"cogentcore.org/core/colors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keymap"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
)

// Inspector represents a [tree.Node] with a [Tree] and a [Form].
type Inspector struct {
	Frame

	// Root is the root of the tree being edited.
	Root tree.Node

	// currentNode is the currently selected node in the tree.
	currentNode tree.Node

	// filename is the current filename for saving / loading
	filename Filename

	treeWidget *Tree
}

func (is *Inspector) Init() {
	is.Frame.Init()
	is.Styler(func(s *styles.Style) {
		s.Grow.Set(1, 1)
		s.Direction = styles.Column
	})

	var titleWidget *Text
	tree.AddChildAt(is, "title", func(w *Text) {
		titleWidget = w
		is.currentNode = is.Root
		w.SetType(TextHeadlineSmall)
		w.Styler(func(s *styles.Style) {
			s.Grow.Set(1, 0)
			s.Align.Self = styles.Center
		})
		w.Updater(func() {
			w.SetText(fmt.Sprintf("Inspector of %s (%s)", is.currentNode.AsTree().Name, labels.FriendlyTypeName(reflect.TypeOf(is.currentNode))))
		})
	})
	renderRebuild := func() {
		sc, ok := is.Root.(*Scene)
		if !ok {
			return
		}
		sc.renderContext().rebuild = true // trigger full rebuild
	}
	tree.AddChildAt(is, "splits", func(w *Splits) {
		w.SetSplits(.3, .7)
		var form *Form
		tree.AddChildAt(w, "tree-frame", func(w *Frame) {
			w.Styler(func(s *styles.Style) {
				s.Background = colors.Scheme.SurfaceContainerLow
				s.Direction = styles.Column
				s.Overflow.Set(styles.OverflowAuto)
				s.Gap.Zero()
			})
			tree.AddChildAt(w, "tree", func(w *Tree) {
				is.treeWidget = w
				w.SetTreeInit(func(tr *Tree) {
					tr.Styler(func(s *styles.Style) {
						s.Max.X.Em(20)
					})
				})
				w.OnSelect(func(e events.Event) {
					if len(w.SelectedNodes) == 0 {
						return
					}
					sn := w.SelectedNodes[0].AsCoreTree().SyncNode
					is.currentNode = sn
					// Note: doing Update on the entire inspector reverts all tree expansion,
					// so we only want to update the title and form
					titleWidget.Update()
					form.SetStruct(sn).Update()

					sc, ok := is.Root.(*Scene)
					if !ok {
						return
					}
					if wb := AsWidget(sn); wb != nil {
						pselw := sc.selectedWidget
						sc.selectedWidget = sn.(Widget)
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
		tree.AddChildAt(w, "struct", func(w *Form) {
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
					is.toggleSelectionMode()
				}
				pselw := sc.selectedWidget
				sc.selectedWidget = nil
				if pselw != nil {
					pselw.AsWidget().NeedsRender()
				}
			})
			w.Updater(func() {
				w.SetStruct(is.currentNode)
			})
		})
	})
}

// save saves the tree to current filename, in a standard JSON-formatted file.
func (is *Inspector) save() error { //types:add
	if is.Root == nil {
		return nil
	}
	if is.filename == "" {
		return nil
	}

	err := jsonx.Save(is.Root, string(is.filename))
	if err != nil {
		return err
	}
	return nil
}

// saveAs saves tree to given filename, in a standard JSON-formatted file
func (is *Inspector) saveAs(filename Filename) error { //types:add
	if is.Root == nil {
		return nil
	}
	err := jsonx.Save(is.Root, string(filename))
	if err != nil {
		return err
	}
	is.filename = filename
	is.NeedsRender() // notify our editor
	return nil
}

// open opens tree from given filename, in a standard JSON-formatted file
func (is *Inspector) open(filename Filename) error { //types:add
	if is.Root == nil {
		return nil
	}
	err := jsonx.Open(is.Root, string(filename))
	if err != nil {
		return err
	}
	is.filename = filename
	is.NeedsRender() // notify our editor
	return nil
}

// toggleSelectionMode toggles the editor between selection mode or not.
// In selection mode, bounding boxes are rendered around each Widget,
// and clicking on a Widget pulls it up in the inspector.
func (is *Inspector) toggleSelectionMode() { //types:add
	sc, ok := is.Root.(*Scene)
	if !ok {
		return
	}
	sc.renderBBoxes = !sc.renderBBoxes
	if sc.renderBBoxes {
		sc.selectedWidgetChan = make(chan Widget)
		go is.selectionMonitor()
	} else {
		if sc.selectedWidgetChan != nil {
			close(sc.selectedWidgetChan)
		}
		sc.selectedWidgetChan = nil
	}
	sc.NeedsLayout()
}

// selectionMonitor monitors for the selected widget
func (is *Inspector) selectionMonitor() {
	sc, ok := is.Root.(*Scene)
	if !ok {
		return
	}
	sc.Stage.raise()
	sw, ok := <-sc.selectedWidgetChan
	if !ok || sw == nil {
		return
	}
	tv := is.treeWidget.FindSyncNode(sw)
	if tv == nil {
		// if we can't be found, we are probably a part,
		// so we keep going up until we find somebody in
		// the tree
		sw.AsTree().WalkUpParent(func(k tree.Node) bool {
			tv = is.treeWidget.FindSyncNode(k)
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
	tv.SelectEvent(events.SelectOne)
	tv.ScrollToThis()
	is.AsyncUnlock()
	is.Scene.Stage.raise()

	sc.AsyncLock()
	sc.renderBBoxes = false
	if sc.selectedWidgetChan != nil {
		close(sc.selectedWidgetChan)
	}
	sc.selectedWidgetChan = nil
	sc.NeedsRender()
	sc.AsyncUnlock()
}

// inspectApp displays [TheApp].
func (is *Inspector) inspectApp() { //types:add
	d := NewBody("Inspect app")
	NewForm(d).SetStruct(TheApp).SetReadOnly(true)
	d.RunFullDialog(is)
}

func (is *Inspector) MakeToolbar(p *tree.Plan) {
	tree.Add(p, func(w *FuncButton) {
		w.SetFunc(is.toggleSelectionMode).SetText("Select element").SetIcon(icons.ArrowSelectorTool)
		w.Updater(func() {
			_, ok := is.Root.(*Scene)
			w.SetEnabled(ok)
		})
	})
	tree.Add(p, func(w *Separator) {})
	tree.Add(p, func(w *FuncButton) {
		w.SetFunc(is.open).SetIcon(icons.Open).SetKey(keymap.Open)
		w.Args[0].SetValue(is.filename).SetTag(`extension:".json"`)
	})
	tree.Add(p, func(w *FuncButton) {
		w.SetFunc(is.save).SetIcon(icons.Save).SetKey(keymap.Save)
		w.Updater(func() {
			w.SetEnabled(is.filename != "")
		})
	})
	tree.Add(p, func(w *FuncButton) {
		w.SetFunc(is.saveAs).SetIcon(icons.SaveAs).SetKey(keymap.SaveAs)
		w.Args[0].SetValue(is.filename).SetTag(`extension:".json"`)
	})
	tree.Add(p, func(w *Separator) {})
	tree.Add(p, func(w *FuncButton) {
		w.SetFunc(is.inspectApp).SetIcon(icons.Devices)
	})
}

// InspectorWindow opens an interactive editor of the given tree
// in a new window.
func InspectorWindow(n tree.Node) {
	if RecycleMainWindow(n) {
		return
	}
	d := NewBody("Inspector")
	makeInspector(d, n)
	d.RunWindow()
}

// makeInspector configures the given body to have an interactive inspector
// of the given tree.
func makeInspector(b *Body, n tree.Node) {
	b.SetTitle("Inspector").SetData(n)
	if n != nil {
		b.Name += "-" + n.AsTree().Name
		b.Title += ": " + n.AsTree().Name
	}
	is := NewInspector(b)
	is.SetRoot(n)
	b.AddTopBar(func(bar *Frame) {
		NewToolbar(bar).Maker(is.MakeToolbar)
	})
}
