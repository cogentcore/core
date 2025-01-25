// Copyright (c) 2025, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/linebender/vello/blob/main/vello/src/lib.rs

// Package rasterg provides a GPU-based rasterizer based on vello.
package rasterg

import (
	"errors"
	"image/color"
)

// AaConfig represents the anti-aliasing method to use during a render pass.
type AaConfig int

const (
	// Area anti-aliasing
	Area AaConfig = iota
	// 8x Multisampling
	Msaa8
	// 16x Multisampling
	Msaa16
)

// AaSupport represents the set of anti-aliasing configurations to enable during pipeline creation.
type AaSupport struct {
	Area   bool
	Msaa8  bool
	Msaa16 bool
}

// All returns an AaSupport with all anti-aliasing methods enabled.
func (a *AaSupport) All() *AaSupport {
	return &AaSupport{
		Area:   true,
		Msaa8:  true,
		Msaa16: true,
	}
}

// AreaOnly returns an AaSupport with only Area anti-aliasing enabled.
func (a *AaSupport) AreaOnly() *AaSupport {
	return &AaSupport{
		Area:   true,
		Msaa8:  false,
		Msaa16: false,
	}
}

// RenderParams represents parameters used in a single render that are configurable by the client.
type RenderParams struct {
	BaseColor          color.Color
	Width, Height      uint32
	AntialiasingMethod AaConfig
}

// RendererOptions represents options which are set at renderer creation time.
type RendererOptions struct {
	SurfaceFormat       *TextureFormat
	UseCPU              bool
	AntialiasingSupport AaSupport
	NumInitThreads      *int
}

// Renderer represents a renderer for rendering scenes.
type Renderer struct {
	Options  RendererOptions
	Engine   WgpuEngine
	Resolver Resolver
	Shaders  FullShaders
	Blit     *BlitPipeline
	Debug    *DebugRenderer
	Target   *TargetTexture
	Profiler *GpuProfiler
}

// NewRenderer creates a new renderer for the specified device.
func NewRenderer(device *Device, options RendererOptions) (*Renderer, error) {
	engine := NewWgpuEngine(options.UseCPU)
	if options.NumInitThreads != nil && *options.NumInitThreads != 1 {
		engine.UseParallelInitialisation()
	}
	shaders, err := FullShaders(device, &engine, &options)
	if err != nil {
		return nil, err
	}
	var blit *BlitPipeline
	if options.SurfaceFormat != nil {
		blit, err = NewBlitPipeline(device, *options.SurfaceFormat, &engine)
		if err != nil {
			return nil, err
		}
	}
	var debug *DebugRenderer
	if options.SurfaceFormat != nil {
		debug = NewDebugRenderer(device, *options.SurfaceFormat, &engine)
	}
	return &Renderer{
		Options:  options,
		Engine:   engine,
		Resolver: NewResolver(),
		Shaders:  shaders,
		Blit:     blit,
		Debug:    debug,
		Target:   nil,
		Profiler: NewGpuProfiler(),
	}, nil
}

// RenderToTexture renders a scene to the target texture.
func (r *Renderer) RenderToTexture(device *Device, queue *Queue, scene *Scene, texture *TextureView, params *RenderParams) error {
	recording, target := RenderFull(scene, &r.Resolver, &r.Shaders, params)
	externalResources := []ExternalResource{NewExternalResourceImage(*target.AsImage(), texture)}
	return r.Engine.RunRecording(device, queue, &recording, externalResources, "render_to_texture", r.Profiler)
}

// RenderToSurface renders a scene to the target surface.
func (r *Renderer) RenderToSurface(device *Device, queue *Queue, scene *Scene, surface *SurfaceTexture, params *RenderParams) error {
	width := params.Width
	height := params.Height
	target := r.Target
	if target == nil || target.Width != width || target.Height != height {
		target = NewTargetTexture(device, width, height)
	}
	if err := r.RenderToTexture(device, queue, scene, &target.View, params); err != nil {
		return err
	}
	blit := r.Blit
	if blit == nil {
		return errors.New("renderer should have configured surface_format to use on a surface")
	}
	recording := NewRecording()
	targetProxy := NewImageProxy(width, height, ImageFormatFromWgpu(target.Format))
	surfaceProxy := NewImageProxy(width, height, ImageFormatFromWgpu(surface.Texture.Format()))
	recording.Draw(DrawParams{
		ShaderID:      blit.ID,
		InstanceCount: 1,
		VertexCount:   6,
		Resources:     []ResourceProxy{NewResourceProxyImage(targetProxy)},
		Target:        surfaceProxy,
		ClearColor:    [4]float32{0, 0, 0, 0},
	})
	surfaceView := surface.Texture.CreateView()
	externalResources := []ExternalResource{
		NewExternalResourceImage(targetProxy, &target.View),
		NewExternalResourceImage(surfaceProxy, &surfaceView),
	}
	if err := r.Engine.RunRecording(device, queue, &recording, externalResources, "blit (render_to_surface)", r.Profiler); err != nil {
		return err
	}
	r.Target = target
	return nil
}

// TargetTexture represents a target texture.
type TargetTexture struct {
	View   TextureView
	Width  uint32
	Height uint32
	Format TextureFormat
}

// NewTargetTexture creates a new target texture.
func NewTargetTexture(device *Device, width, height uint32) *TargetTexture {
	format := TextureFormatRgba8Unorm
	texture := device.CreateTexture(TextureDescriptor{
		Size: Extent3d{
			Width:  width,
			Height: height,
			Depth:  1,
		},
		MipLevelCount: 1,
		SampleCount:   1,
		Dimension:     TextureDimension2D,
		Usage:         TextureUsageStorageBinding | TextureUsageTextureBinding,
		Format:        format,
	})
	view := texture.CreateView()
	return &TargetTexture{
		View:   view,
		Width:  width,
		Height: height,
		Format: format,
	}
}

// BlitPipeline represents a blit pipeline.
type BlitPipeline struct {
	ID ShaderID
}

// NewBlitPipeline creates a new blit pipeline.
func NewBlitPipeline(device *Device, format TextureFormat, engine *WgpuEngine) (*BlitPipeline, error) {
	const shaders = `
        @vertex
        fn vs_main(@builtin(vertex_index) ix: u32) -> @builtin(position) vec4<f32> {
            var vertex = vec2(-1.0, 1.0);
            switch ix {
                case 1u: {
                    vertex = vec2(-1.0, -1.0);
                }
                case 2u, 4u: {
                    vertex = vec2(1.0, -1.0);
                }
                case 5u: {
                    vertex = vec2(1.0, 1.0);
                }
                default: {}
            }
            return vec4(vertex, 0.0, 1.0);
        }

        @group(0) @binding(0)
        var fine_output: texture_2d<f32>;

        @fragment
        fn fs_main(@builtin(position) pos: vec4<f32>) -> @location(0) vec4<f32> {
            let rgba_sep = textureLoad(fine_output, vec2<i32>(pos.xy), 0);
            return vec4(rgba_sep.rgb * rgba_sep.a, rgba_sep.a);
        }
    `
	module := device.CreateShaderModule(ShaderModuleDescriptor{
		Label:  "blit shaders",
		Source: ShaderSourceWgsl(shaders),
	})
	shaderID, err := engine.AddRenderShader(
		device,
		"vello.blit",
		&module,
		"vs_main",
		"fs_main",
		PrimitiveTopologyTriangleList,
		ColorTargetState{
			Format:    format,
			Blend:     nil,
			WriteMask: ColorWritesAll,
		},
		nil,
		[]BindType{
			BindTypeImageRead(ImageFormatFromWgpu(format)),
		},
	)
	if err != nil {
		return nil, err
	}
	return &BlitPipeline{ID: shaderID}, nil
}
