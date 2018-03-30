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

package xgb

import (
	"github.com/BurntSushi/xgb/xproto"
	"github.com/BurntSushi/xgbutil/xcursor"
	"github.com/rcoreilly/goki/gi/oswin"
)

var cursorCache map[oswin.Cursor]xproto.Cursor
var cursorXIds map[oswin.Cursor]uint16

func init() {
	cursorCache = make(map[oswin.Cursor]xproto.Cursor)
	// the default cursor is always cursor 0 - no need to CreateCursor so cache it up front
	cursorCache[oswin.NormalCursor] = 0

	cursorXIds = map[oswin.Cursor]uint16{
		oswin.ResizeNCursor:    xcursor.TopSide,
		oswin.ResizeECursor:    xcursor.RightSide,
		oswin.ResizeSCursor:    xcursor.BottomSide,
		oswin.ResizeWCursor:    xcursor.LeftSide,
		oswin.ResizeEWCursor:   xcursor.SBHDoubleArrow,
		oswin.ResizeNSCursor:   xcursor.SBVDoubleArrow,
		oswin.ResizeNECursor:   xcursor.TopRightCorner,
		oswin.ResizeSECursor:   xcursor.BottomRightCorner,
		oswin.ResizeSWCursor:   xcursor.BottomLeftCorner,
		oswin.ResizeNWCursor:   xcursor.TopLeftCorner,
		oswin.CrosshairCursor:  xcursor.Crosshair,
		oswin.IBeamCursor:      xcursor.XTerm,
		oswin.GrabHoverCursor:  xcursor.Hand2,
		oswin.GrabActiveCursor: xcursor.Hand2,
		// xcursor defines this but no crossed-circle or similar. GUMBY. dafuq?
		oswin.NotAllowedCursor: xcursor.Gumby,
	}
}

func (w *OSWindow) SetCursor(cursor oswin.Cursor) {
	if w.cursor != cursor {
		w.cursor = cursor
		w.win.Change(xproto.CwCursor, uint32(xCursor(w, cursor)))
	}
}

func xCursor(w *OSWindow, c oswin.Cursor) xproto.Cursor {
	xc, ok := cursorCache[c]
	if !ok {
		xc = createCursor(w, c)
		cursorCache[c] = xc
	}
	return xc
}

func createCursor(w *OSWindow, c oswin.Cursor) xproto.Cursor {
	xid, ok := cursorXIds[c]
	if ok {
		xc, err := xcursor.CreateCursor(w.win.X, xid)
		if err == nil {
			return xc
		}
	}
	return 0 // fallback to cursor 0
}
