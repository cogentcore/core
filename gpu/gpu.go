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
	// Debug is a global flag for turning on debug mode, getting
	// more diagnostic output about GPU configuration etc.
	// If it is set to true, [NewSurface] will call
	// [wgpu.SetLogLevel]([wgpu.LogLevelInfo]). Otherwise, it will
	// call [wgpu.SetLogLevel]([wgpu.LogLevelError]).
	// You can also manually set the log level with [wgpu.SetLogLevel]
	// after [NewSurface] is called.
	Debug = false

	// DebugAdapter provides detailed information about the selected
	// GPU adpater device (i.e., the type and limits of the hardware).
	DebugAdapter = false
)

// DefaultOpts are default GPU config options that can be set by any app
// prior to initializing the GPU object -- this may be easier than passing
// options in from the app during the Config call.  Any such options take
// precedence over these options (usually best to avoid direct conflits --
// monitor Debug output to see).
var DefaultOpts *GPUOpts

// GPU represents the GPU hardware
type GPU struct {
	// Instance represents the WebGPU system overall
	Instance *wgpu.Instance

	// GPU represents the specific GPU hardware device used.
	// You can call GetInfo() to get info.
	GPU *wgpu.Adapter

	// options passed in during config
	UserOpts *GPUOpts

	// set of enabled options set post-Config
	EnabledOpts GPUOpts

	// name of the physical GPU device
	DeviceName string

	// name of application -- set during Config and used in init of GPU
	AppName string

	// this is used for computing, not graphics
	Compute bool

	// Properties are the general properties of the GPU adapter.
	Properties wgpu.AdapterInfo

	// Limits are the limits of the current GPU adapter.
	Limits wgpu.SupportedLimits

	// maximum number of compute threads per compute shader invokation, for a 1D number of threads per Warp, which is generally greater than MaxComputeWorkGroup, which allows for the and maxima as well.  This is not defined anywhere in the formal spec, unfortunately, but has been determined empirically for Mac and NVIDIA which are two of the most relevant use-cases.  If not a known case, the MaxComputeWorkGroupvalue is used, which can significantly slow down compute processing if more could actually be used.  Please file an issue or PR for other GPUs with known larger values.
	MaxComputeWorkGroupCount1D int
}

// InitNoDisplay initializes WebGPU system for a purely compute-based
// or headless operation, without any display (i.e., without using glfw).
// Call before doing any vgpu stuff.
// Loads the WebGPU library and sets the Vulkan instance proc addr and calls Init.
// IMPORTANT: must be called on the main initial thread!
func InitNoDisplay() error {
	// err := vkinit.LoadVulkan()
	// if err != nil {
	// 	log.Println(err)
	// 	return err
	// }
	return nil
}

// Defaults sets up default parameters, with the graphics flag
// determining whether graphics-relevant items are added.
func (gp *GPU) Defaults(graphics bool) {
	if graphics {
	} else {
		gp.Compute = true
	}
	PlatformDefaults(gp)
}

// NewGPU returns a new GPU struct with Graphics Defaults set
// configure any additional defaults before calling Config.
// Use NewComputeGPU for a compute-only GPU that doesn't load graphics extensions.
func NewGPU() *GPU {
	gp := &GPU{}
	gp.Defaults(true)
	return gp
}

// NewComputeGPU returns a new GPU struct with Compute Defaults set
// configure any additional defaults before calling Config.
// Use NewGPU for a graphics enabled GPU.
func NewComputeGPU() *GPU {
	gp := &GPU{}
	gp.Defaults(false)
	return gp
}

// Config configures the GPU given the extensions set in InstanceExts,
// DeviceExts, and ValidationLayers, and the given GPUOpts options.
// Only the first such opts will be used -- the variable args is used to enable
// no options to be passed by default.
func (gp *GPU) Config(name string, opts ...*GPUOpts) error {
	gp.AppName = name
	gp.UserOpts = DefaultOpts
	if len(opts) > 0 {
		if gp.UserOpts == nil {
			gp.UserOpts = opts[0]
		} else {
			gp.UserOpts.CopyFrom(opts[0])
		}
	}
	gp.Instance = wgpu.CreateInstance(nil)

	gpus := gp.Instance.EnumerateAdapters(nil)
	gpIndex := gp.SelectGPU(gpus)
	gp.GPU = gpus[gpIndex]
	gp.Properties = gp.GPU.GetInfo()
	gp.DeviceName = gp.Properties.Name

	gp.Limits = gp.GPU.GetLimits()

	if DebugAdapter {
		fmt.Println(gp.PropertiesString())
	}

	// todo:
	// gp.MaxComputeWorkGroupCount1D = int(gp.GPUProperties.Limits.MaxComputeWorkGroupCount[0])
	// note: unclear what the limit is here.
	// if gp.MaxComputeWorkGroupCount1D == 0 { // otherwise set per-platform in defaults (DARWIN)
	// if strings.Contains(gp.DeviceName, "NVIDIA") {
	// 	// according to: https://WebGPU.gpuinfo.org/displaydevicelimit.php?name=maxComputeWorkGroupInvocations&platform=all
	// 	// all NVIDIA are either 1 << 31 or -1 of that.
	// 	gp.MaxComputeWorkGroupCount1D = (1 << 31) - 1 // according to vgpu
	// } else {
	// note: if known to be higher for any specific case, please file an issue or PR
	// }
	// }
	return nil
}

func (gp *GPU) SelectGPU(gpus []*wgpu.Adapter) int {
	n := len(gpus)
	if n == 1 {
		return 0
	}
	// todo: also make available other names!
	trgDevNm := ""
	if ev := os.Getenv("MESA_VK_DEVICE_SELECT"); ev != "" {
		trgDevNm = ev
	} else if ev := os.Getenv("VK_DEVICE_SELECT"); ev != "" {
		trgDevNm = ev
	}
	if gp.Compute {
		if ev := os.Getenv("VK_COMPUTE_DEVICE_SELECT"); ev != "" {
			trgDevNm = ev
		}
	}

	if trgDevNm != "" {
		idx, err := strconv.Atoi(trgDevNm)
		if err == nil && idx >= 0 && idx < n {
			return idx
		}
		for gi := range n {
			// type AdapterInfo struct {
			// 	VendorId          uint32
			// 	VendorName        string
			// 	Architecture      string
			// 	DeviceId          uint32
			// 	Name              string
			// 	DriverDescription string
			// 	AdapterType       AdapterType
			// 	BackendType       BackendType
			// }

			props := gpus[gi].GetInfo()
			if strings.Contains(props.Name, trgDevNm) {
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

	for gi := range n {
		// note: we could potentially check for the optional features here
		// but generally speaking the discrete device is going to be the most
		// feature-full, so the practical benefit is unlikely to be significant.
		props := gpus[gi].GetInfo()
		if props.AdapterType == wgpu.AdapterTypeDiscreteGPU {
			// todo: pick one with best memory
			// var memProperties vk.PhysicalDeviceMemoryProperties
			// vk.GetPhysicalDeviceMemoryProperties(gpus[gi], &memProperties)
			// memProperties.Deref()
			// if Debug {
			// 	log.Printf("vgpu: %d: evaluating discrete device named: %s\n", gi, dnm)
			// }
			// for mi := uint32(0); mi < memProperties.MemoryHeapCount; mi++ {
			// 	heap := &memProperties.MemoryHeaps[mi]
			// 	heap.Deref()
			// 	// if heap.Flags&vk.MemoryHeapFlags(vk.MemoryHeapDeviceLocalBit) != 0 {
			// 	sz := int(heap.Size)
			// 	if sz > maxSz {
			// 		devNm = gp.GetDeviceName(&properties, gi)
			// 		maxSz = sz
			// 		maxIndex = gi
			// 	}
			// }
			// }
			return gi
		}
		// } else {
		// 	if Debug {
		// 		log.Printf("vgpu: %d: skipping device named: %s -- not discrete\n", gi, dnm)
		// 	}
		// }
	}
	return 0
	// gp.DeviceName = devNm
	// if Debug {
	// 	log.Printf("vgpu: %d selected device named: %s, memory size: %d\n", maxIndex, devNm, maxSz)
	// }
	// return maxIndex
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

// NewComputeSystem returns a new system initialized for this GPU,
// exclusively for Compute, not graphics functionality.
// func (gp *GPU) NewComputeSystem(name string) *GraphicsSystem {
// 	return NewComputeGraphicsSystem(gp, name)
// }

// NewDevice returns a new device for given GPU.
// It gets the Queue for this device.
func (gp *GPU) NewDevice() (*Device, error) {
	return NewDevice(gp)
}

// PropertiesString returns a human-readable summary of the GPU properties.
func (gp *GPU) PropertiesString() string {
	return "\n######## GPU Properties\n" + reflectx.StringJSON(&gp.Properties) + reflectx.StringJSON(gp.Limits.Limits)
}

// NoDisplayGPU Initializes the Vulkan GPU and returns that
// and the graphics GPU device, with given name, without connecting
// to the display.
func NoDisplayGPU(nm string) (*GPU, *Device, error) {
	if err := InitNoDisplay(); err != nil {
		return nil, nil, err
	}
	gp := NewGPU()
	if err := gp.Config(nm, nil); err != nil {
		return nil, nil, err
	}
	dev, err := NewGraphicsDevice(gp)
	return gp, dev, err
}
