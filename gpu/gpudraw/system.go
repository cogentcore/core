// Copyright 2024 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpudraw

import (
	"embed"
	"fmt"
	"image"
	"image/draw"
	"unsafe"

	"cogentcore.org/core/base/errors"
	"cogentcore.org/core/base/slicesx"
	"cogentcore.org/core/gpu"
	"cogentcore.org/core/gpu/drawmatrix"
	"github.com/cogentcore/webgpu/wgpu"
)

//go:embed shaders/*.wgsl
var shaders embed.FS

// ConfigPipeline configures graphics settings on the pipeline
func (dw *Drawer) ConfigPipeline(pl *gpu.GraphicsPipeline, blend bool) {
	pl.SetGraphicsDefaults()
	pl.SetCullMode(wgpu.CullModeBack)
	pl.SetFrontFace(wgpu.FrontFaceCCW)
	pl.SetAlphaBlend(blend)
}

// configSystem configures GPUDraw sytem
func (dw *Drawer) configSystem(gp *gpu.GPU, rd gpu.Renderer) {
	dw.opList = slicesx.SetLength(dw.opList, AllocChunk) // allocate
	dw.opList = dw.opList[:0]
	dw.images.init()

	dw.System = gpu.NewGraphicsSystem(gp, "gpudraw", rd)
	sy := dw.System

	// note: requires different pipelines for src vs. over draw op modes
	dopl := sy.AddGraphicsPipeline("drawover")
	dw.ConfigPipeline(dopl, true)

	dspl := sy.AddGraphicsPipeline("drawsrc")
	dw.ConfigPipeline(dspl, false)

	fopl := sy.AddGraphicsPipeline("fillover")
	dw.ConfigPipeline(fopl, true)

	fspl := sy.AddGraphicsPipeline("fillsrc")
	dw.ConfigPipeline(fspl, false)

	sh := dopl.AddShader("draw")
	sh.OpenFileFS(shaders, "shaders/draw.wgsl")
	dopl.AddEntry(sh, gpu.VertexShader, "vs_main")
	dopl.AddEntry(sh, gpu.FragmentShader, "fs_main")

	sh = dspl.AddShader("draw")
	sh.OpenFileFS(shaders, "shaders/draw.wgsl")
	dspl.AddEntry(sh, gpu.VertexShader, "vs_main")
	dspl.AddEntry(sh, gpu.FragmentShader, "fs_main")

	sh = fopl.AddShader("fill")
	sh.OpenFileFS(shaders, "shaders/fill.wgsl")
	fopl.AddEntry(sh, gpu.VertexShader, "vs_main")
	fopl.AddEntry(sh, gpu.FragmentShader, "fs_main")

	sh = fspl.AddShader("fill")
	sh.OpenFileFS(shaders, "shaders/fill.wgsl")
	fspl.AddEntry(sh, gpu.VertexShader, "vs_main")
	fspl.AddEntry(sh, gpu.FragmentShader, "fs_main")

	vgp := sy.Vars().AddVertexGroup()
	mgp := sy.Vars().AddGroup(gpu.Uniform, "Matrix")         // 0
	tgp := sy.Vars().AddGroup(gpu.SampledTexture, "Texture") // 1

	posv := vgp.Add("Pos", gpu.Float32Vector2, 0, gpu.VertexShader)
	idxv := vgp.Add("Index", gpu.Uint16, 0, gpu.VertexShader)
	idxv.Role = gpu.Index

	mv := mgp.AddStruct("Matrix", int(unsafe.Sizeof(drawmatrix.Matrix{})), 1, gpu.VertexShader, gpu.FragmentShader)
	mv.DynamicOffset = true

	tgp.Add("TexSampler", gpu.TextureRGBA32, 1, gpu.FragmentShader)

	vgp.SetNValues(1)
	mgp.SetNValues(1)
	tgp.SetNValues(AllocChunk)

	sy.Config()

	rectPos := posv.Values.Values[0]
	gpu.SetValueFrom(rectPos, []float32{
		0.0, 0.0,
		0.0, 1.0,
		1.0, 0.0,
		1.0, 1.0})

	rectIndex := idxv.Values.Values[0]
	gpu.SetValueFrom(rectIndex, []uint16{0, 1, 2, 2, 1, 3})

	vl := errors.Log1(sy.Vars().ValueByIndex(0, "Matrix", 0))
	vl.SetDynamicN(AllocChunk)

	// need a dummy texture in case only using fill
	dimg := image.NewRGBA(image.Rectangle{Max: image.Point{2, 2}})
	img := errors.Log1(tgp.ValueByIndex("TexSampler", 0))
	img.SetFromGoImage(dimg, 0)
}

// memoryFinalizer is a companion of the RenderPassEncoder.
type memoryFinalizer []func() error

func (o memoryFinalizer) AddFinalizer(finalizers ...func() error) memoryFinalizer {
	if len(finalizers) == 0 {
		return o
	}

	return append(o, finalizers...)
}

func (o memoryFinalizer) Finalize() error {
	var errs []error

	for _, finalize := range o {
		err := finalize()
		if err != nil {
			errs = append(errs, err)
		}
	}

	if err := errors.Join(errs...); err != nil {
		return fmt.Errorf("render pass encoder finalization failed: %w", err)
	}

	return nil
}

func (dw *Drawer) drawAll() error {
	sy := dw.System

	vars := sy.Vars()
	vl := errors.Log1(vars.ValueByIndex(0, "Matrix", 0))
	vl.WriteDynamicBuffer()

	mvr := errors.Log1(vars.VarByName(0, "Matrix"))
	mvl := mvr.Values.Values[0]
	tvr := errors.Log1(vars.VarByName(1, "TexSampler"))
	tvr.SetCurrentValue(0)

	rp, err := sy.BeginRenderPass() // NoClear() // TODO: NoClear not working!
	if errors.Log(err) != nil {
		return err
	}

	// finalizers is a collection of postponed finalizers for objects allocated in CGO world.
	var finalizers memoryFinalizer

	imgIdx := 0
	lastOp := draw.Op(-1)
	_ = lastOp
	for i, op := range dw.opList {
		var pl *gpu.GraphicsPipeline
		switch op {
		case draw.Over:
			pl = sy.GraphicsPipelines["drawover"]
		case draw.Src:
			pl = sy.GraphicsPipelines["drawsrc"]
		case fillOver:
			pl = sy.GraphicsPipelines["fillover"]
		case fillSrc:
			pl = sy.GraphicsPipelines["fillsrc"]
		}

		mvl.DynamicIndex = i
		if op < fillOver {
			tvr.SetCurrentValue(dw.images.used[imgIdx].index)
			imgIdx++
		}

		if op != lastOp {
			pl.BindPipeline(rp)
			lastOp = op
		} else {
			pl.BindAllGroups(rp)
		}

		pl.BindDrawIndexed(rp)

		// we should regularly clean up pipelines.
		finalizers = finalizers.AddFinalizer(pl.DrainFinalizers()...)
	}

	rp.End()
	sy.EndRenderPass(rp)

	// we should actually run finalizers when everything new is up and running on GPU.
	if err := finalizers.Finalize(); err != nil {
		return fmt.Errorf("Drawer.drawAll: finalizer failed: %w", err)
	}

	return nil
}
