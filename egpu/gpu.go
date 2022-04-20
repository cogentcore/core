// Copyright (c) 2022, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is initially adapted from https://github.com/vulkan-go/asche
// Copyright © 2017 Maxim Kupriianov <max@kc.vc>, under the MIT License

package egpu

import (
	"errors"
	"log"
	"unsafe"

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

	APIVersion       vk.Version
	AppVersion       vk.Version
	Name             string
	InstanceExts     []string `desc:"set to required instance exts prior to calling Init"`
	DeviceExts       []string `desc:"set to required device exts prior to calling Init"`
	ValidationLayers []string `desc:"set to required validation layers prior to calling Init"`
	Debug            bool
}

func (gp *GPU) Defaults() {
	gp.APIVersion = vk.Version(vk.MakeVersion(1, 1, 0))
	gp.AppVersion = vk.Version(vk.MakeVersion(1, 0, 0))
}

func (gp *GPU) Init(name string, debug bool) error {
	gp.Name = name
	TheGPU = gp

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
			PApplicationName:   SafeString(gp.Name),
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

	if gp.Debug {
		// Register a debug callback
		ret := vk.CreateDebugReportCallback(instance, &vk.DebugReportCallbackCreateInfo{
			SType:       vk.StructureTypeDebugReportCallbackCreateInfo,
			Flags:       vk.DebugReportFlags(vk.DebugReportErrorBit | vk.DebugReportWarningBit),
			PfnCallback: dbgCallbackFunc,
		}, nil, &gp.DebugCallback)
		IfPanic(NewError(ret))
		log.Println("vulkan: DebugReportCallback enabled by application")
	}

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
