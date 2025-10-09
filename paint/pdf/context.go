// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package pdf

import (
	"fmt"

	"cogentcore.org/core/math32"
	"cogentcore.org/core/styles"
)

// context holds the graphics state context to track the corresponding
// PDF state to optimize setting of styles.
type context struct {

	// Style is current style: copied from parent context initially.
	Style styles.Paint

	// Transform is the current transform that has been set.
	// it is not accumulated.
	Transform math32.Matrix2
}

func newContext(sty *styles.Paint, ctm math32.Matrix2) *context {
	c := &context{}
	c.Style = *sty
	c.Transform = ctm
	return c
}

// PushStack adds a graphics stack push (q), which must
// be paired with a corresponding Pop (Q).
func (w *pdfPage) PushStack() {
	ctx := w.stack.Peek()
	fmt.Fprintf(w, " q")
	w.stack.Push(newContext(&ctx.Style, ctx.Transform))
}

// PopStack adds a graphics stack pop (Q) which must
// be paired with a corresponding Push (q).
func (w *pdfPage) PopStack() {
	fmt.Fprintf(w, " Q")
	w.stack.Pop()
}

// SetTransform adds a cm to set the current matrix transform (CMT).
func (w *pdfPage) SetTransform(m math32.Matrix2) {
	rot := m.ExtractRot()
	m2 := m
	if rot != 0 {
		m2 = m.Mul(math32.Rotate2D(-2 * rot))
	}
	fmt.Fprintf(w, " %s cm", mat2(m2))
	ctx := w.stack.Peek()
	ctx.Transform = ctx.Transform.Mul(m2)
}

// PushTransform adds a graphics stack push (q) and then
// cm to set the current matrix transform (CMT).
func (w *pdfPage) PushTransform(m math32.Matrix2) {
	w.PushStack()
	w.SetTransform(m)
}

// style() returns the currently active style
func (w *pdfPage) style() *styles.Paint {
	ctx := w.stack.Peek()
	return &ctx.Style
}
