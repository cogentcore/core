// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is adapted from https://github.com/tdewolff/canvas
// Copyright (c) 2015 Taco de Wolff, under an MIT License.

package path

import (
	"fmt"
	"io"
	"slices"
	"sort"
	"strings"
	"sync"

	"cogentcore.org/core/math32"
)

// BentleyOttmannEpsilon is the snap rounding grid used by the Bentley-Ottmann algorithm.
// This prevents numerical issues. It must be larger than Epsilon since we use that to calculate
// intersections between segments. It is the number of binary digits to keep.
var BentleyOttmannEpsilon = float32(1e-8)

// RayIntersections returns the intersections of a path with a ray starting at (x,y) to (∞,y).
// An intersection is tangent only when it is at (x,y), i.e. the start of the ray. Intersections
// are sorted along the ray. This function runs in O(n) with n the number of path segments.
func (p Path) RayIntersections(x, y float32) []Intersection {
	var start, end, cp1, cp2 math32.Vector2
	var zs []Intersection
	for i := 0; i < len(p); {
		cmd := p[i]
		switch cmd {
		case MoveTo:
			end = p.EndPoint(i)
		case LineTo, Close:
			end = p.EndPoint(i)
			ymin := math32.Min(start.Y, end.Y)
			ymax := math32.Max(start.Y, end.Y)
			xmax := math32.Max(start.X, end.X)
			if InInterval(y, ymin, ymax) && x <= xmax+Epsilon {
				zs = intersectionLineLine(zs, math32.Vector2{x, y}, math32.Vector2{xmax + 1.0, y}, start, end)
			}
		case QuadTo:
			cp1, end = p.QuadToPoints(i)
			ymin := math32.Min(math32.Min(start.Y, end.Y), cp1.Y)
			ymax := math32.Max(math32.Max(start.Y, end.Y), cp1.Y)
			xmax := math32.Max(math32.Max(start.X, end.X), cp1.X)
			if InInterval(y, ymin, ymax) && x <= xmax+Epsilon {
				zs = intersectionLineQuad(zs, math32.Vector2{x, y}, math32.Vector2{xmax + 1.0, y}, start, cp1, end)
			}
		case CubeTo:
			cp1, cp2, end = p.CubeToPoints(i)
			ymin := math32.Min(math32.Min(start.Y, end.Y), math32.Min(cp1.Y, cp2.Y))
			ymax := math32.Max(math32.Max(start.Y, end.Y), math32.Max(cp1.Y, cp2.Y))
			xmax := math32.Max(math32.Max(start.X, end.X), math32.Max(cp1.X, cp2.X))
			if InInterval(y, ymin, ymax) && x <= xmax+Epsilon {
				zs = intersectionLineCube(zs, math32.Vector2{x, y}, math32.Vector2{xmax + 1.0, y}, start, cp1, cp2, end)
			}
		case ArcTo:
			var rx, ry, phi float32
			var large, sweep bool
			rx, ry, phi, large, sweep, end = p.ArcToPoints(i)
			cx, cy, theta0, theta1 := ellipseToCenter(start.X, start.Y, rx, ry, phi, large, sweep, end.X, end.Y)
			if InInterval(y, cy-math32.Max(rx, ry), cy+math32.Max(rx, ry)) && x <= cx+math32.Max(rx, ry)+Epsilon {
				zs = intersectionLineEllipse(zs, math32.Vector2{x, y}, math32.Vector2{cx + rx + 1.0, y}, math32.Vector2{cx, cy}, math32.Vector2{rx, ry}, phi, theta0, theta1)
			}
		}
		i += CmdLen(cmd)
		start = end
	}
	for i := range zs {
		if zs[i].T[0] != 0.0 {
			zs[i].T[0] = math32.NaN()
		}
	}
	sort.SliceStable(zs, func(i, j int) bool {
		if Equal(zs[i].X, zs[j].X) {
			return false
		}
		return zs[i].X < zs[j].X
	})
	return zs
}

type pathOp int

const (
	opSettle pathOp = iota
	opAND
	opOR
	opNOT
	opXOR
	opDIV
)

func (op pathOp) String() string {
	switch op {
	case opSettle:
		return "Settle"
	case opAND:
		return "AND"
	case opOR:
		return "OR"
	case opNOT:
		return "NOT"
	case opXOR:
		return "XOR"
	case opDIV:
		return "DIV"
	}
	return fmt.Sprintf("pathOp(%d)", op)
}

var boPointPool *sync.Pool
var boNodePool *sync.Pool
var boSquarePool *sync.Pool
var boInitPoolsOnce = sync.OnceFunc(func() {
	boPointPool = &sync.Pool{New: func() any { return &SweepPoint{} }}
	boNodePool = &sync.Pool{New: func() any { return &SweepNode{} }}
	boSquarePool = &sync.Pool{New: func() any { return &toleranceSquare{} }}
})

// Settle returns the "settled" path. It removes all self-intersections, orients all filling paths
// CCW and all holes CW, and tries to split into subpaths if possible. Note that path p is
// flattened unless q is already flat. Path q is implicitly closed. It runs in O((n + k) log n),
// with n the sum of the number of segments, and k the number of intersections.
func (p Path) Settle(fillRule FillRules) Path {
	return bentleyOttmann(p.Split(), nil, opSettle, fillRule)
}

// Settle is the same as Path.Settle, but faster if paths are already split.
func (ps Paths) Settle(fillRule FillRules) Path {
	return bentleyOttmann(ps, nil, opSettle, fillRule)
}

// And returns the boolean path operation of path p AND q, i.e. the intersection of both. It
// removes all self-intersections, orients all filling paths CCW and all holes CW, and tries to
// split into subpaths if possible. Note that path p is flattened unless q is already flat. Path
// q is implicitly closed. It runs in O((n + k) log n), with n the sum of the number of segments,
// and k the number of intersections.
func (p Path) And(q Path) Path {
	return bentleyOttmann(p.Split(), q.Split(), opAND, NonZero)
}

// And is the same as Path.And, but faster if paths are already split.
func (ps Paths) And(qs Paths) Path {
	return bentleyOttmann(ps, qs, opAND, NonZero)
}

// Or returns the boolean path operation of path p OR q, i.e. the union of both. It
// removes all self-intersections, orients all filling paths CCW and all holes CW, and tries to
// split into subpaths if possible. Note that path p is flattened unless q is already flat. Path
// q is implicitly closed. It runs in O((n + k) log n), with n the sum of the number of segments,
// and k the number of intersections.
func (p Path) Or(q Path) Path {
	return bentleyOttmann(p.Split(), q.Split(), opOR, NonZero)
}

// Or is the same as Path.Or, but faster if paths are already split.
func (ps Paths) Or(qs Paths) Path {
	return bentleyOttmann(ps, qs, opOR, NonZero)
}

// Xor returns the boolean path operation of path p XOR q, i.e. the symmetric difference of both.
// It removes all self-intersections, orients all filling paths CCW and all holes CW, and tries to
// split into subpaths if possible. Note that path p is flattened unless q is already flat. Path
// q is implicitly closed. It runs in O((n + k) log n), with n the sum of the number of segments,
// and k the number of intersections.
func (p Path) Xor(q Path) Path {
	return bentleyOttmann(p.Split(), q.Split(), opXOR, NonZero)
}

// Xor is the same as Path.Xor, but faster if paths are already split.
func (ps Paths) Xor(qs Paths) Path {
	return bentleyOttmann(ps, qs, opXOR, NonZero)
}

// Not returns the boolean path operation of path p NOT q, i.e. the difference of both.
// It removes all self-intersections, orients all filling paths CCW and all holes CW, and tries to
// split into subpaths if possible. Note that path p is flattened unless q is already flat. Path
// q is implicitly closed. It runs in O((n + k) log n), with n the sum of the number of segments,
// and k the number of intersections.
func (p Path) Not(q Path) Path {
	return bentleyOttmann(p.Split(), q.Split(), opNOT, NonZero)
}

// Not is the same as Path.Not, but faster if paths are already split.
func (ps Paths) Not(qs Paths) Path {
	return bentleyOttmann(ps, qs, opNOT, NonZero)
}

// DivideBy returns the boolean path operation of path p DIV q, i.e. p divided by q.
// It removes all self-intersections, orients all filling paths CCW and all holes CW, and tries to
// split into subpaths if possible. Note that path p is flattened unless q is already flat. Path
// q is implicitly closed. It runs in O((n + k) log n), with n the sum of the number of segments,
// and k the number of intersections.
func (p Path) DivideBy(q Path) Path {
	return bentleyOttmann(p.Split(), q.Split(), opDIV, NonZero)
}

// DivideBy is the same as PathivideBy, but faster if paths are already split.
func (ps Paths) DivideBy(qs Paths) Path {
	return bentleyOttmann(ps, qs, opDIV, NonZero)
}

type SweepPoint struct {
	// initial data
	math32.Vector2             // position of this endpoint
	other          *SweepPoint // pointer to the other endpoint of the segment
	segment        int         // segment index to distinguish self-overlapping segments

	// processing the queue
	node *SweepNode // used for fast accessing btree node in O(1) (instead of Find in O(log n))

	// computing sweep fields
	windings          int         // windings of the same polygon (excluding this segment)
	otherWindings     int         // windings of the other polygon
	selfWindings      int         // positive if segment goes left-right (or bottom-top when vertical)
	otherSelfWindings int         // used when merging overlapping segments
	prev              *SweepPoint // segment below

	// building the polygon
	index          int // index into result array
	resultWindings int // windings of the resulting polygon

	// bools at the end to optimize memory layout of struct
	clipping   bool  // is clipping path (otherwise is subject path)
	open       bool  // path is not closed (only for subject paths)
	left       bool  // point is left-end of segment
	vertical   bool  // segment is vertical
	increasing bool  // original direction is left-right (or bottom-top)
	overlapped bool  // segment's overlapping was handled
	inResult   uint8 // in final result polygon (1 is once, 2 is twice for opDIV)
}

func (s *SweepPoint) InterpolateY(x float32) float32 {
	t := (x - s.X) / (s.other.X - s.X)
	return s.Lerp(s.other.Vector2, t).Y
}

// ToleranceEdgeY returns the y-value of the SweepPoint at the tolerance edges given by xLeft and
// xRight, or at the endpoints of the SweepPoint, whichever comes first.
func (s *SweepPoint) ToleranceEdgeY(xLeft, xRight float32) (float32, float32) {
	if !s.left {
		s = s.other
	}

	y0 := s.Y
	if s.X < xLeft {
		y0 = s.InterpolateY(xLeft)
	}
	y1 := s.other.Y
	if xRight <= s.other.X {
		y1 = s.InterpolateY(xRight)
	}
	return y0, y1
}

func (s *SweepPoint) SplitAt(z math32.Vector2) (*SweepPoint, *SweepPoint) {
	// split segment at point
	r := boPointPool.Get().(*SweepPoint)
	l := boPointPool.Get().(*SweepPoint)
	*r, *l = *s.other, *s
	r.Vector2, l.Vector2 = z, z

	// update references
	r.other, s.other.other = s, l
	l.other, s.other = s.other, r
	l.node = nil
	return r, l
}

func (s *SweepPoint) Reverse() {
	s.left, s.other.left = !s.left, s.left
	s.increasing, s.other.increasing = !s.increasing, !s.other.increasing
}

func (s *SweepPoint) String() string {
	path := "P"
	if s.clipping {
		path = "Q"
	}
	arrow := "→"
	if !s.left {
		arrow = "←"
	}
	return fmt.Sprintf("%s-%v(%v%v%v)", path, s.segment, s.Vector2, arrow, s.other.Vector2)
}

// SweepEvents is a heap priority queue of sweep events.
type SweepEvents []*SweepPoint

func (q SweepEvents) Less(i, j int) bool {
	return q[i].LessH(q[j])
}

func (q SweepEvents) Swap(i, j int) {
	q[i], q[j] = q[j], q[i]
}

func (q *SweepEvents) AddPathEndpoints(p Path, seg int, clipping bool) int {
	if len(p) == 0 {
		return seg
	}

	// TODO: change this if we allow non-flat paths
	// allocate all memory at once to prevent multiple allocations/memmoves below
	n := len(p) / 4
	if cap(*q) < len(*q)+n {
		q2 := make(SweepEvents, len(*q), len(*q)+n)
		copy(q2, *q)
		*q = q2
	}

	open := !p.Closed()
	start := math32.Vector2{p[1], p[2]}
	if math32.IsNaN(start.X) || math32.IsInf(start.X, 0.0) || math32.IsNaN(start.Y) || math32.IsInf(start.Y, 0.0) {
		panic("path has NaN or Inf")
	}
	for i := 4; i < len(p); {
		if p[i] != LineTo && p[i] != Close {
			panic("non-flat paths not supported")
		}

		n := CmdLen(p[i])
		end := math32.Vector2{p[i+n-3], p[i+n-2]}
		if math32.IsNaN(end.X) || math32.IsInf(end.X, 0.0) || math32.IsNaN(end.Y) || math32.IsInf(end.Y, 0.0) {
			panic("path has NaN or Inf")
		}
		i += n
		seg++

		if start == end {
			// skip zero-length lineTo or close command
			start = end
			continue
		}

		vertical := start.X == end.X
		increasing := start.X < end.X
		if vertical {
			increasing = start.Y < end.Y
		}
		a := boPointPool.Get().(*SweepPoint)
		b := boPointPool.Get().(*SweepPoint)
		*a = SweepPoint{
			Vector2:    start,
			clipping:   clipping,
			open:       open,
			segment:    seg,
			left:       increasing,
			increasing: increasing,
			vertical:   vertical,
		}
		*b = SweepPoint{
			Vector2:    end,
			clipping:   clipping,
			open:       open,
			segment:    seg,
			left:       !increasing,
			increasing: increasing,
			vertical:   vertical,
		}
		a.other = b
		b.other = a
		*q = append(*q, a, b)
		start = end
	}
	return seg
}

func (q SweepEvents) Init() {
	n := len(q)
	for i := n/2 - 1; 0 <= i; i-- {
		q.down(i, n)
	}
}

func (q *SweepEvents) Push(item *SweepPoint) {
	*q = append(*q, item)
	q.up(len(*q) - 1)
}

func (q *SweepEvents) Top() *SweepPoint {
	return (*q)[0]
}

func (q *SweepEvents) Pop() *SweepPoint {
	n := len(*q) - 1
	q.Swap(0, n)
	q.down(0, n)

	items := (*q)[n]
	*q = (*q)[:n]
	return items
}

func (q *SweepEvents) Fix(i int) {
	if !q.down(i, len(*q)) {
		q.up(i)
	}
}

// from container/heap
func (q SweepEvents) up(j int) {
	for {
		i := (j - 1) / 2 // parent
		if i == j || !q.Less(j, i) {
			break
		}
		q.Swap(i, j)
		j = i
	}
}

func (q SweepEvents) down(i0, n int) bool {
	i := i0
	for {
		j1 := 2*i + 1
		if n <= j1 || j1 < 0 { // j1 < 0 after int overflow
			break
		}
		j := j1 // left child
		if j2 := j1 + 1; j2 < n && q.Less(j2, j1) {
			j = j2 // = 2*i + 2  // right child
		}
		if !q.Less(j, i) {
			break
		}
		q.Swap(i, j)
		i = j
	}
	return i0 < i
}

func (q SweepEvents) Print(w io.Writer) {
	q2 := make(SweepEvents, len(q))
	copy(q2, q)
	q = q2

	n := len(q) - 1
	for 0 < n {
		q.Swap(0, n)
		q.down(0, n)
		n--
	}
	width := int(math32.Max(0.0, math32.Log10(float32(len(q)-1)))) + 1
	for k := len(q) - 1; 0 <= k; k-- {
		fmt.Fprintf(w, "%*d %v\n", width, len(q)-1-k, q[k])
	}
	return
}

func (q SweepEvents) String() string {
	sb := strings.Builder{}
	q.Print(&sb)
	str := sb.String()
	if 0 < len(str) {
		str = str[:len(str)-1]
	}
	return str
}

type SweepNode struct {
	parent, left, right *SweepNode
	height              int

	*SweepPoint
}

func (n *SweepNode) Prev() *SweepNode {
	// go left
	if n.left != nil {
		n = n.left
		for n.right != nil {
			n = n.right // find the right-most of current subtree
		}
		return n
	}

	for n.parent != nil && n.parent.left == n {
		n = n.parent // find first parent for which we're right
	}
	return n.parent // can be nil
}

func (n *SweepNode) Next() *SweepNode {
	// go right
	if n.right != nil {
		n = n.right
		for n.left != nil {
			n = n.left // find the left-most of current subtree
		}
		return n
	}

	for n.parent != nil && n.parent.right == n {
		n = n.parent // find first parent for which we're left
	}
	return n.parent // can be nil
}

func (a *SweepNode) swap(b *SweepNode) {
	a.SweepPoint, b.SweepPoint = b.SweepPoint, a.SweepPoint
	a.SweepPoint.node, b.SweepPoint.node = a, b
}

//func (n *SweepNode) fix() (*SweepNode, int) {
//	move := 0
//	if prev := n.Prev(); prev != nil && 0 < prev.CompareV(n.SweepPoint, false) {
//		// move down
//		n.swap(prev)
//		n, prev = prev, n
//		move--
//
//		for prev = prev.Prev(); prev != nil; prev = prev.Prev() {
//			if prev.CompareV(n.SweepPoint, false) < 0 {
//				break
//			}
//			n.swap(prev)
//			n, prev = prev, n
//			move--
//		}
//	} else if next := n.Next(); next != nil && next.CompareV(n.SweepPoint, false) < 0 {
//		// move up
//		n.swap(next)
//		n, next = next, n
//		move++
//
//		for next = next.Next(); next != nil; next = next.Next() {
//			if 0 < next.CompareV(n.SweepPoint, false) {
//				break
//			}
//			n.swap(next)
//			n, next = next, n
//			move++
//		}
//	}
//	return n, move
//}

func (n *SweepNode) balance() int {
	r := 0
	if n.left != nil {
		r -= n.left.height
	}
	if n.right != nil {
		r += n.right.height
	}
	return r
}

func (n *SweepNode) updateHeight() {
	n.height = 0
	if n.left != nil {
		n.height = n.left.height
	}
	if n.right != nil && n.height < n.right.height {
		n.height = n.right.height
	}
	n.height++
}

func (n *SweepNode) swapChild(a, b *SweepNode) {
	if n.right == a {
		n.right = b
	} else {
		n.left = b
	}
	if b != nil {
		b.parent = n
	}
}

func (a *SweepNode) rotateLeft() *SweepNode {
	b := a.right
	if a.parent != nil {
		a.parent.swapChild(a, b)
	} else {
		b.parent = nil
	}
	a.parent = b
	if a.right = b.left; a.right != nil {
		a.right.parent = a
	}
	b.left = a
	return b
}

func (a *SweepNode) rotateRight() *SweepNode {
	b := a.left
	if a.parent != nil {
		a.parent.swapChild(a, b)
	} else {
		b.parent = nil
	}
	a.parent = b
	if a.left = b.right; a.left != nil {
		a.left.parent = a
	}
	b.right = a
	return b
}

func (n *SweepNode) print(w io.Writer, prefix string, cmp int) {
	c := ""
	if cmp < 0 {
		c = "│ "
	} else if 0 < cmp {
		c = "  "
	}
	if n.right != nil {
		n.right.print(w, prefix+c, 1)
	} else if n.left != nil {
		fmt.Fprintf(w, "%v%v┌─nil\n", prefix, c)
	}

	c = ""
	if 0 < cmp {
		c = "┌─"
	} else if cmp < 0 {
		c = "└─"
	}
	fmt.Fprintf(w, "%v%v%v\n", prefix, c, n.SweepPoint)

	c = ""
	if 0 < cmp {
		c = "│ "
	} else if cmp < 0 {
		c = "  "
	}
	if n.left != nil {
		n.left.print(w, prefix+c, -1)
	} else if n.right != nil {
		fmt.Fprintf(w, "%v%v└─nil\n", prefix, c)
	}
}

func (n *SweepNode) Print(w io.Writer) {
	n.print(w, "", 0)
}

// TODO: test performance versus (2,4)-tree (current LEDA implementation), (2,16)-tree (as proposed by S. Naber/Näher in "Comparison of search-tree data structures in LEDA. Personal communication" apparently), RB-tree (likely a good candidate), and an AA-tree (simpler implementation may be faster). Perhaps an unbalanced (e.g. Treap) works well due to the high number of insertions/deletions.
type SweepStatus struct {
	root *SweepNode
}

func (s *SweepStatus) newNode(item *SweepPoint) *SweepNode {
	n := boNodePool.Get().(*SweepNode)
	n.parent = nil
	n.left = nil
	n.right = nil
	n.height = 1
	n.SweepPoint = item
	n.SweepPoint.node = n
	return n
}

func (s *SweepStatus) returnNode(n *SweepNode) {
	n.SweepPoint.node = nil
	n.SweepPoint = nil // help the GC
	boNodePool.Put(n)
}

func (s *SweepStatus) find(item *SweepPoint) (*SweepNode, int) {
	n := s.root
	for n != nil {
		cmp := item.CompareV(n.SweepPoint)
		if cmp < 0 {
			if n.left == nil {
				return n, -1
			}
			n = n.left
		} else if 0 < cmp {
			if n.right == nil {
				return n, 1
			}
			n = n.right
		} else {
			break
		}
	}
	return n, 0
}

func (s *SweepStatus) rebalance(n *SweepNode) {
	for {
		oheight := n.height
		if balance := n.balance(); balance == 2 {
			// Tree is excessively right-heavy, rotate it to the left.
			if n.right != nil && n.right.balance() < 0 {
				// Right tree is left-heavy, which would cause the next rotation to result in
				// overall left-heaviness. Rotate the right tree to the right to counteract this.
				n.right = n.right.rotateRight()
				n.right.right.updateHeight()
			}
			n = n.rotateLeft()
			n.left.updateHeight()
		} else if balance == -2 {
			// Tree is excessively left-heavy, rotate it to the right
			if n.left != nil && 0 < n.left.balance() {
				// The left tree is right-heavy, which would cause the next rotation to result in
				// overall right-heaviness. Rotate the left tree to the left to compensate.
				n.left = n.left.rotateLeft()
				n.left.left.updateHeight()
			}
			n = n.rotateRight()
			n.right.updateHeight()
		} else if balance < -2 || 2 < balance {
			panic("Tree too far out of shape!")
		}

		n.updateHeight()
		if n.parent == nil {
			s.root = n
			return
		}
		if oheight == n.height {
			return
		}
		n = n.parent
	}
}

func (s *SweepStatus) String() string {
	if s.root == nil {
		return "nil"
	}

	sb := strings.Builder{}
	s.root.Print(&sb)
	str := sb.String()
	if 0 < len(str) {
		str = str[:len(str)-1]
	}
	return str
}

func (s *SweepStatus) First() *SweepNode {
	if s.root == nil {
		return nil
	}
	n := s.root
	for n.left != nil {
		n = n.left
	}
	return n
}

func (s *SweepStatus) Last() *SweepNode {
	if s.root == nil {
		return nil
	}
	n := s.root
	for n.right != nil {
		n = n.right
	}
	return n
}

// Find returns the node equal to item. May return nil.
func (s *SweepStatus) Find(item *SweepPoint) *SweepNode {
	n, cmp := s.find(item)
	if cmp == 0 {
		return n
	}
	return nil
}

func (s *SweepStatus) FindPrevNext(item *SweepPoint) (*SweepNode, *SweepNode) {
	if s.root == nil {
		return nil, nil
	}

	n, cmp := s.find(item)
	if cmp < 0 {
		return n.Prev(), n
	} else if 0 < cmp {
		return n, n.Next()
	} else {
		return n.Prev(), n.Next()
	}
}

func (s *SweepStatus) Insert(item *SweepPoint) *SweepNode {
	if s.root == nil {
		s.root = s.newNode(item)
		return s.root
	}

	rebalance := false
	n, cmp := s.find(item)
	if cmp < 0 {
		// lower
		n.left = s.newNode(item)
		n.left.parent = n
		rebalance = n.right == nil
	} else if 0 < cmp {
		// higher
		n.right = s.newNode(item)
		n.right.parent = n
		rebalance = n.left == nil
	} else {
		// equal, replace
		n.SweepPoint.node = nil
		n.SweepPoint = item
		n.SweepPoint.node = n
		return n
	}

	if rebalance && n.parent != nil {
		n.height++
		s.rebalance(n.parent)
	}

	if cmp < 0 {
		return n.left
	} else {
		return n.right
	}
}

func (s *SweepStatus) InsertAfter(n *SweepNode, item *SweepPoint) *SweepNode {
	var cur *SweepNode
	rebalance := false
	if n == nil {
		if s.root == nil {
			s.root = s.newNode(item)
			return s.root
		}

		// insert as left-most node in tree
		n = s.root
		for n.left != nil {
			n = n.left
		}
		n.left = s.newNode(item)
		n.left.parent = n
		rebalance = n.right == nil
		cur = n.left
	} else if n.right == nil {
		// insert directly to the right of n
		n.right = s.newNode(item)
		n.right.parent = n
		rebalance = n.left == nil
		cur = n.right
	} else {
		// insert next to n at a deeper level
		n = n.right
		for n.left != nil {
			n = n.left
		}
		n.left = s.newNode(item)
		n.left.parent = n
		rebalance = n.right == nil
		cur = n.left
	}

	if rebalance && n.parent != nil {
		n.height++
		s.rebalance(n.parent)
	}
	return cur
}

func (s *SweepStatus) Remove(n *SweepNode) {
	ancestor := n.parent
	if n.left == nil || n.right == nil {
		// no children or one child
		child := n.left
		if n.left == nil {
			child = n.right
		}
		if n.parent != nil {
			n.parent.swapChild(n, child)
		} else {
			s.root = child
		}
		if child != nil {
			child.parent = n.parent
		}
	} else {
		// two children
		succ := n.right
		for succ.left != nil {
			succ = succ.left
		}
		ancestor = succ.parent // rebalance from here
		if succ.parent == n {
			// succ is child of n
			ancestor = succ
		}
		succ.parent.swapChild(succ, succ.right)

		// swap succesor with deleted node
		succ.parent, succ.left, succ.right = n.parent, n.left, n.right
		if n.parent != nil {
			n.parent.swapChild(n, succ)
		} else {
			s.root = succ
		}
		if n.left != nil {
			n.left.parent = succ
		}
		if n.right != nil {
			n.right.parent = succ
		}
	}

	// rebalance all ancestors
	for ; ancestor != nil; ancestor = ancestor.parent {
		s.rebalance(ancestor)
	}
	s.returnNode(n)
	return
}

func (s *SweepStatus) Clear() {
	n := s.First()
	for n != nil {
		cur := n
		n = n.Next()
		s.returnNode(cur)
	}
	s.root = nil
}

func (a *SweepPoint) LessH(b *SweepPoint) bool {
	// used for sweep queue
	if a.X != b.X {
		return a.X < b.X // sort left to right
	} else if a.Y != b.Y {
		return a.Y < b.Y // then bottom to top
	} else if a.left != b.left {
		return b.left // handle right-endpoints before left-endpoints
	} else if a.compareTangentsV(b) < 0 {
		return true // sort upwards, this ensures CCW orientation order of result
	}
	return false
}

func (a *SweepPoint) CompareH(b *SweepPoint) int {
	// used for sweep queue
	// sort left-to-right, then bottom-to-top, then right-endpoints before left-endpoints, and then
	// sort upwards to ensure a CCW orientation of the result
	if a.X < b.X {
		return -1
	} else if b.X < a.X {
		return 1
	} else if a.Y < b.Y {
		return -1
	} else if b.Y < a.Y {
		return 1
	} else if !a.left && b.left {
		return -1
	} else if a.left && !b.left {
		return 1
	}
	return a.compareTangentsV(b)
}

func (a *SweepPoint) compareOverlapsV(b *SweepPoint) int {
	// compare segments vertically that overlap (ie. are the same)
	if a.clipping != b.clipping {
		// for equal segments, clipping path is virtually on top (or left if vertical) of subject
		if b.clipping {
			return -1
		} else {
			return 1
		}
	}

	// equal segment on same path, sort by segment index
	if a.segment != b.segment {
		if a.segment < b.segment {
			return -1
		} else {
			return 1
		}
	}
	return 0
}

func (a *SweepPoint) compareTangentsV(b *SweepPoint) int {
	// compare segments vertically at a.X, b.X <= a.X, and a and b coincide at (a.X,a.Y)
	// note that a.left==b.left, we may be comparing right-endpoints
	sign := 1
	if !a.left {
		sign = -1
	}
	if a.vertical {
		// a is vertical
		if b.vertical {
			// a and b are vertical
			if a.Y == b.Y {
				return sign * a.compareOverlapsV(b)
			} else if a.Y < b.Y {
				return -1
			} else {
				return 1
			}
		}
		return 1
	} else if b.vertical {
		// b is vertical
		return -1
	}

	if a.other.X == b.other.X && a.other.Y == b.other.Y {
		return sign * a.compareOverlapsV(b)
	} else if a.left && a.other.X < b.other.X || !a.left && b.other.X < a.other.X {
		by := b.InterpolateY(a.other.X) // b's y at a's other
		if a.other.Y == by {
			return sign * a.compareOverlapsV(b)
		} else if a.other.Y < by {
			return sign * -1
		} else {
			return sign * 1
		}
	} else {
		ay := a.InterpolateY(b.other.X) // a's y at b's other
		if ay == b.other.Y {
			return sign * a.compareOverlapsV(b)
		} else if ay < b.other.Y {
			return sign * -1
		} else {
			return sign * 1
		}
	}
}

func (a *SweepPoint) compareV(b *SweepPoint) int {
	// compare segments vertically at a.X and b.X < a.X
	// note that by may be infinite/large for fully/nearly vertical segments
	by := b.InterpolateY(a.X) // b's y at a's left
	if a.Y == by {
		return a.compareTangentsV(b)
	} else if a.Y < by {
		return -1
	} else {
		return 1
	}
}

func (a *SweepPoint) CompareV(b *SweepPoint) int {
	// used for sweep status, a is the point to be inserted / found
	if a.X == b.X {
		// left-point at same X
		if a.Y == b.Y {
			// left-point the same
			return a.compareTangentsV(b)
		} else if a.Y < b.Y {
			return -1
		} else {
			return 1
		}
	} else if a.X < b.X {
		// a starts to the left of b
		return -b.compareV(a)
	} else {
		// a starts to the right of b
		return a.compareV(b)
	}
}

//type SweepPointPair [2]*SweepPoint
//
//func (pair SweepPointPair) Swapped() SweepPointPair {
//	return SweepPointPair{pair[1], pair[0]}
//}

func addIntersections(zs []math32.Vector2, queue *SweepEvents, event *SweepPoint, prev, next *SweepNode) bool {
	// a and b are always left-endpoints and a is below b
	//pair := SweepPointPair{a, b}
	//if _, ok := handled[pair]; ok {
	//	return
	//} else if _, ok := handled[pair.Swapped()]; ok {
	//	return
	//}
	//handled[pair] = struct{}{}

	var a, b *SweepPoint
	if prev == nil {
		a, b = event, next.SweepPoint
	} else if next == nil {
		a, b = prev.SweepPoint, event
	} else {
		a, b = prev.SweepPoint, next.SweepPoint
	}

	// find all intersections between segment pair
	// this returns either no intersections, or one or more secant/tangent intersections,
	// or exactly two "same" intersections which occurs when the segments overlap.
	zs = intersectionLineLineBentleyOttmann(zs[:0], a.Vector2, a.other.Vector2, b.Vector2, b.other.Vector2)

	// no (valid) intersections
	if len(zs) == 0 {
		return false
	}

	// Non-vertical but downwards-sloped segments may become vertical upon intersection due to
	// floating-point rounding and limited precision. Only the first segment of b can ever become
	// vertical, never the first segment of a:
	// - a and b may be segments in status when processing a right-endpoint. The left-endpoints of
	//   both thus must be to the left of this right-endpoint (unless vertical) and can never
	//   become vertical in their first segment.
	// - a is the segment of the currently processed left-endpoint and b is in status and above it.
	//   a's left-endpoint is to the right of b's left-endpoint and is below b, thus:
	//   - a and b go upwards: a nor b may become vertical, no reversal
	//   - a goes downwards and b upwards: no intersection
	//   - a goes upwards and b downwards: only a may become vertical but no reversal
	//   - a and b go downwards: b may pass a's left-endpoint to its left (no intersection),
	//     through it (tangential intersection, no splitting), or to its right so that a never
	//     becomes vertical and thus no reversal
	// - b is the segment of the currently processed left-endpoint and a is in status and below it.
	//   a's left-endpoint is below or to the left of b's left-endpoint and a is below b, thus:
	//   - a and b go upwards: only a may become vertical, no reversal
	//   - a goes downwards and b upwards: no intersection
	//   - a goes upwards and b downwards: both may become vertical where only b must be reversed
	//   - a and b go downwards: if b passes through a's left-endpoint, it must become vertical and
	//     be reversed, or it passed to the right of a's left-endpoint and a nor b become vertical
	// Conclusion: either may become vertical, but only b ever needs reversal of direction. And
	// note that b is the currently processed left-endpoint and thus isn't in status.
	// Note: handle overlapping segments immediately by checking up and down status for segments
	// that compare equally with weak ordering (ie. overlapping).

	if !event.left {
		// intersection may be to the left (or below) the current event due to floating-point
		// precision which would interfere with the sequence in queue, this is a problem when
		// handling right-endpoints
		for i := range zs {
			zold := zs[i]
			z := &zs[i]
			if z.X < event.X {
				z.X = event.X
			} else if z.X == event.X && z.Y < event.Y {
				z.Y = event.Y
			}

			aMaxY := math32.Max(a.Y, a.other.Y)
			bMaxY := math32.Max(b.Y, b.other.Y)
			if a.other.X < z.X || b.other.X < z.X || aMaxY < z.Y || bMaxY < z.Y {
				fmt.Println("WARNING: intersection moved outside of segment:", zold, "=>", z)
			}
		}
	}

	// split segments a and b, but first find overlapping segments above and below and split them at the same point
	// this prevents a case that causes alternating intersections between overlapping segments and thus slowdown significantly
	//if a.node != nil {
	//	splitOverlappingAtIntersections(zs, queue, a, true)
	//}
	aChanged := splitAtIntersections(zs, queue, a, true)

	//if b.node != nil {
	//	splitOverlappingAtIntersections(zs, queue, b, false)
	//}
	bChanged := splitAtIntersections(zs, queue, b, false)
	return aChanged || bChanged
}

//func splitOverlappingAtIntersections(zs []Point, queue *SweepEvents, s *SweepPoint, isA bool) bool {
//	changed := false
//	for prev := s.node.Prev(); prev != nil; prev = prev.Prev() {
//		if prev.Point == s.Point && prev.other.Point == s.other.Point {
//			splitAtIntersections(zs, queue, prev.SweepPoint, isA)
//			changed = true
//		}
//	}
//	if !changed {
//		for next := s.node.Next(); next != nil; next = next.Next() {
//			if next.Point == s.Point && next.other.Point == s.other.Point {
//				splitAtIntersections(zs, queue, next.SweepPoint, isA)
//				changed = true
//			}
//		}
//	}
//	return changed
//}

func splitAtIntersections(zs []math32.Vector2, queue *SweepEvents, s *SweepPoint, isA bool) bool {
	changed := false
	for i := len(zs) - 1; 0 <= i; i-- {
		z := zs[i]
		if z == s.Vector2 || z == s.other.Vector2 {
			// ignore tangent intersections at the endpoints
			continue
		}

		// split segment at intersection
		right, left := s.SplitAt(z)

		// reverse direction if necessary
		if left.X == left.other.X {
			// segment after the split is vertical
			left.vertical, left.other.vertical = true, true
			if left.other.Y < left.Y {
				left.Reverse()
			}
		} else if right.X == right.other.X {
			// segment before the split is vertical
			right.vertical, right.other.vertical = true, true
			if right.Y < right.other.Y {
				// reverse first segment
				if isA {
					fmt.Println("WARNING: reversing first segment of A")
				}
				if right.other.node != nil {
					panic("impossible: first segment became vertical and needs reversal, but was already in the sweep status")
				}
				right.Reverse()

				// Note that we swap the content of the currently processed left-endpoint of b with
				// the new left-endpoint vertically below. The queue may not be strictly ordered
				// with other vertical segments at the new left-endpoint, but this isn't a problem
				// since we sort the events in each square after the Bentley-Ottmann phase.

				// update references from handled and queue by swapping their contents
				first := right.other
				*right, *first = *first, *right
				first.other, right.other = right, first
			}
		}

		// add to handled
		//handled[SweepPointPair{a, bLeft}] = struct{}{}
		//if aPrevLeft != a {
		//	// there is only one non-tangential intersection
		//	handled[SweepPointPair{aPrevLeft, bLeft}] = struct{}{}
		//}

		// add to queue
		queue.Push(right)
		queue.Push(left)
		changed = true
	}
	return changed
}

//func reorderStatus(queue *SweepEvents, event *SweepPoint, aOld, bOld *SweepNode) {
//	var aNew, bNew *SweepNode
//	var aMove, bMove int
//	if aOld != nil {
//		// a == prev is a node in status that needs to be reordered
//		aNew, aMove = aOld.fix()
//	}
//	if bOld != nil {
//		// b == next is a node in status that needs to be reordered
//		bNew, bMove = bOld.fix()
//	}
//
//	// find new intersections after snapping and moving around, first between the (new) neighbours
//	// of a and b, and then check if any other segment became adjacent due to moving around a or b,
//	// while avoiding superfluous checking for intersections (the aMove/bMove conditions)
//	if aNew != nil {
//		if prev := aNew.Prev(); prev != nil && aMove != bMove+1 {
//			// b is not a's previous
//			addIntersections(queue, event, prev, aNew)
//		}
//		if next := aNew.Next(); next != nil && aMove != bMove-1 {
//			// b is not a's next
//			addIntersections(queue, event, aNew, next)
//		}
//	}
//	if bNew != nil {
//		if prev := bNew.Prev(); prev != nil && bMove != aMove+1 {
//			// a is not b's previous
//			addIntersections(queue, event, prev, bNew)
//		}
//		if next := bNew.Next(); next != nil && bMove != aMove-1 {
//			// a is not b's next
//			addIntersections(queue, event, bNew, next)
//		}
//	}
//	if aOld != nil && aMove != 0 && bMove != -1 {
//		// a's old position is not aNew or bNew
//		if prev := aOld.Prev(); prev != nil && aMove != -1 && bMove != -2 {
//			// a nor b are not old a's previous
//			addIntersections(queue, event, prev, aOld)
//		}
//		if next := aOld.Next(); next != nil && aMove != 1 && bMove != 0 {
//			// a nor b are not old a's next
//			addIntersections(queue, event, aOld, next)
//		}
//	}
//	if bOld != nil && aMove != 1 && bMove != 0 {
//		// b's old position is not aNew or bNew
//		if aOld == nil {
//			if prev := bOld.Prev(); prev != nil && aMove != 0 && bMove != -1 {
//				// a nor b are not old b's previous
//				addIntersections(queue, event, prev, bOld)
//			}
//		}
//		if next := bOld.Next(); next != nil && aMove != 2 && bMove != 1 {
//			// a nor b are not old b's next
//			addIntersections(queue, event, bOld, next)
//		}
//	}
//}

type toleranceSquare struct {
	X, Y   float32       // snapped value
	Events []*SweepPoint // all events in this square

	// reference node inside or near the square
	// after breaking up segments, this is the previous node (ie. completely below the square)
	Node *SweepNode

	// lower and upper node crossing this square
	Lower, Upper *SweepNode
}

type toleranceSquares []*toleranceSquare

func (squares *toleranceSquares) find(x, y float32) (int, bool) {
	// find returns the index of the square at or above (x,y) (or len(squares) if above all)
	// the bool indicates if the square exists, otherwise insert a new square at that index
	for i := len(*squares) - 1; 0 <= i; i-- {
		if (*squares)[i].X < x || (*squares)[i].Y < y {
			return i + 1, false
		} else if (*squares)[i].Y == y {
			return i, true
		}
	}
	return 0, false
}

func (squares *toleranceSquares) Add(x float32, event *SweepPoint, refNode *SweepNode) {
	// refNode is always the node itself for left-endpoints, and otherwise the previous node (ie.
	// the node below) of a right-endpoint, or the next (ie. above) node if the previous is nil.
	// It may be inside or outside the right edge of the square. If outside, it is the first such
	// segment going upwards/downwards from the square (and not just any segment).
	y := snap(event.Y, BentleyOttmannEpsilon)
	if idx, ok := squares.find(x, y); !ok {
		// create new tolerance square
		square := boSquarePool.Get().(*toleranceSquare)
		*square = toleranceSquare{
			X:      x,
			Y:      y,
			Events: []*SweepPoint{event},
			Node:   refNode,
		}
		*squares = append((*squares)[:idx], append(toleranceSquares{square}, (*squares)[idx:]...)...)
	} else {
		// insert into existing tolerance square
		(*squares)[idx].Node = refNode
		(*squares)[idx].Events = append((*squares)[idx].Events, event)
	}

	// (nearly) vertical segments may still be used as the reference segment for squares around
	// in that case, replace with the new reference node (above or below that segment)
	if !event.left {
		orig := event.other.node
		for i := len(*squares) - 1; 0 <= i && (*squares)[i].X == x; i-- {
			if (*squares)[i].Node == orig {
				(*squares)[i].Node = refNode
			}
		}
	}
}

//func (event *SweepPoint) insertIntoSortedH(events *[]*SweepPoint) {
//	// O(log n)
//	lo, hi := 0, len(*events)
//	for lo < hi {
//		mid := (lo + hi) / 2
//		if (*events)[mid].LessH(event, false) {
//			lo = mid + 1
//		} else {
//			hi = mid
//		}
//	}
//
//	sorted := sort.IsSorted(eventSliceH(*events))
//	if !sorted {
//		fmt.Println("WARNING: H not sorted")
//		for i, event := range *events {
//			fmt.Println(i, event, event.Angle())
//		}
//	}
//	*events = append(*events, nil)
//	copy((*events)[lo+1:], (*events)[lo:])
//	(*events)[lo] = event
//	if sorted && !sort.IsSorted(eventSliceH(*events)) {
//		fmt.Println("ERROR: not sorted after inserting into events:", *events)
//	}
//}

func (event *SweepPoint) breakupSegment(events *[]*SweepPoint, index int, x, y float32) *SweepPoint {
	// break up a segment in two parts and let the middle point be (x,y)
	if snap(event.X, BentleyOttmannEpsilon) == x && snap(event.Y, BentleyOttmannEpsilon) == y || snap(event.other.X, BentleyOttmannEpsilon) == x && snap(event.other.Y, BentleyOttmannEpsilon) == y {
		// segment starts or ends in tolerance square, don't break up
		return event
	}

	// original segment should be kept in-place to not alter the queue or status
	r, l := event.SplitAt(math32.Vector2{x, y})
	r.index, l.index = index, index

	// reverse
	//if r.other.X == r.X {
	//	if l.other.Y < r.other.Y {
	//		r.Reverse()
	//	}
	//	r.vertical, r.other.vertical = true, true
	//} else if l.other.X == l.X {
	//	if l.other.Y < r.other.Y {
	//		l.Reverse()
	//	}
	//	l.vertical, l.other.vertical = true, true
	//}

	// update node reference
	if event.node != nil {
		l.node, event.node = event.node, nil
		l.node.SweepPoint = l
	}

	*events = append(*events, r, l)
	return l
}

func (squares toleranceSquares) breakupCrossingSegments(n int, x float32) {
	// find and break up all segments that cross this tolerance square
	// note that we must move up to find all upwards-sloped segments and then move down for the
	// downwards-sloped segments, since they may need to be broken up in other squares first
	x0, x1 := x-BentleyOttmannEpsilon/2.0, x+BentleyOttmannEpsilon/2.0

	// scan squares bottom to top
	for i := n; i < len(squares); i++ {
		square := squares[i] // pointer

		// be aware that a tolerance square is inclusive of the left and bottom edge
		// and only the bottom-left corner
		yTop, yBottom := square.Y+BentleyOttmannEpsilon/2.0, square.Y-BentleyOttmannEpsilon/2.0

		// from reference node find the previous/lower/upper segments for this square
		// the reference node may be any of the segments that cross the right-edge of the square,
		// or a segment below or above the right-edge of the square
		if square.Node != nil {
			y0, y1 := square.Node.ToleranceEdgeY(x0, x1)
			below, above := y0 < yBottom && y1 <= yBottom, yTop <= y0 && yTop <= y1
			if !below && !above {
				// reference node is inside the square
				square.Lower, square.Upper = square.Node, square.Node
			}

			// find upper node
			if !above {
				for next := square.Node.Next(); next != nil; next = next.Next() {
					y0, y1 := next.ToleranceEdgeY(x0, x1)
					if yTop <= y0 && yTop <= y1 {
						// above
						break
					} else if y0 < yBottom && y1 <= yBottom {
						// below
						square.Node = next
						continue
					}
					square.Upper = next
					if square.Lower == nil {
						// this is set if the reference node is below the square
						square.Lower = next
					}
				}
			}

			// find lower node and set reference node to the node completely below the square
			if !below {
				prev := square.Node.Prev()
				for ; prev != nil; prev = prev.Prev() {
					y0, y1 := prev.ToleranceEdgeY(x0, x1)
					if y0 < yBottom && y1 <= yBottom { // exclusive for bottom-right corner
						// below
						break
					} else if yTop <= y0 && yTop <= y1 {
						// above
						square.Node = prev
						continue
					}
					square.Lower = prev
					if square.Upper == nil {
						// this is set if the reference node is above the square
						square.Upper = prev
					}
				}
				square.Node = prev
			}
		}

		// find all segments that cross the tolerance square
		// first find all segments that extend to the right (they are in the sweepline status)
		if square.Lower != nil {
			for node := square.Lower; ; node = node.Next() {
				node.breakupSegment(&squares[i].Events, i, x, square.Y)
				if node == square.Upper {
					break
				}
			}
		}

		// then find which segments that end in this square go through other squares
		for _, event := range square.Events {
			if !event.left {
				y0, _ := event.ToleranceEdgeY(x0, x1)
				s := event.other
				if y0 < yBottom {
					// comes from below, find lowest square and breakup in each square
					j0 := i
					for j := i - 1; 0 <= j; j-- {
						if squares[j].X != x || squares[j].Y+BentleyOttmannEpsilon/2.0 <= y0 {
							break
						}
						j0 = j
					}
					for j := j0; j < i; j++ {
						s = s.breakupSegment(&squares[j].Events, j, x, squares[j].Y)
					}
				} else if yTop <= y0 {
					// comes from above, find highest square and breakup in each square
					j0 := i
					for j := i + 1; j < len(squares); j++ {
						if y0 < squares[j].Y-BentleyOttmannEpsilon/2.0 {
							break
						}
						j0 = j
					}
					for j := j0; i < j; j-- {
						s = s.breakupSegment(&squares[j].Events, j, x, squares[j].Y)
					}
				}
			}
		}
	}
}

type eventSliceV []*SweepPoint

func (a eventSliceV) Len() int {
	return len(a)
}

func (a eventSliceV) Less(i, j int) bool {
	return a[i].CompareV(a[j]) < 0
}

func (a eventSliceV) Swap(i, j int) {
	a[i].node.SweepPoint, a[j].node.SweepPoint = a[j], a[i]
	a[i].node, a[j].node = a[j].node, a[i].node
	a[i], a[j] = a[j], a[i]
}

type eventSliceH []*SweepPoint

func (a eventSliceH) Len() int {
	return len(a)
}

func (a eventSliceH) Less(i, j int) bool {
	return a[i].LessH(a[j])
}

func (a eventSliceH) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (cur *SweepPoint) computeSweepFields(prev *SweepPoint, op pathOp, fillRule FillRules) {
	// cur is left-endpoint
	if !cur.open {
		cur.selfWindings = 1
		if !cur.increasing {
			cur.selfWindings = -1
		}
	}

	// skip vertical segments
	cur.prev = prev
	for prev != nil && prev.vertical {
		prev = prev.prev
	}

	// compute windings
	if prev != nil {
		if cur.clipping == prev.clipping {
			cur.windings = prev.windings + prev.selfWindings
			cur.otherWindings = prev.otherWindings + prev.otherSelfWindings
		} else {
			cur.windings = prev.otherWindings + prev.otherSelfWindings
			cur.otherWindings = prev.windings + prev.selfWindings
		}
	} else {
		// may have been copied when intersected / broken up
		cur.windings, cur.otherWindings = 0, 0
	}

	cur.inResult = cur.InResult(op, fillRule)
	cur.other.inResult = cur.inResult
}

func (s *SweepPoint) InResult(op pathOp, fillRule FillRules) uint8 {
	lowerWindings, lowerOtherWindings := s.windings, s.otherWindings
	upperWindings, upperOtherWindings := s.windings+s.selfWindings, s.otherWindings+s.otherSelfWindings
	if s.clipping {
		lowerWindings, lowerOtherWindings = lowerOtherWindings, lowerWindings
		upperWindings, upperOtherWindings = upperOtherWindings, upperWindings
	}

	if s.open {
		// handle open paths on the subject
		switch op {
		case opSettle, opOR, opDIV:
			return 1
		case opAND:
			if fillRule.Fills(lowerOtherWindings) || fillRule.Fills(upperOtherWindings) {
				return 1
			}
		case opNOT, opXOR:
			if !fillRule.Fills(lowerOtherWindings) || !fillRule.Fills(upperOtherWindings) {
				return 1
			}
		}
		return 0
	}

	// lower/upper windings refers to subject path, otherWindings to clipping path
	var belowFills, aboveFills bool
	switch op {
	case opSettle:
		belowFills = fillRule.Fills(lowerWindings)
		aboveFills = fillRule.Fills(upperWindings)
	case opAND:
		belowFills = fillRule.Fills(lowerWindings) && fillRule.Fills(lowerOtherWindings)
		aboveFills = fillRule.Fills(upperWindings) && fillRule.Fills(upperOtherWindings)
	case opOR:
		belowFills = fillRule.Fills(lowerWindings) || fillRule.Fills(lowerOtherWindings)
		aboveFills = fillRule.Fills(upperWindings) || fillRule.Fills(upperOtherWindings)
	case opNOT:
		belowFills = fillRule.Fills(lowerWindings) && !fillRule.Fills(lowerOtherWindings)
		aboveFills = fillRule.Fills(upperWindings) && !fillRule.Fills(upperOtherWindings)
	case opXOR:
		belowFills = fillRule.Fills(lowerWindings) != fillRule.Fills(lowerOtherWindings)
		aboveFills = fillRule.Fills(upperWindings) != fillRule.Fills(upperOtherWindings)
	case opDIV:
		belowFills = fillRule.Fills(lowerWindings)
		aboveFills = fillRule.Fills(upperWindings)
		if belowFills && aboveFills {
			return 2
		} else if belowFills || aboveFills {
			return 1
		}
		return 0
	}

	// only keep edge if there is a change in filling between both sides
	if belowFills != aboveFills {
		return 1
	}
	return 0
}

func (s *SweepPoint) mergeOverlapping(op pathOp, fillRule FillRules) {
	// When merging overlapping segments, the order of the right-endpoints may have changed and
	// thus be different from the order used to compute the sweep fields, here we reset the values
	// for windings and otherWindings to be taken from the segment below (prev) which was updated
	// after snapping the endpoints.
	// We use event.overlapped to handle segments once and count windings once, in whichever order
	// the events are handled. We also update prev to reflect the segment below the overlapping
	// segments.
	if s.overlapped {
		// already handled
		return
	}
	prev := s.prev
	for ; prev != nil; prev = prev.prev {
		if prev.overlapped || s.Vector2 != prev.Vector2 || s.other.Vector2 != prev.other.Vector2 {
			break
		}

		// combine selfWindings
		if s.clipping == prev.clipping {
			s.selfWindings += prev.selfWindings
			s.otherSelfWindings += prev.otherSelfWindings
		} else {
			s.selfWindings += prev.otherSelfWindings
			s.otherSelfWindings += prev.selfWindings
		}
		prev.windings, prev.selfWindings, prev.otherWindings, prev.otherSelfWindings = 0, 0, 0, 0
		prev.inResult, prev.other.inResult = 0, 0
		prev.overlapped = true
	}
	if prev == s.prev {
		return
	}

	// compute merged windings
	if prev == nil {
		s.windings, s.otherWindings = 0, 0
	} else if s.clipping == prev.clipping {
		s.windings = prev.windings + prev.selfWindings
		s.otherWindings = prev.otherWindings + prev.otherSelfWindings
	} else {
		s.windings = prev.otherWindings + prev.otherSelfWindings
		s.otherWindings = prev.windings + prev.selfWindings
	}
	s.inResult = s.InResult(op, fillRule)
	s.other.inResult = s.inResult
	s.prev = prev
}

func bentleyOttmann(ps, qs Paths, op pathOp, fillRule FillRules) Path {
	// TODO: make public and add grid spacing argument
	// TODO: support OpDIV, keeping only subject, or both subject and clipping subpaths
	// TODO: add Intersects/Touches functions (return bool)
	// TODO: add Intersections function (return []Point)
	// TODO: support Cut to cut a path in subpaths between intersections (not polygons)
	// TODO: support elliptical arcs
	// TODO: use a red-black tree for the sweepline status?
	// TODO: use a red-black or 2-4 tree for the sweepline queue (LessH is 33% of time spent now),
	//       perhaps a red-black tree where the nodes are min-queues of the resulting squares
	// TODO: optimize path data by removing commands, set number of same command (50% less memory)
	// TODO: can we get idempotency (same result after second time) by tracing back each snapped
	//       right-endpoint for the squares it may now intersect? (Hershberger 2013)

	// Implementation of the Bentley-Ottmann algorithm by reducing the complexity of finding
	// intersections to O((n + k) log n), with n the number of segments and k the number of
	// intersections. All special cases are handled by use of:
	// - M. de Berg, et al., "Computational Geometry", Chapter 2, DOI: 10.1007/978-3-540-77974-2
	// - F. Martínez, et al., "A simple algorithm for Boolean operations on polygons", Advances in
	//   Engineering Software 64, p. 11-19, 2013, DOI: 10.1016/j.advengsoft.2013.04.004
	// - J. Hobby, "Practical segment intersection with ﬁnite precision output", Computational
	//   Geometry, 1997
	// - J. Hershberger, "Stable snap rounding", Computational Geometry: Theory and Applications,
	//   2013, DOI: 10.1016/j.comgeo.2012.02.011
	// - https://github.com/verven/contourklip

	// Bentley-Ottmann is the most popular algorithm to find path intersections, which is mainly
	// due to it's relative simplicity and the fact that it is (much) faster than the naive
	// approach. It however does not specify how special cases should be handled (overlapping
	// segments, multiple segment endpoints in one point, vertical segments), which is treated in
	// later works by other authors (e.g. Martínez from which this implementation draws
	// inspiration). I've made some small additions and adjustments to make it work in all cases
	// I encountered. Specifically, this implementation has the following properties:
	// - Subject and clipping paths may consist of any number of contours / subpaths.
	// - Any contour may be oriented clockwise (CW) or counter-clockwise (CCW).
	// - Any path or contour may self-intersect any number of times.
	// - Any point may be crossed multiple times by any path.
	// - Segments may overlap any number of times by any path.
	// - Segments may be vertical.
	// - The clipping path is implicitly closed, it makes no sense if it is an open path.
	// - The subject path is currently implicitly closed, but it is WIP to support open paths.
	// - Paths are currently flattened, but supporting Bézier or elliptical arcs is a WIP.

	// An unaddressed problem in those works is that of numerical accuracies. The main problem is
	// that calculating the intersections is not precise; the imprecision of the initial endpoints
	// of a path can be trivially fixed before the algorithm. Intersections however are calculated
	// during the algorithm and must be addressed. There are a few authors that propose a solution,
	// and Hobby's work inspired this implementation. The approach taken is somewhat different
	// though:
	// - Instead of integers (or rational numbers implemented using integers), floating points are
	//   used for their speed. It isn't even necessary that the grid points can be represented
	//   exactly in the floating point format, as long as all points in the tolerance square around
	//   the grid points snap to the same point. Now we can compare using == instead of an equality
	//   test.
	// - As in Martínez, we treat an intersection as a right- and left-endpoint combination and not
	//   as a third type of event. This avoids rearrangement of events in the sweep status as it is
	//   removed and reinserted into the right position, but at the cost of more delete/insert
	//   operations in the sweep status (potential to improve performance).
	// - As we run the Bentley-Ottmann algorithm, found endpoints must also be snapped to the grid.
	//   Since intersections are found in advance (ie. towards the right), we have no idea how the
	//   sweepline status will be yet, so we cannot snap those intersections to the grid yet. We
	//   must snap all endpoints/intersections when we reach them (ie. pop them off the queue).
	//   When we get to an endpoint, snap all endpoints in the tolerance square around the grid
	//   point to that point, and process all endpoints and intersections. Additionally, we should
	//   break-up all segments that pass through the square into two, and snap them to the grid
	//   point as well. These segments pass very close to another endpoint, and by snapping those
	//   to the grid we avoid the problem where we may or may not find that the segment intersects.
	// - Note that most (not all) intersections on the right are calculated with the left-endpoint
	//   already snapped, which may move the intersection to another grid point. These inaccuracies
	//   depend on the grid spacing and can be made small relative to the size of the input paths.
	//
	// The difference with Hobby's steps is that we advance Bentley-Ottmann for the entire column,
	// and only then do we calculate crossing segments. I'm not sure what reason Hobby has to do
	// this in two fases. Also, Hobby uses a shadow sweep line status structure which contains the
	// segments sorted after snapping. Instead of using two sweep status structures (the original
	// Bentley-Ottmann and the shadow with snapped segments), we sort the status after each column.
	// Additionally, we need to keep the sweep line queue structure ordered as well for the result
	// polygon (instead of the queue we gather the events for each sqaure, and sort those), and we
	// need to calculate the sweep fields for the result polygon.
	//
	// It is best to think of processing the tolerance squares, one at a time moving bottom-to-top,
	// for each column while moving the sweepline from left to right. Since all intersections
	// in this implementation are already converted to two right-endpoints and two left-endpoints,
	// we do all the snapping after each column and snapping the endpoints beforehand is not
	// necessary. We pop off all events from the queue that belong to the same column and process
	// them as we would with Bentley-Ottmann. This ensures that we find all original locations of
	// the intersections (except for intersections between segments in the sweep status structure
	// that are not yet adjacent, see note above) and may introduce new tolerance squares. For each
	// square, we find all segments that pass through and break them up and snap them to the grid.
	// Then snap all endpoints in the
	// square to the grid. We must sort the sweep line status and all events per square to account
	// for the new order after snapping. Some implementation observations:
	// - We must breakup segments that cross the square BEFORE we snap the square's endpoints,
	//   since we depend on the order of in the sweep status (from after processing the column
	//   using the original Bentley-Ottmann sweep line) for finding crossing segments.
	// - We find all original locations of intersections for adjacent segments during and after
	//   processing the column. However, if intersections become adjacent later on, the
	//   left-endpoint has already been snapped and the intersection has moved.
	// - We must be careful with overlapping segments. Since gridsnapping may introduce new
	//   overlapping segments (potentially vertical), we must check for that when processing the
	//   right-endpoints of each square.
	//
	// We thus proceed as follows:
	// - Process all events from left-to-right in a column using the regular Bentley-Ottmann.
	// - Identify all "hot" squares (those that contain endpoints / intersections).
	// - Find all segments that pass through each hot square, break them up and snap to the grid.
	//   These may be segments that start left of the column and end right of it, but also segments
	//   that start or end inside the column, or even start AND end inside the column (eg. vertical
	//   or almost vertical segments).
	// - Snap all endpoints and intersections to the grid.
	// - Compute sweep fields / windings for all new left-endpoints.
	// - Handle segments that are now overlapping for all right-endpoints.
	// Note that we must be careful with vertical segments.

	boInitPoolsOnce() // use pools for SweepPoint and SweepNode to amortize repeated calls to BO

	// return in case of one path is empty
	if op == opSettle {
		qs = nil
	} else if qs.Empty() {
		if op == opAND {
			return Path{}
		}
		return ps.Settle(fillRule)
	}
	if ps.Empty() {
		if qs != nil && (op == opOR || op == opXOR) {
			return qs.Settle(fillRule)
		}
		return Path{}
	}

	// ensure that X-monotone property holds for Béziers and arcs by breaking them up at their
	// extremes along X (ie. their inflection points along X)
	// TODO: handle Béziers and arc segments
	//p = p.XMonotone()
	//q = q.XMonotone()
	for i, iMax := 0, len(ps); i < iMax; i++ {
		split := ps[i].Split()
		if 1 < len(split) {
			ps[i] = split[0]
			ps = append(ps, split[1:]...)
		}
	}
	for i := range ps {
		ps[i] = ps[i].Flatten(Tolerance)
	}
	if qs != nil {
		for i, iMax := 0, len(qs); i < iMax; i++ {
			split := qs[i].Split()
			if 1 < len(split) {
				qs[i] = split[0]
				qs = append(qs, split[1:]...)
			}
		}
		for i := range qs {
			qs[i] = qs[i].Flatten(Tolerance)
		}
	}

	// check for path bounding boxes to overlap
	// TODO: cluster paths that overlap and treat non-overlapping clusters separately, this
	// makes the algorithm "more linear"
	R := Path{}
	var pOverlaps, qOverlaps []bool
	if qs != nil {
		pBounds := make([]math32.Box2, len(ps))
		qBounds := make([]math32.Box2, len(qs))
		for i := range ps {
			pBounds[i] = ps[i].FastBounds()
		}
		for i := range qs {
			qBounds[i] = qs[i].FastBounds()
		}
		pOverlaps = make([]bool, len(ps))
		qOverlaps = make([]bool, len(qs))
		for i := range ps {
			for j := range qs {
				if Touches(pBounds[i], qBounds[j]) {
					pOverlaps[i] = true
					qOverlaps[j] = true
				}
			}
			if !pOverlaps[i] && (op == opOR || op == opXOR || op == opNOT) {
				// path bounding boxes do not overlap, thus no intersections
				R = R.Append(ps[i].Settle(fillRule))
			}
		}
		for j := range qs {
			if !qOverlaps[j] && (op == opOR || op == opXOR) {
				// path bounding boxes do not overlap, thus no intersections
				R = R.Append(qs[j].Settle(fillRule))
			}
		}
	}

	// construct the priority queue of sweep events
	pSeg, qSeg := 0, 0
	queue := &SweepEvents{}
	for i := range ps {
		if qs == nil || pOverlaps[i] {
			pSeg = queue.AddPathEndpoints(ps[i], pSeg, false)
		}
	}
	if qs != nil {
		for i := range qs {
			if qOverlaps[i] {
				// implicitly close all subpaths on Q
				if !qs[i].Closed() {
					qs[i].Close()
				}
				qSeg = queue.AddPathEndpoints(qs[i], qSeg, true)
			}
		}
	}
	queue.Init() // sort from left to right

	// run sweep line left-to-right
	zs := make([]math32.Vector2, 0, 2) // buffer for intersections
	centre := &SweepPoint{}            // allocate here to reduce allocations
	events := []*SweepPoint{}          // buffer used for ordering status
	status := &SweepStatus{}           // contains only left events
	squares := toleranceSquares{}      // sorted vertically, squares and their events
	// TODO: use linked list for toleranceSquares?
	for 0 < len(*queue) {
		// TODO: skip or stop depending on operation if we're to the left/right of subject/clipping polygon

		// We slightly divert from the original Bentley-Ottmann and paper implementation. First
		// we find the top element in queue but do not pop it off yet. If it is a right-event, pop
		// from queue and proceed as usual, but if it's a left-event we first check (and add) all
		// surrounding intersections to the queue. This may change the order from which we should
		// pop off the queue, since intersections may create right-events, or new left-events that
		// are lower (by compareTangentV). If no intersections are found, pop off the queue and
		// proceed as usual.

		// Pass 1
		// process all events of the current column
		n := len(squares)
		x := snap(queue.Top().X, BentleyOttmannEpsilon)
	BentleyOttmannLoop:
		for 0 < len(*queue) && snap(queue.Top().X, BentleyOttmannEpsilon) == x {
			event := queue.Top()
			// TODO: breaking intersections into two right and two left endpoints is not the most
			// efficient. We could keep an intersection-type event and simply swap the order of the
			// segments in status (note there can be multiple segments crossing in one point). This
			// would alleviate a 2*m*log(n) search in status to remove/add the segments (m number
			// of intersections in one point, and n number of segments in status), and instead use
			// an m/2 number of swap operations. This alleviates pressure on the CompareV method.
			if !event.left {
				queue.Pop()

				n := event.other.node
				if n == nil {
					panic("right-endpoint not part of status, probably buggy intersection code")
					// don't put back in boPointPool, rare event
					continue
				} else if n.SweepPoint == nil {
					// this may happen if the left-endpoint is to the right of the right-endpoint
					// for some reason, usually due to a bug in the segment intersection code
					panic("other endpoint already removed, probably buggy intersection code")
					// don't put back in boPointPool, rare event
					continue
				}

				// find intersections between the now adjacent segments
				prev := n.Prev()
				next := n.Next()
				if prev != nil && next != nil {
					addIntersections(zs, queue, event, prev, next)
				}

				// add event to tolerance square
				if prev != nil {
					squares.Add(x, event, prev)
				} else {
					// next can be nil
					squares.Add(x, event, next)
				}

				// remove event from sweep status
				status.Remove(n)
			} else {
				// add intersections to queue
				prev, next := status.FindPrevNext(event)
				if prev != nil {
					addIntersections(zs, queue, event, prev, nil)
				}
				if next != nil {
					addIntersections(zs, queue, event, nil, next)
				}
				if queue.Top() != event {
					// check if the queue order was changed, this happens if the current event
					// is the left-endpoint of a segment that intersects with an existing segment
					// that goes below, or when two segments become fully overlapping, which sets
					// their order in status differently than when one of them extends further
					continue
				}
				queue.Pop()

				// add event to sweep status
				n := status.InsertAfter(prev, event)

				// add event to tolerance square
				squares.Add(x, event, n)
			}
		}

		// Pass 2
		// find all crossing segments, break them up and snap to the grid
		squares.breakupCrossingSegments(n, x)

		// snap events to grid
		// note that this may make segments overlapping from the left and towards the right
		// we handle the former below, but ignore the latter which may result in overlapping
		// segments not being strictly ordered
		for j := n; j < len(squares); j++ {
			del := 0
			square := squares[j] // pointer
			for i := 0; i < len(square.Events); i++ {
				event := square.Events[i]
				event.index = j
				event.X, event.Y = x, square.Y

				other := Gridsnap(event.other.Vector2, BentleyOttmannEpsilon)
				if event.Vector2 == other {
					// remove collapsed segments, we aggregate them with `del` to improve performance when we have many
					// TODO: prevent creating these segments in the first place
					del++
				} else {
					if 0 < del {
						for _, event := range square.Events[i-del : i] {
							if !event.left {
								boPointPool.Put(event.other)
								boPointPool.Put(event)
							}
						}
						square.Events = append(square.Events[:i-del], square.Events[i:]...)
						i -= del
						del = 0
					}
					if event.X == other.X {
						// correct for segments that have become vertical due to snap/breakup
						event.vertical, event.other.vertical = true, true
						if !event.left && event.Y < other.Y {
							// downward sloped, reverse direction
							event.Reverse()
						}
					}
				}
			}
			if 0 < del {
				for _, event := range square.Events[len(square.Events)-del:] {
					if !event.left {
						boPointPool.Put(event.other)
						boPointPool.Put(event)
					}
				}
				square.Events = square.Events[:len(square.Events)-del]
			}
		}

		for _, square := range squares[n:] {
			// reorder sweep status and events for result polygon
			// note that the number of events/nodes is usually small
			// and note that we must first snap all segments in this column before sorting
			if square.Lower != nil {
				events = events[:0]
				for n := square.Lower; ; n = n.Next() {
					events = append(events, n.SweepPoint)
					if n == square.Upper {
						break
					}
				}

				// TODO: test this thoroughly, this below prevents long loops of moving intersections to columns on the right
				for n := square.Lower; n != square.Upper; {
					next := n.Next()
					if 0 < n.CompareV(next.SweepPoint) {
						if next.other.X < n.other.X {
							r, l := n.SplitAt(next.other.Vector2)
							queue.Push(r)
							queue.Push(l)
						} else if n.other.X < next.other.X {
							r, l := next.SplitAt(n.other.Vector2)
							queue.Push(r)
							queue.Push(l)
						}
					}
					n = next
				}

				// keep unsorted events in the same slice
				n := len(events)
				events = append(events, events...)
				origEvents := events[n:]
				events = events[:n]

				sort.Sort(eventSliceV(events))

				// find intersections between neighbouring segments due to snapping
				// TODO: ugly!
				has := false
				centre.Vector2 = math32.Vector2{square.X, square.Y}
				if prev := square.Lower.Prev(); prev != nil {
					has = addIntersections(zs, queue, centre, prev, square.Lower)
				}
				if next := square.Upper.Next(); next != nil {
					has = has || addIntersections(zs, queue, centre, square.Upper, next)
				}

				// find intersections between new neighbours in status after sorting
				for i, event := range events[:len(events)-1] {
					if event != origEvents[i] {
						n := event.node
						var j int
						for origEvents[j] != event {
							j++
						}

						if next := n.Next(); next != nil && (j == 0 || next.SweepPoint != origEvents[j-1]) && (j+1 == len(origEvents) || next.SweepPoint != origEvents[j+1]) {
							// segment changed order and the segment above was not its neighbour
							has = has || addIntersections(zs, queue, centre, n, next)
						}
					}
				}

				if 0 < len(*queue) && snap(queue.Top().X, BentleyOttmannEpsilon) == x {
					//fmt.Println("WARNING: new intersections in this column!")
					goto BentleyOttmannLoop // TODO: is this correct? seems to work
					// TODO: almost parallel combined with overlapping segments may create many intersections considering order of
					//       of overlapping segments and snapping after each column
				} else if has {
					// sort overlapping segments again
					// this is needed when segments get cut and now become equal to the adjacent
					// overlapping segments
					// TODO: segments should be sorted by segment ID when overlapping, even if
					//       one segment extends further than the other, is that due to floating
					//       point accuracy?
					sort.Sort(eventSliceV(events))
				}
			}

			slices.SortFunc(square.Events, (*SweepPoint).CompareH)

			// compute sweep fields on left-endpoints
			for i, event := range square.Events {
				if !event.left {
					event.other.mergeOverlapping(op, fillRule)
				} else if event.node == nil {
					// vertical
					if 0 < i && square.Events[i-1].left {
						// against last left-endpoint in square
						// inside this square there are no crossing segments, they have been broken
						// up and have their left-endpoints sorted
						event.computeSweepFields(square.Events[i-1], op, fillRule)
					} else {
						// against first segment below square
						// square.Node may be nil
						var s *SweepPoint
						if square.Node != nil {
							s = square.Node.SweepPoint
						}
						event.computeSweepFields(s, op, fillRule)
					}
				} else {
					var s *SweepPoint
					if event.node.Prev() != nil {
						s = event.node.Prev().SweepPoint
					}
					event.computeSweepFields(s, op, fillRule)
				}
			}
		}
	}
	status.Clear() // release all nodes (but not SweepPoints)

	// build resulting polygons
	var Ropen Path
	for _, square := range squares {
		for _, cur := range square.Events {
			if cur.inResult == 0 {
				continue
			}

		BuildPath:
			windings := 0
			prev := cur.prev
			if op != opDIV && prev != nil {
				windings = prev.resultWindings
			}

			first := cur
			indexR := len(R)
			R.MoveTo(cur.X, cur.Y)
			cur.resultWindings = windings
			if !first.open {
				// we go to the right/top
				cur.resultWindings++
			}
			cur.other.resultWindings = cur.resultWindings
			for {
				// find segments starting from other endpoint, find the other segment amongst
				// them, the next segment should be the next going CCW
				i0 := 0
				nodes := squares[cur.other.index].Events
				for i := range nodes {
					if nodes[i] == cur.other {
						i0 = i
						break
					}
				}

				// find the next segment in CW order, this will make smaller subpaths
				// instead one large path when multiple segments end at the same position
				var next *SweepPoint
				for i := i0 - 1; ; i-- {
					if i < 0 {
						i += len(nodes)
					}
					if i == i0 {
						break
					} else if 0 < nodes[i].inResult && nodes[i].open == first.open {
						next = nodes[i]
						break
					}
				}
				if next == nil {
					if first.open {
						R.LineTo(cur.other.X, cur.other.Y)
					} else {
						// fmt.Println(ps)
						// fmt.Println(op)
						// fmt.Println(qs)
						// panic("next node for result polygon is nil, probably buggy intersection code")
					}
					break
				} else if next == first {
					break // contour is done
				}
				cur = next

				R.LineTo(cur.X, cur.Y)
				cur.resultWindings = windings
				if cur.left && !first.open {
					// we go to the right/top
					cur.resultWindings++
				}
				cur.other.resultWindings = cur.resultWindings
				cur.other.inResult--
				cur.inResult--
			}
			first.other.inResult--
			first.inResult--

			if first.open {
				if Ropen != nil {
					start := (R[indexR:]).Reverse()
					R = append(R[:indexR], start...)
					R = append(R, Ropen...)
					Ropen = nil
				} else {
					for _, cur2 := range square.Events {
						if 0 < cur2.inResult && cur2.open {
							cur = cur2
							Ropen = make(Path, len(R)-indexR-4)
							copy(Ropen, R[indexR+4:])
							R = R[:indexR]
							goto BuildPath
						}
					}
				}
			} else {
				R.Close()
				if windings%2 != 0 {
					// orient holes clockwise
					hole := R[indexR:].Reverse()
					R = append(R[:indexR], hole...)
				}
			}
		}

		for _, event := range square.Events {
			if !event.left {
				boPointPool.Put(event.other)
				boPointPool.Put(event)
			}
		}
		boSquarePool.Put(square)
	}
	return R
}
