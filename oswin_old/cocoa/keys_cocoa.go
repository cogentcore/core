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

import "github.com/rcoreilly/goki/gi/oswin"

func containsInt(haystack []int, needle int) bool {
	for _, v := range haystack {
		if needle == v {
			return true
		}
	}
	return false
}

var blankLetterCodes = []int{71, 117, 115, 119, 116, 121, 122, 120, 99, 118, 96, 97, 98, 100, 101, 109, 10, 103, 111, 105, 107, 113, 123, 124, 125, 126, 63, 58, 55, 59, 56, 61, 54, 62, 60, 114}
var keyMapping = map[int]string{
	0:   oswin.KeyA,
	11:  oswin.KeyB,
	8:   oswin.KeyC,
	2:   oswin.KeyD,
	14:  oswin.KeyE,
	3:   oswin.KeyF,
	5:   oswin.KeyG,
	4:   oswin.KeyH,
	34:  oswin.KeyI,
	38:  oswin.KeyJ,
	40:  oswin.KeyK,
	37:  oswin.KeyL,
	46:  oswin.KeyM,
	45:  oswin.KeyN,
	31:  oswin.KeyO,
	35:  oswin.KeyP,
	12:  oswin.KeyQ,
	15:  oswin.KeyR,
	1:   oswin.KeyS,
	17:  oswin.KeyT,
	32:  oswin.KeyU,
	9:   oswin.KeyV,
	13:  oswin.KeyW,
	7:   oswin.KeyX,
	16:  oswin.KeyY,
	6:   oswin.KeyZ,
	18:  oswin.Key1,
	19:  oswin.Key2,
	20:  oswin.Key3,
	21:  oswin.Key4,
	23:  oswin.Key5,
	22:  oswin.Key6,
	26:  oswin.Key7,
	28:  oswin.Key8,
	25:  oswin.Key9,
	29:  oswin.Key0,
	50:  oswin.KeyBackTick,
	27:  oswin.KeyMinus,
	24:  oswin.KeyEqual,
	33:  oswin.KeyLeftBracket,
	30:  oswin.KeyRightBracket,
	42:  oswin.KeyBackslash,
	41:  oswin.KeySemicolon,
	39:  oswin.KeyQuote,
	43:  oswin.KeyComma,
	47:  oswin.KeyPeriod,
	44:  oswin.KeySlash,
	36:  oswin.KeyReturn,
	53:  oswin.KeyEscape,
	51:  oswin.KeyBackspace,
	71:  oswin.KeyNumlock,
	117: oswin.KeyDelete,
	115: oswin.KeyHome,
	119: oswin.KeyEnd,
	116: oswin.KeyPrior,
	121: oswin.KeyNext,
	122: oswin.KeyF1,
	120: oswin.KeyF2,
	99:  oswin.KeyF3,
	118: oswin.KeyF4,
	96:  oswin.KeyF5,
	97:  oswin.KeyF6,
	98:  oswin.KeyF7,
	100: oswin.KeyF8,
	101: oswin.KeyF9,
	109: oswin.KeyF10,
	103: oswin.KeyF11,
	111: oswin.KeyF12,
	105: oswin.KeyF13,
	107: oswin.KeyF14,
	113: oswin.KeyF15,
	123: oswin.KeyLeftArrow,
	124: oswin.KeyRightArrow,
	125: oswin.KeyDownArrow,
	126: oswin.KeyUpArrow,
	63:  oswin.KeyFunction,
	58:  oswin.KeyLeftAlt,
	61:  oswin.KeyRightAlt,
	55:  oswin.KeyLeftSuper,
	54:  oswin.KeyRightSuper,
	59:  oswin.KeyLeftControl,
	62:  oswin.KeyRightControl,
	56:  oswin.KeyLeftShift,
	60:  oswin.KeyRightShift,
	114: oswin.KeyInsert,
	48:  oswin.KeyTab,
	49:  oswin.KeySpace,
	83:  oswin.KeyPadHome, // keypad
	84:  oswin.KeyPadDown,
	85:  oswin.KeyPadNext,
	86:  oswin.KeyPadLeft,
	87:  oswin.KeyPadBegin,
	88:  oswin.KeyPadRight,
	89:  oswin.KeyPadEnd,
	91:  oswin.KeyPadUp,
	92:  oswin.KeyPadNext,
	82:  oswin.KeyPadInsert,
	75:  oswin.KeyPadSlash,
	67:  oswin.KeyPadStar,
	78:  oswin.KeyPadMinus,
	69:  oswin.KeyPadPlus,
	65:  oswin.KeyPadDot,
}
