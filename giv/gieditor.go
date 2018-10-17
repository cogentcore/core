// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"

	"github.com/goki/gi"
	"github.com/goki/gi/oswin"
	"github.com/goki/ki"
	"github.com/goki/ki/kit"
)

// GiEditor represents a struct, creating a property editor of the fields --
// constructs Children widgets to show the field names and editor fields for
// each field, within an overall frame with an optional title, and a button
// box at the bottom where methods can be invoked
type GiEditor struct {
	gi.Frame
	KiRoot   ki.Ki       `desc:"root of tree being edited"`
	Changed  bool        `desc:"has the root changed via gui actions?  updated from treeview and structview for changes"`
	Filename gi.FileName `desc:"current filename for saving / loading"`
}

var KiT_GiEditor = kit.Types.AddType(&GiEditor{}, GiEditorProps)

// Update updates the objects being edited (e.g., updating display changes)
func (ge *GiEditor) Update() {
	if ge.KiRoot == nil {
		return
	}
	ge.KiRoot.UpdateSig()
}

// Save saves tree to current filename, in a standard JSON-formatted file
func (ge *GiEditor) Save() {
	if ge.KiRoot == nil {
		return
	}
	if ge.Filename == "" {
		return
	}

	ge.KiRoot.SaveJSON(string(ge.Filename))
	ge.Changed = false
}

// SaveAs saves tree to given filename, in a standard JSON-formatted file
func (ge *GiEditor) SaveAs(filename gi.FileName) {
	if ge.KiRoot == nil {
		return
	}
	ge.KiRoot.SaveJSON(string(filename))
	ge.Changed = false
	ge.Filename = filename
	ge.UpdateSig() // notify our editor
}

// Open opens tree from given filename, in a standard JSON-formatted file
func (ge *GiEditor) Open(filename gi.FileName) {
	if ge.KiRoot == nil {
		return
	}
	ge.KiRoot.OpenJSON(string(filename))
	ge.Filename = filename
	ge.SetFullReRender()
	ge.UpdateSig() // notify our editor
}

// SetRoot sets the source root and ensures everything is configured
func (ge *GiEditor) SetRoot(root ki.Ki) {
	updt := false
	if ge.KiRoot != root {
		updt = ge.UpdateStart()
		ge.KiRoot = root
		// ge.GetAllUpdates(root)
	}
	ge.UpdateFromRoot()
	ge.UpdateEnd(updt)
}

// // GetAllUpdates connects to all nodes in the tree to receive notification of changes
// func (ge *GiEditor) GetAllUpdates(root ki.Ki) {
// 	ge.KiRoot.FuncDownMeFirst(0, ge, func(k ki.Ki, level int, d interface{}) bool {
// 		k.NodeSignal().Connect(ge.This, func(recv, send ki.Ki, sig int64, data interface{}) {
// 			gee := recv.Embed(KiT_GiEditor).(*GiEditor)
// 			if !gee.Changed {
// 				fmt.Printf("GiEditor: Tree changed with signal: %v\n", ki.NodeSignals(sig))
// 				gee.Changed = true
// 			}
// 		})
// 		return true
// 	})
// }

// UpdateFromRoot updates full widget layout
func (ge *GiEditor) UpdateFromRoot() {
	if ge.KiRoot == nil {
		return
	}
	mods, updt := ge.StdConfig()
	ge.SetTitle(fmt.Sprintf("GoGi Editor of Ki Node Tree: %v", ge.KiRoot.Name()))
	ge.ConfigSplitView()
	ge.ConfigToolbar()
	if mods {
		ge.UpdateEnd(updt)
	}
}

// StdFrameConfig returns a TypeAndNameList for configuring a standard Frame
// -- can modify as desired before calling ConfigChildren on Frame using this
func (ge *GiEditor) StdFrameConfig() kit.TypeAndNameList {
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_Label, "title")
	config.Add(gi.KiT_ToolBar, "toolbar")
	config.Add(gi.KiT_SplitView, "splitview")
	return config
}

// StdConfig configures a standard setup of the overall Frame -- returns mods,
// updt from ConfigChildren and does NOT call UpdateEnd
func (ge *GiEditor) StdConfig() (mods, updt bool) {
	ge.Lay = gi.LayoutVert
	ge.SetProp("spacing", gi.StdDialogVSpaceUnits)
	config := ge.StdFrameConfig()
	mods, updt = ge.ConfigChildren(config, false)
	return
}

// SetTitle sets the optional title and updates the Title label
func (ge *GiEditor) SetTitle(title string) {
	lab, _ := ge.TitleWidget()
	if lab != nil {
		lab.Text = title
	}
}

// Title returns the title label widget, and its index, within frame -- nil,
// -1 if not found
func (ge *GiEditor) TitleWidget() (*gi.Label, int) {
	idx, ok := ge.Children().IndexByName("title", 0)
	if !ok {
		return nil, -1
	}
	return ge.KnownChild(idx).(*gi.Label), idx
}

// SplitView returns the main SplitView
func (ge *GiEditor) SplitView() (*gi.SplitView, int) {
	idx, ok := ge.Children().IndexByName("splitview", 2)
	if !ok {
		return nil, -1
	}
	return ge.KnownChild(idx).(*gi.SplitView), idx
}

// TreeView returns the main TreeView
func (ge *GiEditor) TreeView() *TreeView {
	split, _ := ge.SplitView()
	if split != nil {
		tv := split.KnownChild(0).KnownChild(0).(*TreeView)
		return tv
	}
	return nil
}

// StructView returns the main StructView
func (ge *GiEditor) StructView() *StructView {
	split, _ := ge.SplitView()
	if split != nil {
		sv := split.KnownChild(1).(*StructView)
		return sv
	}
	return nil
}

// ToolBar returns the toolbar widget
func (ge *GiEditor) ToolBar() *gi.ToolBar {
	idx, ok := ge.Children().IndexByName("toolbar", 1)
	if !ok {
		return nil
	}
	return ge.KnownChild(idx).(*gi.ToolBar)
}

// ConfigToolbar adds a GiEditor toolbar.
func (ge *GiEditor) ConfigToolbar() {
	tb := ge.ToolBar()
	if tb.HasChildren() {
		return
	}
	tb.SetStretchMaxWidth()
	ToolBarView(ge, ge.Viewport, tb)
}

// ConfigSplitView configures the SplitView.
func (ge *GiEditor) ConfigSplitView() {
	if ge.KiRoot == nil {
		return
	}
	split, _ := ge.SplitView()
	if split == nil {
		return
	}
	// split.Dim = gi.Y
	split.Dim = gi.X

	if len(split.Kids) == 0 {
		tvfr := split.AddNewChild(gi.KiT_Frame, "tvfr").(*gi.Frame)
		tv := tvfr.AddNewChild(KiT_TreeView, "tv").(*TreeView)
		sv := split.AddNewChild(KiT_StructView, "sv").(*StructView)
		tv.TreeViewSig.Connect(ge.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			if data == nil {
				return
			}
			gee, _ := recv.Embed(KiT_GiEditor).(*GiEditor)
			svr := gee.StructView()
			tvn, _ := data.(ki.Ki).Embed(KiT_TreeView).(*TreeView)
			if sig == int64(TreeViewSelected) {
				svr.SetStruct(tvn.SrcNode.Ptr, nil)
			} else if sig == int64(TreeViewChanged) {
				gee.SetChanged()
			}
		})
		sv.ViewSig.Connect(ge.This, func(recv, send ki.Ki, sig int64, data interface{}) {
			gee, _ := recv.Embed(KiT_GiEditor).(*GiEditor)
			gee.SetChanged()
		})
		split.SetSplits(.3, .7)
	}
	tv := ge.TreeView()
	tv.SetRootNode(ge.KiRoot)
	sv := ge.StructView()
	sv.SetStruct(ge.KiRoot, nil)
}

func (ge *GiEditor) SetChanged() {
	ge.Changed = true
	ge.ToolBar().UpdateActions() // nil safe
}

func (ge *GiEditor) Render2D() {
	ge.ToolBar().UpdateActions()
	if win := ge.ParentWindow(); win != nil {
		if !win.IsResizing() {
			win.MainMenuUpdateActives()
		}
	}
	ge.Frame.Render2D()
}

var GiEditorProps = ki.Props{
	"background-color": &gi.Prefs.Colors.Background,
	"color":            &gi.Prefs.Colors.Font,
	"max-width":        -1,
	"max-height":       -1,
	"#title": ki.Props{
		"max-width":        -1,
		"horizontal-align": gi.AlignCenter,
		"vertical-align":   gi.AlignTop,
	},
	"ToolBar": ki.PropSlice{
		{"Update", ki.Props{
			"icon": "update",
			"updtfunc": func(gei interface{}, act *gi.Action) {
				ge := gei.(*GiEditor)
				act.SetActiveStateUpdt(ge.Changed)
			},
		}},
		{"sep-file", ki.BlankProp{}},
		{"Open", ki.Props{
			"label": "Open",
			"icon":  "file-open",
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"default-field": "Filename",
					"ext":           ".json",
				}},
			},
		}},
		{"Save", ki.Props{
			"icon": "file-save",
			"updtfunc": func(gei interface{}, act *gi.Action) {
				ge := gei.(*GiEditor)
				act.SetActiveStateUpdt(ge.Changed && ge.Filename != "")
			},
		}},
		{"SaveAs", ki.Props{
			"label": "Save As...",
			"icon":  "file-save",
			"updtfunc": func(gei interface{}, act *gi.Action) {
				ge := gei.(*GiEditor)
				act.SetActiveStateUpdt(ge.Changed)
			},
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"default-field": "Filename",
					"ext":           ".json",
				}},
			},
		}},
	},
	"MainMenu": ki.PropSlice{
		{"AppMenu", ki.BlankProp{}},
		{"File", ki.PropSlice{
			{"Update", ki.Props{
				"shortcut": "Command+U",
				"updtfunc": func(gei interface{}, act *gi.Action) {
					ge := gei.(*GiEditor)
					act.SetActiveState(ge.Changed)
				},
			}},
			{"sep-file", ki.BlankProp{}},
			{"Open", ki.Props{
				"shortcut": "Command+O",
				"Args": ki.PropSlice{
					{"File Name", ki.Props{
						"default-field": "Filename",
						"ext":           ".json",
					}},
				},
			}},
			{"Save", ki.Props{
				"shortcut": "Command+S",
				"updtfunc": func(gei interface{}, act *gi.Action) {
					ge := gei.(*GiEditor)
					act.SetActiveState(ge.Changed && ge.Filename != "")
				},
			}},
			{"SaveAs", ki.Props{
				"shortcut": "Shift+Command+S",
				"label":    "Save As...",
				"updtfunc": func(gei interface{}, act *gi.Action) {
					ge := gei.(*GiEditor)
					act.SetActiveState(ge.Changed)
				},
				"Args": ki.PropSlice{
					{"File Name", ki.Props{
						"default-field": "Filename",
						"ext":           ".json",
					}},
				},
			}},
			{"sep-close", ki.BlankProp{}},
			{"Close Window", ki.BlankProp{}},
		}},
		{"Edit", "Copy Cut Paste Dupe"},
		{"Window", "Windows"},
	},
}

// GoGiEditorDialog opens an interactive editor of the given Ki tree, at its
// root, returns GiEditor and window
func GoGiEditorDialog(obj ki.Ki) (*GiEditor, *gi.Window) {
	width := 1280
	height := 920
	wnm := "gogi-editor"
	wti := "GoGi Editor"
	if obj != nil {
		wnm += "-" + obj.Name()
		wti += ": " + obj.Name()
	}

	win := gi.NewWindow2D(wnm, wti, width, height, true)

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()
	mfr.Lay = gi.LayoutVert

	ge := mfr.AddNewChild(KiT_GiEditor, "editor").(*GiEditor)
	ge.Viewport = vp
	ge.SetRoot(obj)

	mmen := win.MainMenu
	MainMenuView(ge, win, mmen)

	tb := ge.ToolBar()
	tb.UpdateActions()

	win.OSWin.SetCloseReqFunc(func(w oswin.Window) {
		if ge.Changed {
			gi.ChoiceDialog(vp, gi.DlgOpts{Title: "Close Without Saving?",
				Prompt: "Do you want to save your changes?  If so, Cancel and then Save"},
				[]string{"Close Without Saving", "Cancel"},
				win.This, func(recv, send ki.Ki, sig int64, data interface{}) {
					switch sig {
					case 0:
						w.Close()
					case 1:
						// default is to do nothing, i.e., cancel
					}
				})
		} else {
			w.Close()
		}
	})

	vp.UpdateEndNoSig(updt)
	win.GoStartEventLoop() // in a separate goroutine
	return ge, win
}
