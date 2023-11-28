// Copyright (c) 2023, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"goki.dev/goosi"
)

// StageTypes are the types of Stage containers.
// There are two main categories: MainStage and PopupStage.
// MainStage are Window, Dialog, Sheet: large and potentially
// complex Scenes that persist until dismissed, and can have
// Decor widgets that control display.
// PopupStage are Menu, Tooltip, Snackbar, Chooser that are transitory
// and simple, without additional decor.
// MainStages live in a StageMgr associated with a RenderWin window,
// and manage their own set of PopupStages via a PopupStageMgr.
type StageTypes int32 //enums:enum

const (
	// WindowStage is a MainStage that displays a Scene in a full window.
	// One of these must be created first, as the primary App contents,
	// and it typically persists throughout.  It fills the RenderWin window.
	// Additional Windows can be created either within the same RenderWin
	// (Mobile) or in separate RenderWin windows (Desktop, NewWindow).
	WindowStage StageTypes = iota

	// DialogStage is a MainStage that displays Scene in a smaller dialog window
	// on top of a Window, or in its own RenderWin (on Desktop only).
	// It can be Modal or not.
	DialogStage

	// SheetStage is a MainStage that displays Scene as a
	// partially overlapping panel coming up from the
	// Bottom or LeftSide of the RenderWin main window.
	// It can be Modal or not.
	SheetStage

	// MenuStage is a PopupStage that displays a Scene with Action Widgets
	// overlaid on a MainStage.
	// It is typically Modal and ClickOff, and closes when
	// an Action is selected.
	MenuStage

	// TooltipStage is a PopupStage that displays a Scene with extra info
	// overlaid on a MainStage.
	// It is typically ClickOff and not Modal.
	TooltipStage

	// SnackbarStage is a PopupStage displays a Scene with info and typically
	// an additional optional Action, usually displayed at the bottom.
	// It is typically not ClickOff or Modal, but has a timeout.
	SnackbarStage

	// CompleterStage is a PopupStage that displays a Scene with text completions,
	// spelling corrections, or other such dynamic info.
	// It is typically ClickOff, not Modal, dynamically updating,
	// and closes when something is selected or typing renders
	// it no longer relevant.
	CompleterStage
)

// StageSides are the Sides for Sheet Stages
type StageSides int32 //enums:enum

const (
	// BottomSheet anchors Sheet to the bottom of the window, with handle on the top
	BottomSheet StageSides = iota

	// SideSheet anchors Sheet to the side of the window, with handle on the top
	SideSheet
)

// Stage is a container and manager for displaying a Scene
// in different functional ways, defined by StageTypes, in two categories:
// Main types (Window, Dialog, Sheet) and Popup types
// (Menu, Tooltip, Snackbar, Chooser).
type Stage struct { //gti:add -setters
	// type of Stage: determines behavior and Styling
	Type StageTypes `set:"-"`

	// Scene contents of this Stage -- what it displays
	Scene *Scene `set:"-"`

	// widget in another scene that requested this stage to be created
	// and provides context (stage)
	Context Widget

	// name of the Stage -- generally auto-set based on Scene Name
	Name string

	// Title of the Stage -- generally auto-set based on Scene Title.
	// Used for title of Window and Dialog types.
	Title string

	// if true, blocks input to all other stages.
	Modal bool

	// if true, places a darkening scrim over other stages, if not a full window
	Scrim bool

	// if true dismisses the Stage if user clicks anywhere off the Stage
	ClickOff bool

	// whether to send no events to the stage and just pass them down to lower stages
	IgnoreEvents bool

	// NewWindow: if true, opens a Window or Dialog in its own separate operating
	// system window (RenderWin).  This is by default true for Window on Desktop, otherwise false.
	NewWindow bool

	// if NewWindow is false, then this makes Dialogs and Windows take up
	// the entire window they are created in.
	FullWindow bool

	// for Dialogs: if true includes a close button for closing
	Closeable bool

	// for Dialogs: adds a handle titlebar Decor for moving
	Movable bool

	// for Dialogs: adds a resize handle Decor for resizing
	Resizable bool

	// Side for Stages that can operate on different sides, e.g.,
	// for Sheets: which side does the sheet come out from
	Side StageSides

	// Data is item represented by this main stage -- used for recycling windows
	Data any

	// manager for the popups in this stage
	PopupMgr StageMgr

	// the parent stage manager for this stage, which lives in a RenderWin
	MainMgr *StageMgr

	// sprites are named images that are rendered last overlaying everything else.
	Sprites Sprites `json:"-" xml:"-"`

	// name of sprite that is being dragged -- sprite event function is responsible for setting this.
	SpriteDragging string `json:"-" xml:"-"`

	// Main is the MainStage that owns this Popup (via its PopupMgr)
	Main *Stage

	// if > 0, disappears after a timeout duration
	Timeout time.Duration
}

func (st *Stage) String() string {
	str := fmt.Sprintf("%s Type: %s", st.Name, st.Type)
	if st.Scene != nil {
		str += "  Scene: " + st.Scene.Name()
	}
	rc := st.RenderCtx()
	if rc != nil {
		str += "  Rc: " + rc.String()
	}
	return str
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
	st.Name = sc.Name() + "-" + strings.ToLower(st.Type.String())
	if sc.Body != nil {
		st.Title = sc.Body.Title
	}
	return st
}

func (st *Stage) SetScene(sc *Scene) *Stage {
	st.Scene = sc
	if sc != nil {
		sc.Stage = st
		st.SetNameFromScene()
	}
	return st
}

func (st *Stage) SetType(typ StageTypes) *Stage {
	st.Type = typ
	switch st.Type {
	case WindowStage:
		if !goosi.TheApp.Platform().IsMobile() {
			st.NewWindow = true
		}
		st.FullWindow = true
		st.Modal = true // note: there is no global modal option between RenderWin windows
	case DialogStage:
		st.Modal = true
		st.Scrim = true
		st.ClickOff = true
	case SheetStage:
		st.Modal = true
		st.Scrim = true
		st.ClickOff = true
		st.Resizable = true
	case MenuStage:
		st.Modal = true
		st.Scrim = false
		st.ClickOff = true
	case TooltipStage:
		st.Modal = false
		st.ClickOff = true
		st.Scrim = false
		st.IgnoreEvents = true
	case SnackbarStage:
		st.Modal = false
	case CompleterStage:
		st.Modal = false
		st.Scrim = false
		st.ClickOff = true
	}
	return st
}

// Run does the default run behavior based on the type of stage
func (st *Stage) Run() *Stage {
	if st.Scene == nil {
		slog.Error("stage has nil scene")
	}
	switch st.Type {
	case WindowStage:
		return st.RunWindow()
	case DialogStage:
		return st.RunDialog()
	case SheetStage:
		return st.RunSheet()
	default:
		return st.RunPopup()
	}
}

// Wait waits for the window to close.
// This should be included after the main window Run() call.
func (st *Stage) Wait() *Stage {
	Wait()
	return st
}

// DoUpdate for base just calls DoUpdate on scene
// returns stageMods = true if any Stages have been modified
// and sceneMods = true if any Scenes have been modified.
func (st *Stage) PopupDoUpdate() (stageMods, sceneMods bool) {
	if st.Scene == nil {
		return
	}
	sceneMods = st.Scene.DoUpdate()
	// if sceneMods {
	// 	fmt.Println("scene mod", st.Scene.Name())
	// }
	return
}
