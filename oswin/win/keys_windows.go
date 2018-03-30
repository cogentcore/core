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
	"fmt"
	"github.com/AllenDang/w32"
	"github.com/rcoreilly/goki/gi"
)

func (wnd *OSWindow) checkKeyState() {
	if !wnd.keysStale {
		return
	}
	keyboard := make([]byte, 256)
	if w32.GetKeyboardState(&keyboard) {
		/* virtual keys before 0x08 are mouse buttons; skip them */
		for vk := uintptr(0x08); vk <= 0xff; vk++ {
			isDown := keyboard[vk]&0x80 != 0
			key := keyFromVirtualKeyCode(vk)
			_, wasDown := wnd.keysDown[key]
			if isDown != wasDown {
				if isDown {
					wnd.keysDown[key] = true
					wnd.events <- gi.KeyDownEvent(gi.KeyEvent{key})
				} else {
					delete(wnd.keysDown, key)
					wnd.events <- gi.KeyUpEvent(gi.KeyEvent{key})
				}
			}
		}
		wnd.keysStale = false
	}
}

func (wnd *OSWindow) constructChord() string {
	wnd.checkKeyState()
	return gi.ConstructChord(wnd.keysDown)
}

func keyFromVirtualKeyCode(vk uintptr) string {
	if vk >= '0' && vk <= 'Z' {
		/* alphanumeric range (windows doesn't use 0x3a-0x40) */
		if vk >= 'A' {
			return fmt.Sprintf("%c", vk-'A'+'a') // convert to lower case
		}
		return fmt.Sprintf("%c", vk)
	}
	switch vk {
	case w32.VK_BACK:
		return gi.KeyBackspace
	case w32.VK_TAB:
		return gi.KeyTab
	case w32.VK_RETURN:
		return gi.KeyReturn
	case w32.VK_SHIFT:
		return gi.KeyLeftShift
	case w32.VK_CONTROL:
		return gi.KeyLeftControl
	case w32.VK_MENU:
		return gi.KeyLeftAlt
	case w32.VK_CAPITAL:
		return gi.KeyCapsLock
	case w32.VK_ESCAPE:
		return gi.KeyEscape
	case w32.VK_SPACE:
		return gi.KeySpace
	case w32.VK_PRIOR:
		return gi.KeyPrior
	case w32.VK_NEXT:
		return gi.KeyNext
	case w32.VK_END:
		return gi.KeyEnd
	case w32.VK_HOME:
		return gi.KeyHome
	case w32.VK_LEFT:
		return gi.KeyLeftArrow
	case w32.VK_UP:
		return gi.KeyUpArrow
	case w32.VK_RIGHT:
		return gi.KeyRightArrow
	case w32.VK_DOWN:
		return gi.KeyDownArrow
	case w32.VK_INSERT:
		return gi.KeyInsert
	case w32.VK_DELETE:
		return gi.KeyDelete
	case w32.VK_LWIN:
		return gi.KeyLeftSuper
	case w32.VK_RWIN:
		return gi.KeyRightSuper
	case w32.VK_NUMPAD0:
		return gi.Key0
	case w32.VK_NUMPAD1:
		return gi.Key1
	case w32.VK_NUMPAD2:
		return gi.Key2
	case w32.VK_NUMPAD3:
		return gi.Key3
	case w32.VK_NUMPAD4:
		return gi.Key4
	case w32.VK_NUMPAD5:
		return gi.Key5
	case w32.VK_NUMPAD6:
		return gi.Key6
	case w32.VK_NUMPAD7:
		return gi.Key7
	case w32.VK_NUMPAD8:
		return gi.Key8
	case w32.VK_NUMPAD9:
		return gi.Key9
	case w32.VK_MULTIPLY:
		return gi.KeyPadStar
	case w32.VK_ADD:
		return gi.KeyPadPlus
	case w32.VK_SUBTRACT:
		return gi.KeyPadMinus
	case w32.VK_DECIMAL:
		return gi.KeyPadDot
	case w32.VK_DIVIDE:
		return gi.KeyPadSlash
	case w32.VK_F1:
		return gi.KeyF1
	case w32.VK_F2:
		return gi.KeyF2
	case w32.VK_F3:
		return gi.KeyF3
	case w32.VK_F4:
		return gi.KeyF4
	case w32.VK_F5:
		return gi.KeyF5
	case w32.VK_F6:
		return gi.KeyF6
	case w32.VK_F7:
		return gi.KeyF7
	case w32.VK_F8:
		return gi.KeyF8
	case w32.VK_F9:
		return gi.KeyF9
	case w32.VK_F10:
		return gi.KeyF10
	case w32.VK_F11:
		return gi.KeyF11
	case w32.VK_F12:
		return gi.KeyF12
	case w32.VK_F13:
		return gi.KeyF13
	case w32.VK_F14:
		return gi.KeyF14
	case w32.VK_F15:
		return gi.KeyF15
	case w32.VK_F16:
		return gi.KeyF16
	case w32.VK_NUMLOCK:
		return gi.KeyNumlock
	case w32.VK_LSHIFT:
		return gi.KeyLeftShift
	case w32.VK_RSHIFT:
		return gi.KeyRightShift
	case w32.VK_LCONTROL:
		return gi.KeyLeftControl
	case w32.VK_RCONTROL:
		return gi.KeyRightControl
	case w32.VK_LMENU:
		return gi.KeyLeftAlt
	case w32.VK_RMENU:
		return gi.KeyRightAlt
	case w32.VK_OEM_1:
		return gi.KeySemicolon
	case w32.VK_OEM_PLUS:
		return gi.KeyEqual
	case w32.VK_OEM_COMMA:
		return gi.KeyComma
	case w32.VK_OEM_MINUS:
		return gi.KeyMinus
	case w32.VK_OEM_PERIOD:
		return gi.KeyPeriod
	case w32.VK_OEM_2:
		return gi.KeySlash
	case w32.VK_OEM_3:
		return gi.KeyBackTick
	case w32.VK_OEM_4:
		return gi.KeyLeftBracket
	case w32.VK_OEM_5:
		return gi.KeyBackslash
	case w32.VK_OEM_6:
		return gi.KeyRightBracket
	case w32.VK_OEM_7:
		return gi.KeyQuote

	// the rest lack gi constants. the first few are xgb compatible
	case w32.VK_PAUSE:
		return "Pause"
	case w32.VK_APPS:
		return "Menu"
	case w32.VK_SCROLL:
		return "Scroll_Lock"

	// the rest fallthrough to the default format "vk-0xff"
	case w32.VK_LBUTTON:
	case w32.VK_RBUTTON:
	case w32.VK_CANCEL:
	case w32.VK_MBUTTON:
	case w32.VK_XBUTTON1:
	case w32.VK_XBUTTON2:
	case w32.VK_CLEAR:
	case w32.VK_HANGUL:
	case w32.VK_JUNJA:
	case w32.VK_FINAL:
	case w32.VK_KANJI:
	case w32.VK_CONVERT:
	case w32.VK_NONCONVERT:
	case w32.VK_ACCEPT:
	case w32.VK_MODECHANGE:
	case w32.VK_SELECT:
	case w32.VK_PRINT:
	case w32.VK_EXECUTE:
	case w32.VK_SNAPSHOT:
	case w32.VK_HELP:
	case w32.VK_SLEEP:
	case w32.VK_SEPARATOR:
	case w32.VK_F17:
	case w32.VK_F18:
	case w32.VK_F19:
	case w32.VK_F20:
	case w32.VK_F21:
	case w32.VK_F22:
	case w32.VK_F23:
	case w32.VK_F24:
	case w32.VK_BROWSER_BACK:
	case w32.VK_BROWSER_FORWARD:
	case w32.VK_BROWSER_REFRESH:
	case w32.VK_BROWSER_STOP:
	case w32.VK_BROWSER_SEARCH:
	case w32.VK_BROWSER_FAVORITES:
	case w32.VK_BROWSER_HOME:
	case w32.VK_VOLUME_MUTE:
	case w32.VK_VOLUME_DOWN:
	case w32.VK_VOLUME_UP:
	case w32.VK_MEDIA_NEXT_TRACK:
	case w32.VK_MEDIA_PREV_TRACK:
	case w32.VK_MEDIA_STOP:
	case w32.VK_MEDIA_PLAY_PAUSE:
	case w32.VK_LAUNCH_MAIL:
	case w32.VK_LAUNCH_MEDIA_SELECT:
	case w32.VK_LAUNCH_APP1:
	case w32.VK_LAUNCH_APP2:
	case w32.VK_OEM_8:
	case w32.VK_OEM_AX:
	case w32.VK_OEM_102:
	case w32.VK_ICO_HELP:
	case w32.VK_ICO_00:
	case w32.VK_PROCESSKEY:
	case w32.VK_ICO_CLEAR:
	case w32.VK_OEM_RESET:
	case w32.VK_OEM_JUMP:
	case w32.VK_OEM_PA1:
	case w32.VK_OEM_PA2:
	case w32.VK_OEM_PA3:
	case w32.VK_OEM_WSCTRL:
	case w32.VK_OEM_CUSEL:
	case w32.VK_OEM_ATTN:
	case w32.VK_OEM_FINISH:
	case w32.VK_OEM_COPY:
	case w32.VK_OEM_AUTO:
	case w32.VK_OEM_ENLW:
	case w32.VK_OEM_BACKTAB:
	case w32.VK_ATTN:
	case w32.VK_CRSEL:
	case w32.VK_EXSEL:
	case w32.VK_EREOF:
	case w32.VK_PLAY:
	case w32.VK_ZOOM:
	case w32.VK_NONAME:
	case w32.VK_PA1:
	case w32.VK_OEM_CLEAR:
	}
	return fmt.Sprintf("vk-0x%02x", vk)
}
