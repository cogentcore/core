// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build ios

package ios

/*
#include <stdlib.h>
void setTextBefore(char *before, long sent, int truncated);
long keyboardSentCount(void);
*/
import "C"

import (
	"unicode/utf8"
	"unsafe"

	"cogentcore.org/core/system"
	"cogentcore.org/core/system/driver/base"
)

// Window is the implementation of [system.Window] for the iOS platform.
type Window struct {
	base.WindowSingle[*App]
}

// SetTextInput sets the text of the keyboard to the text before the cursor
// (see [system.TextInput.Before]), for its predictions and corrections,
// unless it does not yet include keys that the keyboard sent: it is then
// set again once they are handled, as they change it.
func (w *Window) SetTextInput(ti *system.TextInput) {
	if ti == nil {
		return
	}
	sent := C.keyboardSentCount() // (before the keys are looked for)
	if base.PendingKeys(&w.App.Event.Deque) {
		return
	}
	truncated := 0
	if utf8.RuneCountInString(ti.Before) >= system.TextInputBeforeMax {
		truncated = 1
	}
	cs := C.CString(ti.Before)
	defer C.free(unsafe.Pointer(cs))
	C.setTextBefore(cs, sent, C.int(truncated))
}
