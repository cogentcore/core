// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

import (
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// scaleShader computes Out[i] = In[i] * Params.Scale + Params.Offset + i
// for all of the indexes less than Params.N.
const scaleShader = `
struct Params {
	Scale: f32,
	Offset: f32,
	N: u32,
	pad: u32,
}

@group(0) @binding(0)
var<uniform> P: Params;

@group(1) @binding(0)
var<storage, read> In: array<f32>;

@group(1) @binding(1)
var<storage, read_write> Out: array<f32>;

@compute
@workgroup_size(64,1,1)
fn main(@builtin(workgroup_id) wgid: vec3<u32>, @builtin(num_workgroups) nwg: vec3<u32>, @builtin(local_invocation_index) loci: u32) {
	let idx = loci + (wgid.x + wgid.y * nwg.x + wgid.z * nwg.x * nwg.y) * 64;
	if (idx >= P.N) {
		return;
	}
	Out[idx] = In[idx] * P.Scale + P.Offset + f32(idx);
}
`

// scaleParams is the Params struct of [scaleShader].
type scaleParams struct {
	Scale  float32
	Offset float32
	N      uint32
	pad    uint32
}

// scaleSystem is a compute system that runs [scaleShader].
type scaleSystem struct {
	gp  *GPU
	sy  *ComputeSystem
	pl  *ComputePipeline
	prm *Var
	in  *Var
	out *Var
}

// newScaleSystem returns a new [scaleSystem] with the given number of
// values for each variable.
func newScaleSystem(t *testing.T, nvals int) *scaleSystem {
	ss := &scaleSystem{}
	ss.gp = NewComputeGPU()
	require.NotNil(t, ss.gp)
	ss.sy = NewComputeSystem(ss.gp, "scale")
	ss.pl = ss.sy.AddComputePipeline("scale")
	sh := ss.pl.AddShader("scale")
	require.NoError(t, sh.OpenCode(scaleShader))
	ss.pl.AddEntry(sh, ComputeShader, "main")

	vars := ss.sy.Vars()
	ugp := vars.AddGroup(Uniform, "Params")
	sgp := vars.AddGroup(Storage, "Data")
	ss.prm = ugp.AddStruct("P", 16, 1, ComputeShader)
	ss.in = sgp.Add("In", Float32, 1, ComputeShader)
	ss.in.ReadOnly = true
	ss.out = sgp.Add("Out", Float32, 1, ComputeShader)
	ugp.SetNValues(nvals)
	sgp.SetNValues(nvals)
	ss.sy.Config()
	return ss
}

// run runs the shader on n items, and returns the Out values.
func (ss *scaleSystem) run(t *testing.T, n int) []float32 {
	ce, err := ss.sy.BeginComputePass()
	require.NoError(t, err)
	require.NoError(t, ss.pl.Dispatch1D(ce, n, 64))
	ce.End()
	out := ss.out.Values.CurrentValue()
	require.NoError(t, out.GPUToRead(ss.sy.CommandEncoder))
	require.NoError(t, ss.sy.EndComputePass())
	require.NoError(t, out.ReadSync())
	res := make([]float32, n)
	require.NoError(t, ReadToBytes(out, res))
	return res
}

func (ss *scaleSystem) release() {
	ss.sy.Release()
	ss.gp.Release()
}

// TestComputeUniformStorage tests a compute shader with a uniform and
// read-only and read-write storage buffers, reading back the results.
func TestComputeUniformStorage(t *testing.T) {
	SkipIfNoGPU(t)
	ss := newScaleSystem(t, 1)
	defer ss.release()

	n := 1000 // not a multiple of the workgroup size
	in := make([]float32, n)
	for i := range in {
		in[i] = float32(i % 7)
	}
	require.NoError(t, SetValueFrom(ss.prm.Values.Values[0], []scaleParams{{Scale: 2, Offset: 0.5, N: uint32(n)}}))
	require.NoError(t, SetValueFrom(ss.in.Values.Values[0], in))
	require.NoError(t, SetValueFrom(ss.out.Values.Values[0], make([]float32, n)))

	res := ss.run(t, n)
	for i, v := range res {
		if !assert.Equal(t, in[i]*2+0.5+float32(i), v, "index %d", i) {
			break
		}
	}

	// a change to the uniform is used by the next run
	require.NoError(t, SetValueFrom(ss.prm.Values.Values[0], []scaleParams{{Scale: -1, Offset: 0, N: uint32(n)}}))
	res = ss.run(t, n)
	assert.Equal(t, -in[10]+10, res[10])
	assert.Equal(t, ss.gp, ss.sy.GPU())
	assert.NotNil(t, ss.sy.Device())
	assert.Nil(t, ss.sy.Render())
	assert.Contains(t, ss.gp.PropertiesString(), "GPU Properties")
}

// TestComputeCurrentValue tests switching between the values of the
// variables of a compute system.
func TestComputeCurrentValue(t *testing.T) {
	SkipIfNoGPU(t)
	ss := newScaleSystem(t, 2)
	defer ss.release()

	n := 64
	for vi := range 2 {
		in := make([]float32, n)
		for i := range in {
			in[i] = float32(vi + 1)
		}
		require.NoError(t, SetValueFrom(ss.prm.Values.Values[vi], []scaleParams{{Scale: 10, N: uint32(n)}}))
		require.NoError(t, SetValueFrom(ss.in.Values.Values[vi], in))
		require.NoError(t, SetValueFrom(ss.out.Values.Values[vi], make([]float32, n)))
	}

	vars := ss.sy.Vars()
	for vi := range 2 {
		_, err := vars.SetCurrentValue(1, "In", vi)
		require.NoError(t, err)
		_, err = vars.SetCurrentValue(1, "Out", vi)
		require.NoError(t, err)
		res := ss.run(t, n)
		assert.Equal(t, float32(vi+1)*10+3, res[3], "value %d", vi)
	}

	_, err := vars.SetCurrentValue(1, "In", 5)
	assert.Error(t, err, "out of range")
	_, err = vars.SetCurrentValue(1, "Nothing", 0)
	assert.Error(t, err)
	_, err = vars.ValueByIndex(1, "Out", 1)
	assert.NoError(t, err)
	_, err = vars.Group(7)
	assert.Error(t, err)
	assert.Equal(t, 2, vars.NGroups())
	assert.Contains(t, vars.StringDoc(), "Out")
}

// TestComputeValueGroups tests reading the values of a [ValueGroups]
// after a compute pass.
func TestComputeValueGroups(t *testing.T) {
	SkipIfNoGPU(t)
	ss := newScaleSystem(t, 1)
	defer ss.release()

	n := 128
	require.NoError(t, SetValueFrom(ss.prm.Values.Values[0], []scaleParams{{Scale: 0, Offset: 1, N: uint32(n)}}))
	require.NoError(t, SetValueFrom(ss.in.Values.Values[0], make([]float32, n)))
	require.NoError(t, SetValueFrom(ss.out.Values.Values[0], make([]float32, n)))

	var vgs ValueGroups
	vgs.Add("out", ss.out.Values.Values[0])
	_, err := vgs.ValuesByName("none")
	assert.Error(t, err)

	ce, err := ss.sy.BeginComputePass()
	require.NoError(t, err)
	require.NoError(t, ss.pl.Dispatch1D(ce, n, 64))
	ce.End()
	require.NoError(t, vgs.GPUToRead("out", ss.sy.CommandEncoder))
	require.NoError(t, ss.sy.EndComputePass())
	require.NoError(t, vgs.ReadSync("out"))
	res := make([]float32, n)
	require.NoError(t, ReadToBytes(ss.out.Values.Values[0], res))
	assert.Equal(t, float32(1+100), res[100])
	assert.Error(t, vgs.ReadSync("none"))
}

// TestVectorizeFunc tests running a kernel function on the CPU with
// threads.
func TestVectorizeFunc(t *testing.T) {
	for _, threads := range []int{0, 1, 3, 16} {
		var sum atomic.Int64
		seen := make([]int32, 100)
		VectorizeFunc(threads, 100, func(idx uint32) {
			sum.Add(int64(idx))
			atomic.AddInt32(&seen[idx], 1)
		})
		assert.Equal(t, int64(99*100/2), sum.Load(), "threads %d", threads)
		for i, s := range seen {
			assert.Equal(t, int32(1), s, "index %d threads %d", i, threads)
		}
	}
	NumThreads = 5
	assert.Equal(t, 5, DefaultNumThreads())
	NumThreads = 0
	assert.Greater(t, DefaultNumThreads(), 0)
}

// TestNumWorkgroups2D tests the number of workgroups in two dimensions.
func TestNumWorkgroups2D(t *testing.T) {
	nx, ny := NumWorkgroups2D(64*64, 8, 8)
	assert.Equal(t, 64, nx*ny)
	nx, ny = NumWorkgroups2D(65, 8, 8)
	assert.GreaterOrEqual(t, nx*ny*64, 65)
}
