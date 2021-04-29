// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"unicode"

	"github.com/goki/gi/gi"
	"github.com/goki/gi/girl"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
	"github.com/goki/mat32"
)

// Path renders SVG data sequences that can render just about anything
type Path struct {
	NodeBase
	Data    []PathData `xml:"-" desc:"the path data to render -- path commands and numbers are serialized, with each command specifying the number of floating-point coord data points that follow"`
	DataStr string     `xml:"d" desc:"string version of the path data"`
}

var KiT_Path = kit.Types.AddType(&Path{}, ki.Props{"EnumType:Flag": gi.KiT_NodeFlags})

// AddNewPath adds a new button to given parent node, with given name and path data.
func AddNewPath(parent ki.Ki, name string, data string) *Path {
	g := parent.AddNewChild(KiT_Path, name).(*Path)
	if data != "" {
		g.SetData(data)
	}
	return g
}

func (g *Path) SVGName() string { return "path" }

func (g *Path) CopyFieldsFrom(frm interface{}) {
	fr := frm.(*Path)
	g.NodeBase.CopyFieldsFrom(&fr.NodeBase)
	g.Data = make([]PathData, len(fr.Data))
	copy(g.Data, fr.Data)
	g.DataStr = fr.DataStr
}

func (g *Path) SetPos(pos mat32.Vec2) {
	// todo: set first point
}

func (g *Path) SetSize(sz mat32.Vec2) {
	// todo: scale bbox
}

// SetData sets the path data to given string, parsing it into an optimized
// form used for rendering
func (g *Path) SetData(data string) error {
	g.DataStr = data
	var err error
	g.Data, err = PathDataParse(data)
	if err != nil {
		return err
	}
	err = PathDataValidate(&g.Pnt, &g.Data, g.Path())
	return err
}

func (g *Path) SVGLocalBBox() mat32.Box2 {
	bb := PathDataBBox(g.Data)
	hlw := 0.5 * g.LocalLineWidth()
	bb.Min.SetSubScalar(hlw)
	bb.Max.SetAddScalar(hlw)
	return bb
}

func (g *Path) Render2D() {
	sz := len(g.Data)
	if sz < 2 {
		return
	}
	vis, rs := g.PushXForm()
	if !vis {
		return
	}
	pc := &g.Pnt
	PathDataRender(g.Data, pc, rs)
	pc.FillStrokeClear(rs)

	g.ComputeBBoxSVG()

	if mrk := MarkerByName(g, "marker-start"); mrk != nil {
		// todo: could look for close-path at end and find angle from there..
		stv, ang := PathDataStart(g.Data)
		mrk.RenderMarker(stv, ang, g.Pnt.StrokeStyle.Width.Dots)
	}
	if mrk := MarkerByName(g, "marker-end"); mrk != nil {
		env, ang := PathDataEnd(g.Data)
		mrk.RenderMarker(env, ang, g.Pnt.StrokeStyle.Width.Dots)
	}
	if mrk := MarkerByName(g, "marker-mid"); mrk != nil {
		var ptm2, ptm1, pt mat32.Vec2
		gotidx := 0
		PathDataIterFunc(g.Data, func(idx int, cmd PathCmds, ptIdx int, cp mat32.Vec2, ctrls []mat32.Vec2) bool {
			ptm2 = ptm1
			ptm1 = pt
			pt = cp
			if gotidx < 2 {
				gotidx++
				return true
			}
			if idx >= sz-3 { // todo: this is approximate...
				return false
			}
			ang := 0.5 * (mat32.Atan2(pt.Y-ptm1.Y, pt.X-ptm1.X) + mat32.Atan2(ptm1.Y-ptm2.Y, ptm1.X-ptm2.X))
			mrk.RenderMarker(ptm1, ang, g.Pnt.StrokeStyle.Width.Dots)
			gotidx++
			return true
		})
	}

	g.Render2DChildren()
	rs.PopXFormLock()
}

// PathCmds are the commands within the path SVG drawing data type
type PathCmds byte

const (
	// move pen, abs coords
	PcM PathCmds = iota
	// move pen, rel coords
	Pcm
	// lineto, abs
	PcL
	// lineto, rel
	Pcl
	// horizontal lineto, abs
	PcH
	// relative lineto, rel
	Pch
	// vertical lineto, abs
	PcV
	// vertical lineto, rel
	Pcv
	// Bezier curveto, abs
	PcC
	// Bezier curveto, rel
	Pcc
	// smooth Bezier curveto, abs
	PcS
	// smooth Bezier curveto, rel
	Pcs
	// quadratic Bezier curveto, abs
	PcQ
	// quadratic Bezier curveto, rel
	Pcq
	// smooth quadratic Bezier curveto, abs
	PcT
	// smooth quadratic Bezier curveto, rel
	Pct
	// elliptical arc, abs
	PcA
	// elliptical arc, rel
	Pca
	// close path
	PcZ
	// close path
	Pcz
	// error -- invalid command
	PcErr
)

//go:generate stringer -type=PathCmds

var KiT_PathCmds = kit.Enums.AddEnumAltLower(PcErr, kit.NotBitFlag, nil, "Pc")

func (ev PathCmds) MarshalJSON() ([]byte, error)  { return kit.EnumMarshalJSON(ev) }
func (ev *PathCmds) UnmarshalJSON(b []byte) error { return kit.EnumUnmarshalJSON(ev, b) }

// PathData encodes the svg path data, using 32-bit floats which are converted
// into uint32 for path commands, and contain the command as the first 5
// bits, and the remaining 27 bits are the number of data points following the
// path command to interpret as numbers.
type PathData float32

// Cmd decodes path data as a command and a number of subsequent values for that command
func (pd PathData) Cmd() (PathCmds, int) {
	iv := uint32(pd)
	cmd := PathCmds(iv & 0x1F)       // only the lowest 5 bits (31 values) for command
	n := int((iv & 0xFFFFFFE0) >> 5) // extract the n from remainder of bits
	return cmd, n
}

// EncCmd encodes command and n into PathData
func (pc PathCmds) EncCmd(n int) PathData {
	nb := int32(n << 5) // n up-shifted
	pd := PathData(int32(pc) | nb)
	return pd
}

// PathDataNext gets the next path data point, incrementing the index
func PathDataNext(data []PathData, i *int) float32 {
	pd := data[*i]
	(*i)++
	return float32(pd)
}

// PathDataNextVec gets the next 2 path data points as a vector
func PathDataNextVec(data []PathData, i *int) mat32.Vec2 {
	v := mat32.Vec2{}
	v.X = float32(data[*i])
	(*i)++
	v.Y = float32(data[*i])
	(*i)++
	return v
}

// PathDataNextRel gets the next 2 path data points as a relative vector
// and returns that relative vector added to current point
func PathDataNextRel(data []PathData, i *int, cp mat32.Vec2) mat32.Vec2 {
	v := mat32.Vec2{}
	v.X = float32(data[*i])
	(*i)++
	v.Y = float32(data[*i])
	(*i)++
	return v.Add(cp)
}

// PathDataNextCmd gets the next path data command, incrementing the index -- ++
// not an expression so its clunky
func PathDataNextCmd(data []PathData, i *int) (PathCmds, int) {
	pd := data[*i]
	(*i)++
	return pd.Cmd()
}

func reflectPt(pt, rp mat32.Vec2) mat32.Vec2 {
	return pt.MulScalar(2).Sub(rp)
}

// PathDataRender traverses the path data and renders it using paint and render state --
// we assume all the data has been validated and that n's are sufficient, etc
func PathDataRender(data []PathData, pc *girl.Paint, rs *girl.State) {
	sz := len(data)
	if sz == 0 {
		return
	}
	lastCmd := PcErr
	var st, cp, xp, ctrl mat32.Vec2
	for i := 0; i < sz; {
		cmd, n := PathDataNextCmd(data, &i)
		rel := false
		switch cmd {
		case PcM:
			cp = PathDataNextVec(data, &i)
			pc.MoveTo(rs, cp.X, cp.Y)
			st = cp
			for np := 1; np < n/2; np++ {
				cp = PathDataNextVec(data, &i)
				pc.LineTo(rs, cp.X, cp.Y)
			}
		case Pcm:
			cp = PathDataNextRel(data, &i, cp)
			pc.MoveTo(rs, cp.X, cp.Y)
			st = cp
			for np := 1; np < n/2; np++ {
				cp = PathDataNextRel(data, &i, cp)
				pc.LineTo(rs, cp.X, cp.Y)
			}
		case PcL:
			for np := 0; np < n/2; np++ {
				cp = PathDataNextVec(data, &i)
				pc.LineTo(rs, cp.X, cp.Y)
			}
		case Pcl:
			for np := 0; np < n/2; np++ {
				cp = PathDataNextRel(data, &i, cp)
				pc.LineTo(rs, cp.X, cp.Y)
			}
		case PcH:
			for np := 0; np < n; np++ {
				cp.X = PathDataNext(data, &i)
				pc.LineTo(rs, cp.X, cp.Y)
			}
		case Pch:
			for np := 0; np < n; np++ {
				cp.X += PathDataNext(data, &i)
				pc.LineTo(rs, cp.X, cp.Y)
			}
		case PcV:
			for np := 0; np < n; np++ {
				cp.Y = PathDataNext(data, &i)
				pc.LineTo(rs, cp.X, cp.Y)
			}
		case Pcv:
			for np := 0; np < n; np++ {
				cp.Y += PathDataNext(data, &i)
				pc.LineTo(rs, cp.X, cp.Y)
			}
		case PcC:
			for np := 0; np < n/6; np++ {
				xp = PathDataNextVec(data, &i)
				ctrl = PathDataNextVec(data, &i)
				cp = PathDataNextVec(data, &i)
				pc.CubicTo(rs, xp.X, xp.Y, ctrl.X, ctrl.Y, cp.X, cp.Y)
			}
		case Pcc:
			for np := 0; np < n/6; np++ {
				xp = PathDataNextRel(data, &i, cp)
				ctrl = PathDataNextRel(data, &i, cp)
				cp = PathDataNextRel(data, &i, cp)
				pc.CubicTo(rs, xp.X, xp.Y, ctrl.X, ctrl.Y, cp.X, cp.Y)
			}
		case Pcs:
			rel = true
			fallthrough
		case PcS:
			for np := 0; np < n/4; np++ {
				switch lastCmd {
				case Pcc, PcC, Pcs, PcS:
					ctrl = reflectPt(cp, ctrl)
				default:
					ctrl = cp
				}
				if rel {
					xp = PathDataNextRel(data, &i, cp)
					cp = PathDataNextRel(data, &i, cp)
				} else {
					xp = PathDataNextVec(data, &i)
					cp = PathDataNextVec(data, &i)
				}
				pc.CubicTo(rs, ctrl.X, ctrl.Y, xp.X, xp.Y, cp.X, cp.Y)
				lastCmd = cmd
				ctrl = xp
			}
		case PcQ:
			for np := 0; np < n/4; np++ {
				ctrl = PathDataNextVec(data, &i)
				cp = PathDataNextVec(data, &i)
				pc.QuadraticTo(rs, ctrl.X, ctrl.Y, cp.X, cp.Y)
			}
		case Pcq:
			for np := 0; np < n/4; np++ {
				ctrl = PathDataNextRel(data, &i, cp)
				cp = PathDataNextRel(data, &i, cp)
				pc.QuadraticTo(rs, ctrl.X, ctrl.Y, cp.X, cp.Y)
			}
		case Pct:
			rel = true
			fallthrough
		case PcT:
			for np := 0; np < n/2; np++ {
				switch lastCmd {
				case Pcq, PcQ, PcT, Pct:
					ctrl = reflectPt(cp, ctrl)
				default:
					ctrl = cp
				}
				if rel {
					cp = PathDataNextRel(data, &i, cp)
				} else {
					cp = PathDataNextVec(data, &i)
				}
				pc.QuadraticTo(rs, ctrl.X, ctrl.Y, cp.X, cp.Y)
				lastCmd = cmd
			}
		case Pca:
			rel = true
			fallthrough
		case PcA:
			for np := 0; np < n/7; np++ {
				rad := PathDataNextVec(data, &i)
				ang := PathDataNext(data, &i)
				largeArc := (PathDataNext(data, &i) != 0)
				sweep := (PathDataNext(data, &i) != 0)
				prv := cp
				if rel {
					cp = PathDataNextRel(data, &i, cp)
				} else {
					cp = PathDataNextVec(data, &i)
				}
				ncx, ncy := girl.FindEllipseCenter(&rad.X, &rad.Y, ang*math.Pi/180, prv.X, prv.Y, cp.X, cp.Y, sweep, largeArc)
				cp.X, cp.Y = pc.DrawEllipticalArcPath(rs, ncx, ncy, cp.X, cp.Y, prv.X, prv.Y, rad.X, rad.Y, ang, largeArc, sweep)
			}
		case PcZ:
			fallthrough
		case Pcz:
			pc.ClosePath(rs)
			cp = st
		}
		lastCmd = cmd
	}
}

// PathDataIterFunc traverses the path data and calls given function on each
// coordinate point, passing overall starting index of coords in data stream,
// command, index of the points within that command, and coord values
// (absolute, not relative, regardless of the command type), including
// special control points for path commands that have them (else nil).
// If function returns false (use ki.Break vs. ki.Continue) then
// traversal is aborted.
// For Control points, order is in same order as in standard path stream
// when multiple, e.g., C,S.
// For A: order is: nc, prv, rad, mat32.Vec2{X: ang}, mat32.Vec2{laf, sf}}
func PathDataIterFunc(data []PathData, fun func(idx int, cmd PathCmds, ptIdx int, cp mat32.Vec2, ctrls []mat32.Vec2) bool) {
	sz := len(data)
	if sz == 0 {
		return
	}
	lastCmd := PcErr
	var st, cp, xp, ctrl, nc mat32.Vec2
	for i := 0; i < sz; {
		cmd, n := PathDataNextCmd(data, &i)
		rel := false
		switch cmd {
		case PcM:
			cp = PathDataNextVec(data, &i)
			if !fun(i-2, cmd, 0, cp, nil) {
				return
			}
			st = cp
			for np := 1; np < n/2; np++ {
				cp = PathDataNextVec(data, &i)
				if !fun(i-2, cmd, np, cp, nil) {
					return
				}
			}
		case Pcm:
			cp = PathDataNextRel(data, &i, cp)
			if !fun(i-2, cmd, 0, cp, nil) {
				return
			}
			st = cp
			for np := 1; np < n/2; np++ {
				cp = PathDataNextRel(data, &i, cp)
				if !fun(i-2, cmd, np, cp, nil) {
					return
				}
			}
		case PcL:
			for np := 0; np < n/2; np++ {
				cp = PathDataNextVec(data, &i)
				if !fun(i-2, cmd, np, cp, nil) {
					return
				}
			}
		case Pcl:
			for np := 0; np < n/2; np++ {
				cp = PathDataNextRel(data, &i, cp)
				if !fun(i-2, cmd, np, cp, nil) {
					return
				}
			}
		case PcH:
			for np := 0; np < n; np++ {
				cp.X = PathDataNext(data, &i)
				if !fun(i-1, cmd, np, cp, nil) {
					return
				}
			}
		case Pch:
			for np := 0; np < n; np++ {
				cp.X += PathDataNext(data, &i)
				if !fun(i-1, cmd, np, cp, nil) {
					return
				}
			}
		case PcV:
			for np := 0; np < n; np++ {
				cp.Y = PathDataNext(data, &i)
				if !fun(i-1, cmd, np, cp, nil) {
					return
				}
			}
		case Pcv:
			for np := 0; np < n; np++ {
				cp.Y += PathDataNext(data, &i)
				if !fun(i-1, cmd, np, cp, nil) {
					return
				}
			}
		case PcC:
			for np := 0; np < n/6; np++ {
				xp = PathDataNextVec(data, &i)
				ctrl = PathDataNextVec(data, &i)
				cp = PathDataNextVec(data, &i)
				if !fun(i-2, cmd, np, cp, []mat32.Vec2{xp, ctrl}) {
					return
				}
			}
		case Pcc:
			for np := 0; np < n/6; np++ {
				xp = PathDataNextRel(data, &i, cp)
				ctrl = PathDataNextRel(data, &i, cp)
				cp = PathDataNextRel(data, &i, cp)
				if !fun(i-2, cmd, np, cp, []mat32.Vec2{xp, ctrl}) {
					return
				}
			}
		case Pcs:
			rel = true
			fallthrough
		case PcS:
			for np := 0; np < n/4; np++ {
				switch lastCmd {
				case Pcc, PcC, Pcs, PcS:
					ctrl = reflectPt(cp, ctrl)
				default:
					ctrl = cp
				}
				if rel {
					xp = PathDataNextRel(data, &i, cp)
					cp = PathDataNextRel(data, &i, cp)
				} else {
					xp = PathDataNextVec(data, &i)
					cp = PathDataNextVec(data, &i)
				}
				if !fun(i-2, cmd, np, cp, []mat32.Vec2{xp, ctrl}) {
					return
				}
				lastCmd = cmd
				ctrl = xp
			}
		case PcQ:
			for np := 0; np < n/4; np++ {
				ctrl = PathDataNextVec(data, &i)
				cp = PathDataNextVec(data, &i)
				if !fun(i-2, cmd, np, cp, []mat32.Vec2{ctrl}) {
					return
				}
			}
		case Pcq:
			for np := 0; np < n/4; np++ {
				ctrl = PathDataNextRel(data, &i, cp)
				cp = PathDataNextRel(data, &i, cp)
				if !fun(i-2, cmd, np, cp, []mat32.Vec2{ctrl}) {
					return
				}
			}
		case Pct:
			rel = true
			fallthrough
		case PcT:
			for np := 0; np < n/2; np++ {
				switch lastCmd {
				case Pcq, PcQ, PcT, Pct:
					ctrl = reflectPt(cp, ctrl)
				default:
					ctrl = cp
				}
				if rel {
					cp = PathDataNextRel(data, &i, cp)
				} else {
					cp = PathDataNextVec(data, &i)
				}
				if !fun(i-2, cmd, np, cp, []mat32.Vec2{ctrl}) {
					return
				}
				lastCmd = cmd
			}
		case Pca:
			rel = true
			fallthrough
		case PcA:
			for np := 0; np < n/7; np++ {
				rad := PathDataNextVec(data, &i)
				ang := PathDataNext(data, &i)
				laf := PathDataNext(data, &i)
				largeArc := (laf != 0)
				sf := PathDataNext(data, &i)
				sweep := (sf != 0)

				prv := cp
				if rel {
					cp = PathDataNextRel(data, &i, cp)
				} else {
					cp = PathDataNextVec(data, &i)
				}
				nc.X, nc.Y = girl.FindEllipseCenter(&rad.X, &rad.Y, ang*math.Pi/180, prv.X, prv.Y, cp.X, cp.Y, sweep, largeArc)
				if !fun(i-2, cmd, np, cp, []mat32.Vec2{nc, prv, rad, mat32.Vec2{X: ang}, mat32.Vec2{laf, sf}}) {
					return
				}
			}
		case PcZ:
			fallthrough
		case Pcz:
			cp = st
		}
		lastCmd = cmd
	}
	return
}

// PathDataBBox traverses the path data and extracts the local bounding box
func PathDataBBox(data []PathData) mat32.Box2 {
	bb := mat32.NewEmptyBox2()
	PathDataIterFunc(data, func(idx int, cmd PathCmds, ptIdx int, cp mat32.Vec2, ctrls []mat32.Vec2) bool {
		bb.ExpandByPoint(cp)
		return ki.Continue
	})
	return bb
}

// PathDataStart gets the starting coords and angle from the path
func PathDataStart(data []PathData) (vec mat32.Vec2, ang float32) {
	gotSt := false
	PathDataIterFunc(data, func(idx int, cmd PathCmds, ptIdx int, cp mat32.Vec2, ctrls []mat32.Vec2) bool {
		if gotSt {
			ang = mat32.Atan2(cp.Y-vec.Y, cp.X-vec.X)
			return ki.Break
		}
		vec = cp
		gotSt = true
		return ki.Continue
	})
	return
}

// PathDataEnd gets the ending coords and angle from the path
func PathDataEnd(data []PathData) (vec mat32.Vec2, ang float32) {
	gotSome := false
	PathDataIterFunc(data, func(idx int, cmd PathCmds, ptIdx int, cp mat32.Vec2, ctrls []mat32.Vec2) bool {
		if gotSome {
			ang = mat32.Atan2(cp.Y-vec.Y, cp.X-vec.X)
		}
		vec = cp
		gotSome = true
		return ki.Continue
	})
	return
}

// PathCmdNMap gives the number of points per each command
var PathCmdNMap = map[PathCmds]int{
	PcM: 2,
	Pcm: 2,
	PcL: 2,
	Pcl: 2,
	PcH: 1,
	Pch: 1,
	PcV: 1,
	Pcv: 1,
	PcC: 6,
	Pcc: 6,
	PcS: 4,
	Pcs: 4,
	PcQ: 4,
	Pcq: 4,
	PcT: 2,
	Pct: 2,
	PcA: 7,
	Pca: 7,
	PcZ: 0,
	Pcz: 0,
}

// PathCmdIsRel returns true if the path command is relative, false for absolute
func PathCmdIsRel(pc PathCmds) bool {
	return pc%2 == 1 // odd ones are relative
}

// PathDataValidate validates the path data and emits error messages on log
func PathDataValidate(pc *girl.Paint, data *[]PathData, errstr string) error {
	sz := len(*data)
	if sz == 0 {
		return nil
	}

	di := 0
	fcmd, _ := PathDataNextCmd(*data, &di)
	if !(fcmd == Pcm || fcmd == PcM) {
		log.Printf("gi.PathDataValidate on %v: doesn't start with M or m -- adding\n", errstr)
		ns := make([]PathData, 3, sz+3)
		ns[0] = PcM.EncCmd(2)
		ns[1], ns[2] = (*data)[1], (*data)[2]
		*data = append(ns, *data...)
	}
	sz = len(*data)

	for i := 0; i < sz; {
		cmd, n := PathDataNextCmd(*data, &i)
		trgn, ok := PathCmdNMap[cmd]
		if !ok {
			err := fmt.Errorf("gi.PathDataValidate on %v: Path Command not valid: %v\n", errstr, cmd)
			log.Println(err)
			return err
		}
		if (trgn == 0 && n > 0) || (trgn > 0 && n%trgn != 0) {
			err := fmt.Errorf("gi.PathDataValidate on %v: Path Command %v has invalid n: %v -- should be: %v\n", errstr, cmd, n, trgn)
			log.Println(err)
			return err
		}
		for np := 0; np < n; np++ {
			PathDataNext(*data, &i)
		}
	}
	return nil
}

// PathRuneToCmd maps rune to path command
var PathRuneToCmd = map[rune]PathCmds{
	'M': PcM,
	'm': Pcm,
	'L': PcL,
	'l': Pcl,
	'H': PcH,
	'h': Pch,
	'V': PcV,
	'v': Pcv,
	'C': PcC,
	'c': Pcc,
	'S': PcS,
	's': Pcs,
	'Q': PcQ,
	'q': Pcq,
	'T': PcT,
	't': Pct,
	'A': PcA,
	'a': Pca,
	'Z': PcZ,
	'z': Pcz,
}

// PathCmdToRune maps command to rune
var PathCmdToRune = map[PathCmds]rune{}

func init() {
	for k, v := range PathRuneToCmd {
		PathCmdToRune[v] = k
	}
}

// PathDecodeCmd decodes rune into corresponding command
func PathDecodeCmd(r rune) PathCmds {
	cmd, ok := PathRuneToCmd[r]
	if ok {
		return cmd
	} else {
		// log.Printf("gi.PathDecodeCmd unrecognized path command: %v %v\n", string(r), r)
		return PcErr
	}
}

// PathDataParse parses a string representation of the path data into compiled path data
func PathDataParse(d string) ([]PathData, error) {
	var pd []PathData
	endi := len(d) - 1
	numSt := -1
	numGotDec := false // did last number already get a decimal point -- if so, then an additional decimal point now acts as a delimiter -- some crazy paths actually leverage that!
	lr := ' '
	lstCmd := -1
	// first pass: just do the raw parse into commands and numbers
	for i, r := range d {
		num := unicode.IsNumber(r) || (r == '.' && !numGotDec) || (r == '-' && lr == 'e') || r == 'e'
		notn := !num
		if i == endi || notn {
			if numSt != -1 || (i == endi && !notn) {
				if numSt == -1 {
					numSt = i
				}
				nstr := d[numSt:i]
				if i == endi && !notn {
					nstr = d[numSt : i+1]
				}
				p, err := strconv.ParseFloat(nstr, 32)
				if err != nil {
					log.Printf("gi.PathDataParse could not parse string: %v into float\n", nstr)
					IconAutoOpen = false
					return nil, err
				}
				pd = append(pd, PathData(p))
			}
			if r == '-' || r == '.' {
				numSt = i
				if r == '.' {
					numGotDec = true
				} else {
					numGotDec = false
				}
			} else {
				numSt = -1
				numGotDec = false
				if lstCmd != -1 { // update number of args for previous command
					lcm, _ := pd[lstCmd].Cmd()
					n := (len(pd) - lstCmd) - 1
					pd[lstCmd] = lcm.EncCmd(n)
				}
				if !unicode.IsSpace(r) && r != ',' {
					cmd := PathDecodeCmd(r)
					if cmd == PcErr {
						if i != endi {
							err := fmt.Errorf("gi.PathDataParse invalid command rune: %v\n", r)
							log.Println(err)
							return nil, err
						}
					} else {
						pc := cmd.EncCmd(0) // encode with 0 length to start
						lstCmd = len(pd)
						pd = append(pd, pc) // push on
					}
				}
			}
		} else if numSt == -1 { // got start of a number
			numSt = i
			if r == '.' {
				numGotDec = true
			} else {
				numGotDec = false
			}
		} else { // inside a number
			if r == '.' {
				numGotDec = true
			}
		}
		lr = r
	}
	return pd, nil
	// todo: add some error checking..
}

// PathDataString returns the string representation of the path data
func PathDataString(data []PathData) string {
	sz := len(data)
	if sz == 0 {
		return ""
	}
	var sb strings.Builder
	var rp, cp, xp, ctrl mat32.Vec2
	for i := 0; i < sz; {
		cmd, n := PathDataNextCmd(data, &i)
		sb.WriteString(fmt.Sprintf("%c ", PathCmdToRune[cmd]))
		switch cmd {
		case PcM, Pcm:
			cp = PathDataNextVec(data, &i)
			sb.WriteString(fmt.Sprintf("%g,%g ", cp.X, cp.Y))
			for np := 1; np < n/2; np++ {
				cp = PathDataNextVec(data, &i)
				sb.WriteString(fmt.Sprintf("%g,%g ", cp.X, cp.Y))
			}
		case PcL, Pcl:
			for np := 0; np < n/2; np++ {
				rp = PathDataNextVec(data, &i)
				sb.WriteString(fmt.Sprintf("%g,%g ", rp.X, rp.Y))
			}
		case PcH, Pch, PcV, Pcv:
			for np := 0; np < n; np++ {
				cp.Y = PathDataNext(data, &i)
				sb.WriteString(fmt.Sprintf("%g ", cp.Y))
			}
		case PcC, Pcc:
			for np := 0; np < n/6; np++ {
				xp = PathDataNextVec(data, &i)
				sb.WriteString(fmt.Sprintf("%g,%g ", xp.X, xp.Y))
				ctrl = PathDataNextVec(data, &i)
				sb.WriteString(fmt.Sprintf("%g,%g ", ctrl.X, ctrl.Y))
				cp = PathDataNextVec(data, &i)
				sb.WriteString(fmt.Sprintf("%g,%g ", cp.X, cp.Y))
			}
		case Pcs, PcS:
			for np := 0; np < n/4; np++ {
				xp = PathDataNextVec(data, &i)
				sb.WriteString(fmt.Sprintf("%g,%g ", xp.X, xp.Y))
				cp = PathDataNextVec(data, &i)
				sb.WriteString(fmt.Sprintf("%g,%g ", cp.X, cp.Y))
			}
		case PcQ, Pcq:
			for np := 0; np < n/4; np++ {
				ctrl = PathDataNextVec(data, &i)
				sb.WriteString(fmt.Sprintf("%g,%g ", ctrl.X, ctrl.Y))
				cp = PathDataNextVec(data, &i)
				sb.WriteString(fmt.Sprintf("%g,%g ", cp.X, cp.Y))
			}
		case PcT, Pct:
			for np := 0; np < n/2; np++ {
				cp = PathDataNextVec(data, &i)
				sb.WriteString(fmt.Sprintf("%g,%g ", cp.X, cp.Y))
			}
		case PcA, Pca:
			for np := 0; np < n/7; np++ {
				rad := PathDataNextVec(data, &i)
				sb.WriteString(fmt.Sprintf("%g,%g ", rad.X, rad.Y))
				ang := PathDataNext(data, &i)
				largeArc := PathDataNext(data, &i)
				sweep := PathDataNext(data, &i)
				sb.WriteString(fmt.Sprintf("%g %g %g ", ang, largeArc, sweep))
				cp = PathDataNextVec(data, &i)
				sb.WriteString(fmt.Sprintf("%g,%g ", cp.X, cp.Y))
			}
		case PcZ, Pcz:
		}
	}
	return sb.String()
}

//////////////////////////////////////////////////////////////////////////////////
//  Transforms

// ApplyXForm applies the given 2D transform to the geometry of this node
// each node must define this for itself
func (g *Path) ApplyXForm(xf mat32.Mat2) {
	// path may have horiz, vert elements -- only gen soln is to transform
	g.Pnt.XForm = g.Pnt.XForm.Mul(xf)
	g.SetProp("transform", g.Pnt.XForm.String())
	g.GradientApplyXForm(xf)
}

// PathDataXFormAbs does the transform of next two data points as absolute coords
func PathDataXFormAbs(data []PathData, i *int, xf mat32.Mat2, lpt mat32.Vec2) mat32.Vec2 {
	cp := PathDataNextVec(data, i)
	tc := xf.MulVec2AsPtCtr(cp, lpt)
	data[*i-2] = PathData(tc.X)
	data[*i-1] = PathData(tc.Y)
	return tc
}

// PathDataXFormRel does the transform of next two data points as relative coords
// compared to given cp coordinate.  returns new *absolute* coordinate
func PathDataXFormRel(data []PathData, i *int, xf mat32.Mat2, cp mat32.Vec2) mat32.Vec2 {
	rp := PathDataNextVec(data, i)
	tc := xf.MulVec2AsVec(rp)
	data[*i-2] = PathData(tc.X)
	data[*i-1] = PathData(tc.Y)
	return cp.Add(tc) // new abs
}

// ApplyDeltaXForm applies the given 2D delta transforms to the geometry of this node
// relative to given point.  Trans translation and point are in top-level coordinates,
// so must be transformed into local coords first.
// Point is upper left corner of selection box that anchors the translation and scaling,
// and for rotation it is the center point around which to rotate
func (g *Path) ApplyDeltaXForm(trans mat32.Vec2, scale mat32.Vec2, rot float32, pt mat32.Vec2) {
	if rot != 0 {
		xf, lpt := g.DeltaXForm(trans, scale, rot, pt, false) // exclude self
		g.Pnt.XForm = g.Pnt.XForm.MulCtr(xf, lpt)
		g.SetProp("transform", g.Pnt.XForm.String())
	} else {
		xf, lpt := g.DeltaXForm(trans, scale, rot, pt, true) // include self
		g.ApplyXFormImpl(xf, lpt)
		g.GradientApplyXFormPt(xf, lpt)
	}
}

// ApplyXFormImpl does the implementation of applying a transform to all points
func (g *Path) ApplyXFormImpl(xf mat32.Mat2, lpt mat32.Vec2) {
	sz := len(g.Data)
	data := g.Data
	lastCmd := PcErr
	var cp, st mat32.Vec2
	var xp, ctrl, rp mat32.Vec2
	for i := 0; i < sz; {
		cmd, n := PathDataNextCmd(data, &i)
		rel := false
		switch cmd {
		case PcM:
			cp = PathDataXFormAbs(data, &i, xf, lpt)
			st = cp
			for np := 1; np < n/2; np++ {
				cp = PathDataXFormAbs(data, &i, xf, lpt)
			}
		case Pcm:
			if i == 1 { // starting
				cp = PathDataXFormAbs(data, &i, xf, lpt)
			} else {
				cp = PathDataXFormRel(data, &i, xf, cp)
			}
			st = cp
			for np := 1; np < n/2; np++ {
				cp = PathDataXFormRel(data, &i, xf, cp)
			}
		case PcL:
			for np := 0; np < n/2; np++ {
				cp = PathDataXFormAbs(data, &i, xf, lpt)
			}
		case Pcl:
			for np := 0; np < n/2; np++ {
				cp = PathDataXFormRel(data, &i, xf, cp)
			}
		case PcH:
			for np := 0; np < n; np++ {
				cp.X = PathDataNext(data, &i)
				tc := xf.MulVec2AsPtCtr(cp, lpt)
				data[i-1] = PathData(tc.X)
			}
		case Pch:
			for np := 0; np < n; np++ {
				rp.X = PathDataNext(data, &i)
				rp.Y = 0
				rp = xf.MulVec2AsVec(rp)
				data[i-1] = PathData(rp.X)
				cp.SetAdd(rp) // new abs
			}
		case PcV:
			for np := 0; np < n; np++ {
				cp.Y = PathDataNext(data, &i)
				tc := xf.MulVec2AsPtCtr(cp, lpt)
				data[i-1] = PathData(tc.Y)
			}
		case Pcv:
			for np := 0; np < n; np++ {
				rp.Y = PathDataNext(data, &i)
				rp.X = 0
				rp = xf.MulVec2AsVec(rp)
				data[i-1] = PathData(rp.Y)
				cp.SetAdd(rp) // new abs
			}
		case PcC:
			for np := 0; np < n/6; np++ {
				xp = PathDataXFormAbs(data, &i, xf, lpt)
				ctrl = PathDataXFormAbs(data, &i, xf, lpt)
				cp = PathDataXFormAbs(data, &i, xf, lpt)
			}
		case Pcc:
			for np := 0; np < n/6; np++ {
				xp = PathDataXFormRel(data, &i, xf, cp)
				ctrl = PathDataXFormRel(data, &i, xf, cp)
				cp = PathDataXFormRel(data, &i, xf, cp)
			}
		case Pcs:
			rel = true
			fallthrough
		case PcS:
			for np := 0; np < n/4; np++ {
				switch lastCmd {
				case Pcc, PcC, Pcs, PcS:
					ctrl = reflectPt(cp, ctrl)
				default:
					ctrl = cp
				}
				if rel {
					xp = PathDataXFormRel(data, &i, xf, cp)
					cp = PathDataXFormRel(data, &i, xf, cp)
				} else {
					xp = PathDataXFormAbs(data, &i, xf, lpt)
					cp = PathDataXFormAbs(data, &i, xf, lpt)
				}
				lastCmd = cmd
				ctrl = xp
			}
		case PcQ:
			for np := 0; np < n/4; np++ {
				ctrl = PathDataXFormAbs(data, &i, xf, lpt)
				cp = PathDataXFormAbs(data, &i, xf, lpt)
			}
		case Pcq:
			for np := 0; np < n/4; np++ {
				ctrl = PathDataXFormRel(data, &i, xf, cp)
				cp = PathDataXFormRel(data, &i, xf, cp)
			}
		case Pct:
			rel = true
			fallthrough
		case PcT:
			for np := 0; np < n/2; np++ {
				switch lastCmd {
				case Pcq, PcQ, PcT, Pct:
					ctrl = reflectPt(cp, ctrl)
				default:
					ctrl = cp
				}
				if rel {
					cp = PathDataXFormRel(data, &i, xf, cp)
				} else {
					cp = PathDataXFormAbs(data, &i, xf, lpt)
				}
				lastCmd = cmd
			}
		case Pca:
			rel = true
			fallthrough
		case PcA:
			for np := 0; np < n/7; np++ {
				rad := PathDataXFormRel(data, &i, xf, mat32.Vec2{})
				ang := PathDataNext(data, &i)
				largeArc := (PathDataNext(data, &i) != 0)
				sweep := (PathDataNext(data, &i) != 0)
				pc := cp
				if rel {
					cp = PathDataXFormRel(data, &i, xf, cp)
				} else {
					cp = PathDataXFormAbs(data, &i, xf, lpt)
				}
				ncx, ncy := girl.FindEllipseCenter(&rad.X, &rad.Y, ang*math.Pi/180, pc.X, pc.Y, cp.X, cp.Y, sweep, largeArc)
				_ = ncx
				_ = ncy
			}
		case PcZ:
			fallthrough
		case Pcz:
			cp = st
		}
		lastCmd = cmd
	}

}

// WriteGeom writes the geometry of the node to a slice of floating point numbers
// the length and ordering of which is specific to each node type.
// Slice must be passed and will be resized if not the correct length.
func (g *Path) WriteGeom(dat *[]float32) {
	sz := len(g.Data)
	SetFloat32SliceLen(dat, sz+6)
	for i := range g.Data {
		(*dat)[i] = float32(g.Data[i])
	}
	g.WriteXForm(*dat, sz)
	g.GradientWritePts(dat)
}

// ReadGeom reads the geometry of the node from a slice of floating point numbers
// the length and ordering of which is specific to each node type.
func (g *Path) ReadGeom(dat []float32) {
	sz := len(g.Data)
	for i := range g.Data {
		g.Data[i] = PathData(dat[i])
	}
	g.ReadXForm(dat, sz)
	g.GradientReadPts(dat)
}
