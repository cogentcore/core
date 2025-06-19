// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package core

import (
	"fmt"
	"image"
	"strings"
	"time"

	"cogentcore.org/core/base/option"
	"cogentcore.org/core/paint"
	"cogentcore.org/core/paint/render"
	"cogentcore.org/core/system"
)

// StageTypes are the types of [Stage] containers.
// There are two main categories: MainStage and PopupStage.
// MainStages are [WindowStage] and [DialogStage], which are
// large and potentially complex [Scene]s that persist until
// dismissed. PopupStages are [MenuStage], [TooltipStage],
// [SnackbarStage], and [CompleterStage], which are transitory
// and simple, without additional decorations. MainStages live
// in a [stages] associated with a [renderWindow] and manage
// their own set of PopupStages via another [stages].
type StageTypes int32 //enums:enum

const (
	// WindowStage is a MainStage that displays a [Scene] in a full window.
	// One of these must be created first, as the primary app content,
	// and it typically persists throughout. It fills the [renderWindow].
	// Additional windows can be created either within the same [renderWindow]
	// on all platforms or in separate [renderWindow]s on desktop platforms.
	WindowStage StageTypes = iota

	// DialogStage is a MainStage that displays a [Scene] in a smaller dialog
	// window on top of a [WindowStage], or in a full or separate window.
	// It can be [Stage.Modal] or not.
	DialogStage

	// MenuStage is a PopupStage that displays a [Scene] typically containing
	// [Button]s overlaid on a MainStage. It is typically [Stage.Modal] and
	// [Stage.ClickOff], and closes when an button is clicked.
	MenuStage

	// TooltipStage is a PopupStage that displays a [Scene] with extra text
	// info for a widget overlaid on a MainStage. It is typically [Stage.ClickOff]
	// and not [Stage.Modal].
	TooltipStage

	// SnackbarStage is a PopupStage that displays a [Scene] with text info
	// and an optional additional button. It is displayed at the bottom of the
	// screen. It is typically not [Stage.ClickOff] or [Stage.Modal], but has a
	// [Stage.Timeout].
	SnackbarStage

	// CompleterStage is a PopupStage that displays a [Scene] with text completion
	// options, spelling corrections, or other such dynamic info. It is typically
	// [Stage.ClickOff], not [Stage.Modal], dynamically updating, and closes when
	// something is selected or typing renders it no longer relevant.
	CompleterStage
)

// isMain returns true if this type of Stage is a Main stage that manages
// its own set of popups
func (st StageTypes) isMain() bool {
	return st <= DialogStage
}

// isPopup returns true if this type of Stage is a Popup, managed by another
// Main stage.
func (st StageTypes) isPopup() bool {
	return !st.isMain()
}

// Stage is a container and manager for displaying a [Scene]
// in different functional ways, defined by [StageTypes].
type Stage struct { //types:add -setters

	// Type is the type of [Stage], which determines behavior and styling.
	Type StageTypes `set:"-"`

	// Scene contents of this [Stage] (what it displays).
	Scene *Scene `set:"-"`

	// Context is a widget in another scene that requested this stage to be created
	// and provides context.
	Context Widget

	// Name is the name of the Stage, which is generally auto-set
	// based on the [Scene.Name].
	Name string

	// Title is the title of the Stage, which is generally auto-set
	// based on the [Body.Title]. It used for the title of [WindowStage]
	// and [DialogStage] types, and for a [Text] title widget if
	// [Stage.DisplayTitle] is true.
	Title string

	// Screen specifies the screen number on which a new window is opened
	// by default on desktop platforms. It defaults to -1, which indicates
	// that the first window should open on screen 0 (the default primary
	// screen) and any subsequent windows should open on the same screen as
	// the currently active window. Regardless, the automatically saved last
	// screen of a window with the same [Stage.Title] takes precedence if it exists;
	// see the website documentation on window geometry saving for more information.
	// Use [TheApp].ScreenByName("name").ScreenNumber to get the screen by name.
	Screen int

	// Modal, if true, blocks input to all other stages.
	Modal bool `set:"-"`

	// Scrim, if true, places a darkening scrim over other stages.
	Scrim bool

	// ClickOff, if true, dismisses the [Stage] if the user clicks anywhere
	// off of the [Stage].
	ClickOff bool

	// ignoreEvents is whether to send no events to the stage and
	// just pass them down to lower stages.
	ignoreEvents bool

	// NewWindow, if true, opens a [WindowStage] or [DialogStage] in its own
	// separate operating system window ([renderWindow]). This is true by
	// default for [WindowStage] on non-mobile platforms, otherwise false.
	NewWindow bool

	// FullWindow, if [Stage.NewWindow] is false, makes [DialogStage]s and
	// [WindowStage]s take up the entire window they are created in.
	FullWindow bool

	// Maximized is whether to make a window take up the entire screen on desktop
	// platforms by default. It is different from [Stage.Fullscreen] in that
	// fullscreen makes the window truly fullscreen without decorations
	// (such as for a video player), whereas maximized keeps decorations and just
	// makes it fill the available space. The automatically saved user previous
	// maximized state takes precedence.
	Maximized bool

	// Fullscreen is whether to make a window fullscreen on desktop platforms.
	// It is different from [Stage.Maximized] in that fullscreen makes
	// the window truly fullscreen without decorations (such as for a video player),
	// whereas maximized keeps decorations and just makes it fill the available space.
	// Not to be confused with [Stage.FullWindow], which is for stages contained within
	// another system window. See [Scene.IsFullscreen] and [Scene.SetFullscreen] to
	// check and update fullscreen state dynamically on desktop and web platforms
	// ([Stage.SetFullscreen] sets the initial state, whereas [Scene.SetFullscreen]
	// sets the current state after the [Stage] is already running).
	Fullscreen bool

	// UseMinSize uses a minimum size as a function of the total available size
	// for sizing new windows and dialogs. Otherwise, only the content size is used.
	// The saved window position and size takes precedence on multi-window platforms.
	UseMinSize bool

	// Resizable specifies whether a window on desktop platforms can
	// be resized by the user, and whether a non-full same-window dialog can
	// be resized by the user on any platform. It defaults to true.
	Resizable bool

	// Timeout, if greater than 0, results in a popup stages disappearing
	// after this timeout duration.
	Timeout time.Duration

	// BackButton is whether to add a back button to the top bar that calls
	// [Scene.Close] when clicked. If it is unset, is will be treated as true
	// on non-[system.Offscreen] platforms for [Stage.FullWindow] but not
	// [Stage.NewWindow] [Stage]s that are not the first in the stack.
	BackButton option.Option[bool] `set:"-"`

	// DisplayTitle is whether to display the [Stage.Title] using a
	// [Text] widget in the top bar. It is on by default for [DialogStage]s
	// and off for all other stages.
	DisplayTitle bool

	// Pos is the default target position for the [Stage] to be placed within
	// the surrounding window or screen in raw pixels. For a new window on desktop
	// platforms, the automatically saved user previous window position takes precedence.
	// For dialogs, this position is the target center position, not the upper-left corner.
	Pos image.Point

	// If a popup stage, this is the main stage that owns it (via its [Stage.popups]).
	// If a main stage, it points to itself.
	Main *Stage `set:"-"`

	// For main stages, this is the stack of the popups within it
	// (created specifically for the main stage).
	// For popups, this is the pointer to the popups within the
	// main stage managing it.
	popups *stages

	// For all stages, this is the main [Stages] that lives in a [renderWindow]
	// and manages the main stages.
	Mains *stages `set:"-"`

	// rendering context which has info about the RenderWindow onto which we render.
	// This should be used instead of the RenderWindow itself for all relevant
	// rendering information. This is only available once a Stage is Run,
	// and must always be checked for nil.
	renderContext *renderContext

	// Sprites are named images that are rendered last overlaying everything else.
	Sprites Sprites `json:"-" xml:"-" set:"-"`

	// spritePainter is the painter for sprite drawing.
	spritePainter *paint.Painter

	// spriteRenderer is the renderer for sprite drawing.
	spriteRenderer render.Renderer
}

func (st *Stage) String() string {
	str := fmt.Sprintf("%s Type: %s", st.Name, st.Type)
	if st.Scene != nil {
		str += "  Scene: " + st.Scene.Name
	}
	rc := st.renderContext
	if rc != nil {
		str += "  Rc: " + rc.String()
	}
	return str
}

// SetBackButton sets [Stage.BackButton] using [option.Option.Set].
func (st *Stage) SetBackButton(b bool) *Stage {
	st.BackButton.Set(b)
	return st
}

// setNameFromScene sets the name of this Stage based on existing
// Scene and Type settings.
func (st *Stage) setNameFromScene() *Stage {
	if st.Scene == nil {
		return nil
	}
	sc := st.Scene
	st.Name = sc.Name + "-" + strings.ToLower(st.Type.String())
	if sc.Body != nil {
		st.Title = sc.Body.Title
	}
	return st
}

func (st *Stage) setScene(sc *Scene) *Stage {
	st.Scene = sc
	if sc != nil {
		sc.Stage = st
		st.setNameFromScene()
	}
	return st
}

// setMains sets the [Stage.Mains] to the given stack of main stages,
// and also sets the RenderContext from that.
func (st *Stage) setMains(sm *stages) *Stage {
	st.Mains = sm
	st.renderContext = sm.renderContext
	return st
}

// setPopups sets the [Stage.Popups] and [Stage.Mains] from the given main
// stage to which this popup stage belongs.
func (st *Stage) setPopups(mainSt *Stage) *Stage {
	st.Main = mainSt
	st.Mains = mainSt.Mains
	st.popups = mainSt.popups
	st.renderContext = st.Mains.renderContext
	return st
}

// setType sets the type and also sets default parameters based on that type
func (st *Stage) setType(typ StageTypes) *Stage {
	st.Type = typ
	st.UseMinSize = true
	st.Resizable = true
	st.Screen = -1
	switch st.Type {
	case WindowStage:
		if !TheApp.Platform().IsMobile() {
			st.NewWindow = true
		}
		st.FullWindow = true
		st.Modal = true // note: there is no global modal option between RenderWindow windows
	case DialogStage:
		st.Modal = true
		st.Scrim = true
		st.ClickOff = true
		st.DisplayTitle = true
	case MenuStage:
		st.Modal = true
		st.Scrim = false
		st.ClickOff = true
	case TooltipStage:
		st.Modal = false
		st.ClickOff = true
		st.Scrim = false
		st.ignoreEvents = true
	case SnackbarStage:
		st.Modal = false
	case CompleterStage:
		st.Modal = false
		st.Scrim = false
		st.ClickOff = true
	}
	return st
}

// SetModal sets modal flag for blocking other input (for dialogs).
// Also updates [Stage.Scrim] accordingly if not modal.
func (st *Stage) SetModal(modal bool) *Stage {
	st.Modal = modal
	if !st.Modal {
		st.Scrim = false
	}
	return st
}

// Run runs the stage using the default run behavior based on the type of stage.
func (st *Stage) Run() *Stage {
	if system.OnSystemWindowCreated == nil {
		return st.run()
	}
	// need to prevent premature quitting by ensuring
	// that WinWait is not done until we run the Stage
	windowWait.Add(1)
	go func() {
		<-system.OnSystemWindowCreated
		system.OnSystemWindowCreated = nil // no longer applicable
		st.run()
		// now that we have run the Stage, WinWait is accurate and
		// we no longer need to prevent it from being done
		windowWait.Done()
	}()
	return st
}

// run is the implementation of [Stage.Run].
func (st *Stage) run() *Stage {
	defer func() { system.HandleRecover(recover()) }()
	switch st.Type {
	case WindowStage:
		return st.runWindow()
	case DialogStage:
		return st.runDialog()
	default:
		return st.runPopup()
	}
}

// doUpdate calls doUpdate on our Scene and UpdateAll on our Popups for Main types.
// returns stageMods = true if any Popup Stages have been modified
// and sceneMods = true if any Scenes have been modified.
func (st *Stage) doUpdate() (stageMods, sceneMods bool) {
	if st.Scene == nil {
		return
	}
	if st.Type.isMain() && st.popups != nil {
		stageMods, sceneMods = st.popups.updateAll()
	}
	scMods := st.Scene.doUpdate()
	sceneMods = sceneMods || scMods
	// if stageMods || sceneMods {
	// 	fmt.Println("scene mod", st.Scene.Name, stageMods, scMods)
	// }
	return
}

// raise moves the Stage to the top of its main [stages]
// and raises the [renderWindow] it is in if necessary.
func (st *Stage) raise() {
	if st.Mains.renderWindow != currentRenderWindow {
		st.Mains.renderWindow.Raise()
	}
	st.Mains.moveToTop(st)
	currentRenderWindow.SetStageTitle(st.Title)
}

func (st *Stage) delete() {
	if st.Type.isMain() && st.popups != nil {
		st.popups.deleteAll()
		st.Sprites.reset()
		st.spriteRenderer = nil
		st.spritePainter = nil
	}
	if st.Scene != nil {
		st.Scene.DeleteChildren()
	}
	st.Scene = nil
	st.Main = nil
	st.popups = nil
	st.Mains = nil
	st.renderContext = nil
}
