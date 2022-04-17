// Copyright (c) 2022, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is initially adapted from https://github.com/vulkan-go/asche
// Copyright © 2017 Maxim Kupriianov <max@kc.vc>, under the MIT License

package egpu

import vk "github.com/vulkan-go/vulkan"

// Mode is the mode of operation
type Mode uint32

const (
	None Mode = (1 << iota) >> 1
	Compute
	Graphics
	Present
)

func (v Mode) Has(mode Mode) bool {
	return v&mode != 0
}

// App is the application for anchoring GPU access
type App interface {
	Init(ctx *Context) error
	APIVersion() vk.Version
	AppVersion() vk.Version
	AppName() string
	Mode() Mode
	Surface(instance vk.Instance) vk.Surface
	InstanceExts() []string
	DeviceExts() []string
	Debug() bool

	// DECORATORS:
	// AppSwapchainDims
	// AppLayers
	// AppContextPrepare
	// AppContextCleanup
	// AppContextInvalidate
}

type AppSwapchainDims interface {
	SwapchainDims() *SwapchainDims
}

type AppLayers interface {
	Layers() []string
}

type AppContextPrepare interface {
	ContextPrepare() error
}

type AppContextCleanup interface {
	ContextCleanup() error
}

type AppContextInvalidate interface {
	ContextInvalidate(imageIdx int) error
}

var (
	DefaultAppVersion = vk.MakeVersion(1, 0, 0)
	DefaultAPIVersion = vk.MakeVersion(1, 0, 0)
	DefaultMode       = Compute | Graphics | Present
)

// SwapchainDims describes the size and format of the swapchain.
type SwapchainDims struct {
	// Width of the swapchain.
	Width uint32
	// Height of the swapchain.
	Height uint32
	// Format is the pixel format of the swapchain.
	Format vk.Format
}

// BaseApp is the base implementation of the App
type BaseApp struct {
	context Context
}

func (app *BaseApp) Context() Context {
	return app.context
}

func (app *BaseApp) Init(ctx Context) error {
	app.context = ctx
	return nil
}

func (app *BaseApp) APIVersion() vk.Version {
	return vk.Version(vk.MakeVersion(1, 0, 0))
}

func (app *BaseApp) AppVersion() vk.Version {
	return vk.Version(vk.MakeVersion(1, 0, 0))
}

func (app *BaseApp) AppName() string {
	return "base"
}

func (app *BaseApp) Mode() Mode {
	return DefaultMode
}

func (app *BaseApp) Surface(instance vk.Instance) vk.Surface {
	return vk.NullSurface
}

func (app *BaseApp) InstanceExts() []string {
	return nil
}

func (app *BaseApp) DeviceExts() []string {
	return nil
}

func (app *BaseApp) Debug() bool {
	return false
}
