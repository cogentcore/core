// Copyright (c) 2023, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"
	"strings"

	"cogentcore.org/core/colors"
	"cogentcore.org/core/events"
	"cogentcore.org/core/fi/uri"
	"cogentcore.org/core/goosi"
	"cogentcore.org/core/grr"
	"cogentcore.org/core/icons"
	"cogentcore.org/core/keyfun"
	"cogentcore.org/core/states"
	"cogentcore.org/core/styles"
	"cogentcore.org/core/svg"
	"cogentcore.org/core/units"
)

// App encapsulates various properties of the overall application,
// including managing an AppBar and associated elements.
type App struct { //gti:add -setters
	// Name can be used in relevant window titles and prompts,
	// and specifies the default application-specific data directory
	Name string

	// About sets the 'about' info for the app, which appears as a menu option
	// in the default app menu.
	About string

	// Icon specifies the app icon, which is passed to [goosi.Window.SetIcon].
	// It should typically be set using [App.SetIconSVG].
	Icon []image.Image

	// AppBarConfig is the function that configures the AppBar,
	// typically put in the [Scene.Bars.Top] (i.e., a TopAppBar).
	// Set to StdAppBarConfig by default, which makes the standard AppBar behavior.
	// Most apps will define their own version to add App-specific
	// functionality, and set this accordingly.
	// If this is nil, then no TopAppBar will be created by default.
	AppBarConfig func(pw Widget)
}

// NewApp returns a new App initialized with the given name.
func NewApp(name string) *App {
	// we load the settings in NewApp so that people can
	// add their own settings to AllSettings first
	grr.Log(LoadAllSettings())

	app := &App{}
	app.Name = name
	app.AppBarConfig = StdAppBarConfig
	app.Config()
	return app
}

// NewAppBody returns a new Body with a new App initialized with
// the given name.
func NewAppBody(name string) *Body {
	b := NewBody(name)
	b.SetApp(NewApp(name))
	return b
}

// SetIconSVG sets the icon of the app to the given SVG icon.
// It automatically logs any errors.
func (a *App) SetIconSVG(r io.Reader) *App {
	a.Icon = make([]image.Image, 3)

	sv := svg.NewSVG(16, 16)
	sv.Color = colors.C(colors.FromRGB(66, 133, 244)) // Google Blue (#4285f4)
	sv.Norm = true
	err := sv.ReadXML(r)
	if grr.Log(err) != nil {
		return a
	}

	sv.Render()
	a.Icon[0] = sv.Pixels

	sv.Resize(image.Pt(32, 32))
	sv.Render()
	a.Icon[1] = sv.Pixels

	sv.Resize(image.Pt(48, 48))
	sv.Render()
	a.Icon[2] = sv.Pixels
	return a
}

// SetIconBytes sets the icon of the app to the given SVG icon bytes.
// It automatically logs any errors.
func (a *App) SetIconBytes(b []byte) *App {
	return a.SetIconSVG(bytes.NewReader(b))
}

// Config performs one-time configuration steps after setting
// properties on the App.
func (a *App) Config() {
	goosi.TheApp.SetName(a.Name)
	goosi.TheApp.SetAbout(a.About)
}

// DataDir returns the application-specific data directory:
// [goosi.DataDir] + [App.Name]. It ensures that the directory exists first.
// Use this directory to store all app-specific data including preferences.
// DataDir is: Mac: ~/Library, Linux: ~/.config, Windows: ~/AppData/Roaming
func (a *App) DataDir() string {
	pdir := filepath.Join(goosi.TheApp.DataDir(), a.Name)
	os.MkdirAll(pdir, 0755)
	return pdir
}

// App returns the [App] this widget is contained in
func (wb *WidgetBase) App() *App {
	return wb.Sc.App
}

// Platform returns the current platform / operating system the app is running on.
func Platform() goosi.Platforms {
	return goosi.TheApp.Platform()
}

// todo: deal with this stuff too:

// SetQuitReqFunc sets the function that is called whenever there is a
// request to quit the app (via a OS or a call to QuitReq() method).  That
// function can then adjudicate whether and when to actually call Quit.
func SetQuitReqFunc(fun func()) {
	goosi.TheApp.SetQuitReqFunc(fun)
}

// AddQuitCleanFunc adds the given function to a list that is called whenever
// app is actually about to quit (irrevocably). Can do any necessary
// last-minute cleanup here.
func AddQuitCleanFunc(fun func()) {
	goosi.TheApp.AddQuitCleanFunc(fun)
}

// Quit closes all windows and exits the program.
func Quit() {
	if !goosi.TheApp.IsQuitting() {
		goosi.TheApp.Quit()
	}
}

// QuitReq requests to Quit -- calls QuitReqFunc if present
func QuitReq() {
	goosi.TheApp.QuitReq()
}

// OpenURL opens the given URL in the user's default browser.  On Linux
// this requires that xdg-utils package has been installed -- uses
// xdg-open command.
func OpenURL(url string) {
	goosi.TheApp.OpenURL(url)
}

// CogentCore returns the Cogent Core data directory: [DataDir] + "CogentCore".
// It ensures that the directory exists first.
func CogentCore() string {
	return goosi.TheApp.CogentCore()
}

// DataDir returns the OS-specific data directory: Mac: ~/Library,
// Linux: ~/.config, Windows: ~/AppData/Roaming
func DataDir() string {
	return goosi.TheApp.DataDir()
}

//////////////////////////////////////////////////////////////////////////////
//		AppBar

// StdAppBarConfig is the standard impl for a [App.AppBarConfig].
// It adds a Back navigation buttons and the AppChooser,
// followed by the [Widget.ConfigToolbar] for the current FullWindow
// Scene being viewed, along with [StdOverflowMenu] items.
// and calls AddDefaultOverflowMenu to provide default menu items,
// which will appear below any other OverflowMenu items added.
func StdAppBarConfig(pw Widget) {
	tb := RecycleToolbar(pw)
	StdAppBarStart(tb) // adds back nav and AppChooser
	StdOverflowMenu(tb)
	CurrentWindowAppBar(tb)
	// apps should add their own app-general functions here
}

// StdAppBarStart adds standard items to start of an AppBar:
// [StdAppBarBack] and [StdAppChooser]
func StdAppBarStart(tb *Toolbar) {
	StdAppBarBack(tb)
	StdAppBarChooser(tb)
}

// StdAppBarBack adds a back button
func StdAppBarBack(tb *Toolbar) *Button {
	bt := NewButton(tb, "back").SetIcon(icons.ArrowBack)
	bt.OnClick(func(e events.Event) {
		stg := tb.Sc.Stage.Main
		mm := stg.MainMgr
		// if we are down to the last window, we don't
		// let people close it with the back button
		if mm.Stack.Len() <= 1 {
			return
		}
		tb.Sc.Close()
	})
	return bt
}

// StdAppBarChooser adds an AppChooser
func StdAppBarChooser(tb *Toolbar) *AppChooser {
	return NewAppChooser(tb, "app-chooser")
}

// todo: use CurrentMainScene instead?

// CurrentWindowAppBar calls ConfigToolbar functions registered on
// the Scene to which the given toolbar belongs.
func CurrentWindowAppBar(tb *Toolbar) {
	tb.Sc.AppBars.Call(tb)
}

// StdOverflowMenu adds the standard overflow menu function.
func StdOverflowMenu(tb *Toolbar) {
	tb.OverflowMenus = append(tb.OverflowMenus, tb.StdOverflowMenu)
}

// note: must be a method on toolbar to get scene

// StdOverflowMenu adds standard overflow menu items.
func (tb *Toolbar) StdOverflowMenu(m *Scene) { //gti:add
	if SettingsWindow != nil {
		NewButton(m).SetText("Settings").SetIcon(icons.Settings).SetKey(keyfun.Prefs).
			OnClick(func(e events.Event) {
				SettingsWindow()
			})
	}
	if InspectorWindow != nil {
		NewButton(m).SetText("Inspect").SetIcon(icons.Edit).SetKey(keyfun.Inspector).
			OnClick(func(e events.Event) {
				InspectorWindow(tb.Sc)
			})
	}
	NewButton(m).SetText("Edit").SetMenu(func(m *Scene) {
		// todo: these need to actually do something -- currently just show keyboard shortcut
		NewButton(m).SetText("Copy").SetIcon(icons.ContentCopy).SetKey(keyfun.Copy)
		NewButton(m).SetText("Cut").SetIcon(icons.ContentCut).SetKey(keyfun.Cut)
		NewButton(m).SetText("Paste").SetIcon(icons.ContentPaste).SetKey(keyfun.Paste)
	})
	NewButton(m).SetText("Window").SetMenu(func(m *Scene) {
		NewButton(m).SetText("Focus next").SetIcon(icons.CenterFocusStrong).
			OnClick(func(e events.Event) {
				AllRenderWins.FocusNext()
			})
		NewButton(m).SetText("Minimize").SetIcon(icons.Minimize).
			OnClick(func(e events.Event) {
				win := tb.Sc.RenderWin()
				if win != nil {
					win.Minimize()
				}
			})
		NewSeparator(m)
		NewButton(m).SetText("Close Window").SetIcon(icons.Close).SetKey(keyfun.WinClose).
			OnClick(func(e events.Event) {
				win := tb.Sc.RenderWin()
				if win != nil {
					win.CloseReq()
				}
			})
		NewButton(m).SetText("Quit").SetIcon(icons.Close).SetShortcut("Command+Q").
			OnClick(func(e events.Event) {
				QuitReq()
			})
		NewSeparator(m)
		for _, w := range MainRenderWins {
			if w != nil {
				NewButton(m).SetText(w.Title).OnClick(func(e events.Event) {
					w.Raise()
				})
			}
		}
		if len(DialogRenderWins) > 0 {
			NewSeparator(m)
			for _, w := range DialogRenderWins {
				if w != nil {
					NewButton(m).SetText(w.Title).OnClick(func(e events.Event) {
						w.Raise()
					})
				}
			}
		}
	})
}

//////////////////////////////////////////////////////////////////////////////
//		AppChooser

// AppChooser is an editable chooser element, typically placed at the start
// of the TopAppBar, that provides direct access to all manner of app resources.
type AppChooser struct {
	Chooser

	// Resources are generators for resources accessible by the AppChooser
	Resources uri.Resources
}

func (ac *AppChooser) CopyFieldsFrom(frm any) {
	fr := frm.(*AppChooser)
	ac.Chooser.CopyFieldsFrom(&fr.Chooser)
	ac.Resources = fr.Resources
}

func (ac *AppChooser) OnInit() {
	ac.Chooser.OnInit()
	ac.SetEditable(true).SetType(ChooserOutlined).SetIcon(icons.Search)
	ac.SetItemsFunc(func() {
		stg := ac.Sc.Stage.Main
		mm := stg.MainMgr
		urs := ac.Resources.Generate()
		iln := mm.Stack.Len() + len(urs)
		ac.Items = make([]any, iln)
		ac.Icons = make([]icons.Icon, iln)
		ac.Tooltips = make([]string, iln)
		for i, kv := range mm.Stack.Order {
			nm := ""
			if kv.Val.Scene.Body != nil && kv.Val.Scene.Body.Title != "" {
				nm = kv.Val.Scene.Body.Title
			} else {
				nm = kv.Val.Scene.Name()
				// -scene is frequently placed at the end of scene names, so we remove it
				nm = strings.TrimSuffix(nm, "-scene")
			}
			u := uri.URI{Label: nm, Icon: icons.Toolbar}
			u.SetURL("scene", nm, fmt.Sprintf("%d", i))
			ac.Items[i] = u
			ac.Icons[i] = u.Icon
			ac.Tooltips[i] = u.URL
		}
		st := len(mm.Stack.Order)
		for i, u := range urs {
			ac.Items[st+i] = u
			ac.Icons[st+i] = u.Icon
			ac.Tooltips[st+i] = u.URL
		}
	})
	ac.OnChange(func(e events.Event) {
		stg := ac.Sc.Stage.Main
		mm := stg.MainMgr
		cv, ok := ac.CurVal.(uri.URI)
		if !ok {
			return
		}
		if cv.HasScheme("scene") {
			e.SetHandled()
			// TODO: optimize this?
			kv := mm.Stack.Order[ac.CurIndex] // todo: bad to rely on index!
			mm.Stack.DeleteIdx(ac.CurIndex, ac.CurIndex+1)
			mm.Stack.InsertAtIdx(mm.Stack.Len(), kv.Key, kv.Val)
			return
		}
		if cv.Func != nil {
			e.SetHandled()
			cv.Func()
			return
		}
		ErrorSnackbar(ac, errors.New("unable to process resource: "+cv.String()))
	})
	ac.Style(func(s *styles.Style) {
		// s.GrowWrap = true // note: this won't work because contents not placed until end
		s.Border.Radius = styles.BorderRadiusFull
		s.Background = colors.C(colors.Scheme.SurfaceContainerHighest)
		if s.Is(states.Focused) {
			s.Border.Width.Set(units.Dp(2))
			s.Border.Color.Set(colors.Scheme.Primary.Base)
		} else {
			s.Border.Width.Zero()
			s.Border.Color.Zero()
		}
	})
}

func (ac *AppChooser) OnAdd() {
	ac.WidgetBase.OnAdd()
	ac.OnShow(func(e events.Event) {
		ac.ItemsFunc()
		// our current scene is always the first item,
		// so we select it on show
		ac.SetCurIndex(0)
	})
}
