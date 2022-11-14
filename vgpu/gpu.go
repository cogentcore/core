// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is initially adapted from https://github.com/vulkan-go/asche
// Copyright © 2017 Maxim Kupriianov <max@kc.vc>, under the MIT License

package vgpu

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"unsafe"

	"github.com/goki/ki/kit"
	vk "github.com/goki/vulkan"
)

// Key docs: https://gpuopen.com/learn/understanding-vulkan-objects/

// Debug is a global flag for turning on debug mode
// Set to true prior to initialization of VGPU.
// It is ignored if false (won't turn off Debug if set by other means).
var Debug = false

// TheGPU is a global for the GPU
var TheGPU *GPU

// GPU represents the GPU hardware
type GPU struct {
	Instance         vk.Instance            `desc:"handle for the vulkan driver instance"`
	GPU              vk.PhysicalDevice      `desc:"handle for the vulkan physical GPU hardware"`
	AppName          string                 `desc:"name of application -- set during Config and used in init of GPU"`
	APIVersion       vk.Version             `desc:"version of vulkan API to target"`
	AppVersion       vk.Version             `desc:"version of application -- optional"`
	InstanceExts     []string               `desc:"use Add method to add required instance extentions prior to calling Config"`
	DeviceExts       []string               `desc:"use Add method to add required device extentions prior to calling Config"`
	ValidationLayers []string               `desc:"set Add method to add required validation layers prior to calling Config"`
	Debug            bool                   `desc:"set to true prior to calling Config to enable debug mode"`
	DebugCallback    vk.DebugReportCallback `desc:"our custom debug callback"`

	GPUProps    vk.PhysicalDeviceProperties       `desc:"properties of physical hardware -- populated after Config"`
	MemoryProps vk.PhysicalDeviceMemoryProperties `desc:"properties of device memory -- populated after Config"`
}

// Defaults sets up default parameters, with the graphics flag
// determining whether graphics-relevant items are added.
func (gp *GPU) Defaults(graphics bool) {
	if Debug {
		gp.Debug = true
	}
	gp.APIVersion = vk.Version(vk.MakeVersion(1, 2, 0))
	gp.AppVersion = vk.Version(vk.MakeVersion(1, 0, 0))
	gp.DeviceExts = []string{"VK_EXT_descriptor_indexing"}
	gp.InstanceExts = []string{"VK_KHR_get_physical_device_properties2"}
	if graphics {
		gp.DeviceExts = append(gp.DeviceExts, []string{"VK_KHR_swapchain"}...)
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

// Config
func (gp *GPU) Config(name string) error {
	TheGPU = gp
	if Debug {
		gp.Debug = true
	}

	gp.AppName = name
	if gp.Debug {
		gp.AddValidationLayer("VK_LAYER_KHRONOS_validation")
		gp.AddInstanceExt("VK_EXT_debug_report") // note _utils is not avail yet
	}

	// Select instance extensions
	requiredInstanceExts := SafeStrings(gp.InstanceExts)
	actualInstanceExts, err := InstanceExts()
	IfPanic(err)
	instanceExts, missing := CheckExisting(actualInstanceExts, requiredInstanceExts)
	if missing > 0 {
		log.Println("vulkan warning: missing", missing, "required instance extensions during Config")
	}
	if gp.Debug {
		log.Printf("vulkan: enabling %d instance extensions", len(instanceExts))
	}

	// Select instance layers
	var validationLayers []string
	if len(gp.ValidationLayers) > 0 {
		requiredValidationLayers := SafeStrings(gp.ValidationLayers)
		actualValidationLayers, err := ValidationLayers()
		IfPanic(err)
		validationLayers, missing = CheckExisting(actualValidationLayers, requiredValidationLayers)
		if missing > 0 {
			log.Println("vulkan warning: missing", missing, "required validation layers during Config")
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
	var gpuCount uint32
	ret = vk.EnumeratePhysicalDevices(gp.Instance, &gpuCount, nil)
	IfPanic(NewError(ret))
	if gpuCount == 0 {
		return errors.New("vulkan error: no GPU devices found")
	}
	gpus := make([]vk.PhysicalDevice, gpuCount)
	ret = vk.EnumeratePhysicalDevices(gp.Instance, &gpuCount, gpus)
	IfPanic(NewError(ret))
	// get the first one, multiple GPUs not supported yet
	gp.GPU = gpus[0]
	vk.GetPhysicalDeviceProperties(gp.GPU, &gp.GPUProps)
	gp.GPUProps.Deref()
	gp.GPUProps.Limits.Deref()
	vk.GetPhysicalDeviceMemoryProperties(gp.GPU, &gp.MemoryProps)
	gp.MemoryProps.Deref()

	// Select device extensions
	requiredDeviceExts := SafeStrings(gp.DeviceExts)
	actualDeviceExts, err := DeviceExts(gp.GPU)
	IfPanic(err)
	deviceExts, missing := CheckExisting(actualDeviceExts, requiredDeviceExts)
	if missing > 0 {
		log.Println("vulkan warning: missing", missing, "required device extensions during Config")
	}
	if gp.Debug {
		log.Printf("vulkan: enabling %d device extensions", len(deviceExts))
	}

	if gp.Debug {
		var debugCallback vk.DebugReportCallback
		// Register a debug callback
		ret := vk.CreateDebugReportCallback(gp.Instance, &vk.DebugReportCallbackCreateInfo{
			SType:       vk.StructureTypeDebugReportCallbackCreateInfo,
			Flags:       vk.DebugReportFlags(vk.DebugReportErrorBit | vk.DebugReportWarningBit | vk.DebugReportInformationBit),
			PfnCallback: dbgCallbackFunc,
		}, nil, &debugCallback)
		IfPanic(NewError(ret))
		log.Println("vulkan: DebugReportCallback enabled by application")
		gp.DebugCallback = debugCallback
	}

	return nil
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
		log.Printf("INFORMATION: [%s] Code %d : %s", pLayerPrefix, messageCode, pMessage)
	case flags&vk.DebugReportFlags(vk.DebugReportWarningBit) != 0:
		log.Printf("WARNING: [%s] Code %d : %s", pLayerPrefix, messageCode, pMessage)
	case flags&vk.DebugReportFlags(vk.DebugReportPerformanceWarningBit) != 0:
		log.Printf("PERFORMANCE WARNING: [%s] Code %d : %s", pLayerPrefix, messageCode, pMessage)
	case flags&vk.DebugReportFlags(vk.DebugReportErrorBit) != 0:
		log.Printf("ERROR: [%s] Code %d : %s", pLayerPrefix, messageCode, pMessage)
	case flags&vk.DebugReportFlags(vk.DebugReportDebugBit) != 0:
		log.Printf("DEBUG: [%s] Code %d : %s", pLayerPrefix, messageCode, pMessage)
	default:
		log.Printf("INFORMATION: [%s] Code %d : %s", pLayerPrefix, messageCode, pMessage)
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
