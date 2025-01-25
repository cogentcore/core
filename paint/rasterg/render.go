// Copyright 2022 the Vello Authors
// SPDX-License-Identifier: Apache-2.0 OR MIT

//! Take an encoded scene and create a graph to render it

package render

import (
    "github.com/oreillymedia/vello/recording"
    "github.com/oreillymedia/vello/shaders"
    "github.com/oreillymedia/vello/encoding"
)

type Render struct {
    fineWgCount   *WorkgroupSize
    fineResources *FineResources
    maskBuf       *recording.ResourceProxy

    // capturedBuffers is only used when the debug_layers feature is enabled.
    capturedBuffers *CapturedBuffers
}

type FineResources struct {
    aaConfig        AaConfig
    configBuf       recording.ResourceProxy
    bumpBuf         recording.ResourceProxy
    tileBuf         recording.ResourceProxy
    segmentsBuf     recording.ResourceProxy
    ptclBuf         recording.ResourceProxy
    gradientImage   recording.ResourceProxy
    infoBinDataBuf  recording.ResourceProxy
    imageAtlas      recording.ResourceProxy
    blendSpillBuf   recording.ResourceProxy
    outImage        recording.ImageProxy
}

type CapturedBuffers struct {
    sizes      encoding.BufferSizes
    pathBboxes recording.BufferProxy
    lines      recording.BufferProxy
}

func RenderFull(scene *Scene, resolver *Resolver, shaders *shaders.FullShaders, params *RenderParams) (*recording.Recording, *recording.ResourceProxy) {
    return RenderEncodingFull(scene.Encoding(), resolver, shaders, params)
}

func RenderEncodingFull(encoding *encoding.Encoding, resolver *Resolver, shaders *shaders.FullShaders, params *RenderParams) (*recording.Recording, *recording.ResourceProxy) {
    render := NewRender()
    recording := render.RenderEncodingCoarse(encoding, resolver, shaders, params, false)
    outImage := render.OutImage()
    render.RecordFine(shaders, recording)
    return recording, outImage
}

func NewRender() *Render {
    return &Render{}
}

func (r *Render) RenderEncodingCoarse(encoding *encoding.Encoding, resolver *Resolver, shaders *shaders.FullShaders, params *RenderParams, robust bool) *recording.Recording {
    recording := recording.NewRecording()
    var packed []byte

    layout, ramps, images := resolver.Resolve(encoding, &packed)
    gradientImage := recording.NewImage(1, 1, recording.ImageFormatRgba8)
    if ramps.Height == 0 {
        gradientImage = recording.NewImage(1, 1, recording.ImageFormatRgba8)
    } else {
        data := ramps.Data
        gradientImage = recording.UploadImage(ramps.Width, ramps.Height, recording.ImageFormatRgba8, data)
    }
    imageAtlas := recording.NewImage(images.Width, images.Height, recording.ImageFormatRgba8)
    for _, image := range images.Images {
        recording.WriteImage(imageAtlas, image.1, image.2, image.0)
    }
    cpuConfig := NewRenderConfig(layout, params.Width, params.Height, params.BaseColor)
    bufferSizes := cpuConfig.BufferSizes
    wgCounts := cpuConfig.WorkgroupCounts

    if len(packed) == 0 {
        packed = make([]byte, 4)
    }
    sceneBuf := recording.Upload("vello.scene", packed)
    configBuf := recording.UploadUniform("vello.config", cpuConfig.GPU)
    infoBinDataBuf := recording.NewBuf(bufferSizes.BinData.SizeInBytes(), "vello.info_bin_data_buf")
    tileBuf := recording.NewBuf(bufferSizes.Tiles.SizeInBytes(), "vello.tile_buf")
    segmentsBuf := recording.NewBuf(bufferSizes.Segments.SizeInBytes(), "vello.segments_buf")
    ptclBuf := recording.NewBuf(bufferSizes.Ptcl.SizeInBytes(), "vello.ptcl_buf")
    reducedBuf := recording.NewBuf(bufferSizes.PathReduced.SizeInBytes(), "vello.reduced_buf")
    recording.Dispatch(shaders.PathtagReduce, wgCounts.PathReduce, []recording.ResourceProxy{configBuf, sceneBuf, reducedBuf})
    pathtagParent := reducedBuf
    var reduced2Buf, reducedScanBuf recording.ResourceProxy
    useLargePathScan := wgCounts.UseLargePathScan && !shaders.PathtagIsCPU
    if useLargePathScan {
        reduced2Buf = recording.NewBuf(bufferSizes.PathReduced2.SizeInBytes(), "vello.reduced2_buf")
        recording.Dispatch(shaders.PathtagReduce2, wgCounts.PathReduce2, []recording.ResourceProxy{reducedBuf, reduced2Buf})
        reducedScanBuf = recording.NewBuf(bufferSizes.PathReducedScan.SizeInBytes(), "reduced_scan_buf")
        recording.Dispatch(shaders.PathtagScan1, wgCounts.PathScan1, []recording.ResourceProxy{reducedBuf, reduced2Buf, reducedScanBuf})
        pathtagParent = reducedScanBuf
    }
    tagmonoidBuf := recording.NewBuf(bufferSizes.PathMonoids.SizeInBytes(), "vello.tagmonoid_buf")
    pathtagScan := shaders.PathtagScan
    if useLargePathScan {
        pathtagScan = shaders.PathtagScanLarge
    }
    recording.Dispatch(pathtagScan, wgCounts.PathScan, []recording.ResourceProxy{configBuf, sceneBuf, pathtagParent, tagmonoidBuf})
    recording.FreeResource(reducedBuf)
    if useLargePathScan {
        recording.FreeResource(reduced2Buf)
        recording.FreeResource(reducedScanBuf)
    }
    pathBboxBuf := recording.NewBuf(bufferSizes.PathBboxes.SizeInBytes(), "vello.path_bbox_buf")
    recording.Dispatch(shaders.BboxClear, wgCounts.BboxClear, []recording.ResourceProxy{configBuf, pathBboxBuf})
    bumpBuf := recording.NewBuf(bufferSizes.BumpAlloc.SizeInBytes(), "vello.bump_buf")
    recording.ClearAll(bumpBuf)
    bumpBufResource := recording.ResourceProxy{Buffer: bumpBuf}
    linesBuf := recording.NewBuf(bufferSizes.Lines.SizeInBytes(), "vello.lines_buf")
    recording.Dispatch(shaders.Flatten, wgCounts.Flatten, []recording.ResourceProxy{configBuf, sceneBuf, tagmonoidBuf, pathBboxBuf, bumpBufResource, linesBuf})
    drawReducedBuf := recording.NewBuf(bufferSizes.DrawReduced.SizeInBytes(), "vello.draw_reduced_buf")
    recording.Dispatch(shaders.DrawReduce, wgCounts.DrawReduce, []recording.ResourceProxy{configBuf, sceneBuf, drawReducedBuf})
    drawMonoidBuf := recording.NewBuf(bufferSizes.DrawMonoids.SizeInBytes(), "vello.draw_monoid_buf")
    recording.Dispatch(shaders.DrawLeaf, wgCounts.DrawLeaf, []recording.ResourceProxy{configBuf, sceneBuf, drawReducedBuf, pathBboxBuf, drawMonoidBuf, infoBinDataBuf, clipInpBuf})
    recording.FreeResource(drawReducedBuf)
            buffer_sizes.draw_monoids.size_in_bytes().into(),
            "vello.draw_monoid_buf",
        );
        let clip_inp_buf = ResourceProxy::new_buf(
            buffer_sizes.clip_inps.size_in_bytes().into(),
            "vello.clip_inp_buf",
        );
        recording.dispatch(
            shaders.draw_leaf,
            wg_counts.draw_leaf,
            [
                config_buf,
                scene_buf,
                draw_reduced_buf,
                path_bbox_buf,
                draw_monoid_buf,
                info_bin_data_buf,
                clip_inp_buf,
            ],
        );
        recording.free_resource(draw_reduced_buf);
        let clip_el_buf = ResourceProxy::new_buf(
            buffer_sizes.clip_els.size_in_bytes().into(),
            "vello.clip_el_buf",
        );
        let clip_bic_buf = ResourceProxy::new_buf(
            buffer_sizes.clip_bics.size_in_bytes().into(),
            "vello.clip_bic_buf",
        );
        if wg_counts.clip_reduce.0 > 0 {
            recording.dispatch(
                shaders.clip_reduce,
                wg_counts.clip_reduce,
                [clip_inp_buf, path_bbox_buf, clip_bic_buf, clip_el_buf],
            );
        }
        let clip_bbox_buf = ResourceProxy::new_buf(
            buffer_sizes.clip_bboxes.size_in_bytes().into(),
            "vello.clip_bbox_buf",
        );
        if wg_counts.clip_leaf.0 > 0 {
            recording.dispatch(
                shaders.clip_leaf,
                wg_counts.clip_leaf,
                [
                    config_buf,
                    clip_inp_buf,
                    path_bbox_buf,
                    clip_bic_buf,
                    clip_el_buf,
                    draw_monoid_buf,
                    clip_bbox_buf,
                ],
            );
        }
        recording.free_resource(clip_inp_buf);
        recording.free_resource(clip_bic_buf);
        recording.free_resource(clip_el_buf);
        let draw_bbox_buf = ResourceProxy::new_buf(
            buffer_sizes.draw_bboxes.size_in_bytes().into(),
            "vello.draw_bbox_buf",
        );
        let bin_header_buf = ResourceProxy::new_buf(
            buffer_sizes.bin_headers.size_in_bytes().into(),
            "vello.bin_header_buf",
        );
        recording.dispatch(
            shaders.binning,
            wg_counts.binning,
            [
                config_buf,
                draw_monoid_buf,
                path_bbox_buf,
                clip_bbox_buf,
                draw_bbox_buf,
                bump_buf,
                info_bin_data_buf,
                bin_header_buf,
            ],
        );
        recording.free_resource(draw_monoid_buf);
        recording.free_resource(clip_bbox_buf);
        // Note: this only needs to be rounded up because of the workaround to store the tile_offset
        // in storage rather than workgroup memory.
        let path_buf =
            ResourceProxy::new_buf(buffer_sizes.paths.size_in_bytes().into(), "vello.path_buf");
        recording.dispatch(
            shaders.tile_alloc,
            wg_counts.tile_alloc,
            [
                config_buf,
                scene_buf,
                draw_bbox_buf,
                bump_buf,
                path_buf,
                tile_buf,
            ],
        );
        recording.free_resource(draw_bbox_buf);
        recording.free_resource(tagmonoid_buf);
        let indirect_count_buf = BufferProxy::new(
            buffer_sizes.indirect_count.size_in_bytes().into(),
            "vello.indirect_count",
        );
        recording.dispatch(
            shaders.path_count_setup,
            wg_counts.path_count_setup,
            [bump_buf, indirect_count_buf.into()],
        );
        let seg_counts_buf = ResourceProxy::new_buf(
            buffer_sizes.seg_counts.size_in_bytes().into(),
            "vello.seg_counts_buf",
        );
        recording.dispatch_indirect(
            shaders.path_count,
            indirect_count_buf,
            0,
            [
                config_buf,
                bump_buf,
                lines_buf,
                path_buf,
                tile_buf,
                seg_counts_buf,
            ],
        );
        recording.dispatch(
            shaders.backdrop,
            wg_counts.backdrop,
            [config_buf, bump_buf, path_buf, tile_buf],
        );
        recording.dispatch(
            shaders.coarse,
            wg_counts.coarse,
            [
                config_buf,
                scene_buf,
                draw_monoid_buf,
                bin_header_buf,
                info_bin_data_buf,
                path_buf,
                tile_buf,
                bump_buf,
                ptcl_buf,
            ],
        );
        recording.dispatch(
            shaders.path_tiling_setup,
            wg_counts.path_tiling_setup,
            [bump_buf, indirect_count_buf.into(), ptcl_buf],
        );
        recording.dispatch_indirect(
            shaders.path_tiling,
            indirect_count_buf,
            0,
            [
                bump_buf,
                seg_counts_buf,
                lines_buf,
                path_buf,
                tile_buf,
                segments_buf,
            ],
        );
        recording.free_buffer(indirect_count_buf);
        recording.free_resource(seg_counts_buf);
        recording.free_resource(scene_buf);
        recording.free_resource(draw_monoid_buf);
        recording.free_resource(bin_header_buf);
        recording.free_resource(path_buf);
        let out_image = ImageProxy::new(params.width, params.height, ImageFormat::Rgba8);
        let blend_spill_buf = BufferProxy::new(
            buffer_sizes.blend_spill.size_in_bytes().into(),
            "vello.blend_spill",
        );
        self.fine_wg_count = Some(wg_counts.fine);
        self.fine_resources = Some(FineResources {
            aa_config: params.antialiasing_method,
            config_buf,
            bump_buf,
            tile_buf,
            segments_buf,
            ptcl_buf,
            gradient_image,
            info_bin_data_buf,
            blend_spill_buf: ResourceProxy::Buffer(blend_spill_buf),
            image_atlas: ResourceProxy::Image(image_atlas),
            out_image,
        });
        if robust {
            recording.download(*bump_buf.as_buf().unwrap());
        }
        recording.free_resource(bump_buf);

        #[cfg(feature = "debug_layers")]
        {
            if robust {
                let path_bboxes = *path_bbox_buf.as_buf().unwrap();
                let lines = *lines_buf.as_buf().unwrap();
                recording.download(lines);

                self.captured_buffers = Some(CapturedBuffers {
                    sizes: cpu_config.buffer_sizes,
                    path_bboxes,
                    lines,
                });
            } else {
                recording.free_resource(path_bbox_buf);
                recording.free_resource(lines_buf);
            }
        }
        #[cfg(not(feature = "debug_layers"))]
        {
            recording.free_resource(path_bbox_buf);
            recording.free_resource(lines_buf);
        }

        recording
    }

    /// Run fine rasterization assuming the coarse phase succeeded.
    pub fn record_fine(&mut self, shaders: &FullShaders, recording: &mut Recording) {
        let fine_wg_count = self.fine_wg_count.take().unwrap();
        let fine = self.fine_resources.take().unwrap();
        match fine.aa_config {
            AaConfig::Area => {
                recording.dispatch(
                    shaders
                        .fine_area
                        .expect("shaders not configured to support AA mode: area"),
                    fine_wg_count,
                    [
                        fine.config_buf,
                        fine.segments_buf,
                        fine.ptcl_buf,
                        fine.info_bin_data_buf,
                        fine.blend_spill_buf,
                        ResourceProxy::Image(fine.out_image),
                        fine.gradient_image,
                        fine.image_atlas,
                    ],
                );
            }
            _ => {
                if self.mask_buf.is_none() {
                    let mask_lut = match fine.aa_config {
                        AaConfig::Msaa16 => make_mask_lut_16(),
                        AaConfig::Msaa8 => make_mask_lut(),
                        _ => unreachable!(),
                    };
                    let buf = recording.upload("vello.mask_lut", mask_lut);
                    self.mask_buf = Some(buf.into());
                }
                let fine_shader = match fine.aa_config {
                    AaConfig::Msaa16 => shaders
                        .fine_msaa16
                        .expect("shaders not configured to support AA mode: msaa16"),
                    AaConfig::Msaa8 => shaders
                        .fine_msaa8
                        .expect("shaders not configured to support AA mode: msaa8"),
                    _ => unreachable!(),
                };
                recording.dispatch(
                    fine_shader,
                    fine_wg_count,
                    [
                        fine.config_buf,
                        fine.segments_buf,
                        fine.ptcl_buf,
                        fine.info_bin_data_buf,
                        fine.blend_spill_buf,
                        ResourceProxy::Image(fine.out_image),
                        fine.gradient_image,
                        fine.image_atlas,
                        self.mask_buf.unwrap(),
                    ],
                );
            }
        }
        recording.free_resource(fine.config_buf);
        recording.free_resource(fine.tile_buf);
        recording.free_resource(fine.segments_buf);
        recording.free_resource(fine.ptcl_buf);
        recording.free_resource(fine.gradient_image);
        recording.free_resource(fine.image_atlas);
        recording.free_resource(fine.info_bin_data_buf);
        recording.free_resource(fine.blend_spill_buf);
        // TODO: make mask buf persistent
        if let Some(mask_buf) = self.mask_buf.take() {
            recording.free_resource(mask_buf);
        }
    }

    /// Get the output image.
    ///
    /// This is going away, as the caller will add the output image to the bind
    /// map.
    pub fn out_image(&self) -> ImageProxy {
        self.fine_resources.as_ref().unwrap().out_image
    }

    pub fn bump_buf(&self) -> BufferProxy {
        *self
            .fine_resources
            .as_ref()
            .unwrap()
            .bump_buf
            .as_buf()
            .unwrap()
    }

    #[cfg(feature = "debug_layers")]
    pub fn take_captured_buffers(&mut self) -> Option<CapturedBuffers> {
        self.captured_buffers.take()
    }
}

/// Resources produced by pipeline, needed for fine rasterization.
struct FineResources {
    aa_config: AaConfig,

    config_buf: ResourceProxy,
    bump_buf: ResourceProxy,
    tile_buf: ResourceProxy,
    segments_buf: ResourceProxy,
    ptcl_buf: ResourceProxy,
    gradient_image: ResourceProxy,
    info_bin_data_buf: ResourceProxy,
    image_atlas: ResourceProxy,
    blend_spill_buf: ResourceProxy,

    out_image: ImageProxy,
}

/// A collection of internal buffers that are used for debug visualization when the
/// `debug_layers` feature is enabled. The contents of these buffers remain GPU resident
/// and must be freed directly by the caller.
///
/// Some of these buffers are also scheduled for a download to allow their contents to be
/// processed for CPU-side validation. These buffers are documented as such.
#[cfg(feature = "debug_layers")]
pub struct CapturedBuffers {
    pub sizes: vello_encoding::BufferSizes,

    /// Buffers that remain GPU-only
    pub path_bboxes: BufferProxy,

    /// Buffers scheduled for download
    pub lines: BufferProxy,
}

#[cfg(feature = "debug_layers")]
impl CapturedBuffers {
    pub fn release_buffers(self, recording: &mut Recording) {
        recording.free_buffer(self.path_bboxes);
        recording.free_buffer(self.lines);
    }
}

#[cfg(feature = "wgpu")]
pub(crate) fn render_full(
    scene: &Scene,
    resolver: &mut Resolver,
    shaders: &FullShaders,
    params: &RenderParams,
) -> (Recording, ResourceProxy) {
    render_encoding_full(scene.encoding(), resolver, shaders, params)
}

#[cfg(feature = "wgpu")]
/// Create a single recording with both coarse and fine render stages.
///
/// This function is not recommended when the scene can be complex, as it does not
/// implement robust dynamic memory.
pub(crate) fn render_encoding_full(
    encoding: &Encoding,
    resolver: &mut Resolver,
    shaders: &FullShaders,
    params: &RenderParams,
) -> (Recording, ResourceProxy) {
    let mut render = Render::new();
    let mut recording = render.render_encoding_coarse(encoding, resolver, shaders, params, false);
    let out_image = render.out_image();
    render.record_fine(shaders, &mut recording);
    (recording, out_image.into())
}

impl Default for Render {
    fn default() -> Self {
        Self::new()
    }
}

impl Render {
    pub fn new() -> Self {
        Self {
            fine_wg_count: None,
            fine_resources: None,
            mask_buf: None,
            #[cfg(feature = "debug_layers")]
            captured_buffers: None,
        }
    }

    /// Prepare a recording for the coarse rasterization phase.
    ///
    /// The `robust` parameter controls whether we're preparing for readback
    /// of the atomic bump buffer, for robust dynamic memory.
    pub fn render_encoding_coarse(
        &mut self,
        encoding: &Encoding,
        resolver: &mut Resolver,
        shaders: &FullShaders,
        params: &RenderParams,
        robust: bool,
    ) -> Recording {
        use vello_encoding::RenderConfig;
        let mut recording = Recording::default();
        let mut packed = vec![];

        let (layout, ramps, images) = resolver.resolve(encoding, &mut packed);
        let gradient_image = if ramps.height == 0 {
            ResourceProxy::new_image(1, 1, ImageFormat::Rgba8)
        } else {
            let data: &[u8] = bytemuck::cast_slice(ramps.data);
            ResourceProxy::Image(recording.upload_image(
                ramps.width,
                ramps.height,
                ImageFormat::Rgba8,
                data,
            ))
        };
        let image_atlas = if images.images.is_empty() {
            ImageProxy::new(1, 1, ImageFormat::Rgba8)
        } else {
            ImageProxy::new(images.width, images.height, ImageFormat::Rgba8)
        };
        for image in images.images {
            recording.write_image(image_atlas, image.1, image.2, image.0.clone());
        }
        let cpu_config =
            RenderConfig::new(&layout, params.width, params.height, &params.base_color);
        // HACK: The coarse workgroup counts is the number of active bins.
        if (cpu_config.workgroup_counts.coarse.0
            * cpu_config.workgroup_counts.coarse.1
            * cpu_config.workgroup_counts.coarse.2)
            > 256
        {
            log::warn!(
                "Trying to paint too large image. {}x{}.\n\
                See https://github.com/linebender/vello/issues/680 for details",
                params.width,
                params.height
            );
        }
        let buffer_sizes = &cpu_config.buffer_sizes;
        let wg_counts = &cpu_config.workgroup_counts;

        if packed.is_empty() {
            // HACK: wgpu doesn't allow empty buffers, so we make sure that the scene buffer we upload
            // can contain at least one array item.
            // The values passed here should never be read, because the scene size in config
            // is zero.
            packed.resize(size_of::<u32>(), u8::MAX);
        }
        let scene_buf = ResourceProxy::Buffer(recording.upload("vello.scene", packed));
        let config_buf = ResourceProxy::Buffer(
            recording.upload_uniform("vello.config", bytemuck::bytes_of(&cpu_config.gpu)),
        );
        let info_bin_data_buf = ResourceProxy::new_buf(
            buffer_sizes.bin_data.size_in_bytes() as u64,
            "vello.info_bin_data_buf",
        );
        let tile_buf =
            ResourceProxy::new_buf(buffer_sizes.tiles.size_in_bytes().into(), "vello.tile_buf");
        let segments_buf = ResourceProxy::new_buf(
            buffer_sizes.segments.size_in_bytes().into(),
            "vello.segments_buf",
        );
        let ptcl_buf =
            ResourceProxy::new_buf(buffer_sizes.ptcl.size_in_bytes().into(), "vello.ptcl_buf");
        let reduced_buf = ResourceProxy::new_buf(
            buffer_sizes.path_reduced.size_in_bytes().into(),
            "vello.reduced_buf",
        );
        // TODO: really only need pathtag_wgs - 1
        recording.dispatch(
            shaders.pathtag_reduce,
            wg_counts.path_reduce,
            [config_buf, scene_buf, reduced_buf],
        );
        let mut pathtag_parent = reduced_buf;
        let mut large_pathtag_bufs = None;
        let use_large_path_scan = wg_counts.use_large_path_scan && !shaders.pathtag_is_cpu;
        if use_large_path_scan {
            let reduced2_buf = ResourceProxy::new_buf(
                buffer_sizes.path_reduced2.size_in_bytes().into(),
                "vello.reduced2_buf",
            );
            recording.dispatch(
                shaders.pathtag_reduce2,
                wg_counts.path_reduce2,
                [reduced_buf, reduced2_buf],
            );
            let reduced_scan_buf = ResourceProxy::new_buf(
                buffer_sizes.path_reduced_scan.size_in_bytes().into(),
                "reduced_scan_buf",
            );
            recording.dispatch(
                shaders.pathtag_scan1,
                wg_counts.path_scan1,
                [reduced_buf, reduced2_buf, reduced_scan_buf],
            );
            pathtag_parent = reduced_scan_buf;
            large_pathtag_bufs = Some((reduced2_buf, reduced_scan_buf));
        }

        let tagmonoid_buf = ResourceProxy::new_buf(
            buffer_sizes.path_monoids.size_in_bytes().into(),
            "vello.tagmonoid_buf",
        );
        let pathtag_scan = if use_large_path_scan {
            shaders.pathtag_scan_large
        } else {
            shaders.pathtag_scan
        };
        recording.dispatch(
            pathtag_scan,
            wg_counts.path_scan,
            [config_buf, scene_buf, pathtag_parent, tagmonoid_buf],
        );
        recording.free_resource(reduced_buf);
        if let Some((reduced2, reduced_scan)) = large_pathtag_bufs {
            recording.free_resource(reduced2);
            recording.free_resource(reduced_scan);
        }
        let path_bbox_buf = ResourceProxy::new_buf(
            buffer_sizes.path_bboxes.size_in_bytes().into(),
            "vello.path_bbox_buf",
        );
        recording.dispatch(
            shaders.bbox_clear,
            wg_counts.bbox_clear,
            [config_buf, path_bbox_buf],
        );
        let bump_buf = BufferProxy::new(
            buffer_sizes.bump_alloc.size_in_bytes().into(),
            "vello.bump_buf",
        );
        recording.clear_all(bump_buf);
        let bump_buf = ResourceProxy::Buffer(bump_buf);
        let lines_buf =
            ResourceProxy::new_buf(buffer_sizes.lines.size_in_bytes().into(), "vello.lines_buf");
        recording.dispatch(
            shaders.flatten,
            wg_counts.flatten,
            [
                config_buf,
                scene_buf,
                tagmonoid_buf,
                path_bbox_buf,
                bump_buf,
                lines_buf,
            ],
        );
        let draw_reduced_buf = ResourceProxy::new_buf(
            buffer_sizes.draw_reduced.size_in_bytes().into(),
            "vello.draw_reduced_buf",
        );
        recording.dispatch(
            shaders.draw_reduce,
            wg_counts.draw_reduce,
            [config_buf, scene_buf, draw_reduced_buf],
        );
        let draw_monoid_buf = ResourceProxy::new_buf(
            func (r *Renderer) RecordCoarse(shaders *FullShaders, recording *Recording) *Recording {
                bufferSizes := r.bufferSizes
                configBuf := NewResourceProxyBuf(bufferSizes.config.sizeInBytes(), "vello.config_buf")
                sceneBuf := NewResourceProxyBuf(bufferSizes.scene.sizeInBytes(), "vello.scene_buf")
                drawReducedBuf := NewResourceProxyBuf(bufferSizes.drawReduced.sizeInBytes(), "vello.draw_reduced_buf")
                pathBBoxBuf := NewResourceProxyBuf(bufferSizes.pathBBox.sizeInBytes(), "vello.path_bbox_buf")
                drawMonoidBuf := NewResourceProxyBuf(bufferSizes.drawMonoid.sizeInBytes(), "vello.draw_monoid_buf")
                infoBinDataBuf := NewResourceProxyBuf(bufferSizes.infoBinData.sizeInBytes(), "vello.info_bin_data_buf")
                clipInpBuf := NewResourceProxyBuf(bufferSizes.clipInps.sizeInBytes(), "vello.clip_inp_buf")
                
                recording.Dispatch(shaders.drawLeaf, wgCounts.drawLeaf, []ResourceProxy{
                    configBuf,
                    sceneBuf,
                    drawReducedBuf,
                    pathBBoxBuf,
                    drawMonoidBuf,
                    infoBinDataBuf,
                    clipInpBuf,
                })
                
                recording.FreeResource(drawReducedBuf)
                
                clipElBuf := NewResourceProxyBuf(bufferSizes.clipEls.sizeInBytes(), "vello.clip_el_buf")
                clipBicBuf := NewResourceProxyBuf(bufferSizes.clipBics.sizeInBytes(), "vello.clip_bic_buf")
                
                if wgCounts.clipReduce > 0 {
                    recording.Dispatch(shaders.clipReduce, wgCounts.clipReduce, []ResourceProxy{
                        clipInpBuf,
                        pathBBoxBuf,
                        clipBicBuf,
                        clipElBuf,
                    })
                }
                
                clipBBoxBuf := NewResourceProxyBuf(bufferSizes.clipBBoxes.sizeInBytes(), "vello.clip_bbox_buf")
                
                if wgCounts.clipLeaf > 0 {
                    recording.Dispatch(shaders.clipLeaf, wgCounts.clipLeaf, []ResourceProxy{
                        configBuf,
                        clipInpBuf,
                        pathBBoxBuf,
                        clipBicBuf,
                        clipElBuf,
                        drawMonoidBuf,
                        clipBBoxBuf,
                    })
                }
                
                recording.FreeResource(clipInpBuf)
                recording.FreeResource(clipBicBuf)
                recording.FreeResource(clipElBuf)
                
                drawBBoxBuf := NewResourceProxyBuf(bufferSizes.drawBBoxes.sizeInBytes(), "vello.draw_bbox_buf")
                binHeaderBuf := NewResourceProxyBuf(bufferSizes.binHeaders.sizeInBytes(), "vello.bin_header_buf")
                
                recording.Dispatch(shaders.binning, wgCounts.binning, []ResourceProxy{
                    configBuf,
                    drawMonoidBuf,
                    pathBBoxBuf,
                    clipBBoxBuf,
                    drawBBoxBuf,
                    bumpBuf,
                    infoBinDataBuf,
                    binHeaderBuf,
                })
                
                recording.FreeResource(drawMonoidBuf)
                recording.FreeResource(clipBBoxBuf)
                
                pathBuf := NewResourceProxyBuf(bufferSizes.paths.sizeInBytes(), "vello.path_buf")
                
                recording.Dispatch(shaders.tileAlloc, wgCounts.tileAlloc, []ResourceProxy{
                    configBuf,
                    sceneBuf,
                    drawBBoxBuf,
                    bumpBuf,
                    pathBuf,
                    tileBuf,
                })
                
                recording.FreeResource(drawBBoxBuf)
                recording.FreeResource(tagmonoidBuf)
                
                indirectCountBuf := NewBufferProxy(bufferSizes.indirectCount.sizeInBytes(), "vello.indirect_count")
                
                recording.Dispatch(shaders.pathCountSetup, wgCounts.pathCountSetup, []ResourceProxy{
                    bumpBuf,
                    indirectCountBuf,
                })
                
                segCountsBuf := NewResourceProxyBuf(bufferSizes.segCounts.sizeInBytes(), "vello.seg_counts_buf")
                
                recording.DispatchIndirect(shaders.pathCount, indirectCountBuf, 0, []ResourceProxy{
                    configBuf,
                    bumpBuf,
                    linesBuf,
                    pathBuf,
                    tileBuf,
                    segCountsBuf,
                })
                
                recording.Dispatch(shaders.backdrop, wgCounts.backdrop, []ResourceProxy{
                    configBuf,
                    bumpBuf,
                    pathBuf,
                    tileBuf,
                })
                
                recording.Dispatch(shaders.coarse, wgCounts.coarse, []ResourceProxy{
                    configBuf,
                    sceneBuf,
                    drawMonoidBuf,
                    binHeaderBuf,
                    infoBinDataBuf,
                    pathBuf,
                    tileBuf,
                    bumpBuf,
                    ptclBuf,
                })
                
                recording.Dispatch(shaders.pathTilingSetup, wgCounts.pathTilingSetup, []ResourceProxy{
                    bumpBuf,
                    indirectCountBuf,
                    ptclBuf,
                })
                
                recording.DispatchIndirect(shaders.pathTiling, indirectCountBuf, 0, []ResourceProxy{
                    bumpBuf,
                    segCountsBuf,
                    linesBuf,
                    pathBuf,
                    tileBuf,
                    segmentsBuf,
                })
                
                recording.FreeBuffer(indirectCountBuf)
                recording.FreeResource(segCountsBuf)
                recording.FreeResource(sceneBuf)
                recording.FreeResource(drawMonoidBuf)
                recording.FreeResource(binHeaderBuf)
                recording.FreeResource(pathBuf)
                
                outImage := NewImageProxy(params.width, params.height, ImageFormat.Rgba8)
                blendSpillBuf := NewBufferProxy(bufferSizes.blendSpill.sizeInBytes(), "vello.blend_spill")
                
                r.fineWgCount = wgCounts.fine
                r.fineResources = FineResources{
                    aaConfig:          params.antialiasingMethod,
                    configBuf:         configBuf,
                    bumpBuf:           bumpBuf,
                    tileBuf:           tileBuf,
                    segmentsBuf:       segmentsBuf,
                    ptclBuf:           ptclBuf,
                    gradientImage:     gradientImage,
                    infoBinDataBuf:    infoBinDataBuf,
                    blendSpillBuf:     ResourceProxyBuffer(blendSpillBuf),
                    imageAtlas:        ResourceProxyImage(imageAtlas),
                    outImage:          outImage,
                }
                
                if robust {
                    recording.Download(*bumpBuf.AsBuf().(*Buffer))
                }
                
                recording.FreeResource(bumpBuf)
                
                #ifdef DEBUG_LAYERS
                if robust {
                    pathBBoxes := *pathBBoxBuf.AsBuf().(*Buffer)
                    lines := *linesBuf.AsBuf().(*Buffer)
                    recording.Download(lines)
                    
                    r.capturedBuffers = CapturedBuffers{
                        sizes:      cpuConfig.bufferSizes,
                        pathBBoxes: pathBBoxes,
                        lines:      lines,
                    }
                } else {
                    recording.FreeResource(pathBBoxBuf)
                    recording.FreeResource(linesBuf)
                }
                #else
                recording.FreeResource(pathBBoxBuf)
                recording.FreeResource(linesBuf)
                #endif
                
                return recording
            }

            func (r *Renderer) RecordFine(shaders *FullShaders, recording *Recording) *Recording {
                fineWgCount := r.fineWgCount
                fine := r.fineResources
                
                switch fine.aaConfig {
                case AaConfigArea:
                    recording.Dispatch(shaders.fineArea, fineWgCount, []ResourceProxy{
                        fine.configBuf,
                        fine.segmentsBuf,
                        fine.ptclBuf,
                        fine.infoBinDataBuf,
                        fine.blendSpillBuf,
                        ResourceProxyImage(fine.outImage),
                        fine.gradientImage,
                        fine.imageAtlas,
                    })
                default:
                    if r.maskBuf == nil {
                        var maskLut []byte
                        switch fine.aaConfig {
                        case AaConfigMsaa16:
                            maskLut = makeMaskLut16()
                        case AaConfigMsaa8:
                            maskLut = makeMaskLut()
                        }
                        buf := recording.Upload("vello.mask_lut", maskLut)
                        r.maskBuf = &buf
                    }
                    
                    var fineShader *Shader
                    switch fine.aaConfig {
                    case AaConfigMsaa16:
                        fineShader = shaders.fineMsaa16
                    case AaConfigMsaa8:
                        fineShader = shaders.fineMsaa8
                    }
                    
                    recording.Dispatch(fineShader, fineWgCount, []ResourceProxy{
                        fine.configBuf,
                        fine.segmentsBuf,
                        fine.ptclBuf,
                        fine.infoBinDataBuf,
                        fine.blendSpillBuf,
                        ResourceProxyImage(fine.outImage),
                        fine.gradientImage,
                        fine.imageAtlas,
                        *r.maskBuf,
                    })
                }
                
                recording.FreeResource(fine.configBuf)
                recording.FreeResource(fine.tileBuf)
                recording.FreeResource(fine.segmentsBuf)
                recording.FreeResource(fine.ptclBuf)
                recording.FreeResource(fine.gradientImage)
                recording.FreeResource(fine.imageAtlas)
                recording.FreeResource(fine.infoBinDataBuf)
                recording.FreeResource(fine.blendSpillBuf)
                
                if r.maskBuf != nil {
                    recording.FreeResource(*r.maskBuf)
                }
                
                return recording
            }

            func (r *Renderer) OutImage() *ImageProxy {
                return &r.fineResources.outImage
            }

            func (r *Renderer) BumpBuf() *BufferProxy {
                return r.fineResources.bumpBuf.AsBuf().(*Buffer)
            }

            #ifdef DEBUG_LAYERS
            func (r *Renderer) TakeCapturedBuffers() *CapturedBuffers {
                return r.capturedBuffers
            }
            #endif

