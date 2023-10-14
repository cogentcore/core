// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"
	"log/slog"
	"reflect"
	"strings"

	"goki.dev/gi/v2/gi"
	"goki.dev/goosi/events/key"
	"goki.dev/grease"
	"goki.dev/gti"
	"goki.dev/icons"
)

// these are special menus that we ignore
var specialMenus = map[string]struct{}{
	"AppMenu": {}, "Copy Cut Paste": {}, "Copy Cut Paste Dupe": {}, "RenderWins": {},
}

// ActionUpdateFunc is a function that updates method active / inactive status
// first argument is the object on which the method is defined (receiver)
type ActionUpdateFunc func(it any, act *gi.Button)

// SubMenuFunc is a function that returns a string slice of submenu items
// used in MethodView submenu-func option
// first argument is the object on which the method is defined (receiver)
type SubMenuFunc func(it any, vp *gi.Scene) []string

// SubSubMenuFunc is a function that returns a slice of string slices
// to create submenu items each having their own submenus.
// used in MethodView submenu-func option
// first argument is the object on which the method is defined (receiver)
type SubSubMenuFunc func(it any, vp *gi.Scene) [][]string

// ShortcutFunc is a function that returns a key.Chord string for a shortcut
// used in MethodView shortcut-func option
// first argument is the object on which the method is defined (receiver)
type ShortcutFunc func(it any, act *gi.Button) key.Chord

// LabelFunc is a function that returns a string to set a label
// first argument is the object on which the method is defined (receiver)
type LabelFunc func(it any, act *gi.Button) string

// ToolbarOpts contains the options for a toolbar button. These are the
// options passed to the `gi:toolbar` comment directive.
//
//gti:add
type ToolbarOpts struct {
	// Label is the label for the toolbar button.
	// It defaults to the sentence case version of the
	// name of the function.
	Label string
	// Icon is the icon for the toolbar button. If there
	// is an icon with the same name as the function, it
	// defaults to that icon.
	Icon icons.Icon
	// Tooltip is the tooltip for the toolbar button.
	// It defaults to the documentation for the function.
	Tooltip string
	// SepBefore is whether to insert a separator before the toolbar button.
	SepBefore bool
	// SepAfter is whether to insert a separator after the toolbar button.
	SepAfter bool
}

// ToolbarView adds the toolbar buttons for the given value to the given toolbar.
// It returns whether any toolbar buttons were added.
func ToolbarView(val any, tb *gi.Toolbar) bool {
	typ := gti.TypeByValue(val)
	if typ == nil {
		return false
	}
	gotAny := false
	for _, kv := range typ.Methods.Order {
		met := kv.Val
		var tbDir *gti.Directive
		for _, dir := range met.Directives {
			if dir.Tool == "gi" && dir.Directive == "toolbar" {
				tbDir = dir
				break
			}
		}
		if tbDir == nil {
			continue
		}
		opts := &ToolbarOpts{
			Label:   met.Name,
			Tooltip: met.Doc,
		}
		// we default to the icon with the same name as
		// the method, if it exists
		ic := icons.Icon(strings.ToLower(met.Name))
		if ic.IsValid() {
			opts.Icon = ic
		}
		_, err := grease.SetFromArgs(opts, tbDir.Args, grease.ErrNotFound)
		if err != nil {
			slog.Error("programmer error: error while parsing args to `gi:toolbar` comment directive", "err", err.Error())
			continue
		}
		gotAny = true
		if opts.SepBefore {
			tb.AddSeparator()
		}
		tb.AddButton(gi.ActOpts{Label: opts.Label, Icon: opts.Icon, Tooltip: opts.Tooltip}, func(bt *gi.Button) {
			fmt.Println("calling method", met.Name)
		})
		if opts.SepAfter {
			tb.AddSeparator()
		}
	}
	return gotAny
}

// ArgData contains the relevant data for each arg, including the
// reflect.Value, name, optional description, and default value
type ArgData struct {
	Val     reflect.Value
	Name    string
	Desc    string
	View    Value
	Default any
	Flags   ArgDataFlags
}

// ArgDataFlags define bitflags for method view action options
type ArgDataFlags int64 //enums:bitflag

const (
	// ArgDataHasDef means that there was a Default value set
	ArgDataHasDef ArgDataFlags = iota

	// ArgDataValSet means that there is a fixed value for this arg, given in
	// the config props and set in the Default, so it does not need to be
	// prompted for
	ArgDataValSet
)

func (ad *ArgData) HasDef() bool {
	return ad.Flags.HasFlag(ArgDataHasDef)
}

func (ad *ArgData) SetHasDef() {
	ad.Flags.SetFlag(true, ArgDataHasDef)
}

func (ad *ArgData) HasValSet() bool {
	return ad.Flags.HasFlag(ArgDataValSet)
}

func CallMethod(val any, method string, vp *gi.Scene) bool {
	return false
}

/*  todo: this needs a full rewrite in light of gti etc.

// MainMenuView configures the given MenuBar according to the "MainMenu"
// properties registered on the type for given value element, through the
// kit.AddType method.  See https://goki.dev/gi/v2/wiki/Views for full
// details on formats and options for configuring the menu.  Returns false if
// there is no main menu defined for this type, or on errors (which are
// programmer errors sent to log).
// gopy:interface=handle
func MainMenuView(val any, win *gi.RenderWin, mbar *gi.MenuBar) bool {
	tpp, vtyp, ok := MethodViewTypeProps(val)
	if !ok {
		return false
	}
	mp, ok := ki.SliceTypeProps(tpp, "MainMenu")
	if !ok {
		return false
	}

	if win == nil {
		return false
	}
	if mbar == nil {
		mbar = win.AddMainMenu()
	}

	mnms := make([]string, len(mp))
	for mmi, mm := range mp {
		if mm.Name == "AppMenu" {
			mnms[mmi] = goosi.TheApp.Name()
		} else {
			mnms[mmi] = mm.Name
		}
	}
	mbar.ConfigMenus(mnms)
	rval := true
	for mmi, mm := range mp {
		ma := mbar.Child(mmi).(*gi.Button)
		if mm.Name == "AppMenu" {
			ma.Menu.AddAppMenu(win)
			continue
		}
		if mm.Name == "Edit" {
			if ms, ok := mm.Value.(string); ok {
				if ms == "Copy Cut Paste" {
					ma.Menu.AddCopyCutPaste(win)
				} else if ms == "Copy Cut Paste Dupe" {
					ma.Menu.AddCopyCutPasteDupe(win)
				} else {
					MethodViewErr(vtyp, fmt.Sprintf("Unrecognized Edit menu special string: %v -- `Copy Cut Paste` is standard", ms))
				}
				continue
			}
		}
		if mm.Name == "RenderWin" {
			if ms, ok := mm.Value.(string); ok {
				if ms == "RenderWins" {
					// automatic
				} else {
					MethodViewErr(vtyp, fmt.Sprintf("Unrecognized RenderWin menu special string: %v -- `RenderWins` is standard", ms))
				}
				continue
			}
		}
		rv := ActionsView(val, vtyp, win.Scene, ma, mm.Value)
		if !rv {
			rval = false
		}
	}
	win.MainMenuUpdated()
	return rval
}

// HasToolbarView returns true if given val has a Toolbar type property
// registered -- call this to check before then calling ToolbarView.
func HasToolbarView(val any) bool {
	tpp, _, ok := MethodViewTypeProps(val)
	if !ok {
		return false
	}
	_, ok = ki.SliceTypeProps(tpp, "Toolbar")
	return ok
}

// ToolbarView configures Toolbar according to the "Toolbar" properties
// registered on the type for given value element, through the kit.AddType
// method.  See https://goki.dev/gi/v2/wiki/Views for full details on
// formats and options for configuring the menu.  Returns false if there is no
// toolbar defined for this type, or on errors (which are programmer errors
// sent to log).
func ToolbarView(val any, vp *gi.Scene, tb *gi.Toolbar) bool {
	tpp, vtyp, ok := MethodViewTypeProps(val)
	if !ok {
		return false
	}
	tp, ok := ki.SliceTypeProps(tpp, "Toolbar")
	if !ok {
		return false
	}

	if vp == nil {
		vp = tb.ParentScene()
		if vp == nil {
			MethodViewErr(vtyp, "Scene is nil in ToolbarView config -- must set scene in widget prior to calling this method!")
		}
		return false
	}

	rval := true
	for _, te := range tp {
		if strings.HasPrefix(te.Name, "sep-") {
			sep := tb.NewChild(gi.TypeSeparator, te.Name).(*gi.Separator)
			sep.Horiz = false
			continue
		}
		var ac *gi.Button
		if aci := tb.ChildByName(te.Name, 0); aci != nil { // allows overriding of defaults etc
			ac = aci.(*gi.Button)
			//			fmt.Printf("Toolbar action override: %v\n", ac.Nm)
			ac.ActionSig.DisconnectAll()
		} else {
			ac = tb.NewChild(gi.ButtonType, te.Name).(*gi.Button)
		}
		rv := ActionsView(val, vtyp, vp, ac, te.Value)
		if !rv {
			rval = false
		}
	}
	return rval
}

// CtxtMenuView configures a popup context menu according to the "CtxtMenu"
// properties registered on the type for given value element, through the
// kit.AddType method.  See https://goki.dev/gi/v2/wiki/Views for full
// details on formats and options for configuring the menu.  It looks first
// for "CtxtMenuActive" or "CtxtMenuInactive" depending on inactive flag
// (which applies to the gui view), so you can have different menus in those
// cases, and then falls back on "CtxtMenu".  Returns false if there is no
// context menu defined for this type, or on errors (which are programmer
// errors sent to log).
func CtxtMenuView(val any, inactive bool, vp *gi.Scene, menu *gi.Menu) bool {
	tpp, vtyp, ok := MethodViewTypeProps(val)
	if !ok {
		return false
	}
	var tp ki.PropSlice
	got := false
	if inactive {
		tp, got = ki.SliceTypeProps(tpp, "CtxtMenuInactive")
	} else {
		tp, got = ki.SliceTypeProps(tpp, "CtxtMenuActive")
	}
	if !got {
		tp, got = ki.SliceTypeProps(tpp, "CtxtMenu")
	}
	if !got {
		return false
	}

	if vp == nil {
		MethodViewErr(vtyp, "Scene is nil in CtxtMenuView config -- must set scene in widget prior to calling this method!")
		return false
	}

	rval := true
	for _, te := range tp {
		if strings.HasPrefix(te.Name, "sep-") {
			menu.AddSeparator(te.Name)
			continue
		}
		ac := menu.AddButton(gi.ActOpts{Label: te.Name}, nil, nil)
		rv := ActionsView(val, vtyp, vp, ac, te.Value)
		if !rv {
			rval = false
		}
	}
	return rval
}

//////////////////////////////////////////////////////////////////////////////////
//    CallMethod -- auto gui

// CallMethod calls given method on given object val, using GUI interface to
// prompt for args.  This only works for methods that have been configured
// either on the CallMethods list or any of the Toolbar, MainMenu, or CtxtMenu
// lists (in that order).  List of available methods is cached in type
// properties after first call.
// gopy:interface=handle
func CallMethod(val any, method string, vp *gi.Scene) bool {
	tpp, vtyp, ok := MethodViewTypeProps(val)
	if !ok {
		MethodViewErr(vtyp, fmt.Sprintf("Type: %v properties not found for CallMethod -- need to register type using kit.AddType\n", vtyp.String()))
		return false
	}
	cmp, ok := ki.SubTypeProps(tpp, MethodViewCallMethsProp)
	if !ok {
		cmp = MethodViewCompileMeths(val, vp)
	}

	acp, has := cmp[method]
	if !has {
		MethodViewErr(vtyp, fmt.Sprintf("Method: %v not found among all different methods registered on type properties -- add to CallMethods to make available for CallMethod\n", method))
		return false
	}
	ac, ok := acp.(*gi.Button)
	if !ok {
		MethodViewErr(vtyp, fmt.Sprintf("Method: %v not a gi.Button -- should be!\n", method))
		return false
	}

	MethodViewSetActionData(ac, val, vp)
	ac.Trigger()
	return true
}

// MethodViewSetActionData sets the MethodViewData associated with the given action
// with values updated from the given val and scene
func MethodViewSetActionData(ac *gi.Button, val any, vp *gi.Scene) {
	if ac.Data == nil {
		fmt.Printf("giv.MethodView no MethodViewData on action: %v\n", ac.Nm)
		return
	}
	md := ac.Data.(*MethodViewData)
	md.Val = val
	md.ValVal = reflect.ValueOf(val)
	md.Sc = vp
	md.MethVal = md.ValVal.MethodByName(md.Method)
	if len(ac.ActionSig.Cons) == 0 {
		fmt.Printf("giv.MethodView CallMethod had no connections: %v\n", ac.Nm)
		ac.ActionSig.Connect(vp.This(), MethodViewCall)
	}
}

var compileMethsOrder = []string{"CallMethods", "Toolbar", "MainMenu", "CtxtMenuActive", "CtxtMenu", "CtxtMenuInactive"}

// MethodViewCompileMeths gets all methods either on the CallMethods list or any
// of the Toolbar, MainMenu, or CtxtMenu lists (in that order).  Returns
// property list of them, which are just names -> Actions
func MethodViewCompileMeths(val any, vp *gi.Scene) ki.Props {
	tpp, vtyp, ok := MethodViewTypeProps(val)
	if !ok {
		return nil
	}
	var cmp ki.Props = make(ki.Props)
	for _, lst := range compileMethsOrder {
		tp, got := ki.SliceTypeProps(tpp, lst)
		if !got {
			continue
		}
		MethodViewCompileActions(cmp, val, vtyp, vp, "", tp)
	}
	// kit.SetTypeProp(tpp, MethodViewCallMethsProp, cmp)
	return cmp
}

// MethodViewCompileActions processes properties for parent action pa for
// overall object val of given type -- could have a sub-menu of further
// actions or might just be a single action
func MethodViewCompileActions(cmp ki.Props, val any, vtyp reflect.Type, vp *gi.Scene, pnm string, pp any) bool {
	rval := true
	if pv, ok := pp.(ki.PropSlice); ok {
		for _, mm := range pv {
			_, isspec := specialMenus[mm.Name]
			if strings.HasPrefix(mm.Name, "sep-") || isspec {
				continue
			} else {
				rv := MethodViewCompileActions(cmp, val, vtyp, vp, mm.Name, mm.Value)
				if !rv {
					rval = false
				}
			}
		}
	} else {
		_, isspec := specialMenus[pnm]
		if strings.HasPrefix(pnm, "sep-") || isspec {
			return rval
		}
		if _, has := cmp[pnm]; has {
			return rval
		}
		ac := &gi.Button{}
		ac.InitName(ac, pnm)
		ac.Text = strings.Replace(strings.Join(camelcase.Split(ac.Nm), " "), "  ", " ", -1)
		cmp[pnm] = ac
		rv := false
		switch pv := pp.(type) {
		case ki.BlankProp:
			rv = ActionView(val, vtyp, vp, ac, nil)
		case ki.Props:
			rv = ActionView(val, vtyp, vp, ac, pv)
		}
		if !rv {
			rval = false
		}
	}
	return rval
}

//////////////////////////////////////////////////////////////////////////////////
//    Utils

// MethodViewErr is error logging function for MethodView system, showing the type info
func MethodViewErr(vtyp reflect.Type, msg string) {
	if vtyp != nil {
		log.Printf("giv.MethodView for type: %v: debug error: %v\n", vtyp.String(), msg)
	} else {
		log.Printf("giv.MethodView debug error: %v\n", msg)
	}
}

// MethodViewTypeProps gets props, typ of val, returns false if not found or
// other err
func MethodViewTypeProps(val any) (ki.Props, reflect.Type, bool) {
	if laser.AnyIsNil(val) {
		return nil, nil, false
	}
	vtyp := reflect.TypeOf(val)
	// tpp := kit.Types.Properties(kit.NonPtrType(vtyp), false)
	// if tpp == nil {
	// 	return nil, vtyp, false
	// }
	return *tpp, vtyp, true
}

// HasMainMenuView returns true if given val has a MainMenu type property
// registered -- call this to check before then calling MainMenuView
func HasMainMenuView(val any) bool {
	tpp, _, ok := MethodViewTypeProps(val)
	if !ok {
		return false
	}
	_, ok = ki.SliceTypeProps(tpp, "MainMenu")
	return ok
}

// MethodViewNoUpdateAfterProp returns true if given val has a top-level "MethodViewNoUpdateAfter"
// type property registered -- some types generically want that and it is much easier to
// just specify once instead of every time..
func MethodViewNoUpdateAfterProp(val any) bool {
	tpp, _, ok := MethodViewTypeProps(val)
	if !ok {
		return false
	}
	// _, nua := kit.TypeProp(tpp, "MethodViewNoUpdateAfter")
	// return nua
}

// This is the name of the property that holds cached map of compiled callable methods
var MethodViewCallMethsProp = "__MethodViewCallMeths"

//////////////////////////////////////////////////////////////////////////////////
//    ActionsView

// ActionsView processes properties for parent action pa for overall object
// val of given type -- could have a sub-menu of further actions or might just
// be a single action
func ActionsView(val any, vtyp reflect.Type, vp *gi.Scene, pa *gi.Button, pp any) bool {
	pa.Text = strings.Replace(strings.Join(camelcase.Split(pa.Nm), " "), "  ", " ", -1)
	rval := true
	switch pv := pp.(type) {
	case ki.PropSlice:
		for _, mm := range pv {
			if strings.HasPrefix(mm.Name, "sep-") {
				pa.Menu.AddSeparator(mm.Name)
			} else {
				nac := &gi.Button{}
				nac.InitName(nac, mm.Name)
				nac.SetAsMenu()
				pa.Menu = append(pa.Menu, nac.This().(gi.Widget))
				rv := ActionsView(val, vtyp, vp, nac, mm.Value)
				if !rv {
					rval = false
				}
			}
		}
	case ki.BlankProp:
		rv := ActionView(val, vtyp, vp, pa, nil)
		if !rv {
			rval = false
		}
	case ki.Props:
		rv := ActionView(val, vtyp, vp, pa, pv)
		if !rv {
			rval = false
		}
	}
	return rval
}

// ActionView configures given action with given props
func ActionView(val any, vtyp reflect.Type, vp *gi.Scene, ac *gi.Button, props ki.Props) bool {
	// special action names
	switch ac.Nm {
	case "Close RenderWin":
		ac.Shortcut = gi.ShortcutForFun(gi.KeyFunWinClose)
		ac.ActionSig.Connect(vp.Win.This(), func(recv, send ki.Ki, sig int64, data any) {
			vp.Win.CloseReq()
		})
		return true
	}

	// other special cases based on props
	nometh := false // set to true if doesn't have an actual method to call, e.g., keyfun
	for pk := range props {
		switch pk {
		case "keyfun":
			nometh = true
		}
	}

	methNm := ac.Nm
	methTyp, hasmeth := vtyp.MethodByName(methNm)
	if !nometh && !hasmeth {
		MethodViewErr(vtyp, fmt.Sprintf("ActionView for Method: %v -- not found in type", methNm))
		return false
	}
	valval := reflect.ValueOf(val)
	methVal := valval.MethodByName(methNm)
	if !nometh && (laser.ValueIsZero(methVal) || methVal.IsNil()) {
		MethodViewErr(vtyp, fmt.Sprintf("ActionView for Method: %v -- method value not valid", methNm))
		return false
	}

	rval := true
	md := &MethodViewData{Val: val, ValVal: valval, Sc: vp, Method: methNm, MethVal: methVal, MethTyp: methTyp}
	ac.Data = md

	if MethodViewNoUpdateAfterProp(val) {
		// bitflag.Set32((*int32)(&md.Flags), int(MethodViewNoUpdateAfter))
	}

	if props == nil {
		ac.ActionSig.Connect(vp.This(), MethodViewCall)
		return true
	}
	for pk, pv := range props {
		switch pk {
		case "shortcut":
			if kf, ok := pv.(gi.KeyFuns); ok {
				ac.Shortcut = gi.ShortcutForFun(kf)
			} else {
				ac.Shortcut = key.Chord(laser.ToString(pv)).OSShortcut()
			}
		case "shortcut-func":
			if sf, ok := pv.(ShortcutFunc); ok {
				ac.Shortcut = sf(md.Val, ac)
			} else if sf, ok := pv.(func(it any, act *gi.Button) key.Chord); ok {
				ac.Shortcut = sf(md.Val, ac)
			} else {
				MethodViewErr(vtyp, fmt.Sprintf("ActionView for Method: %v, shortcut-func must be of type ShortcutFunc", methNm))
			}
		case "keyfun":
			if kf, ok := pv.(gi.KeyFuns); ok {
				ac.Shortcut = gi.ShortcutForFun(kf)
				md.KeyFun = kf
				// bitflag.Set32((*int32)(&md.Flags), int(MethodViewKeyFun))
			}
		case "label":
			ac.Text = laser.ToString(pv)
		case "label-func":
			if sf, ok := pv.(LabelFunc); ok {
				str := sf(md.Val, ac)
				ac.Text = str
			} else if sf, ok := pv.(func(it any, act *gi.Button) string); ok {
				ac.Text = sf(md.Val, ac)
			} else {
				MethodViewErr(vtyp, fmt.Sprintf("ActionView for Method: %v, label-func must be of type LabelFunc", methNm))
			}
		case "icon":
			ac.Icon = icons.Icon(laser.ToString(pv))
		case "desc":
			md.Desc = laser.ToString(pv)
			ac.Tooltip = md.Desc
		case "confirm":
			// bitflag.Set32((*int32)(&md.Flags), int(MethodViewConfirm))
		case "show-return":
			// bitflag.Set32((*int32)(&md.Flags), int(MethodViewShowReturn))
		case "no-update-after":
			// bitflag.Set32((*int32)(&md.Flags), int(MethodViewNoUpdateAfter))
		case "update-after": // if MethodViewNoUpdateAfterProp was set above
			// bitflag.Clear32((*int32)(&md.Flags), int(MethodViewNoUpdateAfter))
		case "updtfunc":
			if uf, ok := pv.(ActionUpdateFunc); ok {
				md.UpdateFunc = uf
				ac.UpdateFunc = MethodViewUpdateFunc
			} else if uf, ok := pv.(func(it any, act *gi.Button)); ok {
				md.UpdateFunc = ActionUpdateFunc(uf)
				ac.UpdateFunc = MethodViewUpdateFunc
			} else {
				MethodViewErr(vtyp, fmt.Sprintf("ActionView for Method: %v, updtfunc must be of type ActionUpdateFunc", methNm))
			}
		case "submenu":
			ac.MakeMenuFunc = MethodViewSubMenuFunc
			if pvs, ok := pv.(string); ok { // field name
				md.SubMenuField = pvs
			} else {
				md.SubMenuSlice = pv
			}
			// bitflag.Set32((*int32)(&md.Flags), int(MethodViewHasSubMenu))
		case "submenu-func":
			if sf, ok := pv.(SubMenuFunc); ok {
				ac.MakeMenuFunc = MethodViewSubMenuFunc
				md.SubMenuFunc = sf
				// bitflag.Set32((*int32)(&md.Flags), int(MethodViewHasSubMenu))
			} else if sf, ok := pv.(func(it any, vp *gi.Scene) []string); ok {
				ac.MakeMenuFunc = MethodViewSubMenuFunc
				md.SubMenuFunc = SubMenuFunc(sf)
				// bitflag.Set32((*int32)(&md.Flags), int(MethodViewHasSubMenu))
			} else {
				MethodViewErr(vtyp, fmt.Sprintf("ActionView for Method: %v, submenu-func must be of type SubMenuFunc", methNm))
			}
		case "subsubmenu-func":
			if sf, ok := pv.(SubSubMenuFunc); ok {
				ac.MakeMenuFunc = MethodViewSubMenuFunc
				md.SubSubMenuFunc = sf
				// bitflag.Set32((*int32)(&md.Flags), int(MethodViewHasSubMenu))
			} else if sf, ok := pv.(func(it any, vp *gi.Scene) [][]string); ok {
				ac.MakeMenuFunc = MethodViewSubMenuFunc
				md.SubSubMenuFunc = SubSubMenuFunc(sf)
				// bitflag.Set32((*int32)(&md.Flags), int(MethodViewHasSubMenu))
			} else {
				MethodViewErr(vtyp, fmt.Sprintf("ActionView for Method: %v, subsubmenu-func must be of type SubMenuFunc", methNm))
			}
		case "Args":
			argv, ok := pv.(ki.PropSlice)
			if !ok {
				MethodViewErr(vtyp, fmt.Sprintf("ActionView for Method: %v, Args property must be of type ki.PropSlice, containing names and other properties for each arg", methNm))
				rval = false
			} else {
				if ActionViewArgsValidate(md, vtyp, methTyp, argv) {
					md.ArgProps = argv
				} else {
					rval = false
				}
			}
		}
	}
	if !rval {
		return false
	}
	// if !bitflag.Has32((int32)(md.Flags), int(MethodViewHasSubMenu)) {
	// 	ac.ActionSig.Connect(vp.This(), MethodViewCall)
	// }
	return true
}

// ActionViewArgsValidate validates the Args properties relative to number of args on type
func ActionViewArgsValidate(md *MethodViewData, vtyp reflect.Type, meth reflect.Method, argprops ki.PropSlice) bool {
	mtyp := meth.Type
	narg := mtyp.NumIn()
	apsz := len(argprops)
	if narg-1 != apsz {
		MethodViewErr(vtyp, fmt.Sprintf("Method: %v takes %v args (beyond the receiver), but Args properties only has %v", meth.Name, narg-1, apsz))
		return false
	}
	// if bitflag.Has32((int32)(md.Flags), int(MethodViewHasSubMenu)) && apsz != 1 {
	// 	MethodViewErr(vtyp, fmt.Sprintf("Method: %v has a submenu of values to use as the one arg for it, but it takes %v args (beyond the receiver) -- should only take 1", meth.Name, narg-1))
	// 	return false
	// }

	return true
}

//////////////////////////////////////////////////////////////////////////////////
//    Method Callbacks -- called when Action fires

// MethodViewFlags define bitflags for method view action options
type MethodViewFlags int64 //enums:bitflag

const (
	// MethodViewConfirm confirms action before proceeding
	MethodViewConfirm MethodViewFlags = iota

	// MethodViewShowReturn shows the return value from the method
	MethodViewShowReturn

	// MethodViewNoUpdateAfter means do not update window after method runs (default is to do so)
	MethodViewNoUpdateAfter

	// MethodViewHasSubMenu means that this action has a submenu option --
	// argument values will be selected from the auto-generated submenu
	MethodViewHasSubMenu

	// MethodViewHasSubMenuVal means that this action was called using a submenu
	// and the SubMenuVal has the selected value
	MethodViewHasSubMenuVal

	// MethodViewKeyFun means this action's only function is to emit the key fun
	MethodViewKeyFun
)

// SubMenuFunc is a function that returns a string slice of submenu items
// used in MethodView submenu-func option
// first argument is the object on which the method is defined (receiver)
type SubMenuFunc func(it any, vp *gi.Scene) []string

// SubSubMenuFunc is a function that returns a slice of string slices
// to create submenu items each having their own submenus.
// used in MethodView submenu-func option
// first argument is the object on which the method is defined (receiver)
type SubSubMenuFunc func(it any, vp *gi.Scene) [][]string

// ShortcutFunc is a function that returns a key.Chord string for a shortcut
// used in MethodView shortcut-func option
// first argument is the object on which the method is defined (receiver)
type ShortcutFunc func(it any, act *gi.Button) key.Chord

// LabelFunc is a function that returns a string to set a label
// first argument is the object on which the method is defined (receiver)
type LabelFunc func(it any, act *gi.Button) string

// ActionUpdateFunc is a function that updates method active / inactive status
// first argument is the object on which the method is defined (receiver)
type ActionUpdateFunc func(it any, act *gi.Button)

// MethodViewData is set to the Action.Data field for all MethodView actions,
// containing info needed to actually call the Method on value Val.
type MethodViewData struct {
	Val     any
	ValVal  reflect.Value
	Sc      *gi.Scene
	Method  string
	MethVal reflect.Value
	MethTyp reflect.Method

	// names and other properties of args, in one-to-one with method args
	ArgProps ki.PropSlice `desc:"names and other properties of args, in one-to-one with method args"`

	// props for special action types, e.g., FileView
	SpecProps ki.Props `desc:"props for special action types, e.g., FileView"`

	// prompt shown in arg dialog or confirm prompt dialog
	Desc string `desc:"prompt shown in arg dialog or confirm prompt dialog"`

	// update function defined in properties -- called by our wrapper update function
	UpdateFunc ActionUpdateFunc `desc:"update function defined in properties -- called by our wrapper update function"`

	// value for submenu generation as a literal slice of items of appropriate type for method being called
	SubMenuSlice any `desc:"value for submenu generation as a literal slice of items of appropriate type for method being called"`

	// value for submenu generation as name of field on obj
	SubMenuField string `desc:"value for submenu generation as name of field on obj"`

	// function that will generate submenu items, as []string slice
	SubMenuFunc SubMenuFunc `desc:"function that will generate submenu items, as []string slice"`

	// function that will generate sub-submenu items, as [][]string slice
	SubSubMenuFunc SubSubMenuFunc `desc:"function that will generate sub-submenu items, as [][]string slice"`

	// value that the user selected from submenu for this action -- this should be assigned to the first (only) arg of the method
	SubMenuVal any `desc:"value that the user selected from submenu for this action -- this should be assigned to the first (only) arg of the method"`

	// key function that we emit, if MethodViewKeyFun type
	KeyFun gi.KeyFuns `desc:"key function that we emit, if MethodViewKeyFun type"`
	Flags  MethodViewFlags
}

func (md *MethodViewData) MethName() string {
	typnm := laser.ShortTypeName(md.ValVal.Type())
	methnm := typnm + ":" + md.Method
	return methnm
}

// MethodViewCall is the receiver func for MethodView actions that call a method
// -- it uses the MethodViewData to call the target method.
func MethodViewCall(recv, send ki.Ki, sig int64, data any) {
	ac := send.(*gi.Button)
	md := ac.Data.(*MethodViewData)
	if md.ArgProps == nil { // no args -- just call
		MethodViewCallNoArgPrompt(ac, md, nil)
		return
	}
	// need to prompt for args
	ads, args, nprompt, ok := MethodViewArgData(md)
	if !ok {
		return
	}
	if nprompt == 0 {
		MethodViewCallNoArgPrompt(ac, md, args)
		return
	}
	// check for single arg with action -- do action directly
	if len(ads) == 1 {
		ad := &ads[0]
		if ad.Desc == "" {
			ad.Desc = md.Desc

		}
		if ad.Desc != "" {
			ad.View.SetTag("desc", ad.Desc)
		}
		if ad.View.HasDialog() {
			ad.View.OpenDialog(ad.View.Widget, func(dlg *gi.Dialog) {
				if dlg.Accepted {
					MethodViewCallMeth(md, args)
				}
			})
			return
		}
	}

	ArgViewDialog(md.Sc, ads, DlgOpts{Title: ac.Text, Prompt: md.Desc},
		md.Sc.This(), func(recv, send ki.Ki, sig int64, data any) {
			if sig == int64(gi.DialogAccepted) {
				// ddlg := send.Embed(gi.TypeDialog).(*gi.Dialog)
				MethodViewCallMeth(md, args)
			}
		})
}

// MethodViewCallNoArgPrompt calls the method in case where there is no
// prompting otherwise of the user for arg values -- checks for Confirm case
// or otherwise directly calls method
func MethodViewCallNoArgPrompt(ac *gi.Button, md *MethodViewData, args []reflect.Value) {
	// if bitflag.Has32(int32(md.Flags), int(MethodViewKeyFun)) {
	// 	if md.Sc != nil && md.Sc.Win != nil {
	// 		md.Sc.Win.EventMgr.SendKeyFunEvent(md.KeyFun, false)
	// 	}
	// 	return
	// }
	// if bitflag.Has32(int32(md.Flags), int(MethodViewConfirm)) {
	// 	gi.PromptDialog(md.Sc, gi.DlgOpts{Title: ac.Text, Prompt: md.Desc, Ok: true, Cancel: true}, func(act *gi.Button) {
	// 			if sig == int64(gi.DialogAccepted) {
	// 				MethodViewCallMeth(md, args)
	// 			}
	// 		})
	// } else {
	// 	MethodViewCallMeth(md, args)
	// }
}

// MethodViewCallMeth calls the method with given args, and processes the
// results as specified in the MethodViewData.
func MethodViewCallMeth(md *MethodViewData, args []reflect.Value) {
	rv := md.MethVal.Call(args)
	methnm := md.MethName()
	mtyp := md.MethTyp.Type
	narg := mtyp.NumIn() - 1
	for ai := 0; ai < narg; ai++ {
		ap := md.ArgProps[ai]
		argnm := methnm + "." + ap.Name
		MethArgHist[argnm] = args[ai].Interface()
	}

	// if !bitflag.Has32(int32(md.Flags), int(MethodViewNoUpdateAfter)) {
	// 	md.Sc.SetNeedsFullRender() // always update after all methods -- almost always want that
	// }
	// if bitflag.Has32(int32(md.Flags), int(MethodViewShowReturn)) {
	// 	if len(rv) >= 1 {
	// 		MethodViewShowValue(md.Sc, rv[0], md.Method+" Result", "")
	// 	}
	// }
}

// MethodViewShowValue displays a value in a dialog window (e.g., for MethodViewShowReturn)
func MethodViewShowValue(ctx gi.Widget, val reflect.Value, title, prompt string) {
	if laser.ValueIsZero(val) {
		return
	}
	npv := laser.NonPtrValue(val)
	if laser.ValueIsZero(npv) {
		return
	}
	tk := npv.Type().Kind()
	switch tk {
	case reflect.Struct:
		StructViewDialog(ctx, DlgOpts{Title: title, Prompt: prompt, Ok: true, Cancel: true}, val.Interface(), nil)
	case reflect.Slice:
		if bs, ok := npv.Interface().([]byte); ok {
			TextViewDialog(ctx, DlgOpts{Title: title, Prompt: prompt, Ok: true}, bs, nil)
		} else if bs, ok := val.Interface().([]byte); ok {
			TextViewDialog(ctx, DlgOpts{Title: title, Prompt: prompt, Ok: true}, bs, nil)
		} else {
			SliceViewDialog(ctx, DlgOpts{Title: title, Prompt: prompt, Ok: true, Cancel: true}, val.Interface(), nil, nil, nil)
		}
	case reflect.Map:
		MapViewDialog(ctx, DlgOpts{Title: title, Prompt: prompt, Ok: true, Cancel: true}, val.Interface(), nil, nil)
	default:
		TextViewDialog(ctx, DlgOpts{Title: title, Prompt: prompt, Ok: true}, []byte(npv.String()))
	}

}

// MethArgHist stores the history of method arg values -- used for setting defaults
// for next time the method is called.  Key is type:method name
var MethArgHist = map[string]any{}

// MethodViewArgData gets the arg data for the method args, returns false if
// errors -- nprompt is the number of args that require prompting from the
// user (minus any cases with value: set directly)
func MethodViewArgData(md *MethodViewData) (ads []ArgData, args []reflect.Value, nprompt int, ok bool) {
	mtyp := md.MethTyp.Type
	narg := mtyp.NumIn() - 1
	ads = make([]ArgData, narg)
	args = make([]reflect.Value, narg)
	nprompt = 0
	ok = true

	methnm := md.MethName()

	for ai := 0; ai < narg; ai++ {
		ad := &ads[ai]
		atyp := mtyp.In(1 + ai)
		av := reflect.New(atyp)
		ad.Val = av
		args[ai] = av.Elem()

		aps := &md.ArgProps[ai]
		ad.Name = aps.Name

		argnm := methnm + "." + ad.Name
		if def, has := MethArgHist[argnm]; has {
			ad.Default = def
			ad.SetHasDef()
		}

		ad.View = ToValue(ad.Val.Interface(), "")
		ad.View.SetSoloValue(ad.Val)
		ad.View.SetName(ad.Name)
		nprompt++ // assume prompt

		switch apv := aps.Value.(type) {
		case ki.BlankProp:
		case ki.Props:
			for pk, pv := range apv {
				switch pk {
				case "desc":
					ad.Desc = laser.ToString(pv)
					ad.View.SetTag("desc", ad.Desc)
				case "default":
					ad.Default = pv
					ad.SetHasDef()
				case "value":
					ad.Default = pv
					ad.SetHasDef()
					ad.Flags.Set(true, ArgDataValSet)
					nprompt--
				case "default-field":
					field := pv.(string)
					if flv, ok := MethodViewFieldValue(md.ValVal, field); ok {
						ad.Default = flv.Interface()
						ad.SetHasDef()
					}
				default:
					ad.View.SetTag(pk, laser.ToString(pv))
				}
			}
		}

		if md.Flags.HasFlag(MethodViewHasSubMenuVal) {
			ad.Default = md.SubMenuVal
			ad.SetHasDef()
			ad.Flags.HasFlag(ArgDataValSet)
			nprompt--
		}

		if ad.HasDef() {
			ad.View.SetValue(ad.Default)
		}
	}
	return
}

// MethodViewArgDefaultVal returns the default value of the given argument index
func MethodViewArgDefaultVal(md *MethodViewData, ai int) (any, bool) {
	aps := &md.ArgProps[ai]
	var def any
	got := false
	switch apv := aps.Value.(type) {
	case ki.BlankProp:
	case ki.Props:
		for pk, pv := range apv {
			switch pk {
			case "default":
				def = pv
				got = true
			case "value":
				def = pv
				got = true
			case "default-field":
				field := pv.(string)
				if flv, ok := MethodViewFieldValue(md.ValVal, field); ok {
					def = flv.Interface()
					got = true
				}
			}
		}
	}
	return def, got
}

// MethodViewFieldValue returns a reflect.Value for the given field name,
// checking safely (false if not found)
func MethodViewFieldValue(vval reflect.Value, field string) (*reflect.Value, bool) {
	fv, ok := laser.FieldValueByPath(laser.NonPtrValue(vval).Interface(), field)
	if !ok {
		log.Printf("giv.MethodViewFieldValue: Could not find field %v in type: %v\n", field, vval.Type().String())
		return nil, false
	}
	return &fv, true
}

// MethodViewUpdateFunc is general Action.UpdateFunc that then calls any
// MethodViewData.UpdateFunc from its data
func MethodViewUpdateFunc(act *gi.Button) {
	md := act.Data.(*MethodViewData)
	if md.UpdateFunc != nil && md.Val != nil {
		md.UpdateFunc(md.Val, act)
	}
}

// MethodViewSubMenuFunc is a MakeMenuFunc for items that have submenus
func MethodViewSubMenuFunc(aki ki.Ki, m *gi.Menu) {
	ac := aki.(*gi.Button)
	md := ac.Data.(*MethodViewData)
	smd := md.SubMenuSlice
	if md.SubMenuFunc != nil {
		smd = md.SubMenuFunc(md.Val, md.Sc)
	} else if md.SubSubMenuFunc != nil {
		smd = md.SubSubMenuFunc(md.Val, md.Sc)
	} else if md.SubMenuField != "" {
		if flv, ok := MethodViewFieldValue(md.ValVal, md.SubMenuField); ok {
			smd = flv.Interface()
		}
	}
	if smd == nil {
		return
	}
	sltp := laser.NonPtrType(reflect.TypeOf(smd))
	if sltp.Kind() != reflect.Slice && sltp.Kind() != reflect.Array {
		log.Printf("giv.MethodViewSubMenuFunc: submenu data must be a slice or array, not: %v\n", sltp.String())
		return
	}

	def, gotDef := MethodViewArgDefaultVal(md, 0) // assume first
	defstr := ""
	if gotDef {
		defstr = laser.ToString(def)
	}

	mv := reflect.ValueOf(smd)
	mvnp := laser.NonPtrValue(mv)
	md.MakeMenuSliceValue(mvnp, m, false, defstr, gotDef)
	md.Sc.Win.MainMenuUpdated()
}

func (md *MethodViewData) MakeMenuSliceValue(mvnp reflect.Value, m *gi.Menu, isSub bool, defstr string, gotDef bool) {
	sz := mvnp.Len()
	if sz == 0 {
		return
	}
	*m = make(gi.Menu, 0, sz)
	e1 := mvnp.Index(0)
	if e1.Kind() == reflect.Slice { // multi-slice -- sub-menus
		for i := 0; i < sz; i++ {
			val := mvnp.Index(i)
			if val.Len() < 2 {
				continue
			}
			s1 := val.Index(0)
			nm := laser.ToString(s1)
			nac := &gi.Button{}
			nac.InitName(nac, nm)
			nac.Text = nm
			nac.SetAsMenu()
			*m = append(*m, nac)
			md.MakeMenuSliceValue(val, &nac.Menu, true, defstr, gotDef) // sub
		}
		return
	}
	st := 0
	subMenuName := ""
	if isSub { // skip the first one -- used as a label for the higher-level menu
		subMenuName = laser.ToString(mvnp.Index(0)) + ": "
		st = 1
	}
	for i := st; i < sz; i++ {
		val := mvnp.Index(i)
		vi := val.Interface()
		nm := laser.ToString(val)
		if nm == gi.MenuTextSeparator {
			sp := &gi.Separator{}
			sp.InitName(sp, "sep")
			sp.Horiz = true
			*m = append(*m, sp)
			continue
		}
		nac := &gi.Button{}
		nac.InitName(nac, nm)
		nac.Text = nm
		nac.SetAsMenu()
		nac.ActionSig.Connect(md.Sc.This(), MethodViewCall)
		nd := *md // copy
		if isSub {
			nd.SubMenuVal = subMenuName + nm // qualified name
		} else {
			nd.SubMenuVal = vi
		}
		if gotDef {
			if laser.ToString(vi) == defstr {
				nac.SetSelected()
			}
		}
		nd.Flags.SetFlag(true, MethodViewHasSubMenuVal)
		nac.Data = &nd
		*m = append(*m, nac)
	}
}

*/
