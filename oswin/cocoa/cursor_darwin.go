/*
   Copyright 2012 the go.wde authors

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package cocoa

// #include "cursor.h"
import "C"

import (
	"github.com/rcoreilly/goki/gi"
	"unsafe"
)

var cursors map[gi.Cursor]unsafe.Pointer

func init() {
	C.initMacCursor()

	cursors = map[gi.Cursor]unsafe.Pointer{
		gi.NoneCursor:     nil,
		gi.NormalCursor:   C.cursors.arrow,
		gi.ResizeNCursor:  C.cursors.resizeUp,
		gi.ResizeECursor:  C.cursors.resizeRight,
		gi.ResizeSCursor:  C.cursors.resizeDown,
		gi.ResizeWCursor:  C.cursors.resizeLeft,
		gi.ResizeEWCursor: C.cursors.resizeLeftRight,
		gi.ResizeNSCursor: C.cursors.resizeUpDown,

		// might be able to improve the diagonal arrow cursors:
		// http://stackoverflow.com/questions/10733228/native-osx-lion-resize-cursor-for-custom-nswindow-or-nsview
		gi.ResizeNECursor: C.cursors.pointingHand,
		gi.ResizeSECursor: C.cursors.pointingHand,
		gi.ResizeSWCursor: C.cursors.pointingHand,
		gi.ResizeNWCursor: C.cursors.pointingHand,

		gi.CrosshairCursor:  C.cursors.crosshair,
		gi.IBeamCursor:      C.cursors.IBeam,
		gi.GrabHoverCursor:  C.cursors.openHand,
		gi.GrabActiveCursor: C.cursors.closedHand,
		gi.NotAllowedCursor: C.cursors.operationNotAllowed,
	}
}

func setCursor(c gi.Cursor) {
	nscursor := cursors[c]
	if nscursor != nil {
		C.setCursor(nscursor)
	}
}

func (w *OSWindow) SetCursor(cursor gi.Cursor) {
	if w.cursor == cursor {
		return
	}
	if w.hasMouse {
		/* the osx set cursor is application wide rather than window specific */
		setCursor(cursor)
	}
	w.cursor = cursor
}
