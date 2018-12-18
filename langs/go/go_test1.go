/* Copyright (c) 2018, The gide / GoKi Authors. All rights reserved. */
/* Use of this source code is governed by a BSD-style
license that can be found in the LICENSE file. */

/* This has /* embedded */ comments which is /* a bit  */ tricky */ 

package main


func adkf() {
	for range sr.Text {
		if unicode.IsSpace(sr.Text[0]) {
		}
	}
}

var StyleValueTypes = map[reflect.Type]struct{}{
	units.KiT_Value: {Key: "value"},
	KiT_Color:       {},
	KiT_ColorSpec:   {},
	KiT_Matrix2D:    {},
}

func extfun() {
	rs.Raster.SetStroke(
		Float32ToFixed(pc.StrokeWidth(rs)),
		Float32ToFixed(pc.StrokeStyle.MiterLimit),
		pc.capfunc(), nil, nil, pc.joinmode(), // todo: supports leading / trailing caps, and "gaps"
		dash, 0	)
	rs.Scanner.SetClip(rs.Bounds)
}

func (w *Window) SendKeyChordEvent(popup bool, r rune, mods ...key.Modifiers) {
	ke := key.ChordEvent{}
	ke.SetTime()
	ke.SetModifiers(mods...)
	ke.Rune = r
	ke.Action = key.Press
	w.SendEventSignal(&ke, popup)
}

func (fl *FontLib) InitFontPaths(paths ...string) {
	if len(fl.FontPaths) > 0 {
		return
	}
	fl.AddFontPaths(paths...)
}

func (ft FileTime) String(reg string, pars int) string {
	return (time.Time)(ft).Format("Mon Jan  2 15:04:05 MST 2006")
}

var _ArgDataFlags_index = [...]uint8{0, 13, 26, 39}

var FileInfoProps = ki.Props{
	"CtxtMenu": ki.PropSlice{
		{"Duplicate", ki.Props{
			"updtfunc": ActionUpdateFunc(func(fii interface{}, act *gi.Action) {
				fi := fii.(*FileInfo)
				act.SetInactiveState(fi.IsDir())
			}),
		}},
		{"Delete", ki.Props{
			"desc":    "Ok to delete this file?  This is not undoable and is not moving to trash / recycle bin",
			"confirm": true,
			"updtfunc": ActionUpdateFunc(func(fii interface{}, act *gi.Action) {
				fi := fii.(*FileInfo)
				act.SetInactiveState(fi.IsDir())
			}),
		}},
		{"Rename", ki.Props{
			"desc": "Rename file to new file name",
			"Args": ki.PropSlice{
				{"New Name", ki.Props{
					"default-field": "Name",
				}},
			},
		}},
	},
}

func baf() {
//	switch apv := aps.Value.(type) {
//	case ki.BlankProp:
//	}
}

func aaa() {
	sf, ok := pv.(func(it interface{}, act *gi.Action) key.Chord)	
}

func ccc() {
	if sf, ok := pv.(ShortcutFunc); ok {
		ac.Shortcut = sf(md.Val, ac)
	} else if sf, ok := pv.(func(it interface{}, act *gi.Action) key.Chord); ok {
		ac.Shortcut = sf(md.Val, ac)
	} else {
		MethViewErr(vtyp, fmt.Sprintf("ActionView for Method: %v, shortcut-func must be of type ShortcutFunc", methNm))
	}
}


func bbb() {
	a := struct{}{}
}

func (tv *TableView) RowGrabFocus(row int) *gi.WidgetBase {
	
	tv.inFocusGrab = slice{}

	defer func() { tv.inFocusGrab = false 	}
	
	defer func() { tv.inFocusGrab = false 	}()
	
	return nil
}

func sfa() {
	for i := 0; i < 100; i++ {
		fmt.Printf("%v %v", a, i)
		a++
		p := a * i
	}
	tv.inFocusGrab = true
	defer func() { tv.inFocusGrab = false }()
	tv.inFocusGrab = true
}

func tst() {
	if kit.Enums.TypeRegistered(nptyp) { // todo: bitfield
		vv := EnumValueView{}
		vv.Init(&vv)
		return &vv
	} else if _, ok := it.(fmt.Stringer); ok { // use stringer
		vv := ValueViewBase{}
		vv.Init(&vv)
		return &vv
	} else {
		vv := IntValueView{}
		vv.Init(&vv)
		return &vv
	}
}


func dkf() {
	goto pil
pil:
	return nil
}

func (tv *TreeView) FocusChanged2D(change gi.FocusChanges) {
	switch change {
	case gi.FocusInactive: // don't care..
	case gi.FocusActive:
	}
}

func adlf() {
	switch pr := bprpi.(type) {
	case map[string]interface{}:
		wb.SetIconProps(ki.Props(pr))
	case ki.Props:
		wb.SetIconProps(pr)
	}	
}

var _ArgDataFlags_index = [...]uint8{0, 13, 26, 39}

func sld() {
	<-TextViewBlinker.C
}

func main() {
	if sz > max {
		*ch = (*ch)[:max]
	}
}

func (tv *TextView) FindNextLink(pos TextPos) (TextPos, TextRegion, bool) {
	
}

func tst() {
	nwSz := gi.Vec2D{mxwd, off + extraHalf}.ToPointCeil()
}

func tst() {
	a := tv.Renders[ln].Links 

	if !tv.HasLinks && tv.Renders[ln].Links > 0 {
		tv.HasLinks = true
	}
}

func tst() {
	tvn, two := data.(ki.Ki).Embed(giv.KiT_TreeView).(*giv.TreeView)
	for a, b := range cde {
	}
}

var PiViewProps = ki.Props{
	"MainMenu": ki.PropSlice{
		"updtfunc": giv.ActionUpdateFunc(func(pvi interface{}, act *gi.Action) {
			pv := pvi.(*PiView)
			act.SetActiveState(pv.Prefs.ProjFile != "")
		}),
		"offguy": true,
	},
}

import "github.com/goki/gi/gi"

import (
	gi "github.com/goki/gi/gi"
	"github.com/goki/gi/gimain"
	"github.com/goki/gi/oswin"
	gogide "github.com/goki/gide/gide"
	"github.com/goki/pi"
	"github.com/goki/pi/piv"
)

/*
var av1, av2 int

type Pvsi struct {
	Af int
	Bf string
}

func (ps *Pvsi) tst() {
	txt += rs[sd-1].String()
	txt += rs[i].String()
	fmt.Println(ps.Errs[len(ps.Errs)-1].Error())
}	
*/

/*
func tst() {
	win.OSWin.SetCloseReqFunc(func(w oswin.Window) {
		if !inClosePrompt {
			inClosePrompt = true
			if pv.Changed {
				gi.ChoiceDialog(vp, gi.DlgOpts{Title: "Close Without Saving?",
					Prompt: "Do you want to save your changes?  If so, Cancel and then Save"},
					[]string{"Close Without Saving", "Cancel"},
					win.This(), func(recv, send ki.Ki, sig int64, data interface{}) {
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
		}
	})
}
*/

/*
func tst(txt string, amt int) (bool, *Rule) {
	txt += rs[sd-1].String()
	txt += rs[i].String()
	fmt.Println(ps.Errs[len(ps.Errs)-1].Error())
}	

func tst() {
	r := &(*rs)[i]
}	

func tst() {
	rs := &ps.Matches[abc]
}

func tst() {
	rs := Matches[scope][scope]
}	

func tst() {
	if !inClosePrompt {
		if pv.Changed {
			ChoiceDialog(func(ab int) {
				break
				return
				})
		} else {
			w.Close()
		}
	}
}


func tst() {
   SetCloseReqFunc(func(w win) {
 		if !inClosePrompt {
			if pv.Changed {
				ChoiceDialog(func(ab int) {
					break
					return
					})
			} else {
				w.Close()
			}
		}
	})
}


func (pv *PiView) ConfigSplitView() {
	Connect(func(sig int64) {
		switch sig {
		case int64(TreeViewSelected):
			break
		}
	})
}

var MakeSlice = make([]Rule, 100) // make and new require special rules b/c take type args

var MakeSlice = make([][][]*Rule, 100)

func (pv *PiView) OpenTestTextTab() {
	if ctv.Buf != &pv.TestBuf {
		ctv.SetBuf(&pv.TestBuf)
	}
}

func (ev Steps) MarshalJSON() ([]byte, error)  {
	return kit.EnumMarshalJSON(ev)
}
*/
/*
func (ev *Steps) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b)
}

// todo: not dealing with all-in-one-line case -- needs to insert EOS before } 
func (ev *Steps) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

func tst() {
	tokSrc := string(ps.TokenSrc(pos))
}

// var ifa interface{} // todo: not working

func tst() {
	pv.SaveParser()
	pv.GetPrefs()
	Trace.Out(ps, pr, Run, creg.St, creg, trcAst, fmt.Sprintf("%v: optional rule: %v failed", ri, rr.Rule.Name()))
}

// this is Go's "most vexing parse" from a top-down perspective:

//var MultSlice = p[2]*Rule // todo: not working
*/
/*
var SliceAr1 = []Rule{}

var SliceAry = [25]*Rule{}

//var SliceAry = []*Rule{} // todo: ? not excluding here

var RuleMap map[string]*Rule // looks like binary *

var TextViewSelectors = []string{":active", ":focus", ":inactive", ":selected", ":highlight"}

// exclude rule -- two rules fwd and back:
// ?'key:map' '[' ? ']' '*' 'Name' ?'.' ?'Name
//  + start at ', go forward to match name, pkg.name -- exclude if no match
//  + go back.. 
// range is 
// start at *, 
// backtrack: if a given parse fails... nah, way too complicated..

var unaryptr = 25 * *(ptr+2)  // directly to rhs or depth sub of it
var multexpr = 25 * (ptr + 2)
var multex = 25 * ptr + 25 * *ptr // 
var a,b,c,d = 32

func (pr *Rule) BaseIface() reflect.Type {
	return reflect.TypeOf((*Parser)(nil)).Elem()
}


func (pr *Rule) AsParseRule() *Rule {
	return pr.This().Embed(KiT_Rule).(*Rule)
}


func test() {
	RuleMap = map[string]*Rule{}
}

// interface{} here not working
func (pr *Rule) CompileAll(ps *State) bool {
	pr.SetRuleMap(ps)
	allok := true
	pr.FuncDownMeFirst(0, pr.This(), func(k ki.Ki, level int, d interface{}) bool {
		pri := k.Embed(KiT_Rule).(*Rule)
		ok := pri.Compile(ps)
		if !ok {
			allok = false
		}
		return true
	})
	return allok
}

func test() {
	if pr.Rule[0] == '-' {
		rstr = rstr[1:]
		pr.Reverse = true
	} else {
		pr.Reverse = false
	}
}
*/

type Rule struct {
	OnePar
	ki.Node
	Off       bool     `desc:"disable this rule -- useful for testing"`
}

/*
func tst() {
	oswin.TheApp.SetQuitCleanFunc(func() {
		fmt.Printf("Doing final Quit cleanup here..\n")
	})
}
*/

func (pr *Parser) LexErrString() string {
	return pr.LexState.Errs.AllString()
}

func tst() {
	a = !pr.LexState.AtEol()

	pr.LexState.Filename = !pr.LexState.AtEol()

	if !pr.Sub.LexState.AtEol() && cpos == pr.LexState.Pos {
		msg := fmt.Sprintf("did not advance position -- need more rules to match current input: %v", string(pr.LexState.Src[cpos:]))
		pr.LexState.Error(cpos, msg)
		return nil
	}
}

var ext = strings.ToLower(filepath.Ext(flag.Arg(0)))

func tst() {
		if path == "" && proj == "" {
			if flag.NArg() > 0 {
	 			ext := strings.ToLower(filepath.Ext(flag.Arg(0)))
				if ext == ".gide" {
	 				proj = flag.Arg(0)
				} else {
	 				path = flag.Arg(0)
	 			}
	 		}
		}
	recv := gi.Node2DBase{}
}

func (pr *Parser) Init() {
	pr.Lexer.InitName(&pr.Lexer, "Lexer")
}

func (pr *Parser) Init2(a int, fname string, amap map[string]string) bool {
	pr.Parser.InitName(&pr.Parser, "Parser")
}

func (pr *Parser) Init3(a, b int, fname string) (bool, string) {
	pr.Ast.InitName(&pr.Ast, "Ast")
}

func (pr *Parser) Init4(a, b int, fname string) (ok bool, name string) {
	pr.LexState.Init()
}

// SetSrc sets source to be parsed, and filename it came from
func (pr *Parser) SetSrc(src [][]rune, fname string) {
}

func main() {

	if this > that {
		break
	} else {
		continue
	}

	if peas++; this > that {
		fmt.Printf("test")
		break
	}

	if this > that {
		break
	} else if something == other {
		continue
	}

	if a := b; b == a {
		fmt.Printf("test")
		break
	} else {
		continue
	}

	if a > b {
		b++
	}

	switch vvv := av; nm {
	case "baby":
		nm = "maybe"
		a++
	case "not":
		i++
		p := a * i
	default:
		non := "anon"
	}

	for i := 0; i < 100; i++ {
		fmt.Printf("%v %v", a, i)
		a++
		p := a * i
	}

	for a, i, j, k := range names {
		for i < 100 {
			for {
				fmt.Printf("%v %v", a, i)
			}
		}
	}

	fmt.Printf("starting test")
	defer my.Widget.UpdateEnd(updt)
	goto bypass

	if a == b {
		fmt.Printf("equal")
	} else if a > b {
		so++
		be--
		it := 20
	} else {
		fmt.Printf("long one")
	}

bypass:
	fmt.Printf("here")
	return
	return something
	{
		nvar := 22
		nvar += function(of + some + others)
	}
}

const neg = -1

const neg2 = -(2+2)

const Prec1 = ((2-1) * 3)

const (
	parn = 1 + (2 + 3)
	PrecedenceS2 = 25 / (3.14 + -(2 + 4)) > ((2 - 5) * 3)
)

const Precedence2 = -(3.14 + 2 * 4)

// The lexical acts
const (
	// Next means advance input position to the next character(s) after the matched characters
	Next Actions = 4.92

	// Name means read in an entire name, which is letters, _ and digits after first letter
	// position will be advanced to just after
	Name
)

type MyFloat float64

type AStruct struct {
	AField int
	TField gi.Widget `desc:"tagged"`
	AField []string
	MField map[string]int
}

var Typeo int

var ExprVar = "testofit"

var ExprTypeVar map[string]string 

var ExprInitMap = map[string]string{
	"does": {Val: "this work?", Bad: "dkfa"},
}

var ExprSlice = abc[20]

var ExprSlice2 = abc[20:30]

var ExprSlice3 = abc[20:30] + abc[:] + abc[20:] + abc[:30] + abc[20:30:2]

var ExprSelect = abc.Def

var ExprCvt = int(abc) // works with basic type names -- others are FunCall

var TypPtr *Fred

var ExprCvt2 = map[string]string(ab)

var tree = map[token.Tokens]struct{}(optMap)

var tree = (map[token.Tokens]struct{})(optMap)

var partyp = (*int)(tree)

var ExprTypeAssert = absfr.(gi.TreeView)

var ExprTypeAssertPtr = absfr.(*gi.TreeView)

var methexpr = abc.meth(a-b * 2 + bf.Meth(22 + 55) / long.meth.Call(tree))

var ExprMeth = abc.meth(c)

var ExprMethLong = long.abc.meth(c)

var ExprFunNil = fun()

var ExprFun = meth(2 + 2)

var ExprFun = meth(2 + 2, fslaf)

var ExprFunElip = meth(2 + 2, fslaf...)


func main() {
	a <- b
	c++
	c[3] = 42 * 17
 	bf := a * b + c[32]
	d += funcall(a, b, c...)
	fmt.Printf("this is ok", gi.CallMe(a.(tree) + b.(*tree) + int(22) * string(17)))
}

func mainrun() {
	oswin.TheApp.SetName("pie")
	oswin.TheApp.SetAbout(`<code>Pie</code> is the interactive parser (pi) editor written in the <b>GoGi</b> graphical interface system, within the <b>GoKi</b> tree framework.  See <a href="https://github.com/goki/pi">Gide on GitHub</a> and <a href="https://github.com/goki/pi/wiki">Gide wiki</a> for documentation.<br>
<br>
Version: ` + pi.VersionInfo())
	if peas++; this > that {
		fmt.Printf("test")
		break
	}
	if this > that {
		fmt.Printf("test")
		break
	} else {
		continue
	}

	if this > that {
		fmt.Printf("test")
		break
	} else if something == other {
		continue
	}

	oswin.TheApp.SetQuitCleanFunc(func() {
		fmt.Printf("Doing final Quit cleanup here..\n")
	})

	pi.InitPrefs()

	var path string
	var proj string
	// process command args
	if len(os.Args) > 1 {
		flag.StringVar(&path, "path", "", "path to open -- can be to a directory or a filename within the directory")
		flag.StringVar(&proj, "proj", "", "project file to open -- typically has .gide extension")
		// todo: other args?
		flag.Parse()
		if path == "" && proj == "" {
			if flag.NArg() > 0 {
	 			ext = strings.ToLower(filepath.Ext(flag.Arg(0)))
				if ext == ".gide" {
	 				proj = flag.Arg(0)
				} else {
	 				path = flag.Arg(0)
	 			}
	 		}
		}
	}

	recv := gi.Node2DBase{}
	recv.InitName(&recv, "pie_dummy")

	inQuitPrompt := false
	oswin.TheApp.SetQuitReqFunc(func() {
		if !inQuitPrompt {
			inQuitPrompt = true
			if gide.QuitReq() {
				oswin.TheApp.Quit()
			} else {
				inQuitPrompt = false
			}
		}
	})

	if proj != "" {
		proj, _ = filepath.Abs(proj)
	 	gide.OpenGideProj(proj)
	} else {
		if path != "" {
			path, _ = filepath.Abs(path)
		}
		gide.NewGideProjPath(path)
	}

	piv.NewPiView()

	// above NewGideProj calls will have added to WinWait..
	gi.WinWait.Wait()
}

var someother int


type Lang interface {
	// Parser returns the pi.Parser for this language
	Parser() *Parser

	// ParseFile does the complete processing of a given single file, as appropriate
	// for the language -- e.g., runs the lexer followed by the parser, and
	// manages any symbol output from parsing as appropriate for the language / format.
	ParseFile(fs *FileState)
	
	// LexLine does the lexing of a given line of the file, using existing context
	// if available from prior lexing / parsing. Line is in 0-indexed "internal" line indexes.
	// The rune source information is assumed to have already been updated in FileState.
	// languages can run the parser on the line to augment the lex token output as appropriate.
	LexLine(fs *FileState, line int) lex.Line
}

var TextViewProps = ki.Props{
	"white-space":      gi.WhiteSpacePreWrap,
	"font-family":      "Go Mono",
	"border-width":     0, // don't render our own border
	"cursor-width":     units.NewValue(3, units.Px),
	"border-color":     &gi.Prefs.Colors.Border,
	"border-style":     gi.BorderSolid,
	"padding":          units.NewValue(2, units.Px),
	"margin":           units.NewValue(2, units.Px),
	"vertical-align":   gi.AlignTop,
	"text-align":       gi.AlignLeft,
	"tab-size":         4,
	"color":            &gi.Prefs.Colors.Font,
	"background-color": &gi.Prefs.Colors.Background,
	TextViewSelectors[TextViewActive]: ki.Props{
		"background-color": "highlight-10",
	},
	TextViewSelectors[TextViewFocus]: ki.Props{
		"background-color": "lighter-0",
	},
	TextViewSelectors[TextViewInactive]: ki.Props{
		"background-color": "highlight-20",
	},
	TextViewSelectors[TextViewSel]: ki.Props{
		"background-color": &gi.Prefs.Colors.Select,
	},
	TextViewSelectors[TextViewHighlight]: ki.Props{
		"background-color": &gi.Prefs.Colors.Highlight,
	},
}


