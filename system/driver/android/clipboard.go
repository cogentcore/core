// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/golang/mobile
// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build android

package android

/*
#cgo LDFLAGS: -landroid -llog

#include <stdlib.h>

char *getClipboardContent(uintptr_t java_vm, uintptr_t jni_env, uintptr_t ctx);
void setClipboardContent(uintptr_t java_vm, uintptr_t jni_env, uintptr_t ctx, char *content);
*/
import "C"
import (
	"unsafe"

	"cogentcore.org/core/base/fileinfo/mimedata"
)

// TheClipboard is the single [system.Clipboard] for the Android platform
var TheClipboard = &Clipboard{}

// Clipboard is the [system.Clipboard] implementation for the Android platform
type Clipboard struct{}

func (cl *Clipboard) IsEmpty() bool {
	return len(cl.Read(nil).Text(mimedata.TextPlain)) == 0
}

func (cl *Clipboard) Read(types []string) mimedata.Mimes {
	str := ""
	RunOnJVM(func(vm, env, ctx uintptr) error {
		chars := C.getClipboardContent(C.uintptr_t(vm), C.uintptr_t(env), C.uintptr_t(ctx))
		if chars == nil {
			return nil
		}

		str = C.GoString(chars)
		C.free(unsafe.Pointer(chars))
		return nil
	})
	return mimedata.NewText(str)
}

func (cl *Clipboard) Write(data mimedata.Mimes) error {
	str := ""
	if len(data) > 1 { // multipart
		mpd := data.ToMultipart()
		str = string(mpd)
	} else {
		d := data[0]
		if mimedata.IsText(d.Type) {
			str = string(d.Data)
		}
	}
	cstr := C.CString(str)
	defer C.free(unsafe.Pointer(cstr))

	RunOnJVM(func(vm, env, ctx uintptr) error {
		C.setClipboardContent(C.uintptr_t(vm), C.uintptr_t(env), C.uintptr_t(ctx), cstr)
		return nil
	})
	return nil
}

func (cl *Clipboard) Clear() {
	// no-op
}
