// Copyright (c) 2022, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

//go:generate core generate

import (
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/rajveermalviya/go-webgpu/wgpu"
)

// Debug is a global flag for turning on debug mode
// Set to true prior to GPU.Config to get validation debugging.
var Debug = false

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
	// You can call AdapterProperties() to get info.
	GPU *wgpu.Adapter

	// options passed in during config
	UserOpts *GPUOpts

	// set of enabled options set post-Config
	EnabledOpts GPUOpts

	// name of the physical GPU device
	DeviceName string

	// name of application -- set during Config and used in init of GPU
	AppName string

	// // version of WebGPU API to target
	// APIVersion vk.Version

	// // version of application -- optional
	// AppVersion vk.Version

	// use Add method to add required instance extentions prior to calling Config
	// InstanceExts []string

	// use Add method to add required device extentions prior to calling Config
	// DeviceExts []string

	// set Add method to add required validation layers prior to calling Config
	// ValidationLayers []string

	// physical device features required -- set per platform as needed
	// DeviceFeaturesNeeded *vk.PhysicalDeviceVulkan12Features

	// this is used for computing, not graphics
	Compute bool

	// our custom debug callback
	// DebugCallback vk.DebugReportCallback

	// properties of physical hardware -- populated after Config
	// GPUProperties vk.PhysicalDeviceProperties

	// features of physical hardware -- populated after Config
	// GPUFeats vk.PhysicalDeviceFeatures

	// properties of device memory -- populated after Config
	// MemoryProperties vk.PhysicalDeviceMemoryProperties

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
	// if Debug {
	// 	gp.AddValidationLayer("VK_LAYER_KHRONOS_validation")
	// 	gp.AddInstanceExt("VK_EXT_debug_report") // note _utils is not avail yet
	// }

	// Select instance extensions
	// requiredInstanceExts := SafeStrings(gp.InstanceExts)
	// actualInstanceExts, err := InstanceExts()
	// IfPanic(err)
	// instanceExts, missing := CheckExisting(actualInstanceExts, requiredInstanceExts)
	// if missing > 0 {
	// 	log.Println("vgpu: warning: missing", missing, "required instance extensions during Config")
	// }
	// if Debug {
	// 	log.Printf("vgpu: enabling %d instance extensions", len(instanceExts))
	// }

	// Select instance layers
	// var validationLayers []string
	// if len(gp.ValidationLayers) > 0 {
	// 	requiredValidationLayers := SafeStrings(gp.ValidationLayers)
	// 	actualValidationLayers, err := ValidationLayers()
	// 	IfPanic(err)
	// 	validationLayers, missing = CheckExisting(actualValidationLayers, requiredValidationLayers)
	// 	if missing > 0 {
	// 		log.Println("vgpu: warning: missing", missing, "required validation layers during Config")
	// 	}
	// }

	gp.Instance = wgpu.CreateInstance(nil)

	gpus := gp.Instance.EnumerateAdapters(nil)
	gpIndex := gp.SelectGPU(gpus, gpuCount)
	gp.Adapter = gpus[gpIndex]
	props := gp.Adapter.AdapterProperties()
	gp.DeviceName = props.Name

	// vk.GetPhysicalDeviceFeatures(gp.GPU, &gp.GPUFeats)
	// gp.GPUFeats.Deref()
	// if !gp.CheckGPUOpts(&gp.GPUFeats, gp.UserOpts, true) {
	// 	return errors.New("vgpu: fatal config error found, see messages above")
	// }
	//
	// vk.GetPhysicalDeviceProperties(gp.GPU, &gp.GPUProperties)
	// gp.GPUProperties.Deref()
	// gp.GPUProperties.Limits.Deref()
	// vk.GetPhysicalDeviceMemoryProperties(gp.GPU, &gp.MemoryProperties)
	// gp.MemoryProperties.Deref()

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

func (gp *GPU) SelectGPU(gpus []wgpu.Adapter) int {
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
			// type AdapterProperties struct {
			// 	VendorId          uint32
			// 	VendorName        string
			// 	Architecture      string
			// 	DeviceId          uint32
			// 	Name              string
			// 	DriverDescription string
			// 	AdapterType       AdapterType
			// 	BackendType       BackendType
			// }

			props := gpus[gi].AdapterProperties()
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

	devNm := ""
	maxSz := 0
	maxIndex := 0
	for gi := range n {
		// note: we could potentially check for the optional features here
		// but generally speaking the discrete device is going to be the most
		// feature-full, so the practical benefit is unlikely to be significant.
		props := gpus[gi].AdapterProperties()
		dnm := props.Name
		if props.AdapterType == wgpu.AdapterType_DiscreteGPU {
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

// Destroy releases GPU resources -- call after everything else has been destroyed
func (gp *GPU) Destroy() {
	if gp.GPU != nil {
		gp.GPU = nil
	}
	if gp.Adapter != nil {
		gp.Adapter.Release()
		gp.Adapter = nil
	}
	if gp.Instance != nil {
		gp.Instance.Release()
		gp.Instance = nil
	}
}

// NewComputeSystem returns a new system initialized for this GPU,
// for Compute, not graphics functionality.
func (gp *GPU) NewComputeSystem(name string) *System {
	sy := &System{}
	sy.InitCompute(gp, name)
	return sy
}

// NewGraphicsSystem returns a new system initialized for this GPU,
// for graphics functionality, using Device from the Surface or
// RenderFrame depending on the target of rendering.
func (gp *GPU) NewGraphicsSystem(name string, dev *Device) *System {
	sy := &System{}
	sy.InitGraphics(gp, name, dev)
	return sy
}

// NewDevice returns a new device for given GPU.
// It gets the Queue for this device.
func (gp *GPU) NewDevice() (*Device, error) {
	return NewDevice(gp)
}

// PropertiesString returns a human-readable summary of the GPU properties.
func (gp *GPU) PropertiesString(print bool) string {
	ps := "\n\n######## GPU Properties\n"
	// prs := reflectx.StringJSON(&gp.GPUProperties)
	// devnm := `  "DeviceName": `
	// ps += prs[:strings.Index(prs, devnm)]
	// ps += devnm + string(gp.GPUProperties.DeviceName[:]) + "\n"
	// ps += prs[strings.Index(prs, `  "Limits":`):]
	// // ps += "\n\n######## GPU Memory Properties\n" // not really useful
	// // ps += reflectx.StringJSON(&gp.MemoryProperties)
	// ps += "\n"
	// if print {
	// 	fmt.Println(ps)
	// }
	return ps
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
