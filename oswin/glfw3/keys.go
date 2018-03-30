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

package glfw3

import (
	glfw "github.com/grd/glfw3"
	"github.com/rcoreilly/goki/gi/oswin"
)

func containsInt(haystack []glfw.Key, needle glfw.Key) bool {
	for _, v := range haystack {
		if needle == v {
			return true
		}
	}
	return false
}

var blankLetterCodes = []glfw.Key{
	glfw.KeyNumLock,
	glfw.KeyDelete,
	glfw.KeyHome,
	glfw.KeyEnd,
	glfw.KeyF1,
	glfw.KeyF2,
	glfw.KeyF3,
	glfw.KeyF4,
	glfw.KeyF5,
	glfw.KeyF6,
	glfw.KeyF7,
	glfw.KeyF8,
	glfw.KeyF9,
	glfw.KeyF10,
	glfw.KeyF11,
	glfw.KeyF12,
	glfw.KeyF13,
	glfw.KeyF14,
	glfw.KeyF15,
	glfw.KeyPageDown,
	glfw.KeyPageUp,
	glfw.KeyLeft,
	glfw.KeyRight,
	glfw.KeyDown,
	glfw.KeyUp,
	glfw.KeyLeftAlt,
	glfw.KeyLeftSuper,
	glfw.KeyLeftControl,
	glfw.KeyLeftShift,
	glfw.KeyRightAlt,
	glfw.KeyRightSuper,
	glfw.KeyRightControl,
	glfw.KeyRightShift,
	glfw.KeyInsert,
	glfw.KeyCapsLock,
}

var keyMapping = map[glfw.Key]string{
	glfw.KeyA:            oswin.KeyA,
	glfw.KeyB:            oswin.KeyB,
	glfw.KeyC:            oswin.KeyC,
	glfw.KeyD:            oswin.KeyD,
	glfw.KeyE:            oswin.KeyE,
	glfw.KeyF:            oswin.KeyF,
	glfw.KeyG:            oswin.KeyG,
	glfw.KeyH:            oswin.KeyH,
	glfw.KeyI:            oswin.KeyI,
	glfw.KeyJ:            oswin.KeyJ,
	glfw.KeyK:            oswin.KeyK,
	glfw.KeyL:            oswin.KeyL,
	glfw.KeyM:            oswin.KeyM,
	glfw.KeyN:            oswin.KeyN,
	glfw.KeyO:            oswin.KeyO,
	glfw.KeyP:            oswin.KeyP,
	glfw.KeyQ:            oswin.KeyQ,
	glfw.KeyR:            oswin.KeyR,
	glfw.KeyS:            oswin.KeyS,
	glfw.KeyT:            oswin.KeyT,
	glfw.KeyU:            oswin.KeyU,
	glfw.KeyV:            oswin.KeyV,
	glfw.KeyW:            oswin.KeyW,
	glfw.KeyX:            oswin.KeyX,
	glfw.KeyY:            oswin.KeyY,
	glfw.KeyZ:            oswin.KeyZ,
	glfw.Key1:            oswin.Key1,
	glfw.Key2:            oswin.Key2,
	glfw.Key3:            oswin.Key3,
	glfw.Key4:            oswin.Key4,
	glfw.Key5:            oswin.Key5,
	glfw.Key6:            oswin.Key6,
	glfw.Key7:            oswin.Key7,
	glfw.Key8:            oswin.Key8,
	glfw.Key9:            oswin.Key9,
	glfw.Key0:            oswin.Key0,
	glfw.KeyGraveAccent:  oswin.KeyBackTick,
	glfw.KeyMinus:        oswin.KeyMinus,
	glfw.KeyEqual:        oswin.KeyEqual,
	glfw.KeyLeftBracket:  oswin.KeyLeftBracket,
	glfw.KeyRightBracket: oswin.KeyRightBracket,
	glfw.KeyBackslash:    oswin.KeyBackslash,
	glfw.KeySemicolon:    oswin.KeySemicolon,
	glfw.KeyApostrophe:   oswin.KeyQuote,
	glfw.KeyComma:        oswin.KeyComma,
	glfw.KeyPeriod:       oswin.KeyPeriod,
	glfw.KeySlash:        oswin.KeySlash,
	glfw.KeyEnter:        oswin.KeyReturn,
	glfw.KeyEscape:       oswin.KeyEscape,
	glfw.KeyBackspace:    oswin.KeyBackspace,
	glfw.KeyNumLock:      oswin.KeyNumlock,
	glfw.KeyDelete:       oswin.KeyDelete,
	glfw.KeyHome:         oswin.KeyHome,
	glfw.KeyEnd:          oswin.KeyEnd,
	glfw.KeyPageUp:       oswin.KeyPrior,
	glfw.KeyPageDown:     oswin.KeyNext,
	glfw.KeyF1:           oswin.KeyF1,
	glfw.KeyF2:           oswin.KeyF2,
	glfw.KeyF3:           oswin.KeyF3,
	glfw.KeyF4:           oswin.KeyF4,
	glfw.KeyF5:           oswin.KeyF5,
	glfw.KeyF6:           oswin.KeyF6,
	glfw.KeyF7:           oswin.KeyF7,
	glfw.KeyF8:           oswin.KeyF8,
	glfw.KeyF9:           oswin.KeyF9,
	glfw.KeyF10:          oswin.KeyF10,
	glfw.KeyF11:          oswin.KeyF11,
	glfw.KeyF12:          oswin.KeyF12,
	glfw.KeyF13:          oswin.KeyF13,
	glfw.KeyF14:          oswin.KeyF14,
	glfw.KeyF15:          oswin.KeyF15,
	glfw.KeyLeft:         oswin.KeyLeftArrow,
	glfw.KeyRight:        oswin.KeyRightArrow,
	glfw.KeyDown:         oswin.KeyDownArrow,
	glfw.KeyUp:           oswin.KeyUpArrow,
	//glfw.KeyFunction:  oswin.KeyFunction,
	glfw.KeyLeftAlt:      oswin.KeyLeftAlt,
	glfw.KeyRightAlt:     oswin.KeyRightAlt,
	glfw.KeyLeftSuper:    oswin.KeyLeftSuper,
	glfw.KeyRightSuper:   oswin.KeyRightSuper,
	glfw.KeyLeftControl:  oswin.KeyLeftControl,
	glfw.KeyRightControl: oswin.KeyRightControl,
	glfw.KeyLeftShift:    oswin.KeyLeftShift,
	glfw.KeyRightShift:   oswin.KeyRightShift,
	glfw.KeyInsert:       oswin.KeyInsert,
	glfw.KeyTab:          oswin.KeyTab,
	glfw.KeySpace:        oswin.KeySpace,
	glfw.KeyKp1:          oswin.KeyPadEnd,
	glfw.KeyKp2:          oswin.KeyPadDown,
	glfw.KeyKp3:          oswin.KeyPadNext,
	glfw.KeyKp4:          oswin.KeyPadLeft,
	glfw.KeyKp5:          oswin.KeyPadNext,
	glfw.KeyKp6:          oswin.KeyPadRight,
	glfw.KeyKp7:          oswin.KeyPadHome,
	glfw.KeyKp8:          oswin.KeyPadUp,
	glfw.KeyKp9:          oswin.KeyPadBegin,
	glfw.KeyKp0:          oswin.KeyPadInsert,
	glfw.KeyKpDivide:     oswin.KeyPadSlash,
	glfw.KeyKpMultiply:   oswin.KeyPadStar,
	glfw.KeyKpSubtract:   oswin.KeyPadMinus,
	glfw.KeyKpAdd:        oswin.KeyPadPlus,
	glfw.KeyKpDecimal:    oswin.KeyPadDot,
	glfw.KeyCapsLock:     oswin.KeyCapsLock,
}

func constructChord(key glfw.Key, mods glfw.ModifierKey) (chord string) {
	keys := make(map[string]bool)

	if mods&glfw.ModSuper != 0 {
		keys[oswin.KeyLeftSuper] = true
	}

	if mods&glfw.ModShift != 0 {
		keys[oswin.KeyLeftShift] = true
	}

	if mods&glfw.ModAlt != 0 {
		keys[oswin.KeyLeftAlt] = true
	}

	if mods&glfw.ModControl != 0 {
		keys[oswin.KeyLeftControl] = true
	}

	keys[keyMapping[key]] = true

	return oswin.ConstructChord(keys)
}
