package main

import (
	"fmt"

	"cogentcore.org/core/goal/gosl/slrand"
	"cogentcore.org/core/goal/gosl/sltype"
	"cogentcore.org/core/math32"
)

//gosl:start

//gosl:vars
var (
	//gosl:read-only
	Seed []Seeds

	// Data
	Data []Rnds
)

type Seeds struct {
	Seed      uint64
	pad, pad1 int32
}

type Rnds struct {
	Uints      sltype.Uint32Vec2
	pad, pad1  int32
	Floats     sltype.Float32Vec2
	pad2, pad3 int32
	Floats11   sltype.Float32Vec2
	pad4, pad5 int32
	Gauss      sltype.Float32Vec2
	pad6, pad7 int32
}

// RndGen calls random function calls to test generator.
// Note that the counter to the outer-most computation function
// is passed by *value*, so the same counter goes to each element
// as it is computed, but within this scope, counter is passed by
// reference (as a pointer) so subsequent calls get a new counter value.
// The counter should be incremented by the number of random calls
// outside of the overall update function.
func (r *Rnds) RndGen(counter uint64, idx uint32) {
	r.Uints = slrand.Uint32Vec2(counter, uint32(0), idx)
	r.Floats = slrand.Float32Vec2(counter, uint32(1), idx)
	r.Floats11 = slrand.Float32Range11Vec2(counter, uint32(2), idx)
	r.Gauss = slrand.Float32NormVec2(counter, uint32(3), idx)
}

func Compute(i uint32) { //gosl:kernel
	Data[i].RndGen(Seed[0].Seed, i)
}

//gosl:end

const Tol = 1.0e-4 // fails at lower tol eventually -- -6 works for many

func FloatSame(f1, f2 float32) (exact, tol bool) {
	exact = f1 == f2
	tol = math32.Abs(f1-f2) < Tol
	return
}

func Float32Vec2Same(f1, f2 sltype.Float32Vec2) (exact, tol bool) {
	e1, t1 := FloatSame(f1.X, f2.X)
	e2, t2 := FloatSame(f1.Y, f2.Y)
	exact = e1 && e2
	tol = t1 && t2
	return
}

// IsSame compares values at two levels: exact and with Tol
func (r *Rnds) IsSame(o *Rnds) (exact, tol bool) {
	e1 := r.Uints == o.Uints
	e2, t2 := Float32Vec2Same(r.Floats, o.Floats)
	e3, t3 := Float32Vec2Same(r.Floats11, o.Floats11)
	_, t4 := Float32Vec2Same(r.Gauss, o.Gauss)
	exact = e1 && e2 && e3 // skip e4 -- know it isn't
	tol = t2 && t3 && t4
	return
}

func (r *Rnds) String() string {
	return fmt.Sprintf("U: %x\t%x\tF: %g\t%g\tF11: %g\t%g\tG: %g\t%g", r.Uints.X, r.Uints.Y, r.Floats.X, r.Floats.Y, r.Floats11.X, r.Floats11.Y, r.Gauss.X, r.Gauss.Y)
}
