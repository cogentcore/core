// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

//go:generate core generate

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"cogentcore.org/core/base/reflectx"
	"github.com/cogentcore/webgpu/wgpu"
)

var (
	// Debug is whether to enable debug mode, getting
	// more diagnostic output about GPU configuration and rendering.
	// It should be set using [SetDebug].
	Debug = false

	// DebugAdapter provides detailed information about the selected
	// GPU adpater device (i.e., the type and limits of the hardware).
	DebugAdapter = false
)

// SetDebug sets [Debug] (debug mode). If it is set to true,
// it calls [wgpu.SetLogLevel]([wgpu.LogLevelDebug]). Otherwise,
// it calls [wgpu.SetLogLevel]([wgpu.LogLevelError]).
// It is called automatically with false in init().
// You can also manually set the log level with
// [wgpu.SetLogLevel].
func SetDebug(debug bool) {
	Debug = debug
	if Debug {
		wgpu.SetLogLevel(wgpu.LogLevelDebug)
	} else {
		wgpu.SetLogLevel(wgpu.LogLevelError)
	}
}

func init() { SetDebug(false) }

// GPU represents the GPU hardware
type GPU struct {
	// Instance represents the WebGPU system overall
	Instance *wgpu.Instance

	// GPU represents the specific GPU hardware device used.
	// You can call GetInfo() to get info.
	GPU *wgpu.Adapter

	// name of the physical GPU device
	DeviceName string

	// Properties are the general properties of the GPU adapter.
	Properties wgpu.AdapterInfo

	// Limits are the limits of the current GPU adapter.
	Limits wgpu.SupportedLimits

	// ComputeOnly indicates if this GPU is only used for compute,
	// which determines if it listens to WEBGPU_COMPUTE_DEVICE_SELECT
	// environment variable, allowing different compute devices to be
	// selected vs. graphics devices.
	ComputeOnly bool

	// maximum number of compute threads per compute shader invocation,
	// for a 1D number of threads per Warp, which is generally greater
	// than MaxComputeWorkGroup, which allows for the maxima as well.
	// This is not defined anywhere in the formal spec, unfortunately,
	// but has been determined empirically for Mac and NVIDIA which are
	// two of the most relevant use-cases.  If not a known case,
	// the MaxComputeWorkGroupvalue is used, which can significantly
	// slow down compute processing if more could actually be used.
	// Please file an issue or PR for other GPUs with known larger values.
	MaxComputeWorkGroupCount1D int
}

// NewGPU returns a new GPU, configured and ready to use.
// If only doing compute, use [NewComputeGPU].
func NewGPU() *GPU {
	gp := &GPU{}
	gp.init()
	return gp
}

// NewComputeGPU returns a new GPU, configured and ready to use,
// for purely compute use, which causes it to listen to
// use the WEBGPU_COMPUTE_DEVICE_SELECT variable for which GPU device to use.
func NewComputeGPU() *GPU {
	gp := &GPU{}
	gp.ComputeOnly = true
	gp.init()
	return gp
}

// init configures the GPU
func (gp *GPU) init() error {
	gp.Instance = wgpu.CreateInstance(nil)

	gpus := gp.Instance.EnumerateAdapters(nil)
	gpIndex := gp.SelectGPU(gpus)
	gp.GPU = gpus[gpIndex]
	gp.Properties = gp.GPU.GetInfo()
	gp.DeviceName = adapterName(&gp.Properties)
	if Debug || DebugAdapter {
		fmt.Println("gpu: Selected Device:", gp.DeviceName, " (set DebugAdapter to get more adapter info)")
	}

	gp.Limits = gp.GPU.GetLimits()

	if DebugAdapter {
		fmt.Println(gp.PropertiesString())
	}

	gp.MaxComputeWorkGroupCount1D = int(gp.Limits.Limits.MaxComputeWorkgroupsPerDimension)
	ldv := strings.ToLower(gp.DeviceName)
	if strings.Contains(ldv, "nvidia") {
		// all NVIDIA are either 1 << 31 or -1 of that.
		gp.MaxComputeWorkGroupCount1D = (1 << 31) - 1
	} else if strings.Contains(ldv, "apple") {
		gp.MaxComputeWorkGroupCount1D = (1 << 31) - 1
	}
	// note: if known to be higher for any specific case, please file an issue or PR
	// todo: where are the errors!?
	return nil
}

func adapterName(ai *wgpu.AdapterInfo) string {
	if ai.Name != "" && ai.Name != "0x0" {
		return ai.Name
	}
	if ai.DriverDescription != "" && ai.DriverDescription != "0x0" {
		return ai.DriverDescription
	}
	return ai.VendorName
}

func (gp *GPU) SelectGPU(gpus []*wgpu.Adapter) int {
	n := len(gpus)
	if n == 1 {
		return 0
	}
	trgDevNm := ""
	if ev := os.Getenv("GPU_DEVICE_SELECT"); ev != "" {
		trgDevNm = ev
	}
	if gp.ComputeOnly {
		if ev := os.Getenv("GPU_COMPUTE_DEVICE_SELECT"); ev != "" {
			trgDevNm = ev
		}
	}

	if trgDevNm != "" {
		idx, err := strconv.Atoi(trgDevNm)
		if err == nil && idx >= 0 && idx < n {
			return idx
		}
		for gi := range n {
			props := gpus[gi].GetInfo()
			if gpuIsBadBackend(props.BackendType) {
				continue
			}
			pnm := adapterName(&props)
			if strings.Contains(pnm, trgDevNm) {
				devNm := props.Name
				if Debug {
					log.Printf("wgpu: selected device named: %s, specified in *_DEVICE_SELECT environment variable, index: %d\n", devNm, gi)
				}
				return gi
			}
		}
		if Debug {
			log.Printf("vgpu: unable to find device named: %s, specified in *_DEVICE_SELECT environment variable\n", trgDevNm)
		}
	}

	// scoring system has 1 point for discrete and 1 for non-gl backend
	hiscore := 0
	best := 0
	for gi := range n {
		score := 0
		props := gpus[gi].GetInfo()
		if gpuIsBadBackend(props.BackendType) {
			continue
		}
		if props.AdapterType == wgpu.AdapterTypeDiscreteGPU {
			score++
		}
		if !gpuIsGLdBackend(props.BackendType) {
			score++
		}
		if score > hiscore {
			hiscore = score
			best = gi
		}
	}
	return best
}

func gpuIsGLdBackend(bet wgpu.BackendType) bool {
	return bet == wgpu.BackendTypeOpenGL || bet == wgpu.BackendTypeOpenGLES
}

func gpuIsBadBackend(bet wgpu.BackendType) bool {
	return bet == wgpu.BackendTypeUndefined || bet == wgpu.BackendTypeNull
}

// Release releases GPU resources -- call after everything else has been destroyed
func (gp *GPU) Release() {
	if gp.GPU != nil {
		gp.GPU.Release()
		gp.GPU = nil
	}
	if gp.Instance != nil {
		gp.Instance.Release()
		gp.Instance = nil
	}
}

// NewDevice returns a new device for given GPU.
// It gets the Queue for this device.
func (gp *GPU) NewDevice() (*Device, error) {
	return NewDevice(gp)
}

// PropertiesString returns a human-readable summary of the GPU properties.
func (gp *GPU) PropertiesString() string {
	return "\n######## GPU Properties\n" + reflectx.StringJSON(&gp.Properties) + reflectx.StringJSON(gp.Limits.Limits)
}

// NoDisplayGPU Initializes WebGPU and returns that and a new
// GPU device, without using an existing surface window.
func NoDisplayGPU() (*GPU, *Device, error) {
	gp := NewGPU()
	dev, err := NewDevice(gp)
	return gp, dev, err
}
