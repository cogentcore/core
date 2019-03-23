// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based extensively on vulkan-go/asche
// The MIT License (MIT)
// Copyright © 2017 Maxim Kupriianov <max@kc.vc>

package oswin

import "github.com/vulkan-go/vulkan"

// TheGPU is the current oswin GPU instance
var TheGPU GPU

// GPU represents provides the main interface to the GPU hardware
// Based on the Vulkan API, in github.com/vulkan-go
type GPU interface {
	// MemoryProperties gets the current Vulkan physical device memory properties.
	MemoryProperties() vulkan.PhysicalDeviceMemoryProperties

	// PhysicalDeviceProperies gets the current Vulkan physical device properties.
	PhysicalDeviceProperies() vulkan.PhysicalDeviceProperties

	// GraphicsQueueFamilyIndex gets the current Vulkan graphics queue family index.
	GraphicsQueueFamilyIndex() uint32

	// PresentQueueFamilyIndex gets the current Vulkan present queue family index.
	PresentQueueFamilyIndex() uint32

	// HasSeparatePresentQueue is true when PresentQueueFamilyIndex differs from GraphicsQueueFamilyIndex.
	HasSeparatePresentQueue() bool

	// GraphicsQueue gets the current Vulkan graphics queue.
	GraphicsQueue() vulkan.Queue

	// PresentQueue gets the current Vulkan present queue.
	PresentQueue() vulkan.Queue

	// Instance gets the current Vulkan instance.
	Instance() vulkan.Instance

	// Device gets the current Vulkan device.
	Device() vulkan.Device

	// PhysicalDevice gets the current Vulkan physical device.
	PhysicalDevice() vulkan.PhysicalDevice

	// Destroy is the destructor for the Platform instance.
	Destroy()

	// VulkanAPIVersion returns the required API version
	VulkanAPIVersion() vulkan.Version

	// VulkanAppVersion returns the required app version
	VulkanAppVersion() vulkan.Version

	/////////////// Extensions

	// SetReqInstanceExts sets the required extensions for the instance
	// this must be set prior to app initialization per platform
	SetReqInstanceExts(exts []string)

	// ReqInstanceExts returns the required extensions for the instance
	ReqInstanceExts() []string

	// ActInstanceExts returns the actual extensions obained for the instance
	ActInstanceExts() []string

	// SetReqDeviceExts sets the required extensions for the device
	// this must be set prior to app initialization per platform
	SetReqDeviceExts(exts []string)

	// ReqDeviceExts returns the required device extensions
	ReqDeviceExts() []string

	// ActDeviceExts returns the actual device extensions obtained
	ActDeviceExts() []string

	// InstanceExts gets a list of instance extensions available on the GPU
	InstanceExts() (names []string, err error)

	// DeviceExts gets a list of device extensions available on the GPU
	DeviceExts() (names []string, err error)

	/////////////// Validation Layers

	// SetReqValidationLayers sets the required validation layers to add -- must be set prior to app init
	SetReqValidationLayers(lays []string)

	// ReqValidationLayers returns the required validation layers
	ReqValidationLayers() []string

	// ActValidationLayers returns the actual validation layers obtained
	ActValidationLayers() []string

	// ValidationLayers gets a list of validation layers available on the platform.
	ValidationLayers() (names []string, err error)
}

// see gpu/base.go for base impl of this interface, which os-specific
// drivers build upon
