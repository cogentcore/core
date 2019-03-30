// Copyright (c) 2019, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// based extensively on vulkan-go/asche
// The MIT License (MIT)
// Copyright © 2017 Maxim Kupriianov <max@kc.vc>

package gpu

import (
	"fmt"

	"github.com/vulkan-go/vulkan"
)

var end = "\x00"
var endChar byte = '\x00'

func SafeString(s string) string {
	if len(s) == 0 {
		return end
	}
	if s[len(s)-1] != endChar {
		return s + end
	}
	return s
}

func SafeStrings(list []string) []string {
	for i := range list {
		list[i] = SafeString(list[i])
	}
	return list
}

func CheckExisting(actual, required []string) (existing []string, missing int) {
	existing = make([]string, 0, len(required))
	for j := range required {
		req := SafeString(required[j])
		for i := range actual {
			if SafeString(actual[i]) == req {
				existing = append(existing, req)
			}
		}
	}
	missing = len(required) - len(existing)
	return existing, missing
}

func IsError(ret vulkan.Result) bool {
	return ret != vulkan.Success
}

func NewError(ret vulkan.Result) error {
	if ret != vulkan.Success {
		// pc, _, _, ok := runtime.Caller(0)
		// if !ok {
		// 	return fmt.Errorf("oswin vulkan error: %s (%d)",
		// 		vulkan.Error(ret).Error(), ret)
		// }
		// todo: traceback
		// frame := newStackFrame(pc)
		return fmt.Errorf("vulkan error: %s (%d) on",
			vulkan.Error(ret).Error(), ret)
	}
	return nil
}

func orPanic(err error, finalizers ...func()) {
	if err != nil {
		for _, fn := range finalizers {
			fn()
		}
		panic(err)
	}
}

func checkErr(err *error) {
	if v := recover(); v != nil {
		*err = fmt.Errorf("%+v", v)
	}
}
