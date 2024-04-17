// Copyright 2023 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build android

package android

/*
#cgo LDFLAGS: -landroid -llog

#include <android/configuration.h>
#include <android/input.h>
#include <android/keycodes.h>
#include <android/looper.h>
#include <android/native_activity.h>
#include <android/native_window.h>
#include <jni.h>
#include <pthread.h>
#include <stdlib.h>
#include <stdbool.h>

int32_t getKeyRune(JNIEnv* env, AInputEvent* e);
*/
import "C"
import (
	"image"
	"log"

	"cogentcore.org/core/events"
	"cogentcore.org/core/events/key"
)

//export keyboardTyped
func keyboardTyped(str *C.char) {
	for _, r := range C.GoString(str) {
		code := ConvAndroidKeyCode(r)
		TheApp.Event.KeyChord(r, code, 0) // TODO: modifiers
	}
}

//export keyboardDelete
func keyboardDelete() {
	TheApp.Event.KeyChord(0, key.CodeBackspace, 0) // TODO: modifiers
}

//export scaled
func scaled(scaleFactor, posX, posY C.float) {
	where := image.Pt(int(posX), int(posY))
	TheApp.Event.Magnify(float32(scaleFactor), where)
}

// ProcessEvents processes input queue events
func ProcessEvents(env *C.JNIEnv, q *C.AInputQueue) {
	var e *C.AInputEvent
	for C.AInputQueue_getEvent(q, &e) >= 0 {
		if C.AInputQueue_preDispatchEvent(q, e) != 0 {
			continue
		}
		ProcessEvent(env, e)
		C.AInputQueue_finishEvent(q, e, 0)
	}
}

// ProcessEvent processes an input queue event
func ProcessEvent(env *C.JNIEnv, e *C.AInputEvent) {
	switch C.AInputEvent_getType(e) {
	case C.AINPUT_EVENT_TYPE_KEY:
		ProcessKey(env, e)
	case C.AINPUT_EVENT_TYPE_MOTION:
		// At most one of the events in this batch is an up or down event; get its index and change.
		upDownIndex := C.size_t(C.AMotionEvent_getAction(e)&C.AMOTION_EVENT_ACTION_POINTER_INDEX_MASK) >> C.AMOTION_EVENT_ACTION_POINTER_INDEX_SHIFT
		upDownType := events.TouchMove
		switch C.AMotionEvent_getAction(e) & C.AMOTION_EVENT_ACTION_MASK {
		case C.AMOTION_EVENT_ACTION_DOWN, C.AMOTION_EVENT_ACTION_POINTER_DOWN:
			upDownType = events.TouchStart
		case C.AMOTION_EVENT_ACTION_UP, C.AMOTION_EVENT_ACTION_POINTER_UP:
			upDownType = events.TouchEnd
		}

		for i, n := C.size_t(0), C.AMotionEvent_getPointerCount(e); i < n; i++ {
			t := events.TouchMove
			if i == upDownIndex {
				t = upDownType
			}
			seq := events.Sequence(C.AMotionEvent_getPointerId(e, i))
			x := int(C.AMotionEvent_getX(e, i))
			y := int(C.AMotionEvent_getY(e, i))
			TheApp.Event.Touch(t, seq, image.Pt(x, y))
		}
	default:
		log.Printf("unknown input event, type=%d", C.AInputEvent_getType(e))
	}
}

// ProcessKey processes an input key event
func ProcessKey(env *C.JNIEnv, e *C.AInputEvent) {
	deviceID := C.AInputEvent_getDeviceId(e)
	if deviceID == 0 {
		// Software keyboard input, leaving for scribe/IME.
		return
	}

	r := rune(C.getKeyRune(env, e))
	code := ConvAndroidKeyCode(int32(C.AKeyEvent_getKeyCode(e)))

	if r >= '0' && r <= '9' { // GBoard generates key events for numbers, but we see them in textChanged
		return
	}
	typ := events.KeyDown
	if C.AKeyEvent_getAction(e) == C.AKEY_STATE_UP {
		typ = events.KeyUp
	}
	// TODO(crawshaw): set Modifiers.
	TheApp.Event.Key(typ, r, code, 0)
}

// AndroidKeyCodes is a map from android system key codes to system key codes
var AndroidKeyCodes = map[int32]key.Codes{
	C.AKEYCODE_HOME:            key.CodeHome,
	C.AKEYCODE_0:               key.Code0,
	C.AKEYCODE_1:               key.Code1,
	C.AKEYCODE_2:               key.Code2,
	C.AKEYCODE_3:               key.Code3,
	C.AKEYCODE_4:               key.Code4,
	C.AKEYCODE_5:               key.Code5,
	C.AKEYCODE_6:               key.Code6,
	C.AKEYCODE_7:               key.Code7,
	C.AKEYCODE_8:               key.Code8,
	C.AKEYCODE_9:               key.Code9,
	C.AKEYCODE_VOLUME_UP:       key.CodeVolumeUp,
	C.AKEYCODE_VOLUME_DOWN:     key.CodeVolumeDown,
	C.AKEYCODE_A:               key.CodeA,
	C.AKEYCODE_B:               key.CodeB,
	C.AKEYCODE_C:               key.CodeC,
	C.AKEYCODE_D:               key.CodeD,
	C.AKEYCODE_E:               key.CodeE,
	C.AKEYCODE_F:               key.CodeF,
	C.AKEYCODE_G:               key.CodeG,
	C.AKEYCODE_H:               key.CodeH,
	C.AKEYCODE_I:               key.CodeI,
	C.AKEYCODE_J:               key.CodeJ,
	C.AKEYCODE_K:               key.CodeK,
	C.AKEYCODE_L:               key.CodeL,
	C.AKEYCODE_M:               key.CodeM,
	C.AKEYCODE_N:               key.CodeN,
	C.AKEYCODE_O:               key.CodeO,
	C.AKEYCODE_P:               key.CodeP,
	C.AKEYCODE_Q:               key.CodeQ,
	C.AKEYCODE_R:               key.CodeR,
	C.AKEYCODE_S:               key.CodeS,
	C.AKEYCODE_T:               key.CodeT,
	C.AKEYCODE_U:               key.CodeU,
	C.AKEYCODE_V:               key.CodeV,
	C.AKEYCODE_W:               key.CodeW,
	C.AKEYCODE_X:               key.CodeX,
	C.AKEYCODE_Y:               key.CodeY,
	C.AKEYCODE_Z:               key.CodeZ,
	C.AKEYCODE_COMMA:           key.CodeComma,
	C.AKEYCODE_PERIOD:          key.CodeFullStop,
	C.AKEYCODE_ALT_LEFT:        key.CodeLeftAlt,
	C.AKEYCODE_ALT_RIGHT:       key.CodeRightAlt,
	C.AKEYCODE_SHIFT_LEFT:      key.CodeLeftShift,
	C.AKEYCODE_SHIFT_RIGHT:     key.CodeRightShift,
	C.AKEYCODE_TAB:             key.CodeTab,
	C.AKEYCODE_SPACE:           key.CodeSpacebar,
	C.AKEYCODE_ENTER:           key.CodeReturnEnter,
	C.AKEYCODE_DEL:             key.CodeBackspace,
	C.AKEYCODE_GRAVE:           key.CodeGraveAccent,
	C.AKEYCODE_MINUS:           key.CodeHyphenMinus,
	C.AKEYCODE_EQUALS:          key.CodeEqualSign,
	C.AKEYCODE_LEFT_BRACKET:    key.CodeLeftSquareBracket,
	C.AKEYCODE_RIGHT_BRACKET:   key.CodeRightSquareBracket,
	C.AKEYCODE_BACKSLASH:       key.CodeBackslash,
	C.AKEYCODE_SEMICOLON:       key.CodeSemicolon,
	C.AKEYCODE_APOSTROPHE:      key.CodeApostrophe,
	C.AKEYCODE_SLASH:           key.CodeSlash,
	C.AKEYCODE_PAGE_UP:         key.CodePageUp,
	C.AKEYCODE_PAGE_DOWN:       key.CodePageDown,
	C.AKEYCODE_ESCAPE:          key.CodeEscape,
	C.AKEYCODE_FORWARD_DEL:     key.CodeDelete,
	C.AKEYCODE_CTRL_LEFT:       key.CodeLeftControl,
	C.AKEYCODE_CTRL_RIGHT:      key.CodeRightControl,
	C.AKEYCODE_CAPS_LOCK:       key.CodeCapsLock,
	C.AKEYCODE_META_LEFT:       key.CodeLeftMeta,
	C.AKEYCODE_META_RIGHT:      key.CodeRightMeta,
	C.AKEYCODE_INSERT:          key.CodeInsert,
	C.AKEYCODE_F1:              key.CodeF1,
	C.AKEYCODE_F2:              key.CodeF2,
	C.AKEYCODE_F3:              key.CodeF3,
	C.AKEYCODE_F4:              key.CodeF4,
	C.AKEYCODE_F5:              key.CodeF5,
	C.AKEYCODE_F6:              key.CodeF6,
	C.AKEYCODE_F7:              key.CodeF7,
	C.AKEYCODE_F8:              key.CodeF8,
	C.AKEYCODE_F9:              key.CodeF9,
	C.AKEYCODE_F10:             key.CodeF10,
	C.AKEYCODE_F11:             key.CodeF11,
	C.AKEYCODE_F12:             key.CodeF12,
	C.AKEYCODE_NUM_LOCK:        key.CodeKeypadNumLock,
	C.AKEYCODE_NUMPAD_0:        key.CodeKeypad0,
	C.AKEYCODE_NUMPAD_1:        key.CodeKeypad1,
	C.AKEYCODE_NUMPAD_2:        key.CodeKeypad2,
	C.AKEYCODE_NUMPAD_3:        key.CodeKeypad3,
	C.AKEYCODE_NUMPAD_4:        key.CodeKeypad4,
	C.AKEYCODE_NUMPAD_5:        key.CodeKeypad5,
	C.AKEYCODE_NUMPAD_6:        key.CodeKeypad6,
	C.AKEYCODE_NUMPAD_7:        key.CodeKeypad7,
	C.AKEYCODE_NUMPAD_8:        key.CodeKeypad8,
	C.AKEYCODE_NUMPAD_9:        key.CodeKeypad9,
	C.AKEYCODE_NUMPAD_DIVIDE:   key.CodeKeypadSlash,
	C.AKEYCODE_NUMPAD_MULTIPLY: key.CodeKeypadAsterisk,
	C.AKEYCODE_NUMPAD_SUBTRACT: key.CodeKeypadHyphenMinus,
	C.AKEYCODE_NUMPAD_ADD:      key.CodeKeypadPlusSign,
	C.AKEYCODE_NUMPAD_DOT:      key.CodeKeypadFullStop,
	C.AKEYCODE_NUMPAD_ENTER:    key.CodeKeypadEnter,
	C.AKEYCODE_NUMPAD_EQUALS:   key.CodeKeypadEqualSign,
	C.AKEYCODE_VOLUME_MUTE:     key.CodeMute,
}

// ConvAndroidKeyCode converts the given android key code to a system key code
func ConvAndroidKeyCode(aKeyCode int32) key.Codes {
	if code, ok := AndroidKeyCodes[aKeyCode]; ok {
		return code
	}
	return key.CodeUnknown
}

/*
	Many Android key codes do not map into USB HID codes.
	For those, key.CodeUnknown is returned. This switch has all
	cases, even the unknown ones, to serve as a documentation
	and search aid.
	C.AKEYCODE_UNKNOWN
	C.AKEYCODE_SOFT_LEFT
	C.AKEYCODE_SOFT_RIGHT
	C.AKEYCODE_BACK
	C.AKEYCODE_CALL
	C.AKEYCODE_ENDCALL
	C.AKEYCODE_STAR
	C.AKEYCODE_POUND
	C.AKEYCODE_DPAD_UP
	C.AKEYCODE_DPAD_DOWN
	C.AKEYCODE_DPAD_LEFT
	C.AKEYCODE_DPAD_RIGHT
	C.AKEYCODE_DPAD_CENTER
	C.AKEYCODE_POWER
	C.AKEYCODE_CAMERA
	C.AKEYCODE_CLEAR
	C.AKEYCODE_SYM
	C.AKEYCODE_EXPLORER
	C.AKEYCODE_ENVELOPE
	C.AKEYCODE_AT
	C.AKEYCODE_NUM
	C.AKEYCODE_HEADSETHOOK
	C.AKEYCODE_FOCUS
	C.AKEYCODE_PLUS
	C.AKEYCODE_MENU
	C.AKEYCODE_NOTIFICATION
	C.AKEYCODE_SEARCH
	C.AKEYCODE_MEDIA_PLAY_PAUSE
	C.AKEYCODE_MEDIA_STOP
	C.AKEYCODE_MEDIA_NEXT
	C.AKEYCODE_MEDIA_PREVIOUS
	C.AKEYCODE_MEDIA_REWIND
	C.AKEYCODE_MEDIA_FAST_FORWARD
	C.AKEYCODE_MUTE
	C.AKEYCODE_PICTSYMBOLS
	C.AKEYCODE_SWITCH_CHARSET
	C.AKEYCODE_BUTTON_A
	C.AKEYCODE_BUTTON_B
	C.AKEYCODE_BUTTON_C
	C.AKEYCODE_BUTTON_X
	C.AKEYCODE_BUTTON_Y
	C.AKEYCODE_BUTTON_Z
	C.AKEYCODE_BUTTON_L1
	C.AKEYCODE_BUTTON_R1
	C.AKEYCODE_BUTTON_L2
	C.AKEYCODE_BUTTON_R2
	C.AKEYCODE_BUTTON_THUMBL
	C.AKEYCODE_BUTTON_THUMBR
	C.AKEYCODE_BUTTON_START
	C.AKEYCODE_BUTTON_SELECT
	C.AKEYCODE_BUTTON_MODE
	C.AKEYCODE_SCROLL_LOCK
	C.AKEYCODE_FUNCTION
	C.AKEYCODE_SYSRQ
	C.AKEYCODE_BREAK
	C.AKEYCODE_MOVE_HOME
	C.AKEYCODE_MOVE_END
	C.AKEYCODE_FORWARD
	C.AKEYCODE_MEDIA_PLAY
	C.AKEYCODE_MEDIA_PAUSE
	C.AKEYCODE_MEDIA_CLOSE
	C.AKEYCODE_MEDIA_EJECT
	C.AKEYCODE_MEDIA_RECORD
	C.AKEYCODE_NUMPAD_COMMA
	C.AKEYCODE_NUMPAD_LEFT_PAREN
	C.AKEYCODE_NUMPAD_RIGHT_PAREN
	C.AKEYCODE_INFO
	C.AKEYCODE_CHANNEL_UP
	C.AKEYCODE_CHANNEL_DOWN
	C.AKEYCODE_ZOOM_IN
	C.AKEYCODE_ZOOM_OUT
	C.AKEYCODE_TV
	C.AKEYCODE_WINDOW
	C.AKEYCODE_GUIDE
	C.AKEYCODE_DVR
	C.AKEYCODE_BOOKMARK
	C.AKEYCODE_CAPTIONS
	C.AKEYCODE_SETTINGS
	C.AKEYCODE_TV_POWER
	C.AKEYCODE_TV_INPUT
	C.AKEYCODE_STB_POWER
	C.AKEYCODE_STB_INPUT
	C.AKEYCODE_AVR_POWER
	C.AKEYCODE_AVR_INPUT
	C.AKEYCODE_PROG_RED
	C.AKEYCODE_PROG_GREEN
	C.AKEYCODE_PROG_YELLOW
	C.AKEYCODE_PROG_BLUE
	C.AKEYCODE_APP_SWITCH
	C.AKEYCODE_BUTTON_1
	C.AKEYCODE_BUTTON_2
	C.AKEYCODE_BUTTON_3
	C.AKEYCODE_BUTTON_4
	C.AKEYCODE_BUTTON_5
	C.AKEYCODE_BUTTON_6
	C.AKEYCODE_BUTTON_7
	C.AKEYCODE_BUTTON_8
	C.AKEYCODE_BUTTON_9
	C.AKEYCODE_BUTTON_10
	C.AKEYCODE_BUTTON_11
	C.AKEYCODE_BUTTON_12
	C.AKEYCODE_BUTTON_13
	C.AKEYCODE_BUTTON_14
	C.AKEYCODE_BUTTON_15
	C.AKEYCODE_BUTTON_16
	C.AKEYCODE_LANGUAGE_SWITCH
	C.AKEYCODE_MANNER_MODE
	C.AKEYCODE_3D_MODE
	C.AKEYCODE_CONTACTS
	C.AKEYCODE_CALENDAR
	C.AKEYCODE_MUSIC
	C.AKEYCODE_CALCULATOR

	Defined in an NDK API version beyond what we use today:
	C.AKEYCODE_ASSIST
	C.AKEYCODE_BRIGHTNESS_DOWN
	C.AKEYCODE_BRIGHTNESS_UP
	C.AKEYCODE_RO
	C.AKEYCODE_YEN
	C.AKEYCODE_ZENKAKU_HANKAKU
*/
