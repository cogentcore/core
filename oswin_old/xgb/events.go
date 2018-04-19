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
	"fmt"
	"image"
	"os"

	"github.com/BurntSushi/xgb/xproto"
	"github.com/BurntSushi/xgbutil/icccm"
	"github.com/BurntSushi/xgbutil/keybind"
	"github.com/BurntSushi/xgbutil/xevent"
	"github.com/BurntSushi/xgbutil/xgraphics"
	"github.com/rcoreilly/goki/gi/oswin"
)

func buttonForDetail(detail xproto.Button) oswin.MouseButton {
	switch detail {
	case 1:
		return oswin.LeftButton
	case 2:
		return oswin.MiddleButton
	case 3:
		return oswin.RightButton
	case 4:
		return oswin.WheelUpButton
	case 5:
		return oswin.WheelDownButton
	case 6:
		return oswin.WheelLeftButton
	case 7:
		return oswin.WheelRightButton
	}
	return 0
}

func (w *OSWindow) handleEvents() {
	var noX int32 = 1<<31 - 1
	noX++
	var lastX, lastY int32 = noX, 0
	var button oswin.MouseButton

	downKeys := map[string]bool{}

	for {
		e, err := w.conn.WaitForEvent()

		if err != nil {
			fmt.Fprintln(os.Stderr, "[go.gi X error] ", err)
			continue
		}

		switch e := e.(type) {

		case xproto.ButtonPressEvent:
			button = button | buttonForDetail(e.Detail)
			var bpe oswin.MouseDownEvent
			bpe.Which = buttonForDetail(e.Detail)
			bpe.Where.X = int(e.EventX)
			bpe.Where.Y = int(e.EventY)
			lastX = int32(e.EventX)
			lastY = int32(e.EventY)
			w.events <- &bpe

		case xproto.ButtonReleaseEvent:
			button = button & ^buttonForDetail(e.Detail)
			var bue oswin.MouseUpEvent
			bue.Which = buttonForDetail(e.Detail)
			bue.Where.X = int(e.EventX)
			bue.Where.Y = int(e.EventY)
			lastX = int32(e.EventX)
			lastY = int32(e.EventY)
			w.events <- &bue

		case xproto.LeaveNotifyEvent:
			var wee oswin.MouseExitedEvent
			wee.Where.X = int(e.EventX)
			wee.Where.Y = int(e.EventY)
			if lastX != noX {
				wee.From.X = int(lastX)
				wee.From.Y = int(lastY)
			} else {
				wee.From.X = wee.Where.X
				wee.From.Y = wee.Where.Y
			}
			lastX = int32(e.EventX)
			lastY = int32(e.EventY)
			w.events <- &wee
		case xproto.EnterNotifyEvent:
			var wee oswin.MouseEnteredEvent
			wee.Where.X = int(e.EventX)
			wee.Where.Y = int(e.EventY)
			if lastX != noX {
				wee.From.X = int(lastX)
				wee.From.Y = int(lastY)
			} else {
				wee.From.X = wee.Where.X
				wee.From.Y = wee.Where.Y
			}
			lastX = int32(e.EventX)
			lastY = int32(e.EventY)
			w.events <- &wee

		case xproto.MotionNotifyEvent:
			var mme oswin.MouseMovedEvent
			mme.Where.X = int(e.EventX)
			mme.Where.Y = int(e.EventY)
			if lastX != noX {
				mme.From.X = int(lastX)
				mme.From.Y = int(lastY)
			} else {
				mme.From.X = mme.Where.X
				mme.From.Y = mme.Where.Y
			}
			lastX = int32(e.EventX)
			lastY = int32(e.EventY)
			if button == 0 {
				w.events <- &mme
			} else {
				var mde oswin.MouseDraggedEvent
				mde.MouseMovedEvent = mme
				mde.Which = button
				w.events <- &mde
			}

		case xproto.KeyPressEvent:
			var ke oswin.KeyEvent
			code := keybind.LookupString(w.xu, e.State, e.Detail)
			ke.Key = keyForCode(code)
			w.events <- &oswin.KeyDownEvent(ke)
			downKeys[ke.Key] = true
			kpe := oswin.KeyTypedEvent{
				KeyEvent: ke,
				Glyph:    letterForCode(code),
				Chord:    oswin.ConstructChord(downKeys),
			}
			w.events <- &kpe

		case xproto.KeyReleaseEvent:
			var ke oswin.KeyUpEvent
			ke.Key = keyForCode(keybind.LookupString(w.xu, e.State, e.Detail))
			delete(downKeys, ke.Key)
			w.events <- &ke

		case xproto.KeymapNotifyEvent:
			newDownKeys := make(map[string]bool)
			for i := 0; i < len(e.Keys); i++ {
				mask := e.Keys[i]
				for j := 0; j < 8; j++ {
					if mask&(1<<uint(j)) != 0 {
						keycode := xproto.Keycode(8*(i+1) + j)
						key := keyForCode(keybind.LookupString(w.xu, 0, keycode))
						newDownKeys[key] = true
					}
				}
			}
			/* remove keys that are no longer pressed */
			for key := range downKeys {
				if _, ok := newDownKeys[key]; !ok {
					var ke oswin.KeyUpEvent
					ke.Key = key
					delete(downKeys, key)
					w.events <- &ke
				}
			}
			/* add keys that are newly pressed */
			for key := range newDownKeys {
				if _, ok := downKeys[key]; !ok {
					var ke oswin.KeyDownEvent
					ke.Key = key
					downKeys[key] = true
					w.events <- &ke
				}
			}

		case xproto.ConfigureNotifyEvent:
			var re oswin.ResizeEvent
			re.Width = int(e.Width)
			re.Height = int(e.Height)
			if re.Width != w.width || re.Height != w.height {
				w.width, w.height = re.Width, re.Height

				w.bufferLck.Lock()
				w.buffer.Destroy()
				w.buffer = xgraphics.New(w.xu, image.Rect(0, 0, re.Width, re.Height))
				w.bufferLck.Unlock()

				w.events <- &re
			}

		case xproto.ClientMessageEvent:
			if icccm.IsDeleteProtocol(w.xu, xevent.ClientMessageEvent{&e}) {
				w.events <- &oswin.CloseEvent{}
			}
		case xproto.DestroyNotifyEvent:
		case xproto.ReparentNotifyEvent:
		case xproto.MapNotifyEvent:
		case xproto.UnmapNotifyEvent:
		case xproto.PropertyNotifyEvent:

		default:
			fmt.Fprintf(os.Stderr, "unhandled event: type %T\n%+v\n", e, e)
		}

	}

	close(w.events)
}

func (w *OSWindow) EventChan() (events <-chan interface{}) {
	events = w.events

	return
}
