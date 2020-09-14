// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package giv

import (
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/fatih/camelcase"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/oswin"
	"github.com/goki/gi/oswin/key"
	"github.com/goki/ki/bitflag"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// these are special menus that we ignore
var specialMenus = map[string]struct{}{
	"AppMenu": {}, "Copy Cut Paste": {}, "Copy Cut Paste Dupe": {}, "Windows": {},
}

// MainMenuView configures the given MenuBar according to the "MainMenu"
// properties registered on the type for given value element, through the
// kit.AddType method.  See https://github.com/goki/gi/wiki/Views for full
// details on formats and options for configuring the menu.  Returns false if
// there is no main menu defined for this type, or on errors (which are
// programmer errors sent to log).
// gopy:interface=handle
func MainMenuView(val interface{}, win *gi.Window, mbar *gi.MenuBar) bool {
	tpp, vtyp, ok := MethViewTypeProps(val)
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
			mnms[mmi] = oswin.TheApp.Name()
		} else {
			mnms[mmi] = mm.Name
		}
	}
	mbar.ConfigMenus(mnms)
	rval := true
	for mmi, mm := range mp {
		ma := mbar.Child(mmi).(*gi.Action)
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
					MethViewErr(vtyp, fmt.Sprintf("Unrecognized Edit menu special string: %v -- `Copy Cut Paste` is standard", ms))
				}
				continue
			}
		}
		if mm.Name == "Window" {
			if ms, ok := mm.Value.(string); ok {
				if ms == "Windows" {
					// automatic
				} else {
					MethViewErr(vtyp, fmt.Sprintf("Unrecognized Window menu special string: %v -- `Windows` is standard", ms))
				}
				continue
			}
		}
		rv := ActionsView(val, vtyp, win.Viewport, ma, mm.Value)
		if !rv {
			rval = false
		}
	}
	win.MainMenuUpdated()
	return rval
}

// HasToolBarView returns true if given val has a ToolBar type property
// registered -- call this to check before then calling ToolBarView.
func HasToolBarView(val interface{}) bool {
	tpp, _, ok := MethViewTypeProps(val)
	if !ok {
		return false
	}
	_, ok = ki.SliceTypeProps(tpp, "ToolBar")
	if !ok {
		return false
	}
	return true
}

// ToolBarView configures ToolBar according to the "ToolBar" properties
// registered on the type for given value element, through the kit.AddType
// method.  See https://github.com/goki/gi/wiki/Views for full details on
// formats and options for configuring the menu.  Returns false if there is no
// toolbar defined for this type, or on errors (which are programmer errors
// sent to log).
func ToolBarView(val interface{}, vp *gi.Viewport2D, tb *gi.ToolBar) bool {
	tpp, vtyp, ok := MethViewTypeProps(val)
	if !ok {
		return false
	}
	tp, ok := ki.SliceTypeProps(tpp, "ToolBar")
	if !ok {
		return false
	}

	if vp == nil {
		vp = tb.ParentViewport()
		if vp == nil {
			MethViewErr(vtyp, "Viewport is nil in ToolBarView config -- must set viewport in widget prior to calling this method!")
		}
		return false
	}

	rval := true
	for _, te := range tp {
		if strings.HasPrefix(te.Name, "sep-") {
			sep := tb.AddNewChild(gi.KiT_Separator, te.Name).(*gi.Separator)
			sep.Horiz = false
			continue
		}
		var ac *gi.Action
		if aci := tb.ChildByName(te.Name, 0); aci != nil { // allows overriding of defaults etc
			ac = aci.(*gi.Action)
			//			fmt.Printf("ToolBar action override: %v\n", ac.Nm)
			ac.ActionSig.DisconnectAll()
		} else {
			ac = tb.AddNewChild(gi.KiT_Action, te.Name).(*gi.Action)
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
// kit.AddType method.  See https://github.com/goki/gi/wiki/Views for full
// details on formats and options for configuring the menu.  It looks first
// for "CtxtMenuActive" or "CtxtMenuInactive" depending on inactive flag
// (which applies to the gui view), so you can have different menus in those
// cases, and then falls back on "CtxtMenu".  Returns false if there is no
// context menu defined for this type, or on errors (which are programmer
// errors sent to log).
func CtxtMenuView(val interface{}, inactive bool, vp *gi.Viewport2D, menu *gi.Menu) bool {
	tpp, vtyp, ok := MethViewTypeProps(val)
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
		MethViewErr(vtyp, "Viewport is nil in CtxtMenuView config -- must set viewport in widget prior to calling this method!")
		return false
	}

	rval := true
	for _, te := range tp {
		if strings.HasPrefix(te.Name, "sep-") {
			menu.AddSeparator(te.Name)
			continue
		}
		ac := menu.AddAction(gi.ActOpts{Label: te.Name}, nil, nil)
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
// either on the CallMethods list or any of the ToolBar, MainMenu, or CtxtMenu
// lists (in that order).  List of available methods is cached in type
// properties after first call.
// gopy:interface=handle
func CallMethod(val interface{}, method string, vp *gi.Viewport2D) bool {
	tpp, vtyp, ok := MethViewTypeProps(val)
	if !ok {
		MethViewErr(vtyp, fmt.Sprintf("Type: %v properties not found for CallMethod -- need to register type using kit.AddType\n", vtyp.String()))
		return false
	}
	cmp, ok := ki.SubTypeProps(tpp, MethodViewCallMethsProp)
	if !ok {
		cmp = MethViewCompileMeths(val, vp)
	}

	acp, has := cmp[method]
	if !has {
		MethViewErr(vtyp, fmt.Sprintf("Method: %v not found among all different methods registered on type properties -- add to CallMethods to make available for CallMethod\n", method))
		return false
	}
	ac, ok := acp.(*gi.Action)
	if !ok {
		MethViewErr(vtyp, fmt.Sprintf("Method: %v not a gi.Action -- should be!\n", method))
		return false
	}

	MethViewSetActionData(ac, val, vp)
	ac.Trigger()
	return true
}

// MethViewSetActionData sets the MethViewData associated with the given action
// with values updated from the given val and viewport
func MethViewSetActionData(ac *gi.Action, val interface{}, vp *gi.Viewport2D) {
	if ac.Data == nil {
		fmt.Printf("giv.MethView no MethViewData on action: %v\n", ac.Nm)
		return
	}
	md := ac.Data.(*MethViewData)
	md.Val = val
	md.ValVal = reflect.ValueOf(val)
	md.Vp = vp
	md.MethVal = md.ValVal.MethodByName(md.Method)
	if len(ac.ActionSig.Cons) == 0 {
		fmt.Printf("giv.MethView CallMethod had no connections: %v\n", ac.Nm)
		ac.ActionSig.Connect(vp.This(), MethViewCall)
	}
}

var compileMethsOrder = []string{"CallMethods", "ToolBar", "MainMenu", "CtxtMenuActive", "CtxtMenu", "CtxtMenuInactive"}

// MethViewCompileMeths gets all methods either on the CallMethods list or any
// of the ToolBar, MainMenu, or CtxtMenu lists (in that order).  Returns
// property list of them, which are just names -> Actions
func MethViewCompileMeths(val interface{}, vp *gi.Viewport2D) ki.Props {
	tpp, vtyp, ok := MethViewTypeProps(val)
	if !ok {
		return nil
	}
	var cmp ki.Props = make(ki.Props)
	for _, lst := range compileMethsOrder {
		tp, got := ki.SliceTypeProps(tpp, lst)
		if !got {
			continue
		}
		MethViewCompileActions(cmp, val, vtyp, vp, "", tp)
	}
	kit.SetTypeProp(tpp, MethodViewCallMethsProp, cmp)
	return cmp
}

// MethViewCompileActions processes properties for parent action pa for
// overall object val of given type -- could have a sub-menu of further
// actions or might just be a single action
func MethViewCompileActions(cmp ki.Props, val interface{}, vtyp reflect.Type, vp *gi.Viewport2D, pnm string, pp interface{}) bool {
	rval := true
	if pv, ok := pp.(ki.PropSlice); ok {
		for _, mm := range pv {
			_, isspec := specialMenus[mm.Name]
			if strings.HasPrefix(mm.Name, "sep-") || isspec {
				continue
			} else {
				rv := MethViewCompileActions(cmp, val, vtyp, vp, mm.Name, mm.Value)
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
		ac := &gi.Action{}
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

// MethViewErr is error logging function for MethView system, showing the type info
func MethViewErr(vtyp reflect.Type, msg string) {
	if vtyp != nil {
		log.Printf("giv.MethodView for type: %v: debug error: %v\n", vtyp.String(), msg)
	} else {
		log.Printf("giv.MethodView debug error: %v\n", msg)
	}
}

// MethViewTypeProps gets props, typ of val, returns false if not found or
// other err
func MethViewTypeProps(val interface{}) (ki.Props, reflect.Type, bool) {
	if kit.IfaceIsNil(val) {
		return nil, nil, false
	}
	vtyp := reflect.TypeOf(val)
	tpp := kit.Types.Properties(kit.NonPtrType(vtyp), false)
	if tpp == nil {
		return nil, vtyp, false
	}
	return *tpp, vtyp, true
}

// HasMainMenuView returns true if given val has a MainMenu type property
// registered -- call this to check before then calling MainMenuView
func HasMainMenuView(val interface{}) bool {
	tpp, _, ok := MethViewTypeProps(val)
	if !ok {
		return false
	}
	_, ok = ki.SliceTypeProps(tpp, "MainMenu")
	if !ok {
		return false
	}
	return true
}

// MethViewNoUpdateAfterProp returns true if given val has a top-level "MethViewNoUpdateAfter"
// type property registered -- some types generically want that and it is much easier to
// just specify once instead of every time..
func MethViewNoUpdateAfterProp(val interface{}) bool {
	tpp, _, ok := MethViewTypeProps(val)
	if !ok {
		return false
	}
	_, nua := kit.TypeProp(tpp, "MethViewNoUpdateAfter")
	return nua
}

// This is the name of the property that holds cached map of compiled callable methods
var MethodViewCallMethsProp = "__MethViewCallMeths"

//////////////////////////////////////////////////////////////////////////////////
//    ActionsView

// ActionsView processes properties for parent action pa for overall object
// val of given type -- could have a sub-menu of further actions or might just
// be a single action
func ActionsView(val interface{}, vtyp reflect.Type, vp *gi.Viewport2D, pa *gi.Action, pp interface{}) bool {
	pa.Text = strings.Replace(strings.Join(camelcase.Split(pa.Nm), " "), "  ", " ", -1)
	rval := true
	switch pv := pp.(type) {
	case ki.PropSlice:
		for _, mm := range pv {
			if strings.HasPrefix(mm.Name, "sep-") {
				pa.Menu.AddSeparator(mm.Name)
			} else {
				nac := &gi.Action{}
				nac.InitName(nac, mm.Name)
				nac.SetAsMenu()
				pa.Menu = append(pa.Menu, nac.This().(gi.Node2D))
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
func ActionView(val interface{}, vtyp reflect.Type, vp *gi.Viewport2D, ac *gi.Action, props ki.Props) bool {
	// special action names
	switch ac.Nm {
	case "Close Window":
		ac.Shortcut = gi.ShortcutForFun(gi.KeyFunWinClose)
		ac.ActionSig.Connect(vp.Win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
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
		MethViewErr(vtyp, fmt.Sprintf("ActionView for Method: %v -- not found in type", methNm))
		return false
	}
	valval := reflect.ValueOf(val)
	methVal := valval.MethodByName(methNm)
	if !nometh && (kit.ValueIsZero(methVal) || methVal.IsNil()) {
		MethViewErr(vtyp, fmt.Sprintf("ActionView for Method: %v -- method value not valid", methNm))
		return false
	}

	rval := true
	md := &MethViewData{Val: val, ValVal: valval, Vp: vp, Method: methNm, MethVal: methVal, MethTyp: methTyp}
	ac.Data = md

	if MethViewNoUpdateAfterProp(val) {
		bitflag.Set32((*int32)(&md.Flags), int(MethViewNoUpdateAfter))
	}

	if props == nil {
		ac.ActionSig.Connect(vp.This(), MethViewCall)
		return true
	}
	for pk, pv := range props {
		switch pk {
		case "shortcut":
			if kf, ok := pv.(gi.KeyFuns); ok {
				ac.Shortcut = gi.ShortcutForFun(kf)
			} else {
				ac.Shortcut = key.Chord(kit.ToString(pv)).OSShortcut()
			}
		case "shortcut-func":
			if sf, ok := pv.(ShortcutFunc); ok {
				ac.Shortcut = sf(md.Val, ac)
			} else if sf, ok := pv.(func(it interface{}, act *gi.Action) key.Chord); ok {
				ac.Shortcut = sf(md.Val, ac)
			} else {
				MethViewErr(vtyp, fmt.Sprintf("ActionView for Method: %v, shortcut-func must be of type ShortcutFunc", methNm))
			}
		case "keyfun":
			if kf, ok := pv.(gi.KeyFuns); ok {
				ac.Shortcut = gi.ShortcutForFun(kf)
				md.KeyFun = kf
				bitflag.Set32((*int32)(&md.Flags), int(MethViewKeyFun))
			}
		case "label":
			ac.Text = kit.ToString(pv)
		case "label-func":
			if sf, ok := pv.(LabelFunc); ok {
				str := sf(md.Val, ac)
				ac.Text = str
			} else if sf, ok := pv.(func(it interface{}, act *gi.Action) string); ok {
				ac.Text = sf(md.Val, ac)
			} else {
				MethViewErr(vtyp, fmt.Sprintf("ActionView for Method: %v, label-func must be of type LabelFunc", methNm))
			}
		case "icon":
			ac.Icon = gi.IconName(kit.ToString(pv))
		case "desc":
			md.Desc = kit.ToString(pv)
			ac.Tooltip = md.Desc
		case "confirm":
			bitflag.Set32((*int32)(&md.Flags), int(MethViewConfirm))
		case "show-return":
			bitflag.Set32((*int32)(&md.Flags), int(MethViewShowReturn))
		case "no-update-after":
			bitflag.Set32((*int32)(&md.Flags), int(MethViewNoUpdateAfter))
		case "update-after": // if MethViewNoUpdateAfterProp was set above
			bitflag.Clear32((*int32)(&md.Flags), int(MethViewNoUpdateAfter))
		case "updtfunc":
			if uf, ok := pv.(ActionUpdateFunc); ok {
				md.UpdateFunc = uf
				ac.UpdateFunc = MethViewUpdateFunc
			} else if uf, ok := pv.(func(it interface{}, act *gi.Action)); ok {
				md.UpdateFunc = ActionUpdateFunc(uf)
				ac.UpdateFunc = MethViewUpdateFunc
			} else {
				MethViewErr(vtyp, fmt.Sprintf("ActionView for Method: %v, updtfunc must be of type ActionUpdateFunc", methNm))
			}
		case "submenu":
			ac.MakeMenuFunc = MethViewSubMenuFunc
			if pvs, ok := pv.(string); ok { // field name
				md.SubMenuField = pvs
			} else {
				md.SubMenuSlice = pv
			}
			bitflag.Set32((*int32)(&md.Flags), int(MethViewHasSubMenu))
		case "submenu-func":
			if sf, ok := pv.(SubMenuFunc); ok {
				ac.MakeMenuFunc = MethViewSubMenuFunc
				md.SubMenuFunc = sf
				bitflag.Set32((*int32)(&md.Flags), int(MethViewHasSubMenu))
			} else if sf, ok := pv.(func(it interface{}, vp *gi.Viewport2D) []string); ok {
				ac.MakeMenuFunc = MethViewSubMenuFunc
				md.SubMenuFunc = SubMenuFunc(sf)
				bitflag.Set32((*int32)(&md.Flags), int(MethViewHasSubMenu))
			} else {
				MethViewErr(vtyp, fmt.Sprintf("ActionView for Method: %v, submenu-func must be of type SubMenuFunc", methNm))
			}
		case "Args":
			argv, ok := pv.(ki.PropSlice)
			if !ok {
				MethViewErr(vtyp, fmt.Sprintf("ActionView for Method: %v, Args property must be of type ki.PropSlice, containing names and other properties for each arg", methNm))
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
	if !bitflag.Has32((int32)(md.Flags), int(MethViewHasSubMenu)) {
		ac.ActionSig.Connect(vp.This(), MethViewCall)
	}
	return true
}

// ActionViewArgsValidate validates the Args properties relative to number of args on type
func ActionViewArgsValidate(md *MethViewData, vtyp reflect.Type, meth reflect.Method, argprops ki.PropSlice) bool {
	mtyp := meth.Type
	narg := mtyp.NumIn()
	apsz := len(argprops)
	if narg-1 != apsz {
		MethViewErr(vtyp, fmt.Sprintf("Method: %v takes %v args (beyond the receiver), but Args properties only has %v", meth.Name, narg-1, apsz))
		return false
	}
	if bitflag.Has32((int32)(md.Flags), int(MethViewHasSubMenu)) && apsz != 1 {
		MethViewErr(vtyp, fmt.Sprintf("Method: %v has a submenu of values to use as the one arg for it, but it takes %v args (beyond the receiver) -- should only take 1", meth.Name, narg-1))
		return false
	}

	return true
}

//////////////////////////////////////////////////////////////////////////////////
//    Method Callbacks -- called when Action fires

// MethViewFlags define bitflags for method view action options
type MethViewFlags int32

const (
	// MethViewConfirm confirms action before proceeding
	MethViewConfirm MethViewFlags = iota

	// MethViewShowReturn shows the return value from the method
	MethViewShowReturn

	// MethViewNoUpdateAfter means do not update window after method runs (default is to do so)
	MethViewNoUpdateAfter

	// MethViewHasSubMenu means that this action has a submenu option --
	// argument values will be selected from the auto-generated submenu
	MethViewHasSubMenu

	// MethViewHasSubMenuVal means that this action was called using a submenu
	// and the SubMenuVal has the selected value
	MethViewHasSubMenuVal

	// MethViewKeyFun means this action's only function is to emit the key fun
	MethViewKeyFun

	MethViewFlagsN
)

//go:generate stringer -type=MethViewFlags

var KiT_MethViewFlags = kit.Enums.AddEnumAltLower(MethViewFlagsN, kit.BitFlag, nil, "MethView")

// SubMenuFunc is a function that returns a string slice of submenu items
// used in MethView submenu-func option
// first argument is the object on which the method is defined (receiver)
type SubMenuFunc func(it interface{}, vp *gi.Viewport2D) []string

// ShortcutFunc is a function that returns a key.Chord string for a shortcut
// used in MethView shortcut-func option
// first argument is the object on which the method is defined (receiver)
type ShortcutFunc func(it interface{}, act *gi.Action) key.Chord

// LabelFunc is a function that returns a string to set a label
// first argument is the object on which the method is defined (receiver)
type LabelFunc func(it interface{}, act *gi.Action) string

// ActionUpdateFunc is a function that updates method active / inactive status
// first argument is the object on which the method is defined (receiver)
type ActionUpdateFunc func(it interface{}, act *gi.Action)

// MethViewData is set to the Action.Data field for all MethView actions,
// containing info needed to actually call the Method on value Val.
type MethViewData struct {
	Val          interface{}
	ValVal       reflect.Value
	Vp           *gi.Viewport2D
	Method       string
	MethVal      reflect.Value
	MethTyp      reflect.Method
	ArgProps     ki.PropSlice     `desc:"names and other properties of args, in one-to-one with method args"`
	SpecProps    ki.Props         `desc:"props for special action types, e.g., FileView"`
	Desc         string           `desc:"prompt shown in arg dialog or confirm prompt dialog"`
	UpdateFunc   ActionUpdateFunc `desc:"update function defined in properties -- called by our wrapper update function"`
	SubMenuSlice interface{}      `desc:"value for submenu generation as a literal slice of items of appropriate type for method being called"`
	SubMenuField string           `desc:"value for submenu generation as name of field on obj"`
	SubMenuFunc  SubMenuFunc      `desc:"function that will generate submenu items, as []string slice"`
	SubMenuVal   interface{}      `desc:"value that the user selected from submenu for this action -- this should be assigned to the first (only) arg of the method"`
	KeyFun       gi.KeyFuns       `desc:"key function that we emit, if MethViewKeyFun type"`
	Flags        MethViewFlags
}

func (md *MethViewData) MethName() string {
	typnm := kit.ShortTypeName(md.ValVal.Type())
	methnm := typnm + ":" + md.Method
	return methnm
}

// MethViewCall is the receiver func for MethView actions that call a method
// -- it uses the MethViewData to call the target method.
func MethViewCall(recv, send ki.Ki, sig int64, data interface{}) {
	ac := send.(*gi.Action)
	md := ac.Data.(*MethViewData)
	if md.ArgProps == nil { // no args -- just call
		MethViewCallNoArgPrompt(ac, md, nil)
		return
	}
	// need to prompt for args
	ads, args, nprompt, ok := MethViewArgData(md)
	if !ok {
		return
	}
	if nprompt == 0 {
		MethViewCallNoArgPrompt(ac, md, args)
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
		if ad.View.HasAction() {
			ad.View.Activate(md.Vp, ad.View, func(recv, send ki.Ki, sig int64, data interface{}) {
				if sig == int64(gi.DialogAccepted) {
					MethViewCallMeth(md, args)
				}
			})
			return
		}
	}

	ArgViewDialog(md.Vp, ads, DlgOpts{Title: ac.Text, Prompt: md.Desc},
		md.Vp.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
			if sig == int64(gi.DialogAccepted) {
				// ddlg := send.Embed(gi.KiT_Dialog).(*gi.Dialog)
				MethViewCallMeth(md, args)
			}
		})
}

// MethViewCallNoArgPrompt calls the method in case where there is no
// prompting otherwise of the user for arg values -- checks for Confirm case
// or otherwise directly calls method
func MethViewCallNoArgPrompt(ac *gi.Action, md *MethViewData, args []reflect.Value) {
	if bitflag.Has32(int32(md.Flags), int(MethViewKeyFun)) {
		if md.Vp != nil && md.Vp.Win != nil {
			md.Vp.Win.EventMgr.SendKeyFunEvent(md.KeyFun, false)
		}
		return
	}
	if bitflag.Has32(int32(md.Flags), int(MethViewConfirm)) {
		gi.PromptDialog(md.Vp, gi.DlgOpts{Title: ac.Text, Prompt: md.Desc}, gi.AddOk, gi.AddCancel,
			md.Vp.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
				if sig == int64(gi.DialogAccepted) {
					MethViewCallMeth(md, args)
				}
			})
	} else {
		MethViewCallMeth(md, args)
	}
}

// MethViewCallMeth calls the method with given args, and processes the
// results as specified in the MethViewData.
func MethViewCallMeth(md *MethViewData, args []reflect.Value) {
	rv := md.MethVal.Call(args)
	methnm := md.MethName()
	mtyp := md.MethTyp.Type
	narg := mtyp.NumIn() - 1
	for ai := 0; ai < narg; ai++ {
		ap := md.ArgProps[ai]
		argnm := methnm + "." + ap.Name
		MethArgHist[argnm] = args[ai].Interface()
	}

	if !bitflag.Has32(int32(md.Flags), int(MethViewNoUpdateAfter)) {
		md.Vp.SetNeedsFullRender() // always update after all methods -- almost always want that
	}
	if bitflag.Has32(int32(md.Flags), int(MethViewShowReturn)) {
		if len(rv) >= 1 {
			MethViewShowValue(md.Vp, rv[0], md.Method+" Result", "")
		}
	}
}

// MethViewShowValue displays a value in a dialog window (e.g., for MethViewShowReturn)
func MethViewShowValue(vp *gi.Viewport2D, val reflect.Value, title, prompt string) {
	if kit.ValueIsZero(val) {
		return
	}
	npv := kit.NonPtrValue(val)
	if kit.ValueIsZero(npv) {
		return
	}
	tk := npv.Type().Kind()
	switch tk {
	case reflect.Struct:
		StructViewDialog(vp, val.Interface(), DlgOpts{Title: title, Prompt: prompt, Ok: true, Cancel: true}, nil, nil)
	case reflect.Slice:
		if bs, ok := npv.Interface().([]byte); ok {
			TextViewDialog(vp, bs, DlgOpts{Title: title, Prompt: prompt, Ok: true})
		} else if bs, ok := val.Interface().([]byte); ok {
			TextViewDialog(vp, bs, DlgOpts{Title: title, Prompt: prompt, Ok: true})
		} else {
			SliceViewDialog(vp, val.Interface(), DlgOpts{Title: title, Prompt: prompt, Ok: true, Cancel: true}, nil, nil, nil)
		}
	case reflect.Map:
		MapViewDialog(vp, val.Interface(), DlgOpts{Title: title, Prompt: prompt, Ok: true, Cancel: true}, nil, nil)
	default:
		TextViewDialog(vp, []byte(npv.String()), DlgOpts{Title: title, Prompt: prompt, Ok: true})
	}

}

// ArgData contains the relevant data for each arg, including the
// reflect.Value, name, optional description, and default value
type ArgData struct {
	Val     reflect.Value
	Name    string
	Desc    string
	View    ValueView
	Default interface{}
	Flags   ArgDataFlags
}

// ArgDataFlags define bitflags for method view action options
type ArgDataFlags int32

const (
	// ArgDataHasDef means that there was a Default value set
	ArgDataHasDef ArgDataFlags = iota

	// ArgDataValSet means that there is a fixed value for this arg, given in
	// the config props and set in the Default, so it does not need to be
	// prompted for
	ArgDataValSet

	ArgDataFlagsN
)

//go:generate stringer -type=ArgDataFlags

var KiT_ArgDataFlags = kit.Enums.AddEnumAltLower(ArgDataFlagsN, kit.BitFlag, nil, "ArgData")

func (ad *ArgData) HasDef() bool {
	return bitflag.Has32(int32(ad.Flags), int(ArgDataHasDef))
}

func (ad *ArgData) SetHasDef() {
	bitflag.Set32((*int32)(&ad.Flags), int(ArgDataHasDef))
}

func (ad *ArgData) HasValSet() bool {
	return bitflag.Has32(int32(ad.Flags), int(ArgDataValSet))
}

// MethArgHist stores the history of method arg values -- used for setting defaults
// for next time the method is called.  Key is type:method name
var MethArgHist = map[string]interface{}{}

// MethViewArgData gets the arg data for the method args, returns false if
// errors -- nprompt is the number of args that require prompting from the
// user (minus any cases with value: set directly)
func MethViewArgData(md *MethViewData) (ads []ArgData, args []reflect.Value, nprompt int, ok bool) {
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

		ad.View = ToValueView(ad.Val.Interface(), "")
		ad.View.SetSoloValue(ad.Val)
		ad.View.SetName(ad.Name)
		nprompt++ // assume prompt

		switch apv := aps.Value.(type) {
		case ki.BlankProp:
		case ki.Props:
			for pk, pv := range apv {
				switch pk {
				case "desc":
					ad.Desc = kit.ToString(pv)
					ad.View.SetTag("desc", ad.Desc)
				case "default":
					ad.Default = pv
					ad.SetHasDef()
				case "value":
					ad.Default = pv
					ad.SetHasDef()
					bitflag.Set32((*int32)(&ad.Flags), int(ArgDataValSet))
					nprompt--
				case "default-field":
					field := pv.(string)
					if flv, ok := MethViewFieldValue(md.ValVal, field); ok {
						ad.Default = flv.Interface()
						ad.SetHasDef()
					}
				default:
					ad.View.SetTag(pk, kit.ToString(pv))
				}
			}
		}

		if bitflag.Has32((int32)(md.Flags), int(MethViewHasSubMenuVal)) {
			ad.Default = md.SubMenuVal
			ad.SetHasDef()
			bitflag.Set32((*int32)(&ad.Flags), int(ArgDataValSet))
			nprompt--
		}

		if ad.HasDef() {
			ad.View.SetValue(ad.Default)
		}
	}
	return
}

// MethViewArgDefaultVal returns the default value of the given argument index
func MethViewArgDefaultVal(md *MethViewData, ai int) (interface{}, bool) {
	aps := &md.ArgProps[ai]
	var def interface{}
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
				if flv, ok := MethViewFieldValue(md.ValVal, field); ok {
					def = flv.Interface()
					got = true
				}
			}
		}
	}
	return def, got
}

// MethViewFieldValue returns a reflect.Value for the given field name,
// checking safely (false if not found)
func MethViewFieldValue(vval reflect.Value, field string) (*reflect.Value, bool) {
	fv, ok := kit.FieldValueByPath(kit.NonPtrValue(vval).Interface(), field)
	if !ok {
		log.Printf("giv.MethViewFieldValue: Could not find field %v in type: %v\n", field, vval.Type().String())
		return nil, false
	}
	return &fv, true
}

// MethViewUpdateFunc is general Action.UpdateFunc that then calls any
// MethViewData.UpdateFunc from its data
func MethViewUpdateFunc(act *gi.Action) {
	md := act.Data.(*MethViewData)
	if md.UpdateFunc != nil && md.Val != nil {
		md.UpdateFunc(md.Val, act)
	}
}

// MethViewSubMenuFunc is a MakeMenuFunc for items that have submenus
func MethViewSubMenuFunc(aki ki.Ki, m *gi.Menu) {
	ac := aki.(*gi.Action)
	md := ac.Data.(*MethViewData)
	smd := md.SubMenuSlice
	if md.SubMenuFunc != nil {
		smd = md.SubMenuFunc(md.Val, md.Vp)
	} else if md.SubMenuField != "" {
		if flv, ok := MethViewFieldValue(md.ValVal, md.SubMenuField); ok {
			smd = flv.Interface()
		}
	}
	if smd == nil {
		return
	}
	sltp := kit.NonPtrType(reflect.TypeOf(smd))
	if sltp.Kind() != reflect.Slice && sltp.Kind() != reflect.Array {
		log.Printf("giv.MethViewSubMenuFunc: submenu data must be a slice or array, not: %v\n", sltp.String())
		return
	}

	def, gotDef := MethViewArgDefaultVal(md, 0) // assume first
	defstr := ""
	if gotDef {
		defstr = kit.ToString(def)
	}

	mv := reflect.ValueOf(smd)
	mvnp := kit.NonPtrValue(mv)
	sz := mvnp.Len()
	*m = make(gi.Menu, sz)
	for i := 0; i < sz; i++ {
		val := mvnp.Index(i)
		vi := val.Interface()
		nm := kit.ToString(val)
		if nm == gi.MenuTextSeparator {
			sp := &gi.Separator{}
			sp.InitName(sp, "sep")
			sp.Horiz = true
			(*m)[i] = sp
			continue
		}
		nac := &gi.Action{}
		nac.InitName(nac, nm)
		nac.Text = nm
		nac.SetAsMenu()
		nac.ActionSig.Connect(md.Vp.This(), MethViewCall)
		nd := *md // copy
		nd.SubMenuVal = vi
		if gotDef {
			if kit.ToString(vi) == defstr {
				nac.SetSelected()
			}
		}
		bitflag.Set32((*int32)(&nd.Flags), int(MethViewHasSubMenuVal))
		nac.Data = &nd
		(*m)[i] = nac
	}
	md.Vp.Win.MainMenuUpdated()
}
