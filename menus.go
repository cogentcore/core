// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gi

import (
	"image"
	"log"

	"github.com/goki/gi/oswin"
	"github.com/goki/gi/units"
	"github.com/goki/ki"
	"github.com/goki/ki/bitflag"
	"github.com/goki/ki/kit"
)

// Menu is a slice list of Node2D actions, which can contain sub-actions
// (though it can contain anything -- it is just added to a column layout and
// displayed in a popup) -- don't use stretchy sizes in general for these
// items!
type Menu ki.Slice

func (m Menu) MarshalJSON() ([]byte, error) {
	ks := (ki.Slice)(m)
	return ks.MarshalJSON()
}

func (m *Menu) UnmarshalJSON(b []byte) error {
	ks := (*ki.Slice)(m)
	return ks.UnmarshalJSON(b)
}

// MakeMenuFunc is a callback for making a menu on demand
type MakeMenuFunc func(m *Menu)

// AddMenuText adds an action to the menu with a text label, shortcut, and
// connects the action signal to given receiver object, along with given data
// which is stored on the action and then passed in the action signal.
func (m *Menu) AddMenuText(txt, shortcut string, sigTo ki.Ki, data interface{}, fun ki.RecvFunc) *Action {
	if m == nil {
		*m = make(Menu, 0, 10)
	}
	ac := Action{}
	ac.InitName(&ac, txt)
	ac.Text = txt
	ac.Shortcut = OSShortcut(shortcut)
	ac.Data = data
	ac.SetAsMenu()
	*m = append(*m, ac.This.(Node2D))
	if sigTo != nil && fun != nil {
		ac.ActionSig.Connect(sigTo, fun)
	}
	return &ac
}

// AddSeparator adds a separator at the next point in the menu (name is just
// internal label of element, defaults to 'sep' if empty)
func (m *Menu) AddSeparator(name string) *Separator {
	if m == nil {
		*m = make(Menu, 0, 10)
	}
	sp := Separator{}
	if name == "" {
		name = "sep"
	}
	sp.InitName(&sp, name)
	sp.SetProp("min-height", units.NewValue(0.5, units.Em))
	sp.SetProp("max-width", -1)
	sp.Horiz = true
	*m = append(*m, sp.This.(Node2D))
	return &sp
}

// AddLabel adds a label to the menu
func (m *Menu) AddLabel(lbl string) *Label {
	if m == nil {
		*m = make(Menu, 0, 10)
	}
	lb := Label{}
	lb.InitName(&lb, lbl)
	lb.SetText(lbl)
	lb.SetProp("background-color", &Prefs.Colors.Control)
	*m = append(*m, lb.This.(Node2D))
	return &lb
}

// SetShortcuts sets the shortcuts to given window -- call when the menu has
// been attached to a window
func (m *Menu) SetShortcuts(win *Window) {
	if win == nil {
		return
	}
	for _, mi := range *m {
		if ac, ok := mi.(*Action); ok {
			win.AddShortcut(ac.Shortcut, ac)
		}
	}
}

////////////////////////////////////////////////////////////////////////////////////////
// Standard menu elements

// AddCopyCutPaste adds a Copy, Cut, Paste actions that just emit the
// corresponding keyboard shortcut -- cutPasteActive determines whether Cut
// and Paste are active, reflecting the modifiability of relevant element
// (i.e., IsActive)
func (m *Menu) AddCopyCutPaste(win *Window, cutPasteActive bool) {
	cpsc := ActiveKeyMap.ChordForFun(KeyFunCopy)
	ctsc := ActiveKeyMap.ChordForFun(KeyFunCut)
	ptsc := ActiveKeyMap.ChordForFun(KeyFunPaste)
	m.AddMenuText("Copy", cpsc, win, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
		ww := recv.Embed(KiT_Window).(*Window)
		ww.SendKeyFunEvent(KeyFunCopy, false) // false = ignore popups -- don't send to menu
	})
	cut := m.AddMenuText("Cut", ctsc, win, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
		ww := recv.Embed(KiT_Window).(*Window)
		ww.SendKeyFunEvent(KeyFunCut, false) // false = ignore popups -- don't send to menu
	})
	paste := m.AddMenuText("Paste", ptsc, win, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
		ww := recv.Embed(KiT_Window).(*Window)
		ww.SendKeyFunEvent(KeyFunPaste, false) // false = ignore popups -- don't send to menu
	})
	if !cutPasteActive {
		cut.SetInactive()
		paste.SetInactive()
	}
}

// AddAppMenu adds a standard set of menu items for application-level control.
func (m *Menu) AddAppMenu(win *Window) {
	aboutitle := "About " + oswin.TheApp.Name()
	m.AddMenuText(aboutitle, "", win, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
		ww := recv.Embed(KiT_Window).(*Window)
		PromptDialog(ww.Viewport, aboutitle, oswin.TheApp.About(), true, false, nil, nil, nil)
	})
	prsc := ActiveKeyMap.ChordForFun(KeyFunPrefs)
	m.AddMenuText("GoGi Preferences", prsc, win, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
		TheViewIFace.PrefsEditor(&Prefs)
	})
	m.AddSeparator("sepq")
	m.AddMenuText("Quit", "Command+Q", win, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
		oswin.TheApp.QuitReq()
	})
}

// AddWindowsMenu adds menu items for current main and dialog windows.
func (m *Menu) AddWindowsMenu(win *Window) {
	m.AddMenuText("Minimize", "", win, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
		ww := recv.Embed(KiT_Window).(*Window)
		ww.OSWin.Minimize()
	})
	m.AddSeparator("sepa")
	for _, w := range MainWindows {
		if w != nil {
			m.AddMenuText(w.Title, "", w, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
				ww := recv.Embed(KiT_Window).(*Window)
				ww.OSWin.Raise()
			})
		}
	}
	m.AddSeparator("sepw")
	for _, w := range DialogWindows {
		if w != nil {
			m.AddMenuText(w.Title, "", w, nil, func(recv, send ki.Ki, sig int64, data interface{}) {
				ww := recv.Embed(KiT_Window).(*Window)
				ww.OSWin.Raise()
			})
		}
	}
}

////////////////////////////////////////////////////////////////////////////////////////
// PopupMenu function

var MenuFrameProps = ki.Props{
	"border-width":        units.NewValue(0, units.Px),
	"border-color":        "none",
	"margin":              units.NewValue(4, units.Px),
	"padding":             units.NewValue(2, units.Px),
	"box-shadow.h-offset": units.NewValue(2, units.Px),
	"box-shadow.v-offset": units.NewValue(2, units.Px),
	"box-shadow.blur":     units.NewValue(2, units.Px),
	"box-shadow.color":    &Prefs.Colors.Shadow,
}

// PopupMenu pops up a viewport with a layout that draws the supplied actions
// positions are relative to given viewport -- name is relevant base name to
// which Menu is appended
func PopupMenu(menu Menu, x, y int, parVp *Viewport2D, name string) *Viewport2D {
	win := parVp.Win
	mainVp := win.Viewport
	if len(menu) == 0 {
		log.Printf("GoGi PopupMenu: empty menu given\n")
		return nil
	}
	pvp := Viewport2D{}
	pvp.InitName(&pvp, name+"Menu")
	pvp.Win = win
	updt := pvp.UpdateStart()
	pvp.SetProp("color", &Prefs.Colors.Font)
	pvp.Fill = true
	bitflag.Set(&pvp.Flag, int(VpFlagPopup))
	bitflag.Set(&pvp.Flag, int(VpFlagMenu))

	pvp.Geom.Pos = image.Point{x, y}
	// note: not setting VpFlagPopopDestroyAll -- we keep the menu list intact
	frame := pvp.AddNewChild(KiT_Frame, "Frame").(*Frame)
	frame.Lay = LayoutVert
	frame.SetProps(MenuFrameProps, false)
	for _, ac := range menu {
		acn, _ := KiToNode2D(ac)
		if acn != nil {
			frame.AddChild(acn)
		}
	}
	frame.Init2DTree()
	frame.Style2DTree()                                // sufficient to get sizes
	frame.LayData.AllocSize = mainVp.LayData.AllocSize // give it the whole vp initially
	frame.Size2DTree()                                 // collect sizes
	pvp.Win = nil
	scextra := frame.Sty.Layout.ScrollBarWidth.Dots
	frame.LayData.Size.Pref.X += scextra // make room for scrollbar..
	vpsz := frame.LayData.Size.Pref.Min(mainVp.LayData.AllocSize).ToPoint()
	x = kit.MinInt(x, mainVp.Geom.Size.X-vpsz.X) // fit
	y = kit.MinInt(y, mainVp.Geom.Size.Y-vpsz.Y) // fit
	pvp.Resize(vpsz)
	pvp.Geom.Pos = image.Point{x, y}
	pvp.UpdateEndNoSig(updt)

	win.NextPopup = pvp.This
	return &pvp
}

////////////////////////////////////////////////////////////////////////////////////////
// MenuButton pops up a menu, has an indicator by default

type MenuButton struct {
	ButtonBase
}

var KiT_MenuButton = kit.Types.AddType(&MenuButton{}, MenuButtonProps)

var MenuButtonProps = ki.Props{
	"border-width":     units.NewValue(1, units.Px),
	"border-radius":    units.NewValue(4, units.Px),
	"border-color":     &Prefs.Colors.Border,
	"border-style":     BorderSolid,
	"padding":          units.NewValue(4, units.Px),
	"margin":           units.NewValue(4, units.Px),
	"box-shadow.color": &Prefs.Colors.Shadow,
	"text-align":       AlignCenter,
	"background-color": &Prefs.Colors.Control,
	"color":            &Prefs.Colors.Font,
	"#icon": ki.Props{
		"width":   units.NewValue(1, units.Em),
		"height":  units.NewValue(1, units.Em),
		"margin":  units.NewValue(0, units.Px),
		"padding": units.NewValue(0, units.Px),
		"fill":    &Prefs.Colors.Icon,
		"stroke":  &Prefs.Colors.Font,
	},
	"#label": ki.Props{
		"margin":  units.NewValue(0, units.Px),
		"padding": units.NewValue(0, units.Px),
	},
	"#indicator": ki.Props{
		"width":          units.NewValue(1.5, units.Ex),
		"height":         units.NewValue(1.5, units.Ex),
		"margin":         units.NewValue(0, units.Px),
		"padding":        units.NewValue(0, units.Px),
		"vertical-align": AlignBottom,
		"fill":           &Prefs.Colors.Icon,
		"stroke":         &Prefs.Colors.Font,
	},
	"#ind-stretch": ki.Props{
		"width": units.NewValue(1, units.Em),
	},
	ButtonSelectors[ButtonActive]: ki.Props{
		"background-color": "linear-gradient(lighter-0, highlight-10)",
	},
	ButtonSelectors[ButtonInactive]: ki.Props{
		"border-color": "highlight-50",
		"color":        "highlight-50",
	},
	ButtonSelectors[ButtonHover]: ki.Props{
		"background-color": "linear-gradient(highlight-10, highlight-10)",
	},
	ButtonSelectors[ButtonFocus]: ki.Props{
		"border-width":     units.NewValue(2, units.Px),
		"background-color": "linear-gradient(samelight-50, highlight-10)",
	},
	ButtonSelectors[ButtonDown]: ki.Props{
		"color":            "highlight-90",
		"background-color": "linear-gradient(highlight-30, highlight-10)",
	},
	ButtonSelectors[ButtonSelected]: ki.Props{
		"background-color": "linear-gradient(pref(SelectColor), highlight-10)",
	},
}

func (g *MenuButton) ButtonAsBase() *ButtonBase {
	return &(g.ButtonBase)
}

func (g *MenuButton) ConfigParts() {
	config, icIdx, lbIdx := g.ConfigPartsIconLabel(string(g.Icon), g.Text)
	indIdx := g.ConfigPartsAddIndicator(&config, true)  // default on
	mods, updt := g.Parts.ConfigChildren(config, false) // not unique names
	g.ConfigPartsSetIconLabel(string(g.Icon), g.Text, icIdx, lbIdx)
	g.ConfigPartsIndicator(indIdx)
	if mods {
		g.UpdateEnd(updt)
	}
}

////////////////////////////////////////////////////////////////////////////////////////
// Separator

// Separator draws a vertical or horizontal line
type Separator struct {
	WidgetBase
	Horiz bool `xml:"horiz" desc:"is this a horizontal separator -- otherwise vertical"`
}

var KiT_Separator = kit.Types.AddType(&Separator{}, SeparatorProps)

var SeparatorProps = ki.Props{
	"padding":          units.NewValue(2, units.Px),
	"margin":           units.NewValue(2, units.Px),
	"vertical-align":   AlignCenter,
	"horizontal-align": AlignCenter,
	"stroke-width":     units.NewValue(2, units.Px),
	"color":            &Prefs.Colors.Font,
	"stroke":           &Prefs.Colors.Font,
	// todo: dotted
}

func (g *Separator) Render2D() {
	if g.PushBounds() {
		rs := &g.Viewport.Render
		pc := &rs.Paint
		st := &g.Sty
		pc.FontStyle = st.Font
		g.RenderStdBox(st)
		pc.StrokeStyle.SetColor(&st.Font.Color) // ink color
		pc.StrokeStyle.Width = st.Border.Width
		pc.FillStyle.SetColor(nil)

		spc := st.BoxSpace()
		pos := g.LayData.AllocPos.AddVal(spc)
		sz := g.LayData.AllocSize.AddVal(-2.0 * spc)

		if g.Horiz {
			pc.DrawLine(rs, pos.X, pos.Y+0.5*sz.Y, pos.X+sz.X, pos.Y+0.5*sz.Y)
		} else {
			pc.DrawLine(rs, pos.X+0.5*sz.X, pos.Y, pos.X+0.5*sz.X, pos.Y+sz.Y)
		}
		pc.FillStrokeClear(rs)
		g.Render2DChildren()
		g.PopBounds()
	}
}
