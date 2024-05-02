package test

import (
	"math"

	"cogentcore.org/core/math32"
	"github.com/emer/gosl/v2/slbool"
)

//gosl: nohlsl basic

// note: this code is included in the go pre-processing output but
// then removed from the final hlsl output.
// Use when you need different versions of the same function for CPU vs. GPU

// MyTrickyFun this is the CPU version of the tricky function
func MyTrickyFun(x float32) float32 {
	return 10 // ok actually not tricky here, but whatever
}

//gosl: end basic

//gosl: hlsl basic

// // note: here is the hlsl version, only included in hlsl

// // MyTrickyFun this is the GPU version of the tricky function
// float MyTrickyFun(float x) {
// 	return 16; // ok actually not tricky here, but whatever
// }

//gosl: end basic

//gosl: start basic

// FastExp is a quartic spline approximation to the Exp function, by N.N. Schraudolph
// It does not have any of the sanity checking of a standard method -- returns
// nonsense when arg is out of range.  Runs in 2.23ns vs. 6.3ns for 64bit which is faster
// than math32.Exp actually.
func FastExp(x float32) float32 {
	if x <= -88.76731 { // this doesn't add anything and -exp is main use-case anyway
		return 0
	}
	i := int32(12102203*x) + 127*(1<<23)
	m := i >> 7 & 0xFFFF // copy mantissa
	i += (((((((((((3537 * m) >> 16) + 13668) * m) >> 18) + 15817) * m) >> 14) - 80470) * m) >> 11)
	return math.Float32frombits(uint32(i))
}

// NeuronFlags are bit-flags encoding relevant binary state for neurons
type NeuronFlags int32

// The neuron flags
const (
	// NeuronOff flag indicates that this neuron has been turned off (i.e., lesioned)
	NeuronOff NeuronFlags = 1

	// NeuronHasExt means the neuron has external input in its Ext field
	NeuronHasExt NeuronFlags = 1 << 2

	// NeuronHasTarg means the neuron has external target input in its Target field
	NeuronHasTarg NeuronFlags = 1 << 3

	// NeuronHasCmpr means the neuron has external comparison input in its Target field -- used for computing
	// comparison statistics but does not drive neural activity ever
	NeuronHasCmpr NeuronFlags = 1 << 4
)

// Modes are evaluation modes (Training, Testing, etc)
type Modes int32

// The evaluation modes
const (
	NoEvalMode Modes = iota

	// AllModes indicates that the log should occur over all modes present in other items.
	AllModes

	// Train is this a training mode for the env
	Train

	// Test is this a test mode for the env
	Test
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
	Pad2 float32
}

// ParamStruct has the test params
type ParamStruct struct {

	// rate constant in msec
	Tau float32

	// 1/Tau
	Dt     float32
	Option slbool.Bool // note: standard bool doesn't work

	pad float32 // comment this out to trigger alignment warning
}

func (ps *ParamStruct) IntegFromRaw(ds *DataStruct, modArg *float32) {
	// note: the following are just to test basic control structures
	newVal := ps.Dt*(ds.Raw-ds.Integ) + *modArg
	if newVal < -10 || ps.Option.IsTrue() {
		newVal = -10
	}
	ds.Integ += newVal
	ds.Exp = math32.Exp(-ds.Integ)
}

// AnotherMeth does more computation
func (ps *ParamStruct) AnotherMeth(ds *DataStruct) {
	for i := 0; i < 10; i++ {
		ds.Integ *= 0.99
	}
	var flag NeuronFlags
	flag &^= NeuronHasExt // clear flag -- op doesn't exist in C

	mode := Test
	switch mode {
	case Test:
		fallthrough
	case Train:
		ab := float32(.5)
		ds.Exp *= ab
	default:
		ab := float32(1)
		ds.Exp *= ab
	}
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
[[vk::binding(0, 0)]] StructuredBuffer<ParamStruct> Params;
[[vk::binding(0, 1)]] RWStructuredBuffer<DataStruct> Data;
[numthreads(1, 1, 1)]
void main(uint3 idx : SV_DispatchThreadID) {
    Params[0].IntegFromRaw(Data[idx.x], Data[idx.x].Pad2);
}
*/
//gosl: end basic
