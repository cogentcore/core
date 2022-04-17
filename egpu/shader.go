// Copyright (c) 2022, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is initially adapted from https://github.com/vulkan-go/asche
// Copyright © 2017 Maxim Kupriianov <max@kc.vc>, under the MIT License

package egpu

import vk "github.com/vulkan-go/vulkan"

func LoadShaderModule(device vk.Device, data []byte) (vk.ShaderModule, error) {
	var module vk.ShaderModule
	ret := vk.CreateShaderModule(device, &vk.ShaderModuleCreateInfo{
		SType:    vk.StructureTypeShaderModuleCreateInfo,
		CodeSize: uint(len(data)),
		PCode:    sliceUint32(data),
	}, nil, &module)
	if IsError(ret) {
		return vk.NullShaderModule, NewError(ret)
	}
	return module, nil
}
