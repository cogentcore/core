// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package views

import (
	"fmt"
	"reflect"

	"cogentcore.org/core/base/iox/jsonx"
	"cogentcore.org/core/base/labels"
	"cogentcore.org/core/core"
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
	core.Frame

	// Root is the root of the tree being edited.
	Root tree.Node

	// CurrentNode is the currently selected node in the tree.
	CurrentNode tree.Node `set:"-"`

	// Filename is the current filename for saving / loading
	Filename core.Filename `set:"-"`
}

func (is *Inspector) OnInit() {
	is.Frame.OnInit()
	is.Style(func(s *styles.Style) {
		s.Grow.Set(1, 1)
		s.Direction = styles.Column
	})
	is.OnWidgetAdded(func(w core.Widget) {
		// TODO(config)
		if tw, ok := w.(*TreeView); ok {
			tw.Style(func(s *styles.Style) {
				s.Max.X.Em(20)
			})
		}
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
func (is *Inspector) SaveAs(filename core.Filename) error { //types:add
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
func (is *Inspector) Open(filename core.Filename) error { //types:add
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
	sc, ok := is.Root.(*core.Scene)
	if !ok {
		return
	}
	sc.SetFlag(!sc.Is(core.ScRenderBBoxes), core.ScRenderBBoxes)
	if sc.Is(core.ScRenderBBoxes) {
		sc.SelectedWidgetChan = make(chan core.Widget)
		go is.SelectionMonitor()
	} else {
		if sc.SelectedWidgetChan != nil {
			close(sc.SelectedWidgetChan)
		}
		sc.SelectedWidgetChan = nil
	}
	sc.NeedsLayout()
}

// SelectionMonitor monitors for the selected widget
func (is *Inspector) SelectionMonitor() {
	sc, ok := is.Root.(*core.Scene)
	if !ok {
		return
	}
	sc.Stage.Raise()
	sw, ok := <-sc.SelectedWidgetChan
	if !ok || sw == nil {
		return
	}
	tv := is.TreeView().FindSyncNode(sw.This())
	if tv == nil {
		// if we can't be found, we are probably a part,
		// so we keep going up until we find somebody in
		// the tree
		sw.WalkUpParent(func(k tree.Node) bool {
			tv = is.TreeView().FindSyncNode(k)
			if tv != nil {
				return tree.Break
			}
			return tree.Continue
		})
		if tv == nil {
			core.MessageSnackbar(is, fmt.Sprintf("Inspector: tree view node missing: %v", sw))
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
	sc.SetFlag(false, core.ScRenderBBoxes)
	if sc.SelectedWidgetChan != nil {
		close(sc.SelectedWidgetChan)
	}
	sc.SelectedWidgetChan = nil
	sc.NeedsRender()
	sc.AsyncUnlock()
}

// InspectApp displays the underlying operating system app
func (is *Inspector) InspectApp() { //types:add
	d := core.NewBody().AddTitle("Inspect app")
	NewStructView(d).SetStruct(system.TheApp).SetReadOnly(true)
	d.RunFullDialog(is)
}

func (is *Inspector) Make(p *core.Plan) {
	if is.Root == nil {
		return
	}
	core.AddAt(p, "title", func(w *core.Text) {
		w.SetType(core.TextHeadlineSmall)
		w.Style(func(s *styles.Style) {
			s.Grow.Set(1, 0)
			s.Align.Self = styles.Center
		})
	}, func(w *core.Text) {
		w.SetText(fmt.Sprintf("Inspector of %s (%s)", is.Root.Name(), labels.FriendlyTypeName(reflect.TypeOf(is.Root))))
	})
	splits := core.AddAt(p, "splits", func(w *core.Splits) {
		w.SetSplits(.3, .7)
	})
	treeFrame := core.AddAt(splits, "tree-frame", func(w *core.Frame) {
		w.Style(func(s *styles.Style) {
			s.Direction = styles.Column
			s.Overflow.Set(styles.OverflowAuto)
			s.Gap.Zero()
		})
	})
	renderRebuild := func() {
		sc, ok := is.Root.(*core.Scene)
		if !ok {
			return
		}
		sc.RenderContext().SetFlag(true, core.RenderRebuild) // trigger full rebuild
	}
	core.AddAt(treeFrame, "tree", func(w *TreeView) {
		is.CurrentNode = is.Root
		w.OnSelect(func(e events.Event) {
			if len(w.SelectedNodes) == 0 {
				return
			}
			sn := w.SelectedNodes[0].AsTreeView().SyncNode
			is.CurrentNode = sn
			// note: doing Update on entire Inspector undoes all tree expansion
			// only want to update the struct view.
			stru := is.FindPath("splits/struct").(*StructView)
			stru.SetStruct(sn)
			stru.Update()

			sc, ok := is.Root.(*core.Scene)
			if !ok {
				return
			}
			if w, wb := core.AsWidget(sn); w != nil {
				pselw := sc.SelectedWidget
				sc.SelectedWidget = w
				wb.NeedsRender()
				if pselw != nil {
					pselw.AsWidget().NeedsRender()
				}
			}
		})
		w.OnChange(func(e events.Event) {
			renderRebuild()
		})
	}, func(w *TreeView) {
		w.SyncTree(is.Root)
	})
	core.AddAt(splits, "struct", func(w *StructView) {
		w.OnChange(func(e events.Event) {
			renderRebuild()
		})
		w.OnClose(func(e events.Event) {
			sc, ok := is.Root.(*core.Scene)
			if !ok {
				return
			}
			if sc.Is(core.ScRenderBBoxes) {
				is.ToggleSelectionMode()
			}
			pselw := sc.SelectedWidget
			sc.SelectedWidget = nil
			if pselw != nil {
				pselw.AsWidget().NeedsRender()
			}
		})
	}, func(w *StructView) {
		w.SetStruct(is.CurrentNode)
	})
}

// TreeView returns the tree view widget.
func (is *Inspector) TreeView() *TreeView {
	return is.FindPath("splits/tree-frame/tree").(*TreeView)
}

func (is *Inspector) MakeToolbar(p *core.Plan) {
	core.Add(p, func(w *FuncButton) {
		w.SetFunc(is.ToggleSelectionMode).SetText("Select element").SetIcon(icons.ArrowSelectorTool)
	}, func(w *FuncButton) {
		_, ok := is.Root.(*core.Scene)
		w.SetEnabled(ok)
	})
	core.Add[*core.Separator](p)
	core.Add(p, func(w *FuncButton) {
		w.SetFunc(is.Open).SetKey(keymap.Open)
		w.Args[0].SetValue(is.Filename)
		w.Args[0].SetTag("ext", ".json")
	})
	core.Add(p, func(w *FuncButton) {
		w.SetFunc(is.Save).SetKey(keymap.Save)
	}, func(w *FuncButton) {
		w.SetEnabled(is.Filename != "")
	})
	core.Add(p, func(w *FuncButton) {
		w.SetFunc(is.SaveAs).SetKey(keymap.SaveAs)
		w.Args[0].SetValue(is.Filename)
		w.Args[0].SetTag("ext", ".json")
	})
	core.Add[*core.Separator](p)
	core.Add(p, func(w *FuncButton) {
		w.SetFunc(is.InspectApp).SetIcon(icons.Devices)
	})
}

// InspectorWindow opens an interactive editor of the given tree
// in a new window.
func InspectorWindow(n tree.Node) {
	if core.RecycleMainWindow(n) {
		return
	}
	d := core.NewBody("Inspector")
	InspectorView(d, n)
	d.NewWindow().SetCloseOnBack(true).Run()
}

// InspectorView configures the given body to have an interactive inspector
// of the given tree.
func InspectorView(b *core.Body, n tree.Node) {
	b.SetTitle("Inspector").SetData(n)
	if n != nil {
		b.Nm += "-" + n.Name()
		b.Title += ": " + n.Name()
	}
	is := NewInspector(b)
	is.SetRoot(n)
	b.AddAppBar(is.MakeToolbar)
}
