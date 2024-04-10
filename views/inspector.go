// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package views

import (
	"fmt"
	"reflect"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/core"
	"cogentcore.org/core/events"
	"cogentcore.org/core/goosi"
	"cogentcore.org/core/grows/jsons"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keyfun"
	"cogentcore.org/core/laser"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/tree"
	"cogentcore.org/core/units"
)

// Inspector represents a struct, creating a property editor of the fields --
// constructs Children widgets to show the field names and editor fields for
// each field, within an overall frame with an optional title, and a button
// box at the bottom where methods can be invoked
type Inspector struct {
	core.Frame

	// root of tree being edited
	KiRoot tree.Node

	// has the root changed via gui actions?  updated from treeview and structview for changes
	Changed bool `set:"-"`

	// current filename for saving / loading
	Filename core.Filename
}

func (is *Inspector) OnInit() {
	is.Frame.OnInit()
	is.SetStyles()
}

func (is *Inspector) SetStyles() {
	is.Style(func(s *styles.Style) {
		s.Color = colors.C(colors.Scheme.OnBackground)
		s.Grow.Set(1, 1)
		s.Margin.Set(units.Dp(8))
	})
	is.OnWidgetAdded(func(w core.Widget) {
		if tw, ok := w.(*TreeView); ok {
			tw.Style(func(s *styles.Style) {
				s.Max.X.Em(20)
			})
			return
		}
		path := w.PathFrom(is)
		switch path {
		case "title":
			title := w.(*core.Label)
			title.Type = core.LabelHeadlineSmall
			title.Style(func(s *styles.Style) {
				s.Grow.Set(1, 0)
				s.Align.Self = styles.Center
			})
		}
	})
}

// Save saves tree to current filename, in a standard JSON-formatted file
func (is *Inspector) Save() error { //gti:add
	if is.KiRoot == nil {
		return nil
	}
	if is.Filename == "" {
		return nil
	}

	err := jsons.Save(is.KiRoot, string(is.Filename))
	if err != nil {
		return err
	}
	is.Changed = false
	return nil
}

// SaveAs saves tree to given filename, in a standard JSON-formatted file
func (is *Inspector) SaveAs(filename core.Filename) error { //gti:add
	if is.KiRoot == nil {
		return nil
	}
	err := jsons.Save(is.KiRoot, string(filename))
	if err != nil {
		return err
	}
	is.Changed = false
	is.Filename = filename
	is.NeedsRender() // notify our editor
	return nil
}

// Open opens tree from given filename, in a standard JSON-formatted file
func (is *Inspector) Open(filename core.Filename) error { //gti:add
	if is.KiRoot == nil {
		return nil
	}
	err := jsons.Open(is.KiRoot, string(filename))
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
func (is *Inspector) ToggleSelectionMode() { //gti:add
	sc := core.AsScene(is.KiRoot)
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
	sc := core.AsScene(is.KiRoot)
	if sc == nil {
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
			core.NewBody().AddSnackbarText(fmt.Sprintf("Inspector: tree view node missing: %v", sw)).NewSnackbar(is).Run()
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
func (is *Inspector) InspectApp() { //gti:add
	d := core.NewBody().AddTitle("Inspect app")
	NewStructView(d).SetStruct(goosi.TheApp).SetReadOnly(true)
	d.NewFullDialog(is).Run()
}

// SetRoot sets the source root and ensures everything is configured
func (is *Inspector) SetRoot(root tree.Node) {
	if is.KiRoot != root {
		is.KiRoot = root
		// ge.GetAllUpdates(root)
	}
	is.Config()
}

// Config configures the widget
func (is *Inspector) Config() {
	if is.KiRoot == nil {
		return
	}
	is.Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})
	config := tree.Config{}
	config.Add(core.LabelType, "title")
	config.Add(core.SplitsType, "splits")
	is.ConfigChildren(config)
	is.SetTitle(is.KiRoot)
	is.ConfigSplits()
}

// SetTitle sets the title to correspond to the given node.
func (is *Inspector) SetTitle(k tree.Node) {
	is.TitleWidget().SetText(fmt.Sprintf("Inspector of %s (%s)", k.Name(), laser.FriendlyTypeName(reflect.TypeOf(k))))
}

// TitleWidget returns the title label widget
func (is *Inspector) TitleWidget() *core.Label {
	return is.ChildByName("title", 0).(*core.Label)
}

// Splits returns the main Splits
func (is *Inspector) Splits() *core.Splits {
	return is.ChildByName("splits", 2).(*core.Splits)
}

// TreeView returns the main TreeSyncView
func (is *Inspector) TreeView() *TreeView {
	return is.Splits().Child(0).Child(0).(*TreeView)
}

// StructView returns the main StructView
func (is *Inspector) StructView() *StructView {
	return is.Splits().Child(1).(*StructView)
}

// ConfigSplits configures the Splits.
func (is *Inspector) ConfigSplits() {
	if is.KiRoot == nil {
		return
	}
	split := is.Splits().SetSplits(.3, .7)

	if len(split.Kids) == 0 {
		tvfr := core.NewFrame(split, "tvfr")
		tvfr.Style(func(s *styles.Style) {
			s.Direction = styles.Column
			s.Overflow.Set(styles.OverflowAuto)
			s.Gap.Zero()
		})
		tv := NewTreeView(tvfr, "tv")
		sv := NewStructView(split, "sv")
		tv.OnSelect(func(e events.Event) {
			if len(tv.SelectedNodes) == 0 {
				return
			}
			sn := tv.SelectedNodes[0].AsTreeView().SyncNode
			sv.SetStruct(sn)

			is.SetTitle(sn)

			sc := core.AsScene(is.KiRoot)
			if sc == nil {
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
		renderRebuild := func() {
			sc := core.AsScene(is.KiRoot)
			if sc == nil {
				return
			}
			sc.RenderContext().SetFlag(true, core.RenderRebuild) // trigger full rebuild
		}
		tv.OnChange(func(e events.Event) {
			renderRebuild()
		})
		sv.OnChange(func(e events.Event) {
			renderRebuild()
		})
		sv.OnClose(func(e events.Event) {
			sc := core.AsScene(is.KiRoot)
			if sc == nil {
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
	}
	tv := is.TreeView()
	tv.SyncTree(is.KiRoot)
	sv := is.StructView()
	sv.SetStruct(is.KiRoot)
}

func (is *Inspector) ConfigToolbar(tb *core.Toolbar) {
	NewFuncButton(tb, is.ToggleSelectionMode).SetText("Select element").SetIcon(icons.ArrowSelectorTool).
		StyleFirst(func(s *styles.Style) {
			_, ok := is.KiRoot.(*core.Scene)
			s.SetEnabled(ok)
		})
	core.NewSeparator(tb)
	op := NewFuncButton(tb, is.Open).SetKey(keyfun.Open)
	op.Args[0].SetValue(is.Filename)
	op.Args[0].SetTag("ext", ".json")
	NewFuncButton(tb, is.Save).SetKey(keyfun.Save).
		StyleFirst(func(s *styles.Style) { s.SetEnabled(is.Changed && is.Filename != "") })
	sa := NewFuncButton(tb, is.SaveAs).SetKey(keyfun.SaveAs)
	sa.Args[0].SetValue(is.Filename)
	sa.Args[0].SetTag("ext", ".json")
	core.NewSeparator(tb)
	NewFuncButton(tb, is.InspectApp).SetIcon(icons.Devices)
}

// InspectorWindow opens an interactive editor of the given Ki tree
// in a new window.
func InspectorWindow(k tree.Node) {
	if core.ActivateExistingMainWindow(k) {
		return
	}
	d := core.NewBody("inspector")
	InspectorView(d, k)
	d.NewWindow().SetCloseOnBack(true).Run()
}

// InspectorView configures the given body to have an interactive inspector
// of the given Ki tree.
func InspectorView(b *core.Body, k tree.Node) {
	b.SetTitle("Inspector").SetData(k).SetName("inspector")
	if k != nil {
		b.Nm += "-" + k.Name()
		b.Title += ": " + k.Name()
	}
	is := NewInspector(b, "inspector")
	is.SetRoot(k)
	b.AddAppBar(is.ConfigToolbar)
}
