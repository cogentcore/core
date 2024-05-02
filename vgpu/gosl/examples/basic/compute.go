// Copyright (c) 2022, The Goki Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import "cogentcore.org/core/math32"

//gosl: hlsl basic
// #include "fastexp.hlsl"
//gosl: end basic

//gosl: start basic

// DataStruct has the test data
type DataStruct struct {

	// raw value
	Raw float32

	// integrated value
	Integ float32

	// exp of integ
	Exp float32

	// must pad to multiple of 4 floats for arrays
	Pad2 float32
}

// ParamStruct has the test params
type ParamStruct struct {

	// rate constant in msec
	Tau float32

	// 1/Tau
	Dt float32

	pad, pad1 float32
}

// IntegFromRaw computes integrated value from current raw value
func (ps *ParamStruct) IntegFromRaw(ds *DataStruct) {
	ds.Integ += ps.Dt * (ds.Raw - ds.Integ)
	ds.Exp = math32.FastExp(-ds.Integ)
}

//gosl: end basic

// note: only core compute code needs to be in shader -- all init is done CPU-side

func (ps *ParamStruct) Defaults() {
	ps.Tau = 5
	ps.Update()
}

func (ps *ParamStruct) Update() {
	ps.Dt = 1.0 / ps.Tau
}

//gosl: hlsl basic
/*
// // note: double-commented lines required here -- binding is var, set
[[vk::binding(0, 0)]] RWStructuredBuffer<ParamStruct> Params;
[[vk::binding(0, 1)]] RWStructuredBuffer<DataStruct> Data;

[numthreads(64, 1, 1)]

void main(uint3 idx : SV_DispatchThreadID) {
    Params[0].IntegFromRaw(Data[idx.x]);
}
*/
//gosl: end basic
