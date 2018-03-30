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
	"github.com/rcoreilly/goki/gi/oswin"
	"unsafe"
)

var cursors map[oswin.Cursor]unsafe.Pointer

func init() {
	C.initMacCursor()

	cursors = map[oswin.Cursor]unsafe.Pointer{
		oswin.NoneCursor:     nil,
		oswin.NormalCursor:   C.cursors.arrow,
		oswin.ResizeNCursor:  C.cursors.resizeUp,
		oswin.ResizeECursor:  C.cursors.resizeRight,
		oswin.ResizeSCursor:  C.cursors.resizeDown,
		oswin.ResizeWCursor:  C.cursors.resizeLeft,
		oswin.ResizeEWCursor: C.cursors.resizeLeftRight,
		oswin.ResizeNSCursor: C.cursors.resizeUpDown,

		// might be able to improve the diagonal arrow cursors:
		// http://stackoverflow.com/questions/10733228/native-osx-lion-resize-cursor-for-custom-nswindow-or-nsview
		oswin.ResizeNECursor: C.cursors.pointingHand,
		oswin.ResizeSECursor: C.cursors.pointingHand,
		oswin.ResizeSWCursor: C.cursors.pointingHand,
		oswin.ResizeNWCursor: C.cursors.pointingHand,

		oswin.CrosshairCursor:  C.cursors.crosshair,
		oswin.IBeamCursor:      C.cursors.IBeam,
		oswin.GrabHoverCursor:  C.cursors.openHand,
		oswin.GrabActiveCursor: C.cursors.closedHand,
		oswin.NotAllowedCursor: C.cursors.operationNotAllowed,
	}
}

func setCursor(c oswin.Cursor) {
	nscursor := cursors[c]
	if nscursor != nil {
		C.setCursor(nscursor)
	}
}

func (w *OSWindow) SetCursor(cursor oswin.Cursor) {
	if w.cursor == cursor {
		return
	}
	if w.hasMouse {
		/* the osx set cursor is application wide rather than window specific */
		setCursor(cursor)
	}
	w.cursor = cursor
}
