// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"
	"strings"
	"time"

	"goki.dev/girl/gist"
)

var (
	// SnackbarTimeout is the default timeout for Snackbar Stage
	SnackbarTimeout = 5 * time.Second // todo: put in prefs
)

// StageTypes are the types of Stage containers
type StageTypes int32 //enums:enum

const (
	// Window displays Scene in a full window
	Window StageTypes = iota

	// Dialog displays Scene in a smaller dialog window
	// above the Window (can alternatively be in its own OS Window)
	Dialog

	// Sheet displays Scene as a partially overlapping panel
	// coming up from the Bottom or LeftSide
	Sheet

	// Menu displays Scene as a panel on top of window
	Menu

	// Tooltip displays Scene as a tooltip
	Tooltip

	// Snackbar displays Scene as a Snackbar
	Snackbar

	// Chooser displays Scene as a dynamic chooser for completing
	// or correcting text
	Chooser
)

// StageSides are the Sides for Sheet Stages
type StageSides int32 //enums:enum

const (
	// Bottom anchors Sheet to the bottom of the window, with handle on the top
	Bottom StageSides = iota

	// LeftSide anchors Sheet to the left side of the window, with handle on the top
	LeftSide
)

// Stage is a container and manager for displaying a Scene
// in different functional ways, defined by StageTypes
type Stage struct {

	// Scene contents of this Stage -- what it displays
	Scene *Scene

	// type of Stage: determines behavior and Styling
	Type StageTypes

	// name of the Stage -- generally auto-set based on Scene Name
	Name string

	// [view: -] the main data element represented by this window -- used for Recycle* methods based on views representing a given data element -- prevents redundant windows
	Data any `json:"-" xml:"-" view:"-" desc:"the main data element represented by this window -- used for Recycle* methods based on views representing a given data element -- prevents redundant windows"`

	// position and size within the parent Render context.
	// Position is absolute offset relative to top left corner of render context.
	Geom gist.Geom2DInt

	// title of the Stage -- generally auto-set based on Scene Title.  used for title of Window and Dialog types
	Title string

	// if true, blocks input to all other stages.
	Modal bool

	// if true, places a darkening scrim over other stages, if not a full window
	Scrim bool

	// if true dismisses the Stage if user clicks anywhere off the Stage
	ClickOff bool

	// if > 0, disappears after a timeout duration
	Timeout time.Duration

	// if non-zero, requested width in standardized 96 DPI Pixel units.  otherwise automatically resizes.
	Width int

	// if non-zero, requested height in standardized 96 DPI Pixel units.  otherwise automatically resizes.
	Height int

	// if true, opens a Window or Dialog in its own separate operating system window (RenderWin).  This is by default true for Window on Desktop, otherwise false.
	OwnWin bool

	// for Windows: add a back button
	Back bool

	// for Dialogs: adds a handle titlebar Decor for moving
	Movable bool

	// for Dialogs: adds a resize handle Decor for resizing
	Resizable bool

	// Side for Stages that can operate on different sides, e.g., for Sheets: which side does the sheet come out from
	Side StageSides

	// manager for the popups in this stage
	PopupMgr PopupMgr

	// the parent stage manager for this stage
	StageMgr *StageMgrBase

	// event manager for this stage
	EventMgr EventMgr
}

func (st *Stage) RenderCtx() *RenderContext {
	if st.StageMgr == nil {
		return nil
	}
	return st.StageMgr.RenderCtx
}

// NewStage returns a new stage with given type and scene contents.
// Make further configuration choices using Set* methods, which
// can be chained directly after the NewStage call.
// Use an appropriate Run call at the end to start the Stage running.
func NewStage(typ StageTypes, sc *Scene) *Stage {
	st := &Stage{}
	st.SetType(typ)
	st.SetScene(sc)
	return st
}

// NewWindow returns a new Window stage with given scene contents.
// Make further configuration choices using Set* methods, which
// can be chained directly after the New call.
// Use an appropriate Run call at the end to start the Stage running.
func NewWindow(sc *Scene) *Stage {
	return NewStage(Window, sc)
}

// NewDialog returns a new Dialog stage with given scene contents.
// Make further configuration choices using Set* methods, which
// can be chained directly after the New call.
// Use an appropriate Run call at the end to start the Stage running.
func NewDialog(sc *Scene) *Stage {
	return NewStage(Dialog, sc)
}

// NewSheet returns a new Sheet stage with given scene contents,
// for given side (e.g., Bottom or LeftSide).
// Make further configuration choices using Set* methods, which
// can be chained directly after the New call.
// Use an appropriate Run call at the end to start the Stage running.
func NewSheet(sc *Scene, side StageSides) *Stage {
	return NewStage(Sheet, sc).SetSide(side)
}

// NewMenu returns a new Menu stage with given scene contents.
// Make further configuration choices using Set* methods, which
// can be chained directly after the New call.
// Use an appropriate Run call at the end to start the Stage running.
func NewMenu(sc *Scene) *Stage {
	return NewStage(Menu, sc)
}

// NewTooltip returns a new Tooltip stage with given scene contents.
// Make further configuration choices using Set* methods, which
// can be chained directly after the New call.
// Use an appropriate Run call at the end to start the Stage running.
func NewTooltip(sc *Scene) *Stage {
	return NewStage(Tooltip, sc)
}

// NewSnackbar returns a new Snackbar stage with given scene contents.
// Make further configuration choices using Set* methods, which
// can be chained directly after the New call.
// Use an appropriate Run call at the end to start the Stage running.
func NewSnackbar(sc *Scene) *Stage {
	return NewStage(Snackbar, sc)
}

// NewChooser returns a new Chooser stage with given scene contents.
// Make further configuration choices using Set* methods, which
// can be chained directly after the New call.
// Use an appropriate Run call at the end to start the Stage running.
func NewChooser(sc *Scene) *Stage {
	return NewStage(Chooser, sc)
}

// Note: Set* methods are designed to be called in sequence to efficiently set
// desired properties.

// SetNameFromString sets the name of this Stage based on existing
// Scene and Type settings.
func (st *Stage) SetNameFromScene() *Stage {
	if st.Scene == nil {
		return nil
	}
	sc := st.Scene
	st.Name = sc.Name + "-" + strings.ToLower(st.Type.String())
	st.Title = sc.Title
	return st
}

func (st *Stage) SetScene(sc *Scene) *Stage {
	st.Scene = sc
	if sc != nil {
		st.SetNameFromScene()
	}
	return st
}

func (st *Stage) SetType(typ StageTypes) *Stage {
	st.Type = typ
	switch st.Type {
	case Window:
		// if !goosi.TheApp.IsMobile() {
		// 	st.OwnWin = true
		// }
		st.Modal = true // note: there is no global modal option between RenderWin windows
	case Dialog:
		st.Modal = true
		st.Scrim = true
		st.ClickOff = true
		st.Movable = true
		st.Resizable = true
	case Sheet:
		st.Modal = true
		st.Scrim = true
		st.ClickOff = true
		st.Resizable = true
	case Menu:
		st.Modal = true
		st.Scrim = false
		st.ClickOff = true
	case Tooltip:
		st.Modal = true
		st.Scrim = false
	case Snackbar:
		st.Modal = true
		st.Timeout = SnackbarTimeout
	case Chooser:
		st.Modal = false
		st.Scrim = false
		st.ClickOff = true
	}
	return st
}

func (st *Stage) SetName(name string) *Stage {
	st.Name = name
	return st
}

func (st *Stage) SetTitle(title string) *Stage {
	st.Title = title
	return st
}

func (st *Stage) SetModal() *Stage {
	st.Modal = true
	return st
}

func (st *Stage) SetScrim() *Stage {
	st.Scrim = true
	return st
}

func (st *Stage) SetClickOff() *Stage {
	st.ClickOff = true
	return st
}

func (st *Stage) SetTimeout(dur time.Duration) *Stage {
	st.Timeout = dur
	return st
}

func (st *Stage) SetWidth(width int) *Stage {
	st.Width = width
	return st
}

func (st *Stage) SetHeight(height int) *Stage {
	st.Height = height
	return st
}

func (st *Stage) SetOwnWin() *Stage {
	st.OwnWin = true
	return st
}

// SetSharedWin sets OwnWin off to override default OwnWin for Desktop Window
func (st *Stage) SetSharedWin() *Stage {
	st.OwnWin = false
	return st
}

func (st *Stage) SetBack() *Stage {
	st.Back = true
	return st
}

func (st *Stage) SetMovable() *Stage {
	st.Movable = true
	return st
}

func (st *Stage) SetResizable() *Stage {
	st.Resizable = true
	return st
}

func (st *Stage) SetSide(side StageSides) *Stage {
	st.Side = side
	return st
}

/////////////////////////////////////////////////////
//		Run

// Run does the default run behavior based on the type of stage
func (st *Stage) Run() *Stage {
	switch st.Type {
	case Window:
		return st.RunWindow()
	case Dialog:
		return st.RunDialog()
	case Sheet:
		return st.Sheet()
	default:
		return st.RunPopup()
	}
	return st
}

// RunWindow runs a Window with current settings.
// RenderWin field will be set to the parent RenderWin window.
func (st *Stage) RunWindow() *Stage {
	if st.OwnWin {
		st.RenderWin = st.NewRenderWin()
		st.RenderWin.GoStartEventLoop()
		return st
	}
	if CurRenderWin == nil {
		st.AddWindowDecor()
		CurRenderWin = RunNewRenderWin(st.Name, st.Title, st)
		st.RenderWin = CurRenderWin
		return st
	}
	CurRenderWin.AddWindow(st)
	return st
}

// RunDialog runs a Dialog with current settings.
// RenderWin field will be set to the parent RenderWin window.
func (st *Stage) RunDialog() *Stage {
	if st.OwnWin {
		st.RenderWin = RunNewRenderWin(st.Name, st.Title, st)
		return st
	}
	if CurRenderWin == nil {
		// todo: fail!
		return st
	}
	st.AddDialogDecor()
	CurRenderWin.AddDialog(st)
	return st
}

// RunSheet runs a Sheet with current settings.
// RenderWin field will be set to the parent RenderWin window.
func (st *Stage) RunSheet() *Stage {
	if CurRenderWin == nil {
		// todo: fail!
		return st
	}
	st.AddSheetDecor()
	CurRenderWin.AddSheet(st)
	return st
}

// RunPopup runs a popup-style Stage on top of current
// active stage in current active RenderWin.
func (st *Stage) RunPopup() *Stage {
	if CurRenderWin == nil {
		// todo: fail!
		return st
	}
	// maybe Snackbar decor?
	CurRenderWin.AddPopup(st)
	return st
}

/////////////////////////////////////////////////////
//		Decorate

// only called when !OwnWin
func (st *Stage) AddWindowDecor() *Stage {
	if st.Back {
		but := NewButton(st.Scene.Decor, "win-back")
		// todo: do more button config
	}
}

func (st *Stage) AddDialogDecor() *Stage {
	// todo: moveable, resizable
}

func (st *Stage) AddSheetDecor() *Stage {
	// todo: handle based on side
}

///////////////////////////////////////////////////
//  	Events

// IsPtIn returns true if given point is inside the Geom Bounds
// of this Stage.
func (st *Stage) IsPtIn(pt image.Point) bool {
	return pt.In(st.Geom.Bounds())
}

func (st *Stage) Delete() {
	st.PopupMgr.CloseAll()
	if st.Scene != nil {
		st.Scene.Delete()
	}
	st.Scene = nil
	st.StageMgr = nil
}
