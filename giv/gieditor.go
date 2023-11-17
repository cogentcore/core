// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"
	"log"

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

// GiEditor represents a struct, creating a property editor of the fields --
// constructs Children widgets to show the field names and editor fields for
// each field, within an overall frame with an optional title, and a button
// box at the bottom where methods can be invoked
type GiEditor struct {
	gi.Frame

	// root of tree being edited
	KiRoot ki.Ki

	// has the root changed via gui actions?  updated from treeview and structview for changes
	Changed bool `set:"-"`

	// current filename for saving / loading
	Filename gi.FileName
}

func (ge *GiEditor) OnInit() {
	ge.Style(func(s *styles.Style) {
		s.Color = colors.Scheme.OnBackground
		s.Grow.Set(1, 1)
		s.Margin.Set(units.Dp(8))
	})
	ge.OnWidgetAdded(func(w gi.Widget) {
		switch w.PathFrom(ge) {
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
func (ge *GiEditor) Update() { //gti:add
	if ge.KiRoot == nil {
		return
	}
	if w, ok := ge.KiRoot.(gi.Widget); ok {
		w.AsWidget().SetNeedsRender()
	}
}

// Save saves tree to current filename, in a standard JSON-formatted file
func (ge *GiEditor) Save() { //gti:add
	if ge.KiRoot == nil {
		return
	}
	if ge.Filename == "" {
		return
	}

	grr.Log0(jsons.Save(ge.KiRoot, string(ge.Filename)))
	ge.Changed = false
}

// SaveAs saves tree to given filename, in a standard JSON-formatted file
func (ge *GiEditor) SaveAs(filename gi.FileName) { //gti:add
	if ge.KiRoot == nil {
		return
	}
	grr.Log0(jsons.Save(ge.KiRoot, string(filename)))
	ge.Changed = false
	ge.Filename = filename
	ge.SetNeedsRender() // notify our editor
}

// Open opens tree from given filename, in a standard JSON-formatted file
func (ge *GiEditor) Open(filename gi.FileName) { //gti:add
	if ge.KiRoot == nil {
		return
	}
	grr.Log0(jsons.Open(ge.KiRoot, string(filename)))
	ge.Filename = filename
	ge.SetNeedsRender() // notify our editor
}

// EditColorScheme pulls up a window to edit the current color scheme
func (ge *GiEditor) EditColorScheme() { //gti:add
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
func (ge *GiEditor) ToggleSelectionMode() { //gti:add
	// sc := gi.AsScene(ge.KiRoot)
	sc, ok := ge.KiRoot.(*gi.Scene)
	if !ok {
		gi.NewSnackbar(ge).Text("SelectionMode is only available on Scene objects").Run()
		return
	}
	updt := sc.UpdateStart()
	sc.SelectedWidget = nil
	sc.SetFlag(!sc.Is(gi.ScRenderBBoxes), gi.ScRenderBBoxes)
	if sc.Is(gi.ScRenderBBoxes) {
		sc.SelectedWidgetChan = make(chan gi.Widget)
		go ge.SelectionMonitor()
	} else {
		if sc.SelectedWidgetChan != nil {
			close(sc.SelectedWidgetChan)
		}
		sc.SelectedWidgetChan = nil
	}
	sc.UpdateEndLayout(updt)
}

// SelectionMonitor
func (ge *GiEditor) SelectionMonitor() {
	for {
		sc, ok := ge.KiRoot.(*gi.Scene)
		if !ok {
			break
		}
		if sc.SelectedWidgetChan == nil {
			sc.SelectedWidget = nil
			break
		}
		sw := <-sc.SelectedWidgetChan
		tv := ge.TreeView().FindSyncNode(sw.This())
		if tv == nil {
			log.Printf("GiEditor on %v: tree view source node missing for", sw)
		} else {
			gi.UpdateTrace = true
			updt := ge.UpdateStart()
			fmt.Println("updt", updt)
			tv.OpenParents()
			tv.ScrollToMe()
			tv.SelectAction(events.SelectOne)
			ge.UpdateEndLayout(updt)
			updt = sc.UpdateStart()
			sc.SelectedWidget = sw
			sw.AsWidget().SetNeedsRenderUpdate(sc, updt)
			sc.UpdateEndRender(updt)
			gi.UpdateTrace = false
		}
	}
}

// SetRoot sets the source root and ensures everything is configured
func (ge *GiEditor) SetRoot(root ki.Ki) {
	updt := false
	if ge.KiRoot != root {
		updt = ge.UpdateStart()
		ge.KiRoot = root
		// ge.GetAllUpdates(root)
	}
	ge.Config(ge.Sc)
	ge.UpdateEnd(updt)
}

// Config configures the widget
func (ge *GiEditor) ConfigWidget(sc *gi.Scene) {
	if ge.KiRoot == nil {
		return
	}
	ge.Style(func(s *styles.Style) {
		s.SetMainAxis(mat32.Y)
	})
	config := ki.Config{}
	config.Add(gi.LabelType, "title")
	config.Add(gi.SplitsType, "splits")
	mods, updt := ge.ConfigChildren(config)
	ge.SetTitle(fmt.Sprintf("GoGi Editor of Ki Node Tree: %v", ge.KiRoot.Name()))
	ge.ConfigSplits()
	if mods {
		ge.UpdateEnd(updt)
	}
}

// SetTitle sets the optional title and updates the Title label
func (ge *GiEditor) SetTitle(title string) {
	lab := ge.TitleWidget()
	lab.Text = title
}

// Title returns the title label widget, and its index, within frame
func (ge *GiEditor) TitleWidget() *gi.Label {
	return ge.ChildByName("title", 0).(*gi.Label)
}

// Splits returns the main Splits
func (ge *GiEditor) Splits() *gi.Splits {
	return ge.ChildByName("splits", 2).(*gi.Splits)
}

// TreeView returns the main TreeSyncView
func (ge *GiEditor) TreeView() *TreeView {
	return ge.Splits().Child(0).Child(0).(*TreeView)
}

// StructView returns the main StructView
func (ge *GiEditor) StructView() *StructView {
	return ge.Splits().Child(1).(*StructView)
}

// ConfigSplits configures the Splits.
func (ge *GiEditor) ConfigSplits() {
	if ge.KiRoot == nil {
		return
	}
	split := ge.Splits()
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
	tv := ge.TreeView()
	tv.SyncRootNode(ge.KiRoot)
	sv := ge.StructView()
	sv.SetStruct(ge.KiRoot)
}

func (ge *GiEditor) SetChanged() {
	ge.Changed = true
}

func (ge *GiEditor) TopAppBar(tb *gi.TopAppBar) {
	if gi.DefaultTopAppBar != nil {
		gi.DefaultTopAppBar(tb)
	} else {
		gi.DefaultTopAppBarStd(tb)
	}

	up := NewFuncButton(tb, ge.Update).SetIcon(icons.Refresh)
	up.SetUpdateFunc(func() {
		up.SetEnabled(ge.Changed)
	})
	sel := NewFuncButton(tb, ge.ToggleSelectionMode).SetText("Select Element").SetIcon(icons.ArrowSelectorTool)
	sel.SetUpdateFunc(func() {
		sc, ok := ge.KiRoot.(*gi.Scene)
		sel.SetEnabled(ok)
		if !ok {
			return
		}
		_ = sc
		// TODO(kai/sel): check if has flag
	})
	gi.NewSeparator(tb)
	op := NewFuncButton(tb, ge.Open).SetKey(keyfun.Open)
	op.Args[0].SetValue(ge.Filename)
	op.Args[0].SetTag("ext", ".json")
	save := NewFuncButton(tb, ge.Save).SetKey(keyfun.Save)
	save.SetUpdateFunc(func() {
		save.SetEnabledUpdt(ge.Changed && ge.Filename != "")
	})
	sa := NewFuncButton(tb, ge.SaveAs).SetKey(keyfun.SaveAs)
	sa.Args[0].SetValue(ge.Filename)
	sa.Args[0].SetTag("ext", ".json")
	gi.NewSeparator(tb)
	NewFuncButton(tb, ge.EditColorScheme).SetIcon(icons.Colors)
}

func (ge *GiEditor) MenuBar(mb *gi.MenuBar) {
	NewFuncButton(mb, ge.Update)
}

// GoGiEditorDialog opens an interactive editor of the given Ki tree, at its
// root, returns GiEditor and window
func GoGiEditorDialog(obj ki.Ki) {
	if gi.ActivateExistingMainWindow(obj) {
		return
	}
	sc := gi.NewScene("gogi-editor")
	sc.Title = "GoGi Editor"
	if obj != nil {
		sc.Nm += "-" + obj.Name()
		sc.Title += ": " + obj.Name()
	}

	ge := NewGiEditor(sc, "editor")
	ge.SetRoot(obj)

	sc.TopAppBar = ge.TopAppBar

	gi.NewWindow(sc).Run()
}
