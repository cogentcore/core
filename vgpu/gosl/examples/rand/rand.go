package main

import (
	"fmt"

	"cogentcore.org/core/math32"
	"github.com/emer/gosl/v2/slrand"
	"github.com/emer/gosl/v2/sltype"
)

//gosl: hlsl rand
// #include "slrand.hlsl"
//gosl: end rand

//gosl: start rand

type Rnds struct {
	Uints      sltype.Uint2
	pad, pad1  int32
	Floats     sltype.Float2
	pad2, pad3 int32
	Floats11   sltype.Float2
	pad4, pad5 int32
	Gauss      sltype.Float2
	pad6, pad7 int32
}

// RndGen calls random function calls to test generator.
// Note that the counter to the outer-most computation function
// is passed by *value*, so the same counter goes to each element
// as it is computed, but within this scope, counter is passed by
// reference (as a pointer) so subsequent calls get a new counter value.
// The counter should be incremented by the number of random calls
// outside of the overall update function.
func (r *Rnds) RndGen(counter sltype.Uint2, idx uint32) {
	r.Uints = slrand.Uint2(&counter, idx)
	r.Floats = slrand.Float2(&counter, idx)
	r.Floats11 = slrand.Float112(&counter, idx)
	r.Gauss = slrand.NormFloat2(&counter, idx)
}

//gosl: end rand

const Tol = 1.0e-4 // fails at lower tol eventually -- -6 works for many

func FloatSame(f1, f2 float32) (exact, tol bool) {
	exact = f1 == f2
	tol = math32.Abs(f1-f2) < Tol
	return
}

func Float2Same(f1, f2 sltype.Float2) (exact, tol bool) {
	e1, t1 := FloatSame(f1.X, f2.X)
	e2, t2 := FloatSame(f1.Y, f2.Y)
	exact = e1 && e2
	tol = t1 && t2
	return
}

// IsSame compares values at two levels: exact and with Tol
func (r *Rnds) IsSame(o *Rnds) (exact, tol bool) {
	e1 := r.Uints == o.Uints
	e2, t2 := Float2Same(r.Floats, o.Floats)
	e3, t3 := Float2Same(r.Floats11, o.Floats11)
	_, t4 := Float2Same(r.Gauss, o.Gauss)
	exact = e1 && e2 && e3 // skip e4 -- know it isn't
	tol = t2 && t3 && t4
	return
}

func (r *Rnds) String() string {
	return fmt.Sprintf("U: %x\t%x\tF: %g\t%g\tF11: %g\t%g\tG: %g\t%g", r.Uints.X, r.Uints.Y, r.Floats.X, r.Floats.Y, r.Floats11.X, r.Floats11.Y, r.Gauss.X, r.Gauss.Y)
}
