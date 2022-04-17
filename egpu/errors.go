// Copyright (c) 2022, The Emergent Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is initially adapted from https://github.com/vulkan-go/asche
// Copyright © 2017 Maxim Kupriianov <max@kc.vc>, under the MIT License

package egpu

import (
	"fmt"
	"runtime"

	vk "github.com/vulkan-go/vulkan"
)

func IsError(ret vk.Result) bool {
	return ret != vk.Success
}

func NewError(ret vk.Result) error {
	if ret != vk.Success {
		pc, _, _, ok := runtime.Caller(0)
		if !ok {
			return fmt.Errorf("vulkan error: %s (%d)",
				vk.Error(ret).Error(), ret)
		}
		frame := newStackFrame(pc)
		return fmt.Errorf("vulkan error: %s (%d) on %s",
			vk.Error(ret).Error(), ret, frame.String())
	}
	return nil
}

func IfPanic(err error, finalizers ...func()) {
	if err != nil {
		for _, fn := range finalizers {
			fn()
		}
		panic(err)
	}
}

func CheckErr(err *error) {
	if v := recover(); v != nil {
		*err = fmt.Errorf("%+v", v)
	}
}
