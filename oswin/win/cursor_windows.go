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
	"github.com/rcoreilly/goki/gi/oswin"
)

var cursorCache map[oswin.Cursor]w32.HCURSOR
var cursorIDC map[oswin.Cursor]uint16

func init() {
	cursorCache = make(map[oswin.Cursor]w32.HCURSOR)

	cursorIDC = map[oswin.Cursor]uint16{
		oswin.NormalCursor:     w32.IDC_ARROW,
		oswin.ResizeNCursor:    w32.IDC_SIZENS,
		oswin.ResizeSCursor:    w32.IDC_SIZENS,
		oswin.ResizeNSCursor:   w32.IDC_SIZENS,
		oswin.ResizeECursor:    w32.IDC_SIZEWE,
		oswin.ResizeWCursor:    w32.IDC_SIZEWE,
		oswin.ResizeEWCursor:   w32.IDC_SIZEWE,
		oswin.ResizeNECursor:   w32.IDC_SIZENESW,
		oswin.ResizeSWCursor:   w32.IDC_SIZENESW,
		oswin.ResizeNWCursor:   w32.IDC_SIZENWSE,
		oswin.ResizeSECursor:   w32.IDC_SIZENWSE,
		oswin.CrosshairCursor:  w32.IDC_CROSS,
		oswin.IBeamCursor:      w32.IDC_IBEAM,
		oswin.GrabHoverCursor:  w32.IDC_HAND,
		oswin.GrabActiveCursor: w32.IDC_HAND,
		oswin.NotAllowedCursor: w32.IDC_NO,
	}
}

func (w *OSWindow) SetCursor(cursor oswin.Cursor) {
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
	if cursor == oswin.NoneCursor {
		cursor = oswin.NormalCursor
	}
	w32.SetCursor(cursorHandle(cursor))
}

func cursorHandle(id oswin.Cursor) w32.HCURSOR {
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
