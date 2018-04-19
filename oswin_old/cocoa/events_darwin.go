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

// #include "gmd.h"
// #include "cursor.h"
import "C"

import (
	"fmt"
	"image"
	"strings"
	"time"

	"github.com/rcoreilly/goki/gi/oswin"
)

func getButton(b int) (which oswin.MouseButton) {
	switch b {
	case 0:
		which = oswin.LeftButton
	case 1:
		which = oswin.RightButton
	case 2:
		which = oswin.MiddleButton
	}
	return
}

func containsGlyph(haystack []string, needle string) bool {
	for _, v := range haystack {
		if needle == v {
			return true
		}
	}
	return false
}

func (w *OSWindow) EventChan() (events <-chan interface{}) {
	downKeys := make(map[string]bool)
	ec := make(chan interface{})
	go func(ec chan<- interface{}) {

		noXY := image.Point{-1, -1}
		lastXY := noXY
		suppressDrag := false

	eventloop:
		for {
			e := C.getNextEvent(w.cw)
			switch e.kind {
			case C.GMDNoop:
				continue
			case C.GMDMouseDown:
				var mde oswin.MouseDownEvent
				mde.Where.X = int(e.data[0])
				mde.Where.Y = int(e.data[1])
				mde.Which = getButton(int(e.data[2]))
				lastXY = mde.Where
				mde.Time = time.Now()
				ec <- &mde
				suppressDrag = true
			case C.GMDMouseUp:
				var mue oswin.MouseUpEvent
				mue.Where.X = int(e.data[0])
				mue.Where.Y = int(e.data[1])
				mue.Which = getButton(int(e.data[2]))
				lastXY = mue.Where
				mue.Time = time.Now()
				ec <- &mue
			case C.GMDMouseDragged:
				if suppressDrag {
					/* Cocoa emits a drag event immediately after a mouse down.
					 * Other backends only do so after the mouse actually moves, which
					 * is the behaviour we emulate here. */
					suppressDrag = false
					continue
				}
				var mde oswin.MouseDraggedEvent
				mde.Where.X = int(e.data[0])
				mde.Where.Y = int(e.data[1])
				mde.Which = getButton(int(e.data[2]))
				if lastXY != noXY {
					mde.From = lastXY
				} else {
					mde.From = mde.Where
				}
				lastXY = mde.Where
				mde.Time = time.Now()
				ec <- &mde
			case C.GMDMouseMoved:
				var mme oswin.MouseMovedEvent
				mme.Where.X = int(e.data[0])
				mme.Where.Y = int(e.data[1])
				if lastXY != noXY {
					mme.From = lastXY
				} else {
					mme.From = mme.Where
				}
				lastXY = mme.Where
				mme.Time = time.Now()
				ec <- &mme
			case C.GMDMouseEntered:
				w.hasMouse = true
				setCursor(w.cursor)
				var me oswin.MouseEnteredEvent
				me.Where.X = int(e.data[0])
				me.Where.Y = int(e.data[1])
				if lastXY != noXY {
					me.From = lastXY
				} else {
					me.From = me.Where
				}
				lastXY = me.Where
				me.Time = time.Now()
				ec <- &me
			case C.GMDMouseExited:
				w.hasMouse = false
				setCursor(oswin.NormalCursor)
				var me oswin.MouseExitedEvent
				me.Where.X = int(e.data[0])
				me.Where.Y = int(e.data[1])
				if lastXY != noXY {
					me.From = lastXY
				} else {
					me.From = me.Where
				}
				lastXY = me.Where
				me.Time = time.Now()
				ec <- &me
			case C.GMDMouseWheel:
				var me oswin.MouseEvent
				me.Where.X = int(e.data[0])
				me.Where.Y = int(e.data[1])
				me.Time = time.Now()
				// TODO e.data[2] contains horizontal scroll info; what do?
				if e.data[3] != 0 {
					button := oswin.WheelUpButton
					if e.data[3] == -1 {
						button = oswin.WheelDownButton
					}
					ec <- &oswin.MouseDownEvent{me, button}
					ec <- &oswin.MouseUpEvent{me, button}
				}
			case C.GMDMagnify:
				var mge oswin.MagnifyEvent
				mge.Where.X = int(e.data[0])
				mge.Where.Y = int(e.data[1])
				mge.Magnification = 1 + float64(e.data[2])/65536
				mge.Time = time.Now()
				ec <- &mge
			case C.GMDRotate:
				var rte oswin.RotateEvent
				rte.Where.X = int(e.data[0])
				rte.Where.Y = int(e.data[1])
				rte.Rotation = float64(e.data[2]) / 65536
				rte.Time = time.Now()
				ec <- &rte
			case C.GMDScroll:
				var se oswin.ScrollEvent
				se.Where.X = int(e.data[0])
				se.Where.Y = int(e.data[1])
				se.Delta.X = int(e.data[2])
				se.Delta.Y = int(e.data[3])
				se.Time = time.Now()
				ec <- &se
			case C.GMDMainFocus:
				// for some reason Cocoa resets the cursor to normal when the window
				// becomes the "main" window, so we have to set it back to what we want
				setCursor(w.cursor)
			case C.GMDKeyDown:
				var letter string
				var ke oswin.KeyEvent
				ke.Time = time.Now()
				keycode := int(e.data[1])

				blankLetter := containsInt(blankLetterCodes, keycode)
				if !blankLetter {
					letter = fmt.Sprintf("%c", e.data[0])
				}

				ke.Key = keyMapping[keycode]

				if !downKeys[ke.Key] {
					kde := oswin.KeyDownEvent{ke}
					ec <- &kde
				}

				downKeys[ke.Key] = true

				ec <- &oswin.KeyTypedEvent{
					KeyEvent: ke,
					Chord:    oswin.ConstructChord(downKeys),
					Glyph:    letter,
				}

			case C.GMDKeyUp:
				var ke oswin.KeyUpEvent
				ke.Key = keyMapping[int(e.data[1])]
				ke.Time = time.Now()
				delete(downKeys, ke.Key)
				// todo: getting stuck keys
				for key, _ := range downKeys {
					if !strings.HasSuffix(key, "_arrow") {
						if strings.HasPrefix(key, "left_") || strings.HasPrefix(key, "right_") {
							continue
						}
					}
					delete(downKeys, key)
				}
				ec <- &ke
			case C.GMDResize:
				var re oswin.ResizeEvent
				re.Width = int(e.data[0])
				re.Height = int(e.data[1])
				re.Time = time.Now()
				ec <- &re
			case C.GMDClose:
				var ce oswin.CloseEvent
				ce.Time = time.Now()
				ec <- &ce
				break eventloop
				return
			}
		}
		close(ec)
	}(ec)
	events = ec
	return
}
