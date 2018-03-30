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
	"image"
	"syscall"
	"unsafe"
)

const WDEM_UI_THREAD = w32.WM_APP

type EventData struct {
	lastX, lastY int
	button       gi.MouseButton
	noX          int
	trackMouse   bool
}

func (this *EventData) InitEventData() {
	this.noX = 1<<31 - 1
	this.noX++
	this.lastX = this.noX
}

func buttonForDetail(button uint32) gi.MouseButton {
	switch button {
	case w32.WM_LBUTTONDOWN, w32.WM_LBUTTONUP:
		return gi.LeftButton
	case w32.WM_RBUTTONDOWN, w32.WM_RBUTTONUP:
		return gi.RightButton
	case w32.WM_MBUTTONDOWN, w32.WM_MBUTTONUP:
		return gi.MiddleButton
	}
	return 0
}

func buttonForWheel(msg uint32, delta int16) gi.MouseButton {
	if msg == w32.WM_MOUSEWHEEL {
		if delta > 0 {
			return gi.WheelUpButton
		} else if delta < 0 {
			return gi.WheelDownButton
		}
	} else if msg == w32.WM_MOUSEHWHEEL {
		if delta > 0 {
			return gi.WheelRightButton
		} else if delta < 0 {
			return gi.WheelLeftButton
		}
	}
	return 0
}

func WndProc(hwnd w32.HWND, msg uint32, wparam, lparam uintptr) uintptr {
	wnd := GetMsgHandler(hwnd)
	if wnd == nil {
		return uintptr(w32.DefWindowProc(hwnd, msg, wparam, lparam))
	}

	var rc uintptr
	switch msg {
	case w32.WM_ACTIVATE:
		if wparam&0xffff != 0 {
			/* This window has just been granted focus, so flag our internal
			** key state as stale. We can't simply refresh our state because
			** win32's GetKeyboardState isn't always accurate at this point
			** in the event stream. */
			wnd.keysStale = true
		}
		rc = w32.DefWindowProc(hwnd, msg, wparam, lparam)

	case w32.WM_SHOWWINDOW:
		w32.SetFocus(hwnd)
		wnd.restoreCursor()

	case w32.WM_LBUTTONDOWN, w32.WM_RBUTTONDOWN, w32.WM_MBUTTONDOWN:
		wnd.button = wnd.button | buttonForDetail(msg)
		var bpe gi.MouseDownEvent
		bpe.Which = buttonForDetail(msg)
		bpe.Where.X = int(lparam) & 0xFFFF
		bpe.Where.Y = int(lparam>>16) & 0xFFFF
		wnd.lastX = bpe.Where.X
		wnd.lastY = bpe.Where.Y
		wnd.events <- bpe

	case w32.WM_LBUTTONUP, w32.WM_RBUTTONUP, w32.WM_MBUTTONUP:
		wnd.button = wnd.button & ^buttonForDetail(msg)
		var bpe gi.MouseUpEvent
		bpe.Which = buttonForDetail(msg)
		bpe.Where.X = int(lparam) & 0xFFFF
		bpe.Where.Y = int(lparam>>16) & 0xFFFF
		wnd.lastX = bpe.Where.X
		wnd.lastY = bpe.Where.Y
		wnd.events <- bpe

	case w32.WM_MOUSEWHEEL, w32.WM_MOUSEHWHEEL:
		var me gi.MouseEvent
		screenX := int(lparam) & 0xFFFF
		screenY := int(lparam>>16) & 0xFFFF
		me.Where.X, me.Where.Y, _ = w32.ScreenToClient(wnd.hwnd, screenX, screenY)
		delta := int16((wparam >> 16) & 0xFFFF)
		button := buttonForWheel(msg, delta)
		wnd.lastX = me.Where.X
		wnd.lastX = me.Where.Y
		wnd.events <- gi.MouseDownEvent{me, button}
		wnd.events <- gi.MouseUpEvent{me, button}

	case w32.WM_MOUSEMOVE:
		var mme gi.MouseMovedEvent
		mme.Where.X = int(lparam) & 0xFFFF
		mme.Where.Y = int(lparam>>16) & 0xFFFF
		if wnd.lastX != wnd.noX {
			mme.From.X = int(wnd.lastX)
			mme.From.Y = int(wnd.lastY)
		} else {
			mme.From.X = mme.Where.X
			mme.From.Y = mme.Where.Y
		}
		wnd.lastX = mme.Where.X
		wnd.lastY = mme.Where.Y

		if !wnd.trackMouse {
			var tme w32.TRACKMOUSEEVENT
			tme.CbSize = uint32(unsafe.Sizeof(tme))
			tme.DwFlags = w32.TME_LEAVE
			tme.HwndTrack = hwnd
			tme.DwHoverTime = w32.HOVER_DEFAULT
			w32.TrackMouseEvent(&tme)
			wnd.trackMouse = true
			wnd.restoreCursor()
			wnd.events <- gi.MouseEnteredEvent(mme)
		} else {
			if wnd.button == 0 {
				wnd.events <- mme
			} else {
				var mde gi.MouseDraggedEvent
				mde.MouseMovedEvent = mme
				mde.Which = wnd.button
				wnd.events <- mde
			}
		}

	case w32.WM_MOUSELEAVE:
		wnd.trackMouse = false

		var wee gi.MouseExitedEvent
		// TODO: get real position
		wee.Where.Y = wnd.lastX
		wee.Where.X = wnd.lastY
		wnd.events <- wee

	case w32.WM_SYSKEYDOWN, w32.WM_KEYDOWN:
		translatable := w32.MapVirtualKeyEx(uint(wparam), w32.MAPVK_VK_TO_CHAR, w32.HKL(0))
		wnd.keyDown = keyFromVirtualKeyCode(wparam)
		wnd.keysDown[wnd.keyDown] = true
		wnd.checkKeyState()
		wnd.events <- gi.KeyDownEvent{wnd.keyDown}
		if translatable == 0 {
			kpe := gi.KeyTypedEvent{
				gi.KeyEvent{wnd.keyDown},
				"",
				wnd.constructChord(),
			}
			wnd.events <- kpe
		}
	case w32.WM_SYSCHAR, w32.WM_CHAR:
		glyph := syscall.UTF16ToString([]uint16{uint16(wparam)})
		kpe := gi.KeyTypedEvent{
			gi.KeyEvent{wnd.keyDown},
			glyph,
			wnd.constructChord(),
		}
		wnd.events <- kpe
	case w32.WM_SYSKEYUP, w32.WM_KEYUP:
		keyUp := keyFromVirtualKeyCode(wparam)
		delete(wnd.keysDown, keyUp)
		wnd.checkKeyState()
		wnd.events <- gi.KeyUpEvent{
			keyUp,
		}

	case w32.WM_SIZE:
		width := int(lparam) & 0xFFFF
		height := int(lparam>>16) & 0xFFFF
		wnd.buffer = NewDIB(image.Rect(0, 0, width, height))
		wnd.events <- gi.ResizeEvent{width, height}
		rc = w32.DefWindowProc(hwnd, msg, wparam, lparam)

	case w32.WM_PAINT:
		wnd.Repaint()
		rc = w32.DefWindowProc(hwnd, msg, wparam, lparam)

	case GIM_UI_THREAD:
		f := <-wnd.uiTasks
		f()

	case w32.WM_CLOSE:
		wnd.events <- gi.CloseEvent{}

	case w32.WM_DESTROY:
		w32.PostQuitMessage(0)

	default:
		rc = w32.DefWindowProc(hwnd, msg, wparam, lparam)
	}

	return rc
}
