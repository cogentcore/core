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

// Platform represents the platform
type Platform struct {
	Context *Context

	Instance vk.Instance
	Gpu      vk.PhysicalDevice
	Device   vk.Device

	GraphicsQueueIndex uint32
	PresentQueueIndex  uint32
	PresQueue          vk.Queue
	GraphicsQueue      vk.Queue

	GpuProperties    vk.PhysicalDeviceProperties
	MemoryProperties vk.PhysicalDeviceMemoryProperties

	Surface       vk.Surface
	DebugCallback vk.DebugReportCallback
}

func NewPlatform(app App) (p *Platform, err error) {
	// defer CheckErr(&err)
	p = &Platform{
		Context: &Context{
			// TODO: make configurable
			// defines count of slots allocated in swapchain
			FrameLag: 3,
		},
	}
	p.Context.Platform = p

	// Select instance extensions
	requiredInstanceExts := SafeStrings(app.InstanceExts())
	actualInstanceExts, err := InstanceExts()
	IfPanic(err)
	instanceExts, missing := CheckExisting(actualInstanceExts, requiredInstanceExts)
	if missing > 0 {
		log.Println("vulkan warning: missing", missing, "required instance extensions during init")
	}
	log.Printf("vulkan: enabling %d instance extensions", len(instanceExts))

	// Select instance layers
	var validationLayers []string
	if iface, ok := app.(AppLayers); ok {
		requiredValidationLayers := SafeStrings(iface.Layers())
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
			ApiVersion:         uint32(app.APIVersion()),
			ApplicationVersion: uint32(app.AppVersion()),
			PApplicationName:   SafeString(app.AppName()),
			PEngineName:        "vulkango.com\x00",
		},
		EnabledExtensionCount:   uint32(len(instanceExts)),
		PpEnabledExtensionNames: instanceExts,
		EnabledLayerCount:       uint32(len(validationLayers)),
		PpEnabledLayerNames:     validationLayers,
	}, nil, &instance)
	IfPanic(NewError(ret))
	p.Instance = instance
	vk.InitInstance(instance)

	if app.Debug() {
		// Register a debug callback
		ret := vk.CreateDebugReportCallback(instance, &vk.DebugReportCallbackCreateInfo{
			SType:       vk.StructureTypeDebugReportCallbackCreateInfo,
			Flags:       vk.DebugReportFlags(vk.DebugReportErrorBit | vk.DebugReportWarningBit),
			PfnCallback: dbgCallbackFunc,
		}, nil, &p.DebugCallback)
		IfPanic(NewError(ret))
		log.Println("vulkan: DebugReportCallback enabled by application")
	}

	// Find a suitable GPU
	var gpuCount uint32
	ret = vk.EnumeratePhysicalDevices(p.Instance, &gpuCount, nil)
	IfPanic(NewError(ret))
	if gpuCount == 0 {
		return nil, errors.New("vulkan error: no GPU devices found")
	}
	gpus := make([]vk.PhysicalDevice, gpuCount)
	ret = vk.EnumeratePhysicalDevices(p.Instance, &gpuCount, gpus)
	IfPanic(NewError(ret))
	// get the first one, multiple GPUs not supported yet
	p.Gpu = gpus[0]
	vk.GetPhysicalDeviceProperties(p.Gpu, &p.GpuProperties)
	p.GpuProperties.Deref()
	vk.GetPhysicalDeviceMemoryProperties(p.Gpu, &p.MemoryProperties)
	p.MemoryProperties.Deref()

	// Select device extensions
	requiredDeviceExts := SafeStrings(app.DeviceExts())
	actualDeviceExts, err := DeviceExts(p.Gpu)
	IfPanic(err)
	deviceExts, missing := CheckExisting(actualDeviceExts, requiredDeviceExts)
	if missing > 0 {
		log.Println("vulkan warning: missing", missing, "required device extensions during init")
	}
	log.Printf("vulkan: enabling %d device extensions", len(deviceExts))

	// Make sure the surface is here if required
	mode := app.Mode()
	if mode.Has(Present) { // so, a surface is required and provided
		p.Surface = app.Surface(p.Instance)
		if p.Surface == vk.NullSurface {
			return nil, errors.New("vulkan error: surface required but not provided")
		}
	}

	// Get queue family properties
	var queueCount uint32
	vk.GetPhysicalDeviceQueueFamilyProperties(p.Gpu, &queueCount, nil)
	queueProperties := make([]vk.QueueFamilyProperties, queueCount)
	vk.GetPhysicalDeviceQueueFamilyProperties(p.Gpu, &queueCount, queueProperties)
	if queueCount == 0 { // probably should try another GPU
		return nil, errors.New("vulkan error: no queue families found on GPU 0")
	}

	// Find a suitable queue family for the target Vulkan mode
	var graphicsFound bool
	var presentFound bool
	var separateQueue bool
	for i := uint32(0); i < queueCount; i++ {
		var (
			required        vk.QueueFlags
			supportsPresent vk.Bool32
			needsPresent    bool
		)
		if graphicsFound {
			// looking for separate present queue
			separateQueue = true
			vk.GetPhysicalDeviceSurfaceSupport(p.Gpu, i, p.Surface, &supportsPresent)
			if supportsPresent.B() {
				p.PresentQueueIndex = i
				presentFound = true
				break
			}
		}
		if mode.Has(Compute) {
			required |= vk.QueueFlags(vk.QueueComputeBit)
		}
		if mode.Has(Graphics) {
			required |= vk.QueueFlags(vk.QueueGraphicsBit)
		}
		if mode.Has(Present) {
			needsPresent = true
			vk.GetPhysicalDeviceSurfaceSupport(p.Gpu, i, p.Surface, &supportsPresent)
		}
		queueProperties[i].Deref()
		if queueProperties[i].QueueFlags&required != 0 {
			if !needsPresent || (needsPresent && supportsPresent.B()) {
				p.GraphicsQueueIndex = i
				graphicsFound = true
				break
			} else if needsPresent {
				p.GraphicsQueueIndex = i
				graphicsFound = true
				// need present, but this one doesn't support
				// continue lookup
			}
		}
	}
	if separateQueue && !presentFound {
		err := errors.New("vulkan error: could not found separate queue with present capabilities")
		return nil, err
	}
	if !graphicsFound {
		err := errors.New("vulkan error: could not find a suitable queue family for the target Vulkan mode")
		return nil, err
	}

	// Create a Vulkan device
	queueInfos := []vk.DeviceQueueCreateInfo{{
		SType:            vk.StructureTypeDeviceQueueCreateInfo,
		QueueFamilyIndex: p.GraphicsQueueIndex,
		QueueCount:       1,
		PQueuePriorities: []float32{1.0},
	}}
	if separateQueue {
		queueInfos = append(queueInfos, vk.DeviceQueueCreateInfo{
			SType:            vk.StructureTypeDeviceQueueCreateInfo,
			QueueFamilyIndex: p.PresentQueueIndex,
			QueueCount:       1,
			PQueuePriorities: []float32{1.0},
		})
	}

	var device vk.Device
	ret = vk.CreateDevice(p.Gpu, &vk.DeviceCreateInfo{
		SType:                   vk.StructureTypeDeviceCreateInfo,
		QueueCreateInfoCount:    uint32(len(queueInfos)),
		PQueueCreateInfos:       queueInfos,
		EnabledExtensionCount:   uint32(len(deviceExts)),
		PpEnabledExtensionNames: deviceExts,
		EnabledLayerCount:       uint32(len(validationLayers)),
		PpEnabledLayerNames:     validationLayers,
	}, nil, &device)
	IfPanic(NewError(ret))
	p.Device = device
	p.Context.Device = device
	app.Init(p.Context)

	var queue vk.Queue
	vk.GetDeviceQueue(p.Device, p.GraphicsQueueIndex, 0, &queue)
	p.GraphicsQueue = queue

	if mode.Has(Present) { // init a swapchain for surface
		if separateQueue {
			var presentQueue vk.Queue
			vk.GetDeviceQueue(p.Device, p.PresentQueueIndex, 0, &presentQueue)
			p.PresQueue = presentQueue
		}
		p.Context.PreparePresent()

		dimensions := &SwapchainDims{
			// some default preferences here
			Width: 640, Height: 480,
			Format: vk.FormatB8g8r8a8Unorm,
		}
		if iface, ok := app.(AppSwapchainDims); ok {
			dimensions = iface.SwapchainDims()
		}
		p.Context.PrepareSwapchain(p.Gpu, p.Surface, dimensions)
	}
	if iface, ok := app.(AppContextPrepare); ok {
		p.Context.OnPrepare = iface.ContextPrepare
	}
	if iface, ok := app.(AppContextCleanup); ok {
		p.Context.OnCleanup = iface.ContextCleanup
	}
	if iface, ok := app.(AppContextInvalidate); ok {
		p.Context.OnInvalidate = iface.ContextInvalidate
	}
	if mode.Has(Present) {
		p.Context.Prepare(false)
	}
	return p, nil
}

func (p *Platform) HasSeparatePresentQueue() bool {
	return p.PresentQueueIndex != p.GraphicsQueueIndex
}

func (p *Platform) PresentQueue() vk.Queue {
	if p.GraphicsQueueIndex != p.PresentQueueIndex {
		return p.PresQueue
	}
	return p.GraphicsQueue
}

func (p *Platform) Destroy() {
	if p.Device != nil {
		vk.DeviceWaitIdle(p.Device)
	}
	p.Context.Destroy()
	p.Context = nil
	if p.Surface != vk.NullSurface {
		vk.DestroySurface(p.Instance, p.Surface, nil)
		p.Surface = vk.NullSurface
	}
	if p.Device != nil {
		vk.DestroyDevice(p.Device, nil)
		p.Device = nil
	}
	if p.DebugCallback != vk.NullDebugReportCallback {
		vk.DestroyDebugReportCallback(p.Instance, p.DebugCallback, nil)
	}
	if p.Instance != nil {
		vk.DestroyInstance(p.Instance, nil)
		p.Instance = nil
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
