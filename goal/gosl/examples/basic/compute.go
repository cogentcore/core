// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"cogentcore.org/core/math32"
	"cogentcore.org/core/tensor"
)

//gosl:start
//gosl:import "cogentcore.org/core/math32"

//gosl:vars
var (
	// Params are the parameters for the computation.
	//gosl:read-only
	Params []ParamStruct

	// Data is the data on which the computation operates.
	// 2D: outer index is data, inner index is: Raw, Integ, Exp vars.
	//gosl:dims 2
	Data tensor.Float32
)

const (
	Raw int = iota
	Integ
	Exp
)

// ParamStruct has the test params
type ParamStruct struct {

	// rate constant in msec
	Tau float32

	// 1/Tau
	Dt float32

	pad  float32
	pad1 float32
}

// IntegFromRaw computes integrated value from current raw value
func (ps *ParamStruct) IntegFromRaw(idx int) {
	integ := Data.Value(idx, Integ)
	integ += ps.Dt * (Data.Value(idx, Raw) - integ)
	Data.Set(integ, idx, Integ)
	Data.Set(math32.FastExp(-integ), idx, Exp)
}

// Compute does the main computation
func Compute(i uint32) { //gosl:kernel
	Params[0].IntegFromRaw(int(i))
}

//gosl:end

// note: only core compute code needs to be in shader -- all init is done CPU-side

func (ps *ParamStruct) Defaults() {
	ps.Tau = 5
	ps.Update()
}

func (ps *ParamStruct) Update() {
	ps.Dt = 1.0 / ps.Tau
}
