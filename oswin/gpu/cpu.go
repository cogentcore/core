// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based extensively on vulkan-go/asche
// The MIT License (MIT)
// Copyright © 2017 Maxim Kupriianov <max@kc.vc>

package gpu

import (
	"errors"
	"log"

	"github.com/goki/gi/oswin"
	"github.com/vulkan-go/vulkan"
)

// GPUBase provides the base gpu implementation
type GPUBase struct {
	instance vulkan.Instance
	physDev  vulkan.PhysicalDevice
	device   vulkan.Device

	graphicsQueueIndex uint32
	presentQueueIndex  uint32
	presentQueue       vulkan.Queue
	graphicsQueue      vulkan.Queue

	gpuProperties    vulkan.PhysicalDeviceProperties
	memoryProperties vulkan.PhysicalDeviceMemoryProperties
	reqInstanceExts  []string
	reqDeviceExts    []string
	reqValidlayers   []string

	// actual obtained ones
	actInstanceExts []string
	actDeviceExts   []string
	actValidLayers  []string
}

func (gp *GPUBase) MemoryProperties() vulkan.PhysicalDeviceMemoryProperties {
	return gp.memoryProperties
}

func (gp *GPUBase) PhysicalDeviceProperies() vulkan.PhysicalDeviceProperties {
	return gp.gpuProperties
}

func (gp *GPUBase) PhysicalDevice() vulkan.PhysicalDevice {
	return gp.physDev
}

func (gp *GPUBase) GraphicsQueueFamilyIndex() uint32 {
	return gp.graphicsQueueIndex
}

func (gp *GPUBase) PresentQueueFamilyIndex() uint32 {
	return gp.presentQueueIndex
}

func (gp *GPUBase) HasSeparatePresentQueue() bool {
	return gp.presentQueueIndex != gp.graphicsQueueIndex
}

func (gp *GPUBase) GraphicsQueue() vulkan.Queue {
	return gp.graphicsQueue
}

func (gp *GPUBase) PresentQueue() vulkan.Queue {
	if gp.graphicsQueueIndex != gp.presentQueueIndex {
		return gp.presentQueue
	}
	return gp.graphicsQueue
}

func (gp *GPUBase) Instance() vulkan.Instance {
	return gp.instance
}

func (gp *GPUBase) Device() vulkan.Device {
	return gp.device
}

func (gp *GPUBase) VulkanAPIVersion() vulkan.Version {
	return vulkan.Version(vulkan.MakeVersion(1, 0, 0))
}

func (gp *GPUBase) VulkanAppVersion() vulkan.Version {
	return vulkan.Version(vulkan.MakeVersion(1, 0, 0))
}

func (gp *GPUBase) SetReqInstanceExts(exts []string) {
	gp.reqInstanceExts = exts
}

func (gp *GPUBase) ReqInstanceExts() []string {
	return gp.reqInstanceExts
}

func (gp *GPUBase) ActInstanceExts() []string {
	return gp.actInstanceExts
}

func (gp *GPUBase) SetReqDeviceExts(exts []string) {
	gp.reqDeviceExts = exts
}

func (gp *GPUBase) ReqDeviceExts() []string {
	return gp.reqDeviceExts
}

func (gp *GPUBase) ActDeviceExts() []string {
	return gp.actDeviceExts
}

// InstanceExts gets a list of instance extensions available on the platform.
func (gp *GPUBase) InstanceExts() (names []string, err error) {
	var count uint32
	err = NewError(vulkan.EnumerateInstanceExtensionProperties("", &count, nil))
	if err != nil {
		return nil, err
	}
	list := make([]vulkan.ExtensionProperties, count)
	err = NewError(vulkan.EnumerateInstanceExtensionProperties("", &count, list))
	if err != nil {
		return nil, err
	}
	for _, ext := range list {
		ext.Deref()
		names = append(names, vulkan.ToString(ext.ExtensionName[:]))
	}
	return names, err
}

// DeviceExts gets a list of instance extensions available on the provided physical device.
func (gp *GPUBase) DeviceExts() (names []string, err error) {
	var count uint32
	err = NewError(vulkan.EnumerateDeviceExtensionProperties(gp.physDev, "", &count, nil))
	if err != nil {
		return nil, err
	}
	list := make([]vulkan.ExtensionProperties, count)
	err = NewError(vulkan.EnumerateDeviceExtensionProperties(gp.physDev, "", &count, list))
	if err != nil {
		return nil, err
	}
	for _, ext := range list {
		ext.Deref()
		names = append(names, vulkan.ToString(ext.ExtensionName[:]))
	}
	return names, err
}

func (gp *GPUBase) SetReqValidationLayers(lays []string) {
	gp.reqValidlayers = lays
}

func (gp *GPUBase) ReqValidationLayers() []string {
	return gp.reqValidlayers
}

func (gp *GPUBase) ActValidationLayers() []string {
	return gp.actValidLayers
}

// ValidationLayers gets a list of validation layers available on the platform.
func (gp *GPUBase) ValidationLayers() (names []string, err error) {
	var count uint32
	err = NewError(vulkan.EnumerateInstanceLayerProperties(&count, nil))
	if err != nil {
		return nil, err
	}
	list := make([]vulkan.LayerProperties, count)
	err = NewError(vulkan.EnumerateInstanceLayerProperties(&count, list))
	if err != nil {
		return nil, err
	}
	for _, layer := range list {
		layer.Deref()
		names = append(names, vulkan.ToString(layer.LayerName[:]))
	}
	return names, err
}

// DestroyBase destroys base vars -- actual Destroy() method should call
func (gp *GPUBase) DestroyBase() {
	if gp.device != nil {
		vulkan.DeviceWaitIdle(gp.device)
	}
	if gp.device != nil {
		vulkan.DestroyDevice(gp.device, nil)
		gp.device = nil
	}
	// if gp.debugCallback != vulkan.NullDebugReportCallback {
	// 	vulkan.DestroyDebugReportCallback(gp.instance, gp.debugCallback, nil)
	// }
	if gp.instance != nil {
		vulkan.DestroyInstance(gp.instance, nil)
		gp.instance = nil
	}
}

///////////////////////////////////////////////////////
//		Standard Init

// InitBase does all standard initializationn after the required
// extensions and layers have been set
func (gp *GPUBase) InitBase() error {
	err := gp.InitInstance()
	if err != nil {
		return err
	}
	err = gp.InitDevice()
	if err != nil {
		return err
	}
	return err
}

// InitInstance initializes instance
func (gp *GPUBase) InitInstance() error {
	// Select instance extensions
	req := SafeStrings(gp.ReqInstanceExts())
	act, err := gp.InstanceExts()
	if err != nil {
		return err
	}
	missing := 0
	gp.actInstanceExts, missing = CheckExisting(act, req)
	if missing > 0 {
		log.Println("oswin vulkan warning: missing", missing, "required instance extensions during init")
	}
	log.Printf("vulkan: enabling %d instance extensions", len(gp.actInstanceExts))

	// Select validation layers
	req = SafeStrings(gp.ReqValidationLayers())
	act, err = gp.ValidationLayers()
	if err != nil {
		return err
	}
	gp.actValidLayers, missing = CheckExisting(act, req)
	if missing > 0 {
		log.Println("vulkan warning: missing", missing, "required validation layers during init")
	}

	// Create instance
	var inst vulkan.Instance
	err = NewError(vulkan.CreateInstance(&vulkan.InstanceCreateInfo{
		SType: vulkan.StructureTypeInstanceCreateInfo,
		PApplicationInfo: &vulkan.ApplicationInfo{
			SType:              vulkan.StructureTypeApplicationInfo,
			ApiVersion:         uint32(gp.VulkanAPIVersion()),
			ApplicationVersion: uint32(gp.VulkanAppVersion()),
			PApplicationName:   SafeString(oswin.TheApp.Name()),
			PEngineName:        "goki\x00",
		},
		EnabledExtensionCount:   uint32(len(gp.actInstanceExts)),
		PpEnabledExtensionNames: gp.actInstanceExts,
		EnabledLayerCount:       uint32(len(gp.actValidLayers)),
		PpEnabledLayerNames:     gp.actValidLayers,
	}, nil, &inst))
	if err != nil {
		return err
	}
	gp.instance = inst
	vulkan.InitInstance(inst)
	return nil
}

// if app.VulkanDebug() {
// 	// Register a debug callback
// 	ret := vulkan.CreateDebugReportCallback(instance, &vulkan.DebugReportCallbackCreateInfo{
// 		SType:       vulkan.StructureTypeDebugReportCallbackCreateInfo,
// 		Flags:       vulkan.DebugReportFlags(vulkan.DebugReportErrorBit | vulkan.DebugReportWarningBit),
// 		PfnCallback: dbgCallbackFunc,
// 	}, nil, &gp.debugCallback)
// 		if err != nil {
// return err
// }

// 	log.Println("vulkan: DebugReportCallback enabled by application")
// }

// InitDevice initializes device
func (gp *GPUBase) InitDevice() error {
	var gpuCount uint32
	err := NewError(vulkan.EnumeratePhysicalDevices(gp.instance, &gpuCount, nil))
	if err != nil {
		return err
	}
	if gpuCount == 0 {
		return errors.New("vulkan error: no GPU devices found")
	}
	gpus := make([]vulkan.PhysicalDevice, gpuCount)
	err = NewError(vulkan.EnumeratePhysicalDevices(gp.instance, &gpuCount, gpus))
	if err != nil {
		return err
	}

	// todo: could nest this and queue below so it gets the gpu that has
	// a graphics queue

	// get the first one, multiple GPUs not supported yet
	gp.physDev = gpus[0]
	vulkan.GetPhysicalDeviceProperties(gp.physDev, &gp.gpuProperties)
	gp.gpuProperties.Deref()
	vulkan.GetPhysicalDeviceMemoryProperties(gp.physDev, &gp.memoryProperties)
	gp.memoryProperties.Deref()

	// Select device extensions
	req := SafeStrings(gp.ReqDeviceExts())
	act, err := gp.DeviceExts()
	if err != nil {
		return err
	}
	missing := 0
	gp.actDeviceExts, missing = CheckExisting(act, req)
	if missing > 0 {
		log.Println("vulkan warning: missing", missing, "required device extensions during init")
	}
	log.Printf("vulkan: enabling %d device extensions", len(gp.actDeviceExts))

	var queueCount uint32
	vulkan.GetPhysicalDeviceQueueFamilyProperties(gp.physDev, &queueCount, nil)
	queueProperties := make([]vulkan.QueueFamilyProperties, queueCount)
	vulkan.GetPhysicalDeviceQueueFamilyProperties(gp.physDev, &queueCount, queueProperties)
	if queueCount == 0 { // probably should try another GPU
		return errors.New("vulkan error: no queue families found on GPU 0")
	}

	// Find a suitable queue family for the target Vulkan mode
	var graphicsFound bool
	for i := uint32(0); i < queueCount; i++ {
		required := vulkan.QueueFlags(vulkan.QueueGraphicsBit)
		queueProperties[i].Deref()
		if queueProperties[i].QueueFlags&required != 0 {
			gp.graphicsQueueIndex = i
			graphicsFound = true
		}
	}
	if !graphicsFound {
		err := errors.New("vulkan error: could not find a suitable queue family for the target Vulkan mode")
		return err
	}

	// Create a Vulkan device
	queueInfos := []vulkan.DeviceQueueCreateInfo{{
		SType:            vulkan.StructureTypeDeviceQueueCreateInfo,
		QueueFamilyIndex: gp.graphicsQueueIndex,
		QueueCount:       1,
		PQueuePriorities: []float32{1.0},
	}}
	// if separateQueue {
	// 	queueInfos = append(queueInfos, vulkan.DeviceQueueCreateInfo{
	// 		SType:            vulkan.StructureTypeDeviceQueueCreateInfo,
	// 		QueueFamilyIndex: gp.presentQueueIndex,
	// 		QueueCount:       1,
	// 		PQueuePriorities: []float32{1.0},
	// 	})
	// }

	var device vulkan.Device
	err = NewError(vulkan.CreateDevice(gp.physDev, &vulkan.DeviceCreateInfo{
		SType:                   vulkan.StructureTypeDeviceCreateInfo,
		QueueCreateInfoCount:    uint32(len(queueInfos)),
		PQueueCreateInfos:       queueInfos,
		EnabledExtensionCount:   uint32(len(gp.actDeviceExts)),
		PpEnabledExtensionNames: gp.actDeviceExts,
		EnabledLayerCount:       uint32(len(gp.actValidLayers)),
		PpEnabledLayerNames:     gp.actValidLayers,
	}, nil, &device))
	if err != nil {
		return err
	}

	gp.device = device

	var queue vulkan.Queue
	vulkan.GetDeviceQueue(gp.device, gp.graphicsQueueIndex, 0, &queue)
	gp.graphicsQueue = queue

	return nil
}
