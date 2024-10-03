// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import "cogentcore.org/core/math32"

//gosl:start
//gosl:import "cogentcore.org/core/math32"

//gosl:vars
var (
	// Params are the parameters for the computation.
	Params []ParamStruct

	// Data is the data on which the computation operates.
	Data []DataStruct
)

// DataStruct has the test data
type DataStruct struct {

	// raw value
	Raw float32

	// integrated value
	Integ float32

	// exp of integ
	Exp float32

	// must pad to multiple of 4 floats for arrays
	pad float32
}

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
func (ps *ParamStruct) IntegFromRaw(ds *DataStruct) {
	ds.Integ += ps.Dt * (ds.Raw - ds.Integ)
	ds.Exp = math32.FastExp(-ds.Integ)
}

// Compute does the main computation
func Compute(i int32) { //gosl:kernel
	Params[0].IntegFromRaw(&Data[i])
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
