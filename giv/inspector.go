// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"
	"reflect"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/goosi"
	"cogentcore.org/core/grows/jsons"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keyfun"
	"cogentcore.org/core/ki"
	"cogentcore.org/core/laser"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/units"
)

// Inspector represents a struct, creating a property editor of the fields --
// constructs Children widgets to show the field names and editor fields for
// each field, within an overall frame with an optional title, and a button
// box at the bottom where methods can be invoked
type Inspector struct {
	gi.Frame

	// root of tree being edited
	KiRoot ki.Ki

	// has the root changed via gui actions?  updated from treeview and structview for changes
	Changed bool `set:"-"`

	// current filename for saving / loading
	Filename gi.Filename
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
	is.OnWidgetAdded(func(w gi.Widget) {
		if tw, ok := w.(*TreeView); ok {
			tw.Style(func(s *styles.Style) {
				s.Max.X.Em(20)
			})
			return
		}
		path := w.PathFrom(is)
		switch path {
		case "title":
			title := w.(*gi.Label)
			title.Type = gi.LabelHeadlineSmall
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
func (is *Inspector) SaveAs(filename gi.Filename) error { //gti:add
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
func (is *Inspector) Open(filename gi.Filename) error { //gti:add
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
	sc := gi.AsScene(is.KiRoot)
	if sc == nil {
		gi.NewBody().AddSnackbarText("SelectionMode is only available on Scene objects").NewSnackbar(is).Run()
		return
	}
	sc.SetFlag(!sc.Is(gi.ScRenderBBoxes), gi.ScRenderBBoxes)
	if sc.Is(gi.ScRenderBBoxes) {
		sc.SelectedWidgetChan = make(chan gi.Widget)
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
	sc := gi.AsScene(is.KiRoot)
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
		sw.WalkUpParent(func(k ki.Ki) bool {
			tv = is.TreeView().FindSyncNode(k)
			if tv != nil {
				return ki.Break
			}
			return ki.Continue
		})
		if tv == nil {
			gi.NewBody().AddSnackbarText(fmt.Sprintf("Inspector: tree view node missing: %v", sw)).NewSnackbar(is).Run()
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
	sc.SetFlag(false, gi.ScRenderBBoxes)
	if sc.SelectedWidgetChan != nil {
		close(sc.SelectedWidgetChan)
	}
	sc.SelectedWidgetChan = nil
	sc.NeedsRender()
	sc.AsyncUnlock()
}

// InspectApp displays the underlying operating system app
func (is *Inspector) InspectApp() { //gti:add
	d := gi.NewBody().AddTitle("Inspect app")
	NewStructView(d).SetStruct(goosi.TheApp).SetReadOnly(true)
	d.NewFullDialog(is).Run()
}

// SetRoot sets the source root and ensures everything is configured
func (is *Inspector) SetRoot(root ki.Ki) {
	if is.KiRoot != root {
		is.KiRoot = root
		// ge.GetAllUpdates(root)
	}
	is.ConfigWidget()
}

// ConfigWidget configures the widget
func (is *Inspector) ConfigWidget() {
	if is.KiRoot == nil {
		return
	}
	is.Style(func(s *styles.Style) {
		s.Direction = styles.Column
	})
	config := ki.Config{}
	config.Add(gi.LabelType, "title")
	config.Add(gi.SplitsType, "splits")
	is.ConfigChildren(config)
	is.SetTitle(is.KiRoot)
	is.ConfigSplits()
}

// SetTitle sets the title to correspond to the given node.
func (is *Inspector) SetTitle(k ki.Ki) {
	is.TitleWidget().SetText(fmt.Sprintf("Inspector of %s (%s)", k.Name(), laser.FriendlyTypeName(reflect.TypeOf(k))))
}

// TitleWidget returns the title label widget
func (is *Inspector) TitleWidget() *gi.Label {
	return is.ChildByName("title", 0).(*gi.Label)
}

// Splits returns the main Splits
func (is *Inspector) Splits() *gi.Splits {
	return is.ChildByName("splits", 2).(*gi.Splits)
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
	split := is.Splits()

	if len(split.Kids) == 0 {
		tvfr := gi.NewFrame(split, "tvfr")
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

			sc := gi.AsScene(is.KiRoot)
			if sc == nil {
				return
			}
			if w, wb := gi.AsWidget(sn); w != nil {
				pselw := sc.SelectedWidget
				sc.SelectedWidget = w
				wb.NeedsRender()
				if pselw != nil {
					pselw.AsWidget().NeedsRender()
				}
			}
		})
		renderRebuild := func() {
			sc := gi.AsScene(is.KiRoot)
			if sc == nil {
				return
			}
			sc.RenderContext().SetFlag(true, gi.RenderRebuild) // trigger full rebuild
		}
		tv.OnChange(func(e events.Event) {
			renderRebuild()
		})
		sv.OnChange(func(e events.Event) {
			renderRebuild()
		})
		sv.OnClose(func(e events.Event) {
			sc := gi.AsScene(is.KiRoot)
			if sc == nil {
				return
			}
			pselw := sc.SelectedWidget
			sc.SelectedWidget = nil
			if pselw != nil {
				pselw.AsWidget().NeedsRender()
			}
		})
		split.SetSplits(.3, .7)
	}
	tv := is.TreeView()
	tv.SyncRootNode(is.KiRoot)
	sv := is.StructView()
	sv.SetStruct(is.KiRoot)
}

func (is *Inspector) ConfigToolbar(tb *gi.Toolbar) {
	NewFuncButton(tb, is.ToggleSelectionMode).SetText("Select element").SetIcon(icons.ArrowSelectorTool).
		StyleFirst(func(s *styles.Style) {
			_, ok := is.KiRoot.(*gi.Scene)
			s.SetEnabled(ok)
		})
	gi.NewSeparator(tb)
	op := NewFuncButton(tb, is.Open).SetKey(keyfun.Open)
	op.Args[0].SetValue(is.Filename)
	op.Args[0].SetTag("ext", ".json")
	NewFuncButton(tb, is.Save).SetKey(keyfun.Save).
		StyleFirst(func(s *styles.Style) { s.SetEnabled(is.Changed && is.Filename != "") })
	sa := NewFuncButton(tb, is.SaveAs).SetKey(keyfun.SaveAs)
	sa.Args[0].SetValue(is.Filename)
	sa.Args[0].SetTag("ext", ".json")
	gi.NewSeparator(tb)
	NewFuncButton(tb, is.InspectApp).SetIcon(icons.Devices)
}

// InspectorWindow opens an interactive editor of the given Ki tree
// in a new window.
func InspectorWindow(k ki.Ki) {
	if gi.ActivateExistingMainWindow(k) {
		return
	}
	d := gi.NewBody("inspector")
	InspectorView(d, k)
	d.NewWindow().SetCloseOnBack(true).Run()
}

// InspectorView configures the given body to have an interactive inspector
// of the given Ki tree.
func InspectorView(b *gi.Body, k ki.Ki) {
	b.SetTitle("Inspector").SetData(k).SetName("inspector")
	if k != nil {
		b.Nm += "-" + k.Name()
		b.Title += ": " + k.Name()
	}
	is := NewInspector(b, "inspector")
	is.SetRoot(k)
	b.AddAppBar(is.ConfigToolbar)
}
