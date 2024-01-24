// Copyright (c) 2018, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/gi"
	"cogentcore.org/core/goosi"
	"cogentcore.org/core/grows/jsons"
	"cogentcore.org/core/grr"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keyfun"
	"cogentcore.org/core/ki"
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
		s.Color = colors.Scheme.OnBackground
		s.Grow.Set(1, 1)
		s.Margin.Set(units.Dp(8))
	})
	is.OnWidgetAdded(func(w gi.Widget) {
		switch w.PathFrom(is) {
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

// UpdateItems updates the objects being edited (e.g., updating display changes)
func (is *Inspector) UpdateItems() { //gti:add
	if is.KiRoot == nil {
		return
	}
	if w, ok := is.KiRoot.(gi.Widget); ok {
		w.AsWidget().SetNeedsRender(true)
	}
}

// Save saves tree to current filename, in a standard JSON-formatted file
func (is *Inspector) Save() { //gti:add
	if is.KiRoot == nil {
		return
	}
	if is.Filename == "" {
		return
	}

	grr.Log(jsons.Save(is.KiRoot, string(is.Filename)))
	is.Changed = false
}

// SaveAs saves tree to given filename, in a standard JSON-formatted file
func (is *Inspector) SaveAs(filename gi.Filename) { //gti:add
	if is.KiRoot == nil {
		return
	}
	grr.Log(jsons.Save(is.KiRoot, string(filename)))
	is.Changed = false
	is.Filename = filename
	is.SetNeedsRender(true) // notify our editor
}

// Open opens tree from given filename, in a standard JSON-formatted file
func (is *Inspector) Open(filename gi.Filename) { //gti:add
	if is.KiRoot == nil {
		return
	}
	grr.Log(jsons.Open(is.KiRoot, string(filename)))
	is.Filename = filename
	is.SetNeedsRender(true) // notify our editor
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
	updt := sc.UpdateStart()
	sc.SelectedWidget = nil
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
	sc.UpdateEndLayout(updt)
}

// SelectionMonitor monitors the selected widget
func (is *Inspector) SelectionMonitor() {
	for {
		sc := gi.AsScene(is.KiRoot)
		if sc == nil {
			break
		}
		if sc.SelectedWidgetChan == nil {
			sc.SelectedWidget = nil
			break
		}
		sw := <-sc.SelectedWidgetChan
		if sw == nil {
			break
		}
		tv := is.TreeView().FindSyncNode(sw.This())
		if tv == nil {
			gi.NewBody().AddSnackbarText(fmt.Sprintf("Inspector: tree view node missing: %v", sw)).NewSnackbar(is).Run()
		} else {
			updt := is.UpdateStartAsync() // coming from other tree
			tv.OpenParents()
			tv.ScrollToMe()
			tv.SelectAction(events.SelectOne)
			is.UpdateEndAsyncLayout(updt)

			updt = sc.UpdateStartAsync()
			sc.SelectedWidget = sw
			sw.AsWidget().SetNeedsRender(updt)
			sc.UpdateEndAsyncRender(updt)
		}
	}
}

// InspectApp displays the underlying operating system app
func (is *Inspector) InspectApp() { //gti:add
	d := gi.NewBody()
	NewStructView(d).SetStruct(goosi.TheApp).SetReadOnly(true)
	d.NewFullDialog(is).Run()
}

// SetRoot sets the source root and ensures everything is configured
func (is *Inspector) SetRoot(root ki.Ki) {
	updt := false
	if is.KiRoot != root {
		updt = is.UpdateStart()
		is.KiRoot = root
		// ge.GetAllUpdates(root)
	}
	is.Config()
	is.UpdateEnd(updt)
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
	mods, updt := is.ConfigChildren(config)
	is.SetTitle(fmt.Sprintf("Inspector of %v", is.KiRoot.Name()))
	is.ConfigSplits()
	if mods {
		is.UpdateEnd(updt)
	}
}

// SetTitle sets the optional title and updates the Title label
func (is *Inspector) SetTitle(title string) {
	lab := is.TitleWidget()
	lab.Text = title
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
			if len(tv.SelectedNodes) > 0 {
				sv.SetStruct(tv.SelectedNodes[0].AsTreeView().SyncNode)
			}
		})
		split.SetSplits(.3, .7)
	}
	tv := is.TreeView()
	tv.SyncRootNode(is.KiRoot)
	sv := is.StructView()
	sv.SetStruct(is.KiRoot)
}

func (is *Inspector) SetChanged() {
	is.Changed = true
}

func (is *Inspector) ConfigToolbar(tb *gi.Toolbar) {
	NewFuncButton(tb, is.UpdateItems).SetIcon(icons.Refresh)
	// StyleFirst(func(s *styles.Style) { s.SetEnabled(is.Changed) })
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
	d.NewWindow().Run()
}

// InspectorView configures the given body to have an interactive inspector
// of the given Ki tree.
func InspectorView(b *gi.Body, k ki.Ki) {
	b.SetTitle("Inspector").SetName("inspector")
	if k != nil {
		b.Nm += "-" + k.Name()
		b.Title += ": " + k.Name()
	}
	is := NewInspector(b, "inspector")
	is.SetRoot(k)
	b.AddAppBar(is.ConfigToolbar)
}
