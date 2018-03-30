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
	"github.com/rcoreilly/goki/gi"
)

var cursorCache map[gi.Cursor]xproto.Cursor
var cursorXIds map[gi.Cursor]uint16

func init() {
	cursorCache = make(map[gi.Cursor]xproto.Cursor)
	// the default cursor is always cursor 0 - no need to CreateCursor so cache it up front
	cursorCache[gi.NormalCursor] = 0

	cursorXIds = map[gi.Cursor]uint16{
		gi.ResizeNCursor:    xcursor.TopSide,
		gi.ResizeECursor:    xcursor.RightSide,
		gi.ResizeSCursor:    xcursor.BottomSide,
		gi.ResizeWCursor:    xcursor.LeftSide,
		gi.ResizeEWCursor:   xcursor.SBHDoubleArrow,
		gi.ResizeNSCursor:   xcursor.SBVDoubleArrow,
		gi.ResizeNECursor:   xcursor.TopRightCorner,
		gi.ResizeSECursor:   xcursor.BottomRightCorner,
		gi.ResizeSWCursor:   xcursor.BottomLeftCorner,
		gi.ResizeNWCursor:   xcursor.TopLeftCorner,
		gi.CrosshairCursor:  xcursor.Crosshair,
		gi.IBeamCursor:      xcursor.XTerm,
		gi.GrabHoverCursor:  xcursor.Hand2,
		gi.GrabActiveCursor: xcursor.Hand2,
		// xcursor defines this but no crossed-circle or similar. GUMBY. dafuq?
		gi.NotAllowedCursor: xcursor.Gumby,
	}
}

func (w *OSWindow) SetCursor(cursor gi.Cursor) {
	if w.cursor != cursor {
		w.cursor = cursor
		w.win.Change(xproto.CwCursor, uint32(xCursor(w, cursor)))
	}
}

func xCursor(w *OSWindow, c gi.Cursor) xproto.Cursor {
	xc, ok := cursorCache[c]
	if !ok {
		xc = createCursor(w, c)
		cursorCache[c] = xc
	}
	return xc
}

func createCursor(w *OSWindow, c gi.Cursor) xproto.Cursor {
	xid, ok := cursorXIds[c]
	if ok {
		xc, err := xcursor.CreateCursor(w.win.X, xid)
		if err == nil {
			return xc
		}
	}
	return 0 // fallback to cursor 0
}
