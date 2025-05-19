// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is adapted from https://github.com/tdewolff/canvas
// Copyright (c) 2015 Taco de Wolff, under an MIT License.

package ppath

import (
	"fmt"
	"math"
	"strings"

	"cogentcore.org/core/math32"
	"github.com/tdewolff/parse/v2/strconv"
)

type num float32

func (f num) String() string {
	s := fmt.Sprintf("%.*g", Precision, f)
	if num(math.MaxInt32) < f || f < num(math.MinInt32) {
		if i := strings.IndexAny(s, ".eE"); i == -1 {
			s += ".0"
		}
	}
	return string(MinifyNumber([]byte(s), Precision))
}

type dec float32

func (f dec) String() string {
	s := fmt.Sprintf("%.*f", Precision, f)
	s = string(MinifyDecimal([]byte(s), Precision))
	if dec(math.MaxInt32) < f || f < dec(math.MinInt32) {
		if i := strings.IndexByte(s, '.'); i == -1 {
			s += ".0"
		}
	}
	return s
}

func skipCommaWhitespace(path []byte) int {
	i := 0
	for i < len(path) && (path[i] == ' ' || path[i] == ',' || path[i] == '\n' || path[i] == '\r' || path[i] == '\t') {
		i++
	}
	return i
}

// MustParseSVGPath parses an SVG path data string and panics if it fails.
func MustParseSVGPath(s string) Path {
	p, err := ParseSVGPath(s)
	if err != nil {
		panic(err)
	}
	return p
}

// ParseSVGPath parses an SVG path data string.
func ParseSVGPath(s string) (Path, error) {
	if len(s) == 0 {
		return Path{}, nil
	}

	i := 0
	path := []byte(s)
	i += skipCommaWhitespace(path[i:])
	if path[0] == ',' || path[i] < 'A' {
		return nil, fmt.Errorf("bad path: path should start with command")
	}

	cmdLens := map[byte]int{
		'M': 2,
		'Z': 0,
		'L': 2,
		'H': 1,
		'V': 1,
		'C': 6,
		'S': 4,
		'Q': 4,
		'T': 2,
		'A': 7,
	}
	f := [7]float32{}

	p := Path{}
	var q, c math32.Vector2
	var p0, p1 math32.Vector2
	prevCmd := byte('z')
	for {
		i += skipCommaWhitespace(path[i:])
		if len(path) <= i {
			break
		}

		cmd := prevCmd
		repeat := true
		if cmd == 'z' || cmd == 'Z' || !(path[i] >= '0' && path[i] <= '9' || path[i] == '.' || path[i] == '-' || path[i] == '+') {
			cmd = path[i]
			repeat = false
			i++
			i += skipCommaWhitespace(path[i:])
		}

		CMD := cmd
		if 'a' <= cmd && cmd <= 'z' {
			CMD -= 'a' - 'A'
		}
		for j := 0; j < cmdLens[CMD]; j++ {
			if CMD == 'A' && (j == 3 || j == 4) {
				// parse largeArc and sweep booleans for A command
				if i < len(path) && path[i] == '1' {
					f[j] = 1.0
				} else if i < len(path) && path[i] == '0' {
					f[j] = 0.0
				} else {
					return nil, fmt.Errorf("bad path: largeArc and sweep flags should be 0 or 1 in command '%c' at position %d", cmd, i+1)
				}
				i++
			} else {
				num, n := strconv.ParseFloat(path[i:])
				if n == 0 {
					if repeat && j == 0 && i < len(path) {
						return nil, fmt.Errorf("bad path: unknown command '%c' at position %d", path[i], i+1)
					} else if 1 < cmdLens[CMD] {
						return nil, fmt.Errorf("bad path: sets of %d numbers should follow command '%c' at position %d", cmdLens[CMD], cmd, i+1)
					} else {
						return nil, fmt.Errorf("bad path: number should follow command '%c' at position %d", cmd, i+1)
					}
				}
				f[j] = float32(num)
				i += n
			}
			i += skipCommaWhitespace(path[i:])
		}

		switch cmd {
		case 'M', 'm':
			p1 = math32.Vector2{f[0], f[1]}
			if cmd == 'm' {
				p1 = p1.Add(p0)
				cmd = 'l'
			} else {
				cmd = 'L'
			}
			p.MoveTo(p1.X, p1.Y)
		case 'Z', 'z':
			p1 = p.StartPos()
			p.Close()
		case 'L', 'l':
			p1 = math32.Vector2{f[0], f[1]}
			if cmd == 'l' {
				p1 = p1.Add(p0)
			}
			p.LineTo(p1.X, p1.Y)
		case 'H', 'h':
			p1.X = f[0]
			if cmd == 'h' {
				p1.X += p0.X
			}
			p.LineTo(p1.X, p1.Y)
		case 'V', 'v':
			p1.Y = f[0]
			if cmd == 'v' {
				p1.Y += p0.Y
			}
			p.LineTo(p1.X, p1.Y)
		case 'C', 'c':
			cp1 := math32.Vector2{f[0], f[1]}
			cp2 := math32.Vector2{f[2], f[3]}
			p1 = math32.Vector2{f[4], f[5]}
			if cmd == 'c' {
				cp1 = cp1.Add(p0)
				cp2 = cp2.Add(p0)
				p1 = p1.Add(p0)
			}
			p.CubeTo(cp1.X, cp1.Y, cp2.X, cp2.Y, p1.X, p1.Y)
			c = cp2
		case 'S', 's':
			cp1 := p0
			cp2 := math32.Vector2{f[0], f[1]}
			p1 = math32.Vector2{f[2], f[3]}
			if cmd == 's' {
				cp2 = cp2.Add(p0)
				p1 = p1.Add(p0)
			}
			if prevCmd == 'C' || prevCmd == 'c' || prevCmd == 'S' || prevCmd == 's' {
				cp1 = p0.MulScalar(2.0).Sub(c)
			}
			p.CubeTo(cp1.X, cp1.Y, cp2.X, cp2.Y, p1.X, p1.Y)
			c = cp2
		case 'Q', 'q':
			cp := math32.Vector2{f[0], f[1]}
			p1 = math32.Vector2{f[2], f[3]}
			if cmd == 'q' {
				cp = cp.Add(p0)
				p1 = p1.Add(p0)
			}
			p.QuadTo(cp.X, cp.Y, p1.X, p1.Y)
			q = cp
		case 'T', 't':
			cp := p0
			p1 = math32.Vector2{f[0], f[1]}
			if cmd == 't' {
				p1 = p1.Add(p0)
			}
			if prevCmd == 'Q' || prevCmd == 'q' || prevCmd == 'T' || prevCmd == 't' {
				cp = p0.MulScalar(2.0).Sub(q)
			}
			p.QuadTo(cp.X, cp.Y, p1.X, p1.Y)
			q = cp
		case 'A', 'a':
			rx := f[0]
			ry := f[1]
			rot := f[2]
			large := f[3] == 1.0
			sweep := f[4] == 1.0
			p1 = math32.Vector2{f[5], f[6]}
			if cmd == 'a' {
				p1 = p1.Add(p0)
			}
			p.ArcToDeg(rx, ry, rot, large, sweep, p1.X, p1.Y)
		default:
			return nil, fmt.Errorf("bad path: unknown command '%c' at position %d", cmd, i+1)
		}
		prevCmd = cmd
		p0 = p1
	}
	return p, nil
}

// String returns a string that represents the path similar to the SVG
// path data format (but not necessarily valid SVG).
func (p Path) String() string {
	sb := strings.Builder{}
	for i := 0; i < len(p); {
		cmd := p[i]
		switch cmd {
		case MoveTo:
			fmt.Fprintf(&sb, "M%g %g", p[i+1], p[i+2])
		case LineTo:
			fmt.Fprintf(&sb, "L%g %g", p[i+1], p[i+2])
		case QuadTo:
			fmt.Fprintf(&sb, "Q%g %g %g %g", p[i+1], p[i+2], p[i+3], p[i+4])
		case CubeTo:
			fmt.Fprintf(&sb, "C%g %g %g %g %g %g", p[i+1], p[i+2], p[i+3], p[i+4], p[i+5], p[i+6])
		case ArcTo:
			rot := math32.RadToDeg(p[i+3])
			large, sweep := ToArcFlags(p[i+4])
			sLarge := "0"
			if large {
				sLarge = "1"
			}
			sSweep := "0"
			if sweep {
				sSweep = "1"
			}
			fmt.Fprintf(&sb, "A%g %g %g %s %s %g %g", p[i+1], p[i+2], rot, sLarge, sSweep, p[i+5], p[i+6])
		case Close:
			fmt.Fprintf(&sb, "z")
		}
		i += CmdLen(cmd)
	}
	return sb.String()
}

// ToSVG returns a string that represents the path in the SVG path data format with minification.
func (p Path) ToSVG() string {
	if p.Empty() {
		return ""
	}

	sb := strings.Builder{}
	var x, y float32
	for i := 0; i < len(p); {
		cmd := p[i]
		switch cmd {
		case MoveTo:
			x, y = p[i+1], p[i+2]
			fmt.Fprintf(&sb, "M%v %v", num(x), num(y))
		case LineTo:
			xStart, yStart := x, y
			x, y = p[i+1], p[i+2]
			if Equal(x, xStart) && Equal(y, yStart) {
				// nothing
			} else if Equal(x, xStart) {
				fmt.Fprintf(&sb, "V%v", num(y))
			} else if Equal(y, yStart) {
				fmt.Fprintf(&sb, "H%v", num(x))
			} else {
				fmt.Fprintf(&sb, "L%v %v", num(x), num(y))
			}
		case QuadTo:
			x, y = p[i+3], p[i+4]
			fmt.Fprintf(&sb, "Q%v %v %v %v", num(p[i+1]), num(p[i+2]), num(x), num(y))
		case CubeTo:
			x, y = p[i+5], p[i+6]
			fmt.Fprintf(&sb, "C%v %v %v %v %v %v", num(p[i+1]), num(p[i+2]), num(p[i+3]), num(p[i+4]), num(x), num(y))
		case ArcTo:
			rx, ry := p[i+1], p[i+2]
			rot := math32.RadToDeg(p[i+3])
			large, sweep := ToArcFlags(p[i+4])
			x, y = p[i+5], p[i+6]
			sLarge := "0"
			if large {
				sLarge = "1"
			}
			sSweep := "0"
			if sweep {
				sSweep = "1"
			}
			if 90.0 <= rot {
				rx, ry = ry, rx
				rot -= 90.0
			}
			fmt.Fprintf(&sb, "A%v %v %v %s%s%v %v", num(rx), num(ry), num(rot), sLarge, sSweep, num(p[i+5]), num(p[i+6]))
		case Close:
			x, y = p[i+1], p[i+2]
			fmt.Fprintf(&sb, "z")
		}
		i += CmdLen(cmd)
	}
	return sb.String()
}

// ToPS returns a string that represents the path in the PostScript data format.
func (p Path) ToPS() string {
	if p.Empty() {
		return ""
	}

	sb := strings.Builder{}
	var x, y float32
	for i := 0; i < len(p); {
		cmd := p[i]
		switch cmd {
		case MoveTo:
			x, y = p[i+1], p[i+2]
			fmt.Fprintf(&sb, " %v %v moveto", dec(x), dec(y))
		case LineTo:
			x, y = p[i+1], p[i+2]
			fmt.Fprintf(&sb, " %v %v lineto", dec(x), dec(y))
		case QuadTo, CubeTo:
			var start, cp1, cp2 math32.Vector2
			start = math32.Vector2{x, y}
			if cmd == QuadTo {
				x, y = p[i+3], p[i+4]
				cp1, cp2 = QuadraticToCubicBezier(start, math32.Vec2(p[i+1], p[i+2]), math32.Vector2{x, y})
			} else {
				cp1 = math32.Vec2(p[i+1], p[i+2])
				cp2 = math32.Vec2(p[i+3], p[i+4])
				x, y = p[i+5], p[i+6]
			}
			fmt.Fprintf(&sb, " %v %v %v %v %v %v curveto", dec(cp1.X), dec(cp1.Y), dec(cp2.X), dec(cp2.Y), dec(x), dec(y))
		case ArcTo:
			x0, y0 := x, y
			rx, ry, phi := p[i+1], p[i+2], p[i+3]
			large, sweep := ToArcFlags(p[i+4])
			x, y = p[i+5], p[i+6]

			cx, cy, theta0, theta1 := EllipseToCenter(x0, y0, rx, ry, phi, large, sweep, x, y)
			theta0 = math32.RadToDeg(theta0)
			theta1 = math32.RadToDeg(theta1)
			rot := math32.RadToDeg(phi)

			fmt.Fprintf(&sb, " %v %v %v %v %v %v %v ellipse", dec(cx), dec(cy), dec(rx), dec(ry), dec(theta0), dec(theta1), dec(rot))
			if !sweep {
				fmt.Fprintf(&sb, "n")
			}
		case Close:
			x, y = p[i+1], p[i+2]
			fmt.Fprintf(&sb, " closepath")
		}
		i += CmdLen(cmd)
	}
	return sb.String()[1:] // remove the first space
}

// ToPDF returns a string that represents the path in the PDF data format.
func (p Path) ToPDF() string {
	if p.Empty() {
		return ""
	}
	p = p.ReplaceArcs()

	sb := strings.Builder{}
	var x, y float32
	for i := 0; i < len(p); {
		cmd := p[i]
		switch cmd {
		case MoveTo:
			x, y = p[i+1], p[i+2]
			fmt.Fprintf(&sb, " %v %v m", dec(x), dec(y))
		case LineTo:
			x, y = p[i+1], p[i+2]
			fmt.Fprintf(&sb, " %v %v l", dec(x), dec(y))
		case QuadTo, CubeTo:
			var start, cp1, cp2 math32.Vector2
			start = math32.Vector2{x, y}
			if cmd == QuadTo {
				x, y = p[i+3], p[i+4]
				cp1, cp2 = QuadraticToCubicBezier(start, math32.Vec2(p[i+1], p[i+2]), math32.Vector2{x, y})
			} else {
				cp1 = math32.Vec2(p[i+1], p[i+2])
				cp2 = math32.Vec2(p[i+3], p[i+4])
				x, y = p[i+5], p[i+6]
			}
			fmt.Fprintf(&sb, " %v %v %v %v %v %v c", dec(cp1.X), dec(cp1.Y), dec(cp2.X), dec(cp2.Y), dec(x), dec(y))
		case ArcTo:
			panic("arcs should have been replaced")
		case Close:
			x, y = p[i+1], p[i+2]
			fmt.Fprintf(&sb, " h")
		}
		i += CmdLen(cmd)
	}
	return sb.String()[1:] // remove the first space
}
