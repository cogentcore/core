// Copyright (c) 2022, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This is initially adapted from https://github.com/vulkan-go/asche
// Copyright © 2017 Maxim Kupriianov <max@kc.vc>, under the MIT License

package vgpu

import (
	"fmt"
	"log"
	"runtime/debug"

	vk "github.com/goki/vulkan"
)

func IsError(ret vk.Result) bool {
	return ret != vk.Success
}

func NewError(ret vk.Result) error {
	if ret != vk.Success {
		err := fmt.Errorf("vulkan error: %s (%d)", vk.Error(ret).Error(), ret)
		if Debug {
			log.Println(err)
			debug.PrintStack()
		}
		return err
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
