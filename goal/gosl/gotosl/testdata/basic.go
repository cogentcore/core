package test

import (
	"math"

	"cogentcore.org/core/goal/gosl/slbool"
	"cogentcore.org/core/math32"
)

//gosl:start

//gosl:vars
var (
	// Params are the parameters for the computation.
	//gosl:read-only
	Params []ParamStruct

	// Data is the data on which the computation operates.
	Data []DataStruct
)

// FastExp is a quartic spline approximation to the Exp function, by N.N. Schraudolph
// It does not have any of the sanity checking of a standard method -- returns
// nonsense when arg is out of range.  Runs in 2.23ns vs. 6.3ns for 64bit which is faster
// than math32.Exp actually.
func FastExp(x float32) float32 {
	if x <= -88.76731 { // this doesn't add anything and -exp is main use-case anyway
		return 0
	}
	i := int32(12102203*x) + int32(127)*(int32(1)<<23)
	m := i >> 7 & 0xFFFF // copy mantissa
	i += (((((((((((3537 * m) >> 16) + 13668) * m) >> 18) + 15817) * m) >> 14) - 80470) * m) >> 11)
	return math.Float32frombits(uint32(i))
}

// NeuronFlags are bit-flags encoding relevant binary state for neurons
type NeuronFlags int32

// The neuron flags
const (
	// NeuronOff flag indicates that this neuron has been turned off (i.e., lesioned)
	NeuronOff NeuronFlags = 0x01

	// NeuronHasExt means the neuron has external input in its Ext field
	NeuronHasExt NeuronFlags = 0x02 // note: 1<<2 does NOT work

	// NeuronHasTarg means the neuron has external target input in its Target field
	NeuronHasTarg NeuronFlags = 0x04

	// NeuronHasCmpr means the neuron has external comparison input in its Target field -- used for computing
	// comparison statistics but does not drive neural activity ever
	NeuronHasCmpr NeuronFlags = 0x08
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

	pad float32
}

// SubParamStruct has the test sub-params
type SubParamStruct struct {
	A, B, C, D float32
}

func (sp *SubParamStruct) Sum() float32 {
	return sp.A + sp.B + sp.C + sp.D
}

func (sp *SubParamStruct) SumPlus(extra float32) float32 {
	return sp.Sum() + extra
}

// ParamStruct has the test params
type ParamStruct struct {

	// rate constant in msec
	Tau float32

	// 1/Tau
	Dt     float32
	Option slbool.Bool // note: standard bool doesn't work

	pad float32 // comment this out to trigger alignment warning

	// extra parameters
	Subs SubParamStruct
}

func (ps *ParamStruct) IntegFromRaw(ds *DataStruct) float32 {
	// note: the following are just to test basic control structures
	newVal := ps.Dt * (ds.Raw - ds.Integ)
	if newVal < -10 || ps.Option.IsTrue() {
		newVal = -10
	}
	ds.Integ += newVal
	ds.Exp = math32.Exp(-ds.Integ)
	var a float32
	ps.AnotherMeth(ds, &a)
	return ds.Exp
}

// AnotherMeth does more computation
func (ps *ParamStruct) AnotherMeth(ds *DataStruct, ptrarg *float32) {
	for i := 0; i < 10; i++ {
		ds.Integ *= 0.99
	}
	var flag NeuronFlags
	flag &^= NeuronHasExt // clear flag -- op doesn't exist in C

	mode := Test
	switch mode { // note: no fallthrough!
	case Test:
		ab := float32(42)
		ds.Exp /= ab
	case Train:
		ab := float32(.5)
		ds.Exp *= ab
	default:
		ab := float32(1)
		ds.Exp *= ab
	}

	var a, b float32
	b = 42
	a = ps.Subs.Sum()
	ds.Exp = ps.Subs.SumPlus(b)
	ds.Integ = a

	*ptrarg = -1
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

func (ps *ParamStruct) String() string {
	return "params!"
}

//gosl:start

// Compute does the main computation
func Compute(i uint32) { //gosl:kernel
	data := GetData(i)
	Params[0].IntegFromRaw(data)
}

//gosl:end
