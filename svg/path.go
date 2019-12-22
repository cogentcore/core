// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package svg

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"unicode"

	"github.com/chewxy/math32"
	"github.com/goki/gi/gi"
	"github.com/goki/ki/ki"
	"github.com/goki/ki/kit"
)

// Path renders SVG data sequences that can render just about anything
type Path struct {
	NodeBase
	Data     []PathData `xml:"-" desc:"the path data to render -- path commands and numbers are serialized, with each command specifying the number of floating-point coord data points that follow"`
	DataStr  string     `xml:"d" desc:"string version of the path data"`
	MinCoord gi.Vec2D   `desc:"minimum coord in path -- computed in BBox2D"`
	MaxCoord gi.Vec2D   `desc:"maximum coord in path -- computed in BBox2D"`
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

func (g *Path) CopyFieldsFrom(frm interface{}) {
	fr := frm.(*Path)
	g.NodeBase.CopyFieldsFrom(&fr.NodeBase)
	g.Data = make([]PathData, len(fr.Data))
	copy(g.Data, fr.Data)
	g.DataStr = fr.DataStr
	g.MinCoord = fr.MinCoord
	g.MaxCoord = fr.MaxCoord
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
	err = PathDataValidate(&g.Pnt, &g.Data, g.PathUnique())
	return err
}

func (g *Path) Render2D() {
	if g.Viewport == nil {
		g.This().(gi.Node2D).Init2D()
	}
	sz := len(g.Data)
	if sz < 2 {
		return
	}

	pc := &g.Pnt
	rs := &g.Viewport.Render
	rs.Lock()
	rs.PushXForm(pc.XForm)
	PathDataRender(g.Data, pc, rs)
	pc.FillStrokeClear(rs)
	rs.Unlock()

	g.ComputeBBoxSVG()

	if mrk := g.Marker("marker-start"); mrk != nil {
		// todo: could look for close-path at end and find angle from there..
		stv, ang := PathDataStart(g.Data)
		mrk.RenderMarker(stv, ang, g.Pnt.StrokeStyle.Width.Dots)
	}
	if mrk := g.Marker("marker-end"); mrk != nil {
		env, ang := PathDataEnd(g.Data)
		mrk.RenderMarker(env, ang, g.Pnt.StrokeStyle.Width.Dots)
	}
	if mrk := g.Marker("marker-mid"); mrk != nil {
		var ptm2, ptm1, pt gi.Vec2D
		gotidx := 0
		PathDataIterFunc(g.Data, func(idx int, cmd PathCmds, ptIdx int, cx, cy float32) bool {
			ptm2 = ptm1
			ptm1 = pt
			pt = gi.Vec2D{cx, cy}
			if gotidx < 2 {
				gotidx++
				return true
			}
			if idx >= sz-3 { // todo: this is approximate...
				return false
			}
			ang := 0.5 * (math32.Atan2(pt.Y-ptm1.Y, pt.X-ptm1.X) + math32.Atan2(ptm1.Y-ptm2.Y, ptm1.X-ptm2.X))
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

// PathDataNext gets the next path data point, incrementing the index -- ++
// not an expression so its clunky
func PathDataNext(data []PathData, i *int) float32 {
	pd := data[*i]
	(*i)++
	return float32(pd)
}

// PathDataNextCmd gets the next path data command, incrementing the index -- ++
// not an expression so its clunky
func PathDataNextCmd(data []PathData, i *int) (PathCmds, int) {
	pd := data[*i]
	(*i)++
	return pd.Cmd()
}

func reflectPt(px, py, rx, ry float32) (x, y float32) {
	return px*2 - rx, py*2 - ry
}

// PathDataRender traverses the path data and renders it using paint and render state --
// we assume all the data has been validated and that n's are sufficient, etc
func PathDataRender(data []PathData, pc *gi.Paint, rs *gi.RenderState) {
	sz := len(data)
	if sz == 0 {
		return
	}
	lastCmd := PcErr
	var stx, sty, cx, cy, x1, y1, ctrlx, ctrly float32
	for i := 0; i < sz; {
		cmd, n := PathDataNextCmd(data, &i)
		rel := false
		switch cmd {
		case PcM:
			cx = PathDataNext(data, &i)
			cy = PathDataNext(data, &i)
			pc.MoveTo(rs, cx, cy)
			stx, sty = cx, cy
			for np := 1; np < n/2; np++ {
				cx = PathDataNext(data, &i)
				cy = PathDataNext(data, &i)
				pc.LineTo(rs, cx, cy)
			}
		case Pcm:
			cx += PathDataNext(data, &i)
			cy += PathDataNext(data, &i)
			pc.MoveTo(rs, cx, cy)
			stx, sty = cx, cy
			for np := 1; np < n/2; np++ {
				cx += PathDataNext(data, &i)
				cy += PathDataNext(data, &i)
				pc.LineTo(rs, cx, cy)
			}
		case PcL:
			for np := 0; np < n/2; np++ {
				cx = PathDataNext(data, &i)
				cy = PathDataNext(data, &i)
				pc.LineTo(rs, cx, cy)
			}
		case Pcl:
			for np := 0; np < n/2; np++ {
				cx += PathDataNext(data, &i)
				cy += PathDataNext(data, &i)
				pc.LineTo(rs, cx, cy)
			}
		case PcH:
			for np := 0; np < n; np++ {
				cx = PathDataNext(data, &i)
				pc.LineTo(rs, cx, cy)
			}
		case Pch:
			for np := 0; np < n; np++ {
				cx += PathDataNext(data, &i)
				pc.LineTo(rs, cx, cy)
			}
		case PcV:
			for np := 0; np < n; np++ {
				cy = PathDataNext(data, &i)
				pc.LineTo(rs, cx, cy)
			}
		case Pcv:
			for np := 0; np < n; np++ {
				cy += PathDataNext(data, &i)
				pc.LineTo(rs, cx, cy)
			}
		case PcC:
			for np := 0; np < n/6; np++ {
				x1 = PathDataNext(data, &i)
				y1 = PathDataNext(data, &i)
				ctrlx = PathDataNext(data, &i)
				ctrly = PathDataNext(data, &i)
				cx = PathDataNext(data, &i)
				cy = PathDataNext(data, &i)
				pc.CubicTo(rs, x1, y1, ctrlx, ctrly, cx, cy)
			}
		case Pcc:
			for np := 0; np < n/6; np++ {
				x1 = cx + PathDataNext(data, &i)
				y1 = cy + PathDataNext(data, &i)
				ctrlx = cx + PathDataNext(data, &i)
				ctrly = cy + PathDataNext(data, &i)
				cx += PathDataNext(data, &i)
				cy += PathDataNext(data, &i)
				pc.CubicTo(rs, x1, y1, ctrlx, ctrly, cx, cy)
			}
		case Pcs:
			rel = true
			fallthrough
		case PcS:
			for np := 0; np < n/4; np++ {
				switch lastCmd {
				case Pcc, PcC, Pcs, PcS:
					ctrlx, ctrly = reflectPt(cx, cy, ctrlx, ctrly)
				default:
					ctrlx, ctrly = cx, cy
				}
				if rel {
					x1 = cx + PathDataNext(data, &i)
					y1 = cy + PathDataNext(data, &i)
					cx += PathDataNext(data, &i)
					cy += PathDataNext(data, &i)
				} else {
					x1 = PathDataNext(data, &i)
					y1 = PathDataNext(data, &i)
					cx = PathDataNext(data, &i)
					cy = PathDataNext(data, &i)
				}
				pc.CubicTo(rs, ctrlx, ctrly, x1, y1, cx, cy)
				lastCmd = cmd
				ctrlx = x1
				ctrly = y1
			}
		case PcQ:
			for np := 0; np < n/4; np++ {
				ctrlx = PathDataNext(data, &i)
				ctrly = PathDataNext(data, &i)
				cx = PathDataNext(data, &i)
				cy = PathDataNext(data, &i)
				pc.QuadraticTo(rs, ctrlx, ctrly, cx, cy)
			}
		case Pcq:
			for np := 0; np < n/4; np++ {
				ctrlx = cx + PathDataNext(data, &i)
				ctrly = cy + PathDataNext(data, &i)
				cx += PathDataNext(data, &i)
				cy += PathDataNext(data, &i)
				pc.QuadraticTo(rs, ctrlx, ctrly, cx, cy)
			}
		case Pct:
			rel = true
			fallthrough
		case PcT:
			for np := 0; np < n/2; np++ {
				switch lastCmd {
				case Pcq, PcQ, PcT, Pct:
					ctrlx, ctrly = reflectPt(cx, cy, ctrlx, ctrly)
				default:
					ctrlx, ctrly = cx, cy
				}
				if rel {
					cx += PathDataNext(data, &i)
					cy += PathDataNext(data, &i)
				} else {
					cx = PathDataNext(data, &i)
					cy = PathDataNext(data, &i)
				}
				pc.QuadraticTo(rs, ctrlx, ctrly, cx, cy)
				lastCmd = cmd
			}
		case Pca:
			rel = true
			fallthrough
		case PcA:
			for np := 0; np < n/7; np++ {
				rx := PathDataNext(data, &i)
				ry := PathDataNext(data, &i)
				ang := PathDataNext(data, &i)
				largeArc := (PathDataNext(data, &i) != 0)
				sweep := (PathDataNext(data, &i) != 0)
				pcx := cx
				pcy := cy
				if rel {
					cx += PathDataNext(data, &i)
					cy += PathDataNext(data, &i)
				} else {
					cx = PathDataNext(data, &i)
					cy = PathDataNext(data, &i)
				}
				ncx, ncy := gi.FindEllipseCenter(&rx, &ry, ang*math.Pi/180, pcx, pcy, cx, cy, sweep, largeArc)
				cx, cy = pc.DrawEllipticalArcPath(rs, ncx, ncy, cx, cy, pcx, pcy, rx, ry, ang, largeArc, sweep)
			}
		case PcZ:
			fallthrough
		case Pcz:
			pc.ClosePath(rs)
			cx, cy = stx, sty
		}
		lastCmd = cmd
	}
}

// PathDataIterFunc traverses the path data and calls given function on each
// coordinate point, passing overall starting index of coords in data stream,
// command, index of the points within that command, and coord values
// (absolute, not relative, regardless of the command type) -- if function
// returns false, then traversal is aborted
func PathDataIterFunc(data []PathData, fun func(idx int, cmd PathCmds, ptIdx int, cx, cy float32) bool) {
	sz := len(data)
	if sz == 0 {
		return
	}
	var cx, cy, x1, y1 float32
	for i := 0; i < sz; {
		cmd, n := PathDataNextCmd(data, &i)
		rel := false
		switch cmd {
		case PcM:
			cx = PathDataNext(data, &i)
			cy = PathDataNext(data, &i)
			if !fun(i-2, cmd, 0, cx, cy) {
				return
			}
			for np := 1; np < n/2; np++ {
				cx = PathDataNext(data, &i)
				cy = PathDataNext(data, &i)
				if !fun(i-2, cmd, np, cx, cy) {
					return
				}
			}
		case Pcm:
			cx += PathDataNext(data, &i)
			cy += PathDataNext(data, &i)
			if !fun(i, cmd, 0, cx, cy) {
				return
			}
			for np := 1; np < n/2; np++ {
				cx += PathDataNext(data, &i)
				cy += PathDataNext(data, &i)
				if !fun(i-2, cmd, np, cx, cy) {
					return
				}
			}
		case PcL:
			for np := 0; np < n/2; np++ {
				cx = PathDataNext(data, &i)
				cy = PathDataNext(data, &i)
				if !fun(i-2, cmd, np, cx, cy) {
					return
				}
			}
		case Pcl:
			for np := 0; np < n/2; np++ {
				cx += PathDataNext(data, &i)
				cy += PathDataNext(data, &i)
				if !fun(i-2, cmd, np, cx, cy) {
					return
				}
			}
		case PcH:
			for np := 0; np < n; np++ {
				cx = PathDataNext(data, &i)
				if !fun(i-1, cmd, np, cx, cy) {
					return
				}
			}
		case Pch:
			for np := 0; np < n; np++ {
				cx += PathDataNext(data, &i)
				if !fun(i-1, cmd, np, cx, cy) {
					return
				}
			}
		case PcV:
			for np := 0; np < n; np++ {
				cy = PathDataNext(data, &i)
				if !fun(i-1, cmd, np, cx, cy) {
					return
				}
			}
		case Pcv:
			for np := 0; np < n; np++ {
				cy += PathDataNext(data, &i)
				if !fun(i-1, cmd, np, cx, cy) {
					return
				}
			}
		case Pcc:
			rel = true
			fallthrough
		case PcC:
			for np := 0; np < n/6; np++ {
				if rel {
					x1 = PathDataNext(data, &i)
					y1 = PathDataNext(data, &i)
				} else {
					x1 = cx + PathDataNext(data, &i)
					y1 = cy + PathDataNext(data, &i)
				}
				PathDataNext(data, &i)
				PathDataNext(data, &i)
				if rel {
					cx += PathDataNext(data, &i)
					cy += PathDataNext(data, &i)
				} else {
					cx = PathDataNext(data, &i)
					cy = PathDataNext(data, &i)
				}
				x1 = y1 // just to use them -- not sure if should pass to fun
				if !fun(i-2, cmd, np, cx, cy) {
					return
				}
			}
		case PcS:
			for np := 0; np < n/4; np++ {
				x1 = PathDataNext(data, &i)
				y1 = PathDataNext(data, &i)
				cx = PathDataNext(data, &i)
				cy = PathDataNext(data, &i)
				y1 = x1 // just to use them -- not sure if should pass to fun
				if !fun(i-2, cmd, np, cx, cy) {
					return
				}
			}
		case Pcs:
			for np := 0; np < n/4; np++ {
				x1 = cx + PathDataNext(data, &i)
				y1 = cy + PathDataNext(data, &i)
				cx += PathDataNext(data, &i)
				cy += PathDataNext(data, &i)
				if !fun(i-2, cmd, np, cx, cy) {
					return
				}
			}
		case PcQ:
			for np := 0; np < n/4; np++ {
				PathDataNext(data, &i)
				PathDataNext(data, &i)
				cx = PathDataNext(data, &i)
				cy = PathDataNext(data, &i)
				if !fun(i-2, cmd, np, cx, cy) {
					return
				}
			}
		case Pcq:
			for np := 0; np < n/4; np++ {
				PathDataNext(data, &i)
				PathDataNext(data, &i)
				cx += PathDataNext(data, &i)
				cy += PathDataNext(data, &i)
				if !fun(i-2, cmd, np, cx, cy) {
					return
				}
			}
		case PcT:
			for np := 0; np < n/2; np++ {
				cx = PathDataNext(data, &i)
				cy = PathDataNext(data, &i)
				if !fun(i-2, cmd, np, cx, cy) {
					return
				}
			}
		case Pct:
			for np := 0; np < n/2; np++ {
				cx += PathDataNext(data, &i)
				cy += PathDataNext(data, &i)
				if !fun(i-2, cmd, np, cx, cy) {
					return
				}
			}
		case Pca:
			rel = true
			fallthrough
		case PcA:
			for np := 0; np < n/7; np++ {
				PathDataNext(data, &i) // rx
				PathDataNext(data, &i) // ry
				PathDataNext(data, &i) // ang
				PathDataNext(data, &i) // large-arc-flag
				PathDataNext(data, &i) // sweep-flag
				if rel {
					cx += PathDataNext(data, &i)
					cy += PathDataNext(data, &i)
				} else {
					cx = PathDataNext(data, &i)
					cy = PathDataNext(data, &i)
				}
				if !fun(i-2, cmd, np, cx, cy) {
					return
				}
			}
		case PcZ:
		case Pcz:
		}
	}
	return
}

// PathDataMinMax traverses the path data and extracts the min and max point coords
func PathDataMinMax(data []PathData) (min, max gi.Vec2D) {
	PathDataIterFunc(data, func(idx int, cmd PathCmds, ptIdx int, cx, cy float32) bool {
		c := gi.Vec2D{cx, cy}
		if min == gi.Vec2DZero && max == gi.Vec2DZero {
			min = c
			max = c
		} else {
			min.SetMin(c)
			max.SetMax(c)
		}
		return true
	})
	return
}

// PathDataStart gets the starting coords and angle from the path
func PathDataStart(data []PathData) (vec gi.Vec2D, ang float32) {
	gotSt := false
	PathDataIterFunc(data, func(idx int, cmd PathCmds, ptIdx int, cx, cy float32) bool {
		c := gi.Vec2D{cx, cy}
		if gotSt {
			ang = math32.Atan2(c.Y-vec.Y, c.X-vec.X)
			return false // stop
		}
		vec = c
		gotSt = true
		return true
	})
	return
}

// PathDataEnd gets the ending coords and angle from the path
func PathDataEnd(data []PathData) (vec gi.Vec2D, ang float32) {
	gotSome := false
	PathDataIterFunc(data, func(idx int, cmd PathCmds, ptIdx int, cx, cy float32) bool {
		c := gi.Vec2D{cx, cy}
		if gotSome {
			ang = math32.Atan2(c.Y-vec.Y, c.X-vec.X)
		}
		vec = c
		gotSome = true
		return true
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

// PathDataValidate validates the path data and emits error messages on log
func PathDataValidate(pc *gi.Paint, data *[]PathData, errstr string) error {
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

// PathCmdMap maps rune to path command
var PathCmdMap = map[rune]PathCmds{
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

// PathDecodeCmd decodes rune into corresponding command
func PathDecodeCmd(r rune) PathCmds {
	cmd, ok := PathCmdMap[r]
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
