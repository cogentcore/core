// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/gist"
	"github.com/goki/gi/oswin"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
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

// AddNewGiEditor adds a new gieditor to given parent node, with given name.
func AddNewGiEditor(parent ki.Ki, name string) *GiEditor {
	return parent.AddNewChild(KiT_GiEditor, name).(*GiEditor)
}

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
	ge.Config()
	ge.UpdateEnd(updt)
}

// // GetAllUpdates connects to all nodes in the tree to receive notification of changes
// func (ge *GiEditor) GetAllUpdates(root ki.Ki) {
// 	ge.KiRoot.FuncDownMeFirst(0, ge, func(k ki.Ki, level int, d interface{}) bool {
// 		k.NodeSignal().Connect(ge.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
// 			gee := recv.Embed(KiT_GiEditor).(*GiEditor)
// 			if !gee.Changed {
// 				fmt.Printf("GiEditor: Tree changed with signal: %v\n", ki.NodeSignals(sig))
// 				gee.Changed = true
// 			}
// 		})
// 		return ki.Continue
// 	})
// }

// Config configures the widget
func (ge *GiEditor) Config() {
	if ge.KiRoot == nil {
		return
	}
	ge.Lay = gi.LayoutVert
	ge.SetProp("spacing", gi.StdDialogVSpaceUnits)
	config := kit.TypeAndNameList{}
	config.Add(gi.KiT_Label, "title")
	config.Add(gi.KiT_ToolBar, "toolbar")
	config.Add(gi.KiT_SplitView, "splitview")
	mods, updt := ge.ConfigChildren(config)
	ge.SetTitle(fmt.Sprintf("GoGi Editor of Ki Node Tree: %v", ge.KiRoot.Name()))
	ge.ConfigSplitView()
	ge.ConfigToolbar()
	if mods {
		ge.UpdateEnd(updt)
	}
	return
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

// SplitView returns the main SplitView
func (ge *GiEditor) SplitView() *gi.SplitView {
	return ge.ChildByName("splitview", 2).(*gi.SplitView)
}

// TreeView returns the main TreeView
func (ge *GiEditor) TreeView() *TreeView {
	return ge.SplitView().Child(0).Child(0).(*TreeView)
}

// StructView returns the main StructView
func (ge *GiEditor) StructView() *StructView {
	return ge.SplitView().Child(1).(*StructView)
}

// ToolBar returns the toolbar widget
func (ge *GiEditor) ToolBar() *gi.ToolBar {
	return ge.ChildByName("toolbar", 1).(*gi.ToolBar)
}

// ConfigToolbar adds a GiEditor toolbar.
func (ge *GiEditor) ConfigToolbar() {
	tb := ge.ToolBar()
	if tb != nil && tb.HasChildren() {
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
	split := ge.SplitView()
	// split.Dim = mat32.Y
	split.Dim = mat32.X

	if len(split.Kids) == 0 {
		tvfr := gi.AddNewFrame(split, "tvfr", gi.LayoutHoriz)
		tvfr.SetReRenderAnchor()
		tv := AddNewTreeView(tvfr, "tv")
		sv := AddNewStructView(split, "sv")
		tv.TreeViewSig.Connect(ge.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			if data == nil {
				return
			}
			gee, _ := recv.Embed(KiT_GiEditor).(*GiEditor)
			svr := gee.StructView()
			tvn, _ := data.(ki.Ki).Embed(KiT_TreeView).(*TreeView)
			if sig == int64(TreeViewSelected) {
				svr.SetStruct(tvn.SrcNode)
			} else if sig == int64(TreeViewChanged) {
				gee.SetChanged()
			}
		})
		sv.ViewSig.Connect(ge.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			gee, _ := recv.Embed(KiT_GiEditor).(*GiEditor)
			gee.SetChanged()
		})
		split.SetSplits(.3, .7)
	}
	tv := ge.TreeView()
	tv.SetRootNode(ge.KiRoot)
	sv := ge.StructView()
	sv.SetStruct(ge.KiRoot)
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
	"EnumType:Flag":    gi.KiT_NodeFlags,
	"background-color": &gi.Prefs.Colors.Background,
	"color":            &gi.Prefs.Colors.Font,
	"max-width":        -1,
	"max-height":       -1,
	"#title": ki.Props{
		"max-width":        -1,
		"horizontal-align": gist.AlignCenter,
		"vertical-align":   gist.AlignTop,
	},
	"ToolBar": ki.PropSlice{
		{"Update", ki.Props{
			"icon": "update",
			"updtfunc": ActionUpdateFunc(func(gei interface{}, act *gi.Action) {
				ge := gei.(*GiEditor)
				act.SetActiveStateUpdt(ge.Changed)
			}),
		}},
		{"sep-file", ki.BlankProp{}},
		{"Open", ki.Props{
			"label": "Open",
			"icon":  "file-open",
			"desc":  "Open a json-formatted Ki tree structure",
			"Args": ki.PropSlice{
				{"File Name", ki.Props{
					"default-field": "Filename",
					"ext":           ".json",
				}},
			},
		}},
		{"Save", ki.Props{
			"icon": "file-save",
			"desc": "Save json-formatted Ki tree structure to existing filename",
			"updtfunc": ActionUpdateFunc(func(gei interface{}, act *gi.Action) {
				ge := gei.(*GiEditor)
				act.SetActiveStateUpdt(ge.Changed && ge.Filename != "")
			}),
		}},
		{"SaveAs", ki.Props{
			"label": "Save As...",
			"icon":  "file-save",
			"desc":  "Save as a json-formatted Ki tree structure",
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
				"updtfunc": ActionUpdateFunc(func(gei interface{}, act *gi.Action) {
					ge := gei.(*GiEditor)
					act.SetActiveState(ge.Changed)
				}),
			}},
			{"sep-file", ki.BlankProp{}},
			{"Open", ki.Props{
				"shortcut": gi.KeyFunMenuOpen,
				"desc":     "Open a json-formatted Ki tree structure",
				"Args": ki.PropSlice{
					{"File Name", ki.Props{
						"default-field": "Filename",
						"ext":           ".json",
					}},
				},
			}},
			{"Save", ki.Props{
				"shortcut": gi.KeyFunMenuSave,
				"desc":     "Save json-formatted Ki tree structure to existing filename",
				"updtfunc": ActionUpdateFunc(func(gei interface{}, act *gi.Action) {
					ge := gei.(*GiEditor)
					act.SetActiveState(ge.Changed && ge.Filename != "")
				}),
			}},
			{"SaveAs", ki.Props{
				"shortcut": gi.KeyFunMenuSaveAs,
				"label":    "Save As...",
				"desc":     "Save as a json-formatted Ki tree structure",
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
func GoGiEditorDialog(obj ki.Ki) *GiEditor {
	width := 1280
	height := 920
	wnm := "gogi-editor"
	wti := "GoGi Editor"
	if obj != nil {
		wnm += "-" + obj.Name()
		wti += ": " + obj.Name()
	}

	win, recyc := gi.RecycleMainWindow(obj, wnm, wti, width, height)
	if recyc {
		mfr, err := win.MainFrame()
		if err == nil {
			return mfr.Child(0).(*GiEditor)
		}
	}

	vp := win.WinViewport2D()
	updt := vp.UpdateStart()

	mfr := win.SetMainFrame()
	mfr.Lay = gi.LayoutVert

	ge := AddNewGiEditor(mfr, "editor")
	ge.Viewport = vp
	ge.SetRoot(obj)

	mmen := win.MainMenu
	MainMenuView(ge, win, mmen)

	tb := ge.ToolBar()
	tb.UpdateActions()

	inClosePrompt := false
	win.OSWin.SetCloseReqFunc(func(w oswin.Window) {
		if !ge.Changed {
			win.Close()
			return
		}
		if inClosePrompt {
			return
		}
		inClosePrompt = true
		gi.ChoiceDialog(vp, gi.DlgOpts{Title: "Close Without Saving?",
			Prompt: "Do you want to save your changes?  If so, Cancel and then Save"},
			[]string{"Close Without Saving", "Cancel"},
			win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
				switch sig {
				case 0:
					win.Close()
				case 1:
					// default is to do nothing, i.e., cancel
					inClosePrompt = false
				}
			})
	})

	vp.UpdateEndNoSig(updt)
	win.GoStartEventLoop() // in a separate goroutine
	return ge
}
