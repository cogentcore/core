// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is initially adapted from https://github.com/vulkan-go/asche
// Copyright © 2017 Maxim Kupriianov <max@kc.vc>, under the MIT License

package vgpu

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"unsafe"

	"github.com/goki/ki/kit"
	vk "github.com/goki/vulkan"
	"goki.dev/vgpu/vkinit"
)

// Key docs: https://gpuopen.com/learn/understanding-vulkan-objects/

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

	// handle for the vulkan driver instance
	Instance vk.Instance `desc:"handle for the vulkan driver instance"`

	// handle for the vulkan physical GPU hardware
	GPU vk.PhysicalDevice `desc:"handle for the vulkan physical GPU hardware"`

	// options passed in during config
	UserOpts *GPUOpts `desc:"options passed in during config"`

	// set of enabled options set post-Config
	EnabledOpts GPUOpts `desc:"set of enabled options set post-Config"`

	// name of the physical GPU device
	DeviceName string `desc:"name of the physical GPU device"`

	// name of application -- set during Config and used in init of GPU
	AppName string `desc:"name of application -- set during Config and used in init of GPU"`

	// version of vulkan API to target
	APIVersion vk.Version `desc:"version of vulkan API to target"`

	// version of application -- optional
	AppVersion vk.Version `desc:"version of application -- optional"`

	// use Add method to add required instance extentions prior to calling Config
	InstanceExts []string `desc:"use Add method to add required instance extentions prior to calling Config"`

	// use Add method to add required device extentions prior to calling Config
	DeviceExts []string `desc:"use Add method to add required device extentions prior to calling Config"`

	// set Add method to add required validation layers prior to calling Config
	ValidationLayers []string `desc:"set Add method to add required validation layers prior to calling Config"`

	// this is used for computing, not graphics
	Compute bool `desc:"this is used for computing, not graphics"`

	// our custom debug callback
	DebugCallback vk.DebugReportCallback `desc:"our custom debug callback"`

	// properties of physical hardware -- populated after Config
	GPUProps vk.PhysicalDeviceProperties `desc:"properties of physical hardware -- populated after Config"`

	// features of physical hardware -- populated after Config
	GPUFeats vk.PhysicalDeviceFeatures `desc:"features of physical hardware -- populated after Config"`

	// properties of device memory -- populated after Config
	MemoryProps vk.PhysicalDeviceMemoryProperties `desc:"properties of device memory -- populated after Config"`

	// maximum number of compute threads per compute shader invokation, for a 1D number of threads per Warp, which is generally greater than MaxComputeWorkGroup[0], which allows for the [1] and [2] maxima as well.  This is not defined anywhere in the formal spec, unfortunately, but has been determined empirically for Mac and NVIDIA which are two of the most relevant use-cases.  If not a known case, the MaxComputeWorkGroup[0] value is used, which can significantly slow down compute processing if more could actually be used.  Please file an issue or PR for other GPUs with known larger values.
	MaxComputeWorkGroupCount1D int `desc:"maximum number of compute threads per compute shader invokation, for a 1D number of threads per Warp, which is generally greater than MaxComputeWorkGroup[0], which allows for the [1] and [2] maxima as well.  This is not defined anywhere in the formal spec, unfortunately, but has been determined empirically for Mac and NVIDIA which are two of the most relevant use-cases.  If not a known case, the MaxComputeWorkGroup[0] value is used, which can significantly slow down compute processing if more could actually be used.  Please file an issue or PR for other GPUs with known larger values."`

	// [view: -] platform-specific PNext for CreateDevice call
	PlatformDeviceNext unsafe.Pointer `view:"-" desc:"platform-specific PNext for CreateDevice call"`
}

// InitNoDisplay initializes vulkan system for a purely compute-based
// or headless operation, without any display (i.e., without using glfw).
// Call before doing any vgpu stuff.
// Loads the vulkan library and sets the Vulkan instance proc addr and calls Init.
// IMPORTANT: must be called on the main initial thread!
func InitNoDisplay() error {

	err := vkinit.LoadVulkan()
	if err != nil {
		log.Println(err)
		return err
	}
	return nil
}

// Defaults sets up default parameters, with the graphics flag
// determining whether graphics-relevant items are added.
func (gp *GPU) Defaults(graphics bool) {
	gp.APIVersion = vk.Version(vk.MakeVersion(1, 2, 0))
	gp.AppVersion = vk.Version(vk.MakeVersion(1, 0, 0))
	// TODO: these don't work on mobile, but might be needed on desktop
	// gp.DeviceExts = []string{"VK_EXT_descriptor_indexing"}
	// gp.InstanceExts = []string{"VK_KHR_get_physical_device_properties2"}
	if graphics {
		gp.DeviceExts = append(gp.DeviceExts, []string{"VK_KHR_swapchain"}...)
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

// FindString returns index of string if in list, else -1
func FindString(str string, strs []string) int {
	for i, s := range strs {
		if str == s {
			return i
		}
	}
	return -1
}

// AddInstanceExt adds given extension(s), only if not already set
// returns true if added.
func (gp *GPU) AddInstanceExt(ext ...string) bool {
	for _, ex := range ext {
		i := FindString(ex, gp.InstanceExts)
		if i >= 0 {
			continue
		}
		gp.InstanceExts = append(gp.InstanceExts, ex)
	}
	return true
}

// AddDeviceExt adds given extension(s), only if not already set
// returns true if added.
func (gp *GPU) AddDeviceExt(ext ...string) bool {
	for _, ex := range ext {
		i := FindString(ex, gp.DeviceExts)
		if i >= 0 {
			continue
		}
		gp.DeviceExts = append(gp.DeviceExts, ex)
	}
	return true
}

// AddValidationLayer adds given validation layer, only if not already set
// returns true if added.
func (gp *GPU) AddValidationLayer(ext string) bool {
	i := FindString(ext, gp.ValidationLayers)
	if i >= 0 {
		return false
	}
	gp.ValidationLayers = append(gp.ValidationLayers, ext)
	return true
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
	if Debug {
		gp.AddValidationLayer("VK_LAYER_KHRONOS_validation")
		gp.AddInstanceExt("VK_EXT_debug_report") // note _utils is not avail yet
	}

	// Select instance extensions
	requiredInstanceExts := SafeStrings(gp.InstanceExts)
	actualInstanceExts, err := InstanceExts()
	IfPanic(err)
	instanceExts, missing := CheckExisting(actualInstanceExts, requiredInstanceExts)
	if missing > 0 {
		log.Println("vgpu: warning: missing", missing, "required instance extensions during Config")
	}
	if Debug {
		log.Printf("vgpu: enabling %d instance extensions", len(instanceExts))
	}

	// Select instance layers
	var validationLayers []string
	if len(gp.ValidationLayers) > 0 {
		requiredValidationLayers := SafeStrings(gp.ValidationLayers)
		actualValidationLayers, err := ValidationLayers()
		IfPanic(err)
		validationLayers, missing = CheckExisting(actualValidationLayers, requiredValidationLayers)
		if missing > 0 {
			log.Println("vgpu: warning: missing", missing, "required validation layers during Config")
		}
	}

	// Create instance
	var instance vk.Instance
	ret := vk.CreateInstance(&vk.InstanceCreateInfo{
		SType: vk.StructureTypeInstanceCreateInfo,
		PApplicationInfo: &vk.ApplicationInfo{
			SType:              vk.StructureTypeApplicationInfo,
			ApiVersion:         uint32(gp.APIVersion),
			ApplicationVersion: uint32(gp.AppVersion),
			PApplicationName:   SafeString(gp.AppName),
			PEngineName:        "vgpu\x00",
		},
		EnabledExtensionCount:   uint32(len(instanceExts)),
		PpEnabledExtensionNames: instanceExts,
		EnabledLayerCount:       uint32(len(validationLayers)),
		PpEnabledLayerNames:     validationLayers,
		Flags:                   vk.InstanceCreateFlags(vk.InstanceCreateEnumeratePortabilityBit),
	}, nil, &instance)
	IfPanic(NewError(ret))
	gp.Instance = instance

	vk.InitInstance(instance)

	// Find a suitable GPU
	var gpuCountU uint32
	ret = vk.EnumeratePhysicalDevices(gp.Instance, &gpuCountU, nil)
	IfPanic(NewError(ret))
	if gpuCountU == 0 {
		return errors.New("vgpu: error: no GPU devices found")
	}
	gpuCount := int(gpuCountU)
	gpus := make([]vk.PhysicalDevice, gpuCount)
	ret = vk.EnumeratePhysicalDevices(gp.Instance, &gpuCountU, gpus)
	IfPanic(NewError(ret))

	gpIdx := gp.SelectGPU(gpus, gpuCount)
	gp.GPU = gpus[gpIdx]

	vk.GetPhysicalDeviceFeatures(gp.GPU, &gp.GPUFeats)
	gp.GPUFeats.Deref()
	if !gp.CheckGPUOpts(&gp.GPUFeats, gp.UserOpts, true) {
		return errors.New("vgpu: fatal config error found, see messages above")
	}

	vk.GetPhysicalDeviceProperties(gp.GPU, &gp.GPUProps)
	gp.GPUProps.Deref()
	gp.GPUProps.Limits.Deref()
	vk.GetPhysicalDeviceMemoryProperties(gp.GPU, &gp.MemoryProps)
	gp.MemoryProps.Deref()

	gp.MaxComputeWorkGroupCount1D = int(gp.GPUProps.Limits.MaxComputeWorkGroupCount[0])
	// note: unclear what the limit is here.
	// if gp.MaxComputeWorkGroupCount1D == 0 { // otherwise set per-platform in defaults (DARWIN)
	// if strings.Contains(gp.DeviceName, "NVIDIA") {
	// 	// according to: https://vulkan.gpuinfo.org/displaydevicelimit.php?name=maxComputeWorkGroupInvocations&platform=all
	// 	// all NVIDIA are either 1 << 31 or -1 of that.
	// 	gp.MaxComputeWorkGroupCount1D = (1 << 31) - 1 // according to vgpu
	// } else {
	// note: if known to be higher for any specific case, please file an issue or PR
	// }
	// }

	// Select device extensions
	requiredDeviceExts := SafeStrings(gp.DeviceExts)
	actualDeviceExts, err := DeviceExts(gp.GPU)
	IfPanic(err)
	deviceExts, missing := CheckExisting(actualDeviceExts, requiredDeviceExts)
	if missing > 0 {
		log.Println("vgpu: warning: missing", missing, "required device extensions during Config")
	}
	if Debug {
		log.Printf("vgpu: enabling %d device extensions", len(deviceExts))
	}

	if Debug {
		var debugCallback vk.DebugReportCallback
		// Register a debug callback
		ret := vk.CreateDebugReportCallback(gp.Instance, &vk.DebugReportCallbackCreateInfo{
			SType:       vk.StructureTypeDebugReportCallbackCreateInfo,
			Flags:       vk.DebugReportFlags(vk.DebugReportErrorBit | vk.DebugReportWarningBit | vk.DebugReportInformationBit),
			PfnCallback: dbgCallbackFunc,
		}, nil, &debugCallback)
		IfPanic(NewError(ret))
		log.Println("vgpu: DebugReportCallback enabled by application")
		gp.DebugCallback = debugCallback
	}

	return nil
}

func (gp *GPU) GetDeviceName(props *vk.PhysicalDeviceProperties, idx int) string {
	nm := CleanString(string(props.DeviceName[:]))
	return fmt.Sprintf("%s: id=%d idx=%d", nm, props.DeviceID, idx)
}

func (gp *GPU) SelectGPU(gpus []vk.PhysicalDevice, gpuCount int) int {
	if gpuCount == 1 {
		var props vk.PhysicalDeviceProperties
		vk.GetPhysicalDeviceProperties(gpus[0], &props)
		props.Deref()
		gp.DeviceName = gp.GetDeviceName(&props, 0)
		if Debug {
			log.Printf("vgpu: selected only device named: %s\n", gp.DeviceName)
		}
		return 0
	}
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
		if err == nil && idx >= 0 && idx < gpuCount {
			curIdx := 0
			for gi := 0; gi < gpuCount; gi++ {
				var props vk.PhysicalDeviceProperties
				vk.GetPhysicalDeviceProperties(gpus[gi], &props)
				props.Deref()
				if props.DeviceType == vk.PhysicalDeviceTypeDiscreteGpu {
					if curIdx == idx {
						gp.DeviceName = gp.GetDeviceName(&props, gi)
						if Debug {
							log.Printf("vgpu: selected device named: %s, specified by index in *_DEVICE_SELECT environment variable, index: %d\n", gp.DeviceName, gi)
						}
						return gi
					} else {
						curIdx++
					}
				}
			}
			panic(fmt.Sprintf("vgpu: device specified by index in *_DEVICE_SELECT environment variable, index: %d, NOT FOUND\n", idx))
		}
		for gi := 0; gi < gpuCount; gi++ {
			var props vk.PhysicalDeviceProperties
			vk.GetPhysicalDeviceProperties(gpus[gi], &props)
			props.Deref()
			if bytes.Contains(props.DeviceName[:], []byte(trgDevNm)) {
				devNm := gp.GetDeviceName(&props, gi)
				if Debug {
					log.Printf("vgpu: selected device named: %s, specified in *_DEVICE_SELECT environment variable, index: %d\n", devNm, gi)
				}
				gp.DeviceName = devNm
				return gi
			}
		}
		if Debug {
			log.Printf("vgpu: unable to find device named: %s, specified in *_DEVICE_SELECT environment variable\n", trgDevNm)
		}
	}

	devNm := ""
	maxSz := 0
	maxIdx := 0
	for gi := 0; gi < gpuCount; gi++ {
		// note: we could potentially check for the optional features here
		// but generally speaking the discrete device is going to be the most
		// feature-full, so the practical benefit is unlikely to be significant.
		var props vk.PhysicalDeviceProperties
		vk.GetPhysicalDeviceProperties(gpus[gi], &props)
		props.Deref()
		dnm := gp.GetDeviceName(&props, gi)
		if props.DeviceType == vk.PhysicalDeviceTypeDiscreteGpu {
			var memProps vk.PhysicalDeviceMemoryProperties
			vk.GetPhysicalDeviceMemoryProperties(gpus[gi], &memProps)
			memProps.Deref()
			if Debug {
				log.Printf("vgpu: %d: evaluating discrete device named: %s\n", gi, dnm)
			}
			for mi := uint32(0); mi < memProps.MemoryHeapCount; mi++ {
				heap := &memProps.MemoryHeaps[mi]
				heap.Deref()
				// if heap.Flags&vk.MemoryHeapFlags(vk.MemoryHeapDeviceLocalBit) != 0 {
				sz := int(heap.Size)
				if sz > maxSz {
					devNm = gp.GetDeviceName(&props, gi)
					maxSz = sz
					maxIdx = gi
				}
				// }
			}
		} else {
			if Debug {
				log.Printf("vgpu: %d: skipping device named: %s -- not discrete\n", gi, dnm)
			}
		}
	}
	gp.DeviceName = devNm
	if Debug {
		log.Printf("vgpu: %d selected device named: %s, memory size: %d\n", maxIdx, devNm, maxSz)
	}

	return maxIdx
}

// Destroy destroys GPU resources -- call after everything else has been destroyed
func (gp *GPU) Destroy() {
	if gp.DebugCallback != vk.NullDebugReportCallback {
		vk.DestroyDebugReportCallback(gp.Instance, gp.DebugCallback, nil)
	}
	if gp.Instance != nil {
		vk.DestroyInstance(gp.Instance, nil)
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

// PropsString returns a human-readable summary of the GPU properties.
func (gp *GPU) PropsString(print bool) string {
	ps := "\n\n######## GPU Props\n"
	prs := kit.StringJSON(&gp.GPUProps)
	devnm := `  "DeviceName": `
	ps += prs[:strings.Index(prs, devnm)]
	ps += devnm + string(gp.GPUProps.DeviceName[:]) + "\n"
	ps += prs[strings.Index(prs, `  "Limits":`):]
	// ps += "\n\n######## GPU Memory Props\n" // not really useful
	// ps += kit.StringJSON(&gp.MemoryProps)
	ps += "\n"
	if print {
		fmt.Println(ps)
	}
	return ps
}

func dbgCallbackFunc(flags vk.DebugReportFlags, objectType vk.DebugReportObjectType,
	object uint64, location uint64, messageCode int32, pLayerPrefix string,
	pMessage string, pUserData unsafe.Pointer) vk.Bool32 {

	switch {
	case flags&vk.DebugReportFlags(vk.DebugReportInformationBit) != 0:
		if !(strings.Contains(pLayerPrefix, "Loader") && strings.Contains(pMessage, "Device Extension")) {
			log.Printf("INFORMATION: [%s] Code %d : %s", pLayerPrefix, messageCode, pMessage)
		}
	case flags&vk.DebugReportFlags(vk.DebugReportWarningBit) != 0:
		log.Printf("WARNING: [%s] Code %d : %s", pLayerPrefix, messageCode, pMessage)
	case flags&vk.DebugReportFlags(vk.DebugReportPerformanceWarningBit) != 0:
		log.Printf("PERFORMANCE WARNING: [%s] Code %d : %s", pLayerPrefix, messageCode, pMessage)
	case flags&vk.DebugReportFlags(vk.DebugReportErrorBit) != 0:
		log.Printf("ERROR: [%s] Code %d : %s", pLayerPrefix, messageCode, pMessage)
	case flags&vk.DebugReportFlags(vk.DebugReportDebugBit) != 0:
		log.Printf("DEBUG: [%s] Code %d : %s", pLayerPrefix, messageCode, pMessage)
	default:
		if !(strings.Contains(pLayerPrefix, "Loader") && strings.Contains(pMessage, "Device Extension")) {
			log.Printf("INFORMATION: [%s] Code %d : %s", pLayerPrefix, messageCode, pMessage)
		}
	}
	return vk.Bool32(vk.False)
}

// InstanceExts gets a list of instance extensions available on the platform.
func InstanceExts() (names []string, err error) {
	defer CheckErr(&err)

	var count uint32
	ret := vk.EnumerateInstanceExtensionProperties("", &count, nil)
	IfPanic(NewError(ret))
	list := make([]vk.ExtensionProperties, count)
	ret = vk.EnumerateInstanceExtensionProperties("", &count, list)
	IfPanic(NewError(ret))
	for _, ext := range list {
		ext.Deref()
		names = append(names, vk.ToString(ext.ExtensionName[:]))
	}
	return names, err
}

// DeviceExts gets a list of instance extensions available on the provided physical device.
func DeviceExts(gpu vk.PhysicalDevice) (names []string, err error) {
	defer CheckErr(&err)

	var count uint32
	ret := vk.EnumerateDeviceExtensionProperties(gpu, "", &count, nil)
	IfPanic(NewError(ret))
	list := make([]vk.ExtensionProperties, count)
	ret = vk.EnumerateDeviceExtensionProperties(gpu, "", &count, list)
	IfPanic(NewError(ret))
	for _, ext := range list {
		ext.Deref()
		names = append(names, vk.ToString(ext.ExtensionName[:]))
	}
	return names, err
}

// ValidationLayers gets a list of validation layers available on the platform.
func ValidationLayers() (names []string, err error) {
	defer CheckErr(&err)

	var count uint32
	ret := vk.EnumerateInstanceLayerProperties(&count, nil)
	IfPanic(NewError(ret))
	list := make([]vk.LayerProperties, count)
	ret = vk.EnumerateInstanceLayerProperties(&count, list)
	IfPanic(NewError(ret))
	for _, layer := range list {
		layer.Deref()
		names = append(names, vk.ToString(layer.LayerName[:]))
	}
	return names, err
}
