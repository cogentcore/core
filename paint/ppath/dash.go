// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is adapted from https://github.com/tdewolff/canvas
// Copyright (c) 2015 Taco de Wolff, under an MIT License.

package ppath

// Dash returns a new path that consists of dashes.
// The elements in d specify the width of the dashes and gaps.
// It will alternate between dashes and gaps when picking widths.
// If d is an array of odd length, it is equivalent of passing d
// twice in sequence. The offset specifies the offset used into d
// (or negative offset into the path).
// Dash will be applied to each subpath independently.
func (p Path) Dash(offset float32, d ...float32) Path {
	offset, d = dashCanonical(offset, d)
	if len(d) == 0 {
		return p
	} else if len(d) == 1 && d[0] == 0.0 {
		return Path{}
	}

	if len(d)%2 == 1 {
		// if d is uneven length, dash and space lengths alternate. Duplicate d so that uneven indices are always spaces
		d = append(d, d...)
	}

	i0, pos0 := dashStart(offset, d)

	q := Path{}
	for _, ps := range p.Split() {
		i := i0
		pos := pos0

		t := []float32{}
		length := ps.Length()
		for pos+d[i]+Epsilon < length {
			pos += d[i]
			if 0.0 < pos {
				t = append(t, pos)
			}
			i++
			if i == len(d) {
				i = 0
			}
		}

		j0 := 0
		endsInDash := i%2 == 0
		if len(t)%2 == 1 && endsInDash || len(t)%2 == 0 && !endsInDash {
			j0 = 1
		}

		qd := Path{}
		pd := ps.SplitAt(t...)
		for j := j0; j < len(pd)-1; j += 2 {
			qd = qd.Append(pd[j])
		}
		if endsInDash {
			if ps.Closed() {
				qd = pd[len(pd)-1].Join(qd)
			} else {
				qd = qd.Append(pd[len(pd)-1])
			}
		}
		q = q.Append(qd)
	}
	return q
}

func dashStart(offset float32, d []float32) (int, float32) {
	i0 := 0 // index in d
	for d[i0] <= offset {
		offset -= d[i0]
		i0++
		if i0 == len(d) {
			i0 = 0
		}
	}
	pos0 := -offset // negative if offset is halfway into dash
	if offset < 0.0 {
		dTotal := float32(0.0)
		for _, dd := range d {
			dTotal += dd
		}
		pos0 = -(dTotal + offset) // handle negative offsets
	}
	return i0, pos0
}

// dashCanonical returns an optimized dash array.
func dashCanonical(offset float32, d []float32) (float32, []float32) {
	if len(d) == 0 {
		return 0.0, []float32{}
	}

	// remove zeros except first and last
	for i := 1; i < len(d)-1; i++ {
		if Equal(d[i], 0.0) {
			d[i-1] += d[i+1]
			d = append(d[:i], d[i+2:]...)
			i--
		}
	}

	// remove first zero, collapse with second and last
	if Equal(d[0], 0.0) {
		if len(d) < 3 {
			return 0.0, []float32{0.0}
		}
		offset -= d[1]
		d[len(d)-1] += d[1]
		d = d[2:]
	}

	// remove last zero, collapse with fist and second to last
	if Equal(d[len(d)-1], 0.0) {
		if len(d) < 3 {
			return 0.0, []float32{}
		}
		offset += d[len(d)-2]
		d[0] += d[len(d)-2]
		d = d[:len(d)-2]
	}

	// if there are zeros or negatives, don't draw any dashes
	for i := 0; i < len(d); i++ {
		if d[i] < 0.0 || Equal(d[i], 0.0) {
			return 0.0, []float32{0.0}
		}
	}

	// remove repeated patterns
REPEAT:
	for len(d)%2 == 0 {
		mid := len(d) / 2
		for i := 0; i < mid; i++ {
			if !Equal(d[i], d[mid+i]) {
				break REPEAT
			}
		}
		d = d[:mid]
	}
	return offset, d
}

func (p Path) checkDash(offset float32, d []float32) ([]float32, bool) {
	offset, d = dashCanonical(offset, d)
	if len(d) == 0 {
		return d, true // stroke without dashes
	} else if len(d) == 1 && d[0] == 0.0 {
		return d[:0], false // no dashes, no stroke
	}

	length := p.Length()
	i, pos := dashStart(offset, d)
	if length <= d[i]-pos {
		if i%2 == 0 {
			return d[:0], true // first dash covers whole path, stroke without dashes
		}
		return d[:0], false // first space covers whole path, no stroke
	}
	return d, true
}
