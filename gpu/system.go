// Copyright (c) 2024, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gpu

// System provides the general interface for
// [GraphicSystem] and [ComputeSystem].
type System interface {
	// vars represents all the data variables used by the system,
	// with one Var for each resource that is made visible to the shader,
	// indexed by Group (@group) and Binding (@binding).
	// Each Var has Value(s) containing specific instance values.
	Vars() *Vars

	// Device is the logical device for this system, typically from
	// the Renderer (Surface) or owned by a ComputeSystem.
	Device() *Device

	// GPU is our GPU device, which has properties
	// and alignment factors.
	GPU() *GPU

	// Render returns the Render object, for a GraphicsSystem
	// (nil for a ComputeSystem).
	Render() *Render
}
