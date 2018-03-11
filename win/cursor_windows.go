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

package win

import (
	"github.com/AllenDang/w32"
	"github.com/rcoreilly/goki/gi"
)

var cursorCache map[gi.Cursor]w32.HCURSOR
var cursorIDC map[gi.Cursor]uint16

func init() {
	cursorCache = make(map[gi.Cursor]w32.HCURSOR)

	cursorIDC = map[gi.Cursor]uint16{
		gi.NormalCursor:     w32.IDC_ARROW,
		gi.ResizeNCursor:    w32.IDC_SIZENS,
		gi.ResizeSCursor:    w32.IDC_SIZENS,
		gi.ResizeNSCursor:   w32.IDC_SIZENS,
		gi.ResizeECursor:    w32.IDC_SIZEWE,
		gi.ResizeWCursor:    w32.IDC_SIZEWE,
		gi.ResizeEWCursor:   w32.IDC_SIZEWE,
		gi.ResizeNECursor:   w32.IDC_SIZENESW,
		gi.ResizeSWCursor:   w32.IDC_SIZENESW,
		gi.ResizeNWCursor:   w32.IDC_SIZENWSE,
		gi.ResizeSECursor:   w32.IDC_SIZENWSE,
		gi.CrosshairCursor:  w32.IDC_CROSS,
		gi.IBeamCursor:      w32.IDC_IBEAM,
		gi.GrabHoverCursor:  w32.IDC_HAND,
		gi.GrabActiveCursor: w32.IDC_HAND,
		gi.NotAllowedCursor: w32.IDC_NO,
	}
}

func (w *OSWindow) SetCursor(cursor gi.Cursor) {
	if w.cursor != cursor {
		w.cursor = cursor
		handle := cursorHandle(cursor)
		w.onUiThread(func() {
			w32.SetCursor(handle)
		})
	}
}

// restores current cursor. must be called from UI(event) thread.
func (w *OSWindow) restoreCursor() {
	cursor := w.cursor
	if cursor == gi.NoneCursor {
		cursor = gi.NormalCursor
	}
	w32.SetCursor(cursorHandle(cursor))
}

func cursorHandle(id gi.Cursor) w32.HCURSOR {
	h, ok := cursorCache[id]
	if !ok {
		idc, ok := cursorIDC[id]
		if !ok {
			idc = w32.IDC_ARROW
		}
		h = w32.LoadCursor(0, w32.MakeIntResource(idc))
		cursorCache[id] = h
	}
	return h
}
