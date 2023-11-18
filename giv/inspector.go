// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"

	"goki.dev/colors"
	"goki.dev/colors/matcolor"
	"goki.dev/gi/v2/gi"
	"goki.dev/gi/v2/keyfun"
	"goki.dev/girl/styles"
	"goki.dev/girl/units"
	"goki.dev/goosi/events"
	"goki.dev/grows/jsons"
	"goki.dev/grr"
	"goki.dev/icons"
	"goki.dev/ki/v2"
	"goki.dev/mat32/v2"
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
	Filename gi.FileName
}

func (is *Inspector) OnInit() {
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
				s.Align.X = styles.AlignCenter
				s.Align.Y = styles.AlignStart
			})
		}
	})
}

// Update updates the objects being edited (e.g., updating display changes)
func (is *Inspector) Update() { //gti:add
	if is.KiRoot == nil {
		return
	}
	if w, ok := is.KiRoot.(gi.Widget); ok {
		w.AsWidget().SetNeedsRender()
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

	grr.Log0(jsons.Save(is.KiRoot, string(is.Filename)))
	is.Changed = false
}

// SaveAs saves tree to given filename, in a standard JSON-formatted file
func (is *Inspector) SaveAs(filename gi.FileName) { //gti:add
	if is.KiRoot == nil {
		return
	}
	grr.Log0(jsons.Save(is.KiRoot, string(filename)))
	is.Changed = false
	is.Filename = filename
	is.SetNeedsRender() // notify our editor
}

// Open opens tree from given filename, in a standard JSON-formatted file
func (is *Inspector) Open(filename gi.FileName) { //gti:add
	if is.KiRoot == nil {
		return
	}
	grr.Log0(jsons.Open(is.KiRoot, string(filename)))
	is.Filename = filename
	is.SetNeedsRender() // notify our editor
}

// EditColorScheme pulls up a window to edit the current color scheme
func (is *Inspector) EditColorScheme() { //gti:add
	if gi.ActivateExistingMainWindow(&colors.Schemes) {
		return
	}

	sc := gi.NewScene("gogi-color-scheme")
	sc.Title = "GoGi Color Scheme"
	sc.Data = &colors.Schemes

	key := &matcolor.Key{
		Primary:        colors.FromRGB(123, 135, 122),
		Secondary:      colors.FromRGB(106, 196, 178),
		Tertiary:       colors.FromRGB(106, 196, 178),
		Error:          colors.FromRGB(219, 46, 37),
		Neutral:        colors.FromRGB(133, 131, 121),
		NeutralVariant: colors.FromRGB(107, 106, 101),
	}
	p := matcolor.NewPalette(key)
	schemes := matcolor.NewSchemes(p)

	kv := NewStructView(sc, "kv")
	kv.SetStruct(key)
	// kv.Style(func(s *styles.Style) {
	// 	kv.Grow.Set(1,1)
	// })

	split := gi.NewSplits(sc, "split")
	split.Dim = mat32.X

	svl := NewStructView(split, "svl")
	svl.SetStruct(&schemes.Light)
	// svl.Style(func(s *styles.Style) {
	// 	svl.Grow.Set(1,1)
	// })

	svd := NewStructView(split, "svd")
	svd.SetStruct(&schemes.Dark)

	kv.OnChange(func(e events.Event) {
		p = matcolor.NewPalette(key)
		schemes = matcolor.NewSchemes(p)
		colors.Schemes = schemes
		gi.Prefs.UpdateAll()
		svl.UpdateFields()
		svd.UpdateFields()
	})

	gi.NewWindow(sc).Run()
}

// ToggleSelectionMode toggles the editor between selection mode or not.
// In selection mode, bounding boxes are rendered around each Widget,
// and clicks
func (is *Inspector) ToggleSelectionMode() { //gti:add
	sc := gi.AsScene(is.KiRoot)
	if sc == nil {
		gi.NewSnackbar(is).Text("SelectionMode is only available on Scene objects").Run()
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

// SelectionMonitor
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
			gi.NewSnackbar(is).Text(fmt.Sprintf("Inspector: tree view node missing: %v", sw)).Run()
		} else {
			gi.UpdateTrace = true
			updt := is.UpdateStart()
			tv.OpenParents()
			tv.ScrollToMe()
			tv.SelectAction(events.SelectOne)
			is.UpdateEndLayout(updt)
			updt = sc.UpdateStart()
			sc.SelectedWidget = sw
			sw.AsWidget().SetNeedsRenderUpdate(sc, updt)
			sc.UpdateEndRender(updt)
			gi.UpdateTrace = false
		}
	}
}

// SetRoot sets the source root and ensures everything is configured
func (is *Inspector) SetRoot(root ki.Ki) {
	updt := false
	if is.KiRoot != root {
		updt = is.UpdateStart()
		is.KiRoot = root
		// ge.GetAllUpdates(root)
	}
	is.Config(is.Sc)
	is.UpdateEnd(updt)
}

// Config configures the widget
func (is *Inspector) ConfigWidget(sc *gi.Scene) {
	if is.KiRoot == nil {
		return
	}
	is.Style(func(s *styles.Style) {
		s.SetMainAxis(mat32.Y)
	})
	config := ki.Config{}
	config.Add(gi.LabelType, "title")
	config.Add(gi.SplitsType, "splits")
	mods, updt := is.ConfigChildren(config)
	is.SetTitle(fmt.Sprintf("Inspector of Ki Node Tree: %v", is.KiRoot.Name()))
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

// Title returns the title label widget, and its index, within frame
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
	// split.Dim = mat32.Y
	split.Dim = mat32.X

	if len(split.Kids) == 0 {
		tvfr := gi.NewFrame(split, "tvfr")
		tvfr.Style(func(s *styles.Style) {
			s.MainAxis = mat32.Y
			s.Overflow.Set(styles.OverflowAuto)
			s.Gap.Zero()
		})
		tv := NewTreeView(tvfr, "tv")
		sv := NewStructView(split, "sv")
		tv.OnSelect(func(e events.Event) {
			if len(tv.SelectedNodes) > 0 {
				sv.SetStruct(tv.SelectedNodes[0].SyncNode)
				// todo: connect
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

func (is *Inspector) TopAppBar(tb *gi.TopAppBar) {
	if gi.DefaultTopAppBar != nil {
		gi.DefaultTopAppBar(tb)
	} else {
		gi.DefaultTopAppBarStd(tb)
	}

	up := NewFuncButton(tb, is.Update).SetIcon(icons.Refresh)
	up.SetUpdateFunc(func() {
		up.SetEnabled(is.Changed)
	})
	sel := NewFuncButton(tb, is.ToggleSelectionMode).SetText("Select Element").SetIcon(icons.ArrowSelectorTool)
	sel.SetUpdateFunc(func() {
		sc, ok := is.KiRoot.(*gi.Scene)
		sel.SetEnabled(ok)
		if !ok {
			return
		}
		_ = sc
		// TODO(kai/sel): check if has flag
	})
	gi.NewSeparator(tb)
	op := NewFuncButton(tb, is.Open).SetKey(keyfun.Open)
	op.Args[0].SetValue(is.Filename)
	op.Args[0].SetTag("ext", ".json")
	save := NewFuncButton(tb, is.Save).SetKey(keyfun.Save)
	save.SetUpdateFunc(func() {
		save.SetEnabledUpdt(is.Changed && is.Filename != "")
	})
	sa := NewFuncButton(tb, is.SaveAs).SetKey(keyfun.SaveAs)
	sa.Args[0].SetValue(is.Filename)
	sa.Args[0].SetTag("ext", ".json")
	gi.NewSeparator(tb)
	NewFuncButton(tb, is.EditColorScheme).SetIcon(icons.Colors)
}

func (is *Inspector) MenuBar(mb *gi.MenuBar) {
	NewFuncButton(mb, is.Update)
}

// InspectorDialog opens an interactive editor of the given Ki tree, at its
// root, returns Inspector and window
func InspectorDialog(obj ki.Ki) {
	if gi.ActivateExistingMainWindow(obj) {
		return
	}
	sc := gi.NewScene("inspector")
	sc.Title = "Inspector"
	if obj != nil {
		sc.Nm += "-" + obj.Name()
		sc.Title += ": " + obj.Name()
	}

	is := NewInspector(sc, "inspector")
	is.SetRoot(obj)

	sc.TopAppBar = is.TopAppBar

	gi.NewWindow(sc).Run()
}
