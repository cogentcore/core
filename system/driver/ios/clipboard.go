// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://github.com/golang/mobile
// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build ios

package ios

/*
#cgo CFLAGS: -x objective-c -DGL_SILENCE_DEPRECATION
#cgo LDFLAGS: -framework Foundation -framework UIKit -framework MobileCoreServices

#include <stdlib.h>

void setClipboardContent(char *content);
char *getClipboardContent();
*/
import "C"
import (
	"unsafe"

	"cogentcore.org/core/base/fileinfo/mimedata"
	"cogentcore.org/core/system"
)

// TheClipboard is the single [system.Clipboard] for the iOS platform
var TheClipboard = &Clipboard{}

// Clipboard is the [system.Clipboard] implementation for the iOS platform
type Clipboard struct {
	system.ClipboardBase
}

func (cl *Clipboard) Read(types []string) mimedata.Mimes {
	cstr := C.getClipboardContent()
	str := C.GoString(cstr)
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

	C.setClipboardContent(cstr)
	return nil
}
