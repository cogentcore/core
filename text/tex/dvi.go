// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// note: adapted from https://github.com/tdewolff/canvas,
// Copyright (c) 2015 Taco de Wolff, under an MIT License.

package tex

import (
	"encoding/binary"
	"fmt"

	"cogentcore.org/core/paint/ppath"
)

// DVIFonts gets a font according to its font name and font size in points. Font names include:
//
//	cmr: Roman (5--10pt)
//	cmmi: Math Italic (5--10pt)
//	cmsy: Math Symbols (5--10pt)
//	cmex: Math Extension (10pt)
//	cmss: Sans serif (10pt)
//	cmssqi: Sans serif quote italic (8pt)
//	cmssi: Sans serif Italic (10pt)
//	cmbx: Bold Extended (10pt)
//	cmtt: Typewriter (8--10pt)
//	cmsltt: Slanted typewriter (10pt)
//	cmsl: Slanted roman (8--10pt)
//	cmti: Text italic (7--10pt)
//	cmu: Unslanted text italic (10pt)
//	cmmib: Bold math italic (10pt)
//	cmbsy: Bold math symbols (10pt)
//	cmcsc: Caps and Small caps (10pt)
//	cmssbx: Sans serif bold extended (10pt)
//	cmdunh: Dunhill style (10pt)
type DVIFonts interface {
	Get(string, float32) DVIFont
}

// DVIFont draws a rune/glyph to the Pather at a position in millimeters.
type DVIFont interface {
	Draw(*ppath.Path, float32, float32, uint32) float32
}

type state struct {
	h, v, w, x, y, z int32
}

// DVI2Path parses a DVI file (output from TeX) and returns *ppath.Path.
func DVI2Path(b []byte, fonts DVIFonts) (*ppath.Path, error) {
	// state
	var fnt uint32 // font index
	s := state{}
	stack := []state{}

	f := float32(1.0)            // scale factor in mm/units
	mag := uint32(1000)          // is set explicitly in preamble
	fnts := map[uint32]DVIFont{} // selected fonts for indices

	// first position of baseline which will be the path's origin
	firstChar := true
	h0 := int32(0)
	v0 := int32(0)

	p := &ppath.Path{}
	r := &dviReader{b, 0}
	for 0 < r.len() {
		cmd := r.readByte()
		if cmd <= 127 {
			// set_char
			if firstChar {
				h0, v0 = s.h, s.v
				firstChar = false
			}
			c := uint32(cmd)
			if _, ok := fnts[fnt]; !ok {
				return nil, fmt.Errorf("bad command: font %v undefined at position %v", fnt, r.i)
			}
			w := int32(fnts[fnt].Draw(p, f*float32(s.h), -f*float32(s.v), c) / f)
			s.h += w
		} else if 128 <= cmd && cmd <= 131 {
			// set
			if firstChar {
				h0, v0 = s.h, s.v
				firstChar = false
			}
			n := int(cmd - 127)
			if r.len() < n {
				return nil, fmt.Errorf("bad command: %v at position %v", cmd, r.i)
			}
			c := r.readUint32N(n)
			if _, ok := fnts[fnt]; !ok {
				return nil, fmt.Errorf("bad command: font %v undefined at position %v", fnt, r.i)
			}
			s.h += int32(fnts[fnt].Draw(p, f*float32(s.h), -f*float32(s.v), c) / f)
		} else if cmd == 132 {
			// set_rule
			height := r.readInt32()
			width := r.readInt32()
			if 0 < width && 0 < height {
				p.MoveTo(f*float32(s.h), -f*float32(s.v))
				p.LineTo(f*float32(s.h+width), -f*float32(s.v))
				p.LineTo(f*float32(s.h+width), -f*float32(s.v-height))
				p.LineTo(f*float32(s.h), -f*float32(s.v-height))
				p.Close()
			}
			s.h += width
		} else if 133 <= cmd && cmd <= 136 {
			// put
			if firstChar {
				h0, v0 = s.h, s.v
				firstChar = false
			}
			n := int(cmd - 132)
			if r.len() < n {
				return nil, fmt.Errorf("bad command: %v at position %v", cmd, r.i)
			}
			c := r.readUint32N(n)
			if _, ok := fnts[fnt]; !ok {
				return nil, fmt.Errorf("bad command: font %v undefined at position %v", fnt, r.i)
			}
			fnts[fnt].Draw(p, f*float32(s.h), -f*float32(s.v), c)
		} else if cmd == 137 {
			// put_rule
			height := r.readInt32()
			width := r.readInt32()
			if 0 < width && 0 < height {
				p.MoveTo(f*float32(s.h), -f*float32(s.v))
				p.LineTo(f*float32(s.h+width), -f*float32(s.v))
				p.LineTo(f*float32(s.h+width), -f*float32(s.v-height))
				p.LineTo(f*float32(s.h), -f*float32(s.v-height))
				p.Close()
			}
		} else if cmd == 138 {
			// nop
		} else if cmd == 139 {
			// bop
			fnt = 0
			s = state{0, 0, 0, 0, 0, 0}
			stack = stack[:0]
			_ = r.readBytes(10 * 4)
			_ = r.readUint32() // pointer
		} else if cmd == 140 {
			// eop
		} else if cmd == 141 {
			// push
			stack = append(stack, s)
		} else if cmd == 142 {
			// pop
			if len(stack) == 0 {
				return nil, fmt.Errorf("bad command: stack is empty at position %v", r.i)
			}
			s = stack[len(stack)-1]
			stack = stack[:len(stack)-1]
		} else if 143 <= cmd && cmd <= 146 {
			// right
			n := int(cmd - 142)
			if r.len() < n {
				return nil, fmt.Errorf("bad command: %v at position %v", cmd, r.i)
			}
			d := r.readInt32N(n)
			s.h += d
		} else if 147 <= cmd && cmd <= 151 {
			// w
			if cmd == 147 {
				s.h += s.w
			} else {
				n := int(cmd - 147)
				if r.len() < n {
					return nil, fmt.Errorf("bad command: %v at position %v", cmd, r.i)
				}
				d := r.readInt32N(n)
				s.w = d
				s.h += d
			}
		} else if 152 <= cmd && cmd <= 156 {
			// x
			if cmd == 152 {
				s.h += s.x
			} else {
				n := int(cmd - 152)
				if r.len() < n {
					return nil, fmt.Errorf("bad command: %v at position %v", cmd, r.i)
				}
				d := r.readInt32N(n)
				s.x = d
				s.h += d
			}
		} else if 157 <= cmd && cmd <= 160 {
			// down
			n := int(cmd - 156)
			if r.len() < n {
				return nil, fmt.Errorf("bad command: %v at position %v", cmd, r.i)
			}
			d := r.readInt32N(n)
			s.v += d
		} else if 161 <= cmd && cmd <= 165 {
			// y
			if cmd == 161 {
				s.v += s.y
			} else {
				n := int(cmd - 152)
				if r.len() < n {
					return nil, fmt.Errorf("bad command: %v at position %v", cmd, r.i)
				}
				d := r.readInt32N(n)
				s.y = d
				s.v += d
			}
		} else if 166 <= cmd && cmd <= 170 {
			// z
			if cmd == 166 {
				s.v += s.z
			} else {
				n := int(cmd - 166)
				if r.len() < n {
					return nil, fmt.Errorf("bad command: %v at position %v", cmd, r.i)
				}
				d := r.readInt32N(n)
				s.z = d
				s.v += d
			}
		} else if 171 <= cmd && cmd <= 234 {
			// fnt_num
			fnt = uint32(cmd - 171)
		} else if 235 <= cmd && cmd <= 242 {
			// fnt
			n := int(cmd - 234)
			if r.len() < n {
				return nil, fmt.Errorf("bad command: %v at position %v", cmd, r.i)
			}
			fnt = r.readUint32N(n)
		} else if 239 <= cmd && cmd <= 242 {
			// xxx
			n := int(cmd - 242)
			if r.len() < n {
				return nil, fmt.Errorf("bad command: %v at position %v", cmd, r.i)
			}
			k := int(r.readUint32N(n))
			if r.len() < k {
				return nil, fmt.Errorf("bad command: %v at position %v", cmd, r.i)
			}
			_ = r.readBytes(k)
		} else if 243 <= cmd && cmd <= 246 {
			// fnt_def
			n := int(cmd - 242)
			if r.len() < n+14 {
				return nil, fmt.Errorf("bad command: %v at position %v", cmd, r.i)
			}
			k := r.readUint32N(n)
			_ = r.readBytes(4) // checksum
			size := r.readUint32()
			design := r.readUint32() // design
			a := r.readByte()
			l := r.readByte()
			if r.len() < int(a+l) {
				return nil, fmt.Errorf("bad command: %v at position %v", cmd, r.i)
			}
			_ = r.readString(int(a)) // area
			name := r.readString(int(l))
			fnts[k] = fonts.Get(name, float32(mag)*float32(size)/1000.0/float32(design))
		} else if cmd == 247 {
			// pre
			_ = r.readByte() // version
			num := r.readUint32()
			den := r.readUint32()
			mag = r.readUint32()
			f = float32(num) / float32(den) * float32(mag) / 1000.0 / 10000.0 // in units/mm
			n := int(r.readByte())
			_ = r.readString(n) // comment
		} else if cmd == 248 {
			_ = r.readUint32() // pointer to final bop
			_ = r.readUint32() // num
			_ = r.readUint32() // den
			_ = r.readUint32() // mag
			_ = r.readUint32() // largest height
			_ = r.readUint32() // largest width
			_ = r.readUint16() // maximum stack depth
			_ = r.readUint16() // number of pages
		} else if cmd == 249 {
			_ = r.readUint32() // pointer to post
			_ = r.readByte()   // version
			for 0 < r.len() {
				if r.readByte() != 223 {
					break
				}
			}
		} else {
			return nil, fmt.Errorf("bad command: %v at position %v", cmd, r.i)
		}
	}
	// _ = h0
	// _ = v0
	*p = p.Translate(-f*float32(h0), f*float32(v0))
	// *p = p.Translate(100, 100)
	return p, nil
}

type dviReader struct {
	b []byte
	i int
}

func (r *dviReader) len() int {
	return len(r.b) - r.i
}

func (r *dviReader) readByte() byte {
	r.i++
	return r.b[r.i-1]
}

func (r *dviReader) readUint16() uint16 {
	num := binary.BigEndian.Uint16(r.b[r.i : r.i+2])
	r.i += 2
	return num
}

func (r *dviReader) readUint32() uint32 {
	num := binary.BigEndian.Uint32(r.b[r.i : r.i+4])
	r.i += 4
	return num
}

func (r *dviReader) readInt32() int32 {
	return int32(r.readUint32())
}

func (r *dviReader) readUint32N(n int) uint32 {
	if n == 1 {
		return uint32(r.readByte())
	} else if n == 2 {
		return uint32(r.readUint16())
	} else if n == 3 {
		a := r.readByte()
		b := r.readByte()
		c := r.readByte()
		return uint32(a)<<16 | uint32(b)<<8 | uint32(c)
	} else if n == 4 {
		return r.readUint32()
	}
	r.i += n
	return 0
}

func (r *dviReader) readInt32N(n int) int32 {
	if n == 3 {
		a := r.readByte()
		b := r.readByte()
		c := r.readByte()
		if a < 128 {
			return int32(uint32(a)<<16 | uint32(b)<<8 | uint32(c))
		}
		return int32((uint32(a)-256)<<16 | uint32(b)<<8 | uint32(c))
	}
	return int32(r.readUint32N(n))
}

func (r *dviReader) readBytes(n int) []byte {
	b := r.b[r.i : r.i+n]
	r.i += n
	return b
}

func (r *dviReader) readString(n int) string {
	return string(r.readBytes(n))
}
