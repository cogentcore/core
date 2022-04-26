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
	vk "github.com/vulkan-go/vulkan"
)

// Key docs: https://gpuopen.com/learn/understanding-vulkan-objects/

// TheGPU is a global for the GPU
var TheGPU *GPU

// GPU represents the GPU hardware
type GPU struct {
	Instance vk.Instance
	GPU      vk.PhysicalDevice
	Device   Device `desc:"generic graphics device, for framebuffer rendering etc"`

	GpuProps    vk.PhysicalDeviceProperties
	MemoryProps vk.PhysicalDeviceMemoryProperties

	DebugCallback vk.DebugReportCallback

	AppName          string `desc:"name of application -- used in init of GPU"`
	APIVersion       vk.Version
	AppVersion       vk.Version
	InstanceExts     []string `desc:"set to required instance exts prior to calling Init"`
	DeviceExts       []string `desc:"set to required device exts prior to calling Init"`
	ValidationLayers []string `desc:"set to required validation layers prior to calling Init"`
	Debug            bool
}

func (gp *GPU) Defaults() {
	gp.APIVersion = vk.Version(vk.MakeVersion(1, 1, 0))
	gp.AppVersion = vk.Version(vk.MakeVersion(1, 0, 0))
	gp.DeviceExts = []string{"VK_KHR_portability_subset"}
	gp.InstanceExts = []string{"VK_KHR_get_physical_device_properties2"}
}

// NewGPU returns a new GPU struct with Defaults set
// configure any additional defaults before calling Init
func NewGPU() *GPU {
	gp := &GPU{}
	gp.Defaults()
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

// AddInstanceExt adds given extension, only if not already set
// returns true if added.
func (gp *GPU) AddInstanceExt(ext string) bool {
	i := FindString(ext, gp.InstanceExts)
	if i >= 0 {
		return false
	}
	gp.InstanceExts = append(gp.InstanceExts, ext)
	return true
}

// AddDeviceExt adds given extension, only if not already set
// returns true if added.
func (gp *GPU) AddDeviceExt(ext string) bool {
	i := FindString(ext, gp.DeviceExts)
	if i >= 0 {
		return false
	}
	gp.DeviceExts = append(gp.DeviceExts, ext)
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

func (gp *GPU) Init(name string, debug bool) error {
	TheGPU = gp

	gp.AppName = name
	gp.Debug = debug
	if debug {
		gp.AddValidationLayer("VK_LAYER_KHRONOS_validation")
		gp.AddInstanceExt("VK_EXT_debug_report")
	}

	// Select instance extensions
	requiredInstanceExts := SafeStrings(gp.InstanceExts)
	actualInstanceExts, err := InstanceExts()
	IfPanic(err)
	instanceExts, missing := CheckExisting(actualInstanceExts, requiredInstanceExts)
	if missing > 0 {
		log.Println("vulkan warning: missing", missing, "required instance extensions during init")
	}
	log.Printf("vulkan: enabling %d instance extensions", len(instanceExts))

	// Select instance layers
	var validationLayers []string
	if len(gp.ValidationLayers) > 0 {
		requiredValidationLayers := SafeStrings(gp.ValidationLayers)
		actualValidationLayers, err := ValidationLayers()
		IfPanic(err)
		validationLayers, missing = CheckExisting(actualValidationLayers, requiredValidationLayers)
		if missing > 0 {
			log.Println("vulkan warning: missing", missing, "required validation layers during init")
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
			PEngineName:        "egpu\x00",
		},
		EnabledExtensionCount:   uint32(len(instanceExts)),
		PpEnabledExtensionNames: instanceExts,
		EnabledLayerCount:       uint32(len(validationLayers)),
		PpEnabledLayerNames:     validationLayers,
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
	vk.GetPhysicalDeviceProperties(gp.GPU, &gp.GpuProps)
	gp.GpuProps.Deref()
	gp.GpuProps.Limits.Deref()
	vk.GetPhysicalDeviceMemoryProperties(gp.GPU, &gp.MemoryProps)
	gp.MemoryProps.Deref()

	// Select device extensions
	requiredDeviceExts := SafeStrings(gp.DeviceExts)
	actualDeviceExts, err := DeviceExts(gp.GPU)
	IfPanic(err)
	deviceExts, missing := CheckExisting(actualDeviceExts, requiredDeviceExts)
	if missing > 0 {
		log.Println("vulkan warning: missing", missing, "required device extensions during init")
	}
	log.Printf("vulkan: enabling %d device extensions", len(deviceExts))

	if gp.Debug {
		var debugCallback vk.DebugReportCallback
		// Register a debug callback
		ret := vk.CreateDebugReportCallback(gp.Instance, &vk.DebugReportCallbackCreateInfo{
			SType:       vk.StructureTypeDebugReportCallbackCreateInfo,
			Flags:       vk.DebugReportFlags(vk.DebugReportErrorBit | vk.DebugReportWarningBit),
			PfnCallback: dbgCallbackFunc,
		}, nil, &debugCallback)
		IfPanic(NewError(ret))
		log.Println("vulkan: DebugReportCallback enabled by application")
		gp.DebugCallback = debugCallback
	}

	return nil
}

// InitGraphicsDevice initializes the generic graphics device
func (gp *GPU) InitGraphicsDevice() error {
	return gp.Device.Init(gp, vk.QueueGraphicsBit)
}

func (gp *GPU) Destroy() {
	if gp.DebugCallback != vk.NullDebugReportCallback {
		vk.DestroyDebugReportCallback(gp.Instance, gp.DebugCallback, nil)
	}
	gp.Device.Destroy()
	if gp.Instance != nil {
		vk.DestroyInstance(gp.Instance, nil)
		gp.Instance = nil
	}
}

// NewSystem returns a new system initialized for this GPU
// compute = make a compute device instead of a Graphics device.
func (gp *GPU) NewSystem(name string, compute bool) *System {
	sy := &System{}
	sy.Init(gp, name, compute)
	return sy
}

func (gp *GPU) PropsString(print bool) string {
	ps := "\n\n######## GPU Props\n"
	prs := kit.StringJSON(&gp.GpuProps)
	devnm := `  "DeviceName": `
	ps += prs[:strings.Index(prs, devnm)]
	ps += devnm + string(gp.GpuProps.DeviceName[:]) + "\n"
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
	object uint64, location uint, messageCode int32, pLayerPrefix string,
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
