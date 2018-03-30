package xgb

import (
	"github.com/rcoreilly/goki/gi/oswin"
)

func keyForCode(code string) (key string) {
	key = codeKeys[code]
	if key == "" {
		key = code
	}
	return
}

func letterForCode(code string) (letter string) {
	if len(code) == 1 {
		letter = code
	} else {
		letter = longLetters[code]
	}
	return
}

var codeKeys map[string]string
var longLetters = map[string]string{
	"quoteleft":  "`",
	"quoteright": "'",
}

func init() {
	codeKeys = map[string]string{
		"Shift_L":          oswin.KeyLeftShift,
		"Shift_R":          oswin.KeyRightShift,
		"Control_L":        oswin.KeyLeftControl,
		"Control_R":        oswin.KeyRightControl,
		"Hangul_switch":    oswin.KeyLeftAlt,
		"Alt_L":            oswin.KeyLeftAlt,
		"Alt_R":            oswin.KeyRightAlt,
		"Meta_L":           oswin.KeyLeftSuper,
		"Meta_R":           oswin.KeyRightSuper,
		"Super_L":          oswin.KeyLeftSuper,
		"Super_R":          oswin.KeyRightSuper,
		"Tab":              oswin.KeyTab,
		"ISO_Left_Tab":     oswin.KeyTab,
		"Return":           oswin.KeyReturn,
		"Up":               oswin.KeyUpArrow,
		"Down":             oswin.KeyDownArrow,
		"Left":             oswin.KeyLeftArrow,
		"Right":            oswin.KeyRightArrow,
		" ":                oswin.KeySpace,
		"Escape":           oswin.KeyEscape,
		"!":                oswin.Key1,
		"@":                oswin.Key2,
		"#":                oswin.Key3,
		"$":                oswin.Key4,
		"%":                oswin.Key5,
		"^":                oswin.Key6,
		"&":                oswin.Key7,
		"*":                oswin.Key8,
		"(":                oswin.Key9,
		")":                oswin.Key0,
		"_":                oswin.KeyMinus,
		"+":                oswin.KeyEqual,
		"|":                oswin.KeyBackslash,
		"BackSpace":        oswin.KeyBackspace,
		"Delete":           oswin.KeyDelete,
		"quoteleft":        oswin.KeyBackTick,
		"`":                oswin.KeyBackTick,
		"~":                oswin.KeyBackTick,
		"quoteright":       oswin.KeyQuote,
		"\"":               oswin.KeyQuote,
		"{":                oswin.KeyLeftBracket,
		"}":                oswin.KeyRightBracket,
		":":                oswin.KeySemicolon,
		"<":                oswin.KeyComma,
		">":                oswin.KeyPeriod,
		"?":                oswin.KeySlash,
		"F1":               oswin.KeyF1,
		"F2":               oswin.KeyF2,
		"F3":               oswin.KeyF3,
		"F4":               oswin.KeyF4,
		"F5":               oswin.KeyF5,
		"F6":               oswin.KeyF6,
		"F7":               oswin.KeyF7,
		"F8":               oswin.KeyF8,
		"F9":               oswin.KeyF9,
		"F10":              oswin.KeyF10,
		"F11":              oswin.KeyF11,
		"F12":              oswin.KeyF12,
		"F13":              oswin.KeyF13,
		"F14":              oswin.KeyF14,
		"F15":              oswin.KeyF15,
		"F16":              oswin.KeyF16,
		"L1":               oswin.KeyF11,
		"L2":               oswin.KeyF12,
		"XF86Tools":        oswin.KeyF13,
		"XF86Launch5":      oswin.KeyF14,
		"XF86Launch6":      oswin.KeyF15,
		"XF86Launch7":      oswin.KeyF16,
		"Num_Lock":         oswin.KeyNumlock,
		"KP_Equal":         oswin.KeyPadEqual,
		"Insert":           oswin.KeyInsert,
		"Home":             oswin.KeyHome,
		"Prior":            oswin.KeyPrior,
		"Next":             oswin.KeyNext,
		"Page_Up":          oswin.KeyPrior,
		"Page_Down":        oswin.KeyNext,
		"End":              oswin.KeyEnd,
		"KP_Insert":        oswin.KeyPadInsert,
		"KP_Delete":        oswin.KeyPadDot,
		"KP_Enter":         oswin.KeyPadEnter,
		"KP_End":           oswin.KeyPadEnd,
		"KP_Down":          oswin.KeyPadDown,
		"KP_Page_Down":     oswin.KeyPadNext,
		"KP_Next":          oswin.KeyPadNext,
		"KP_Left":          oswin.KeyPadLeft,
		"KP_Begin":         oswin.KeyPadBegin,
		"KP_Right":         oswin.KeyPadRight,
		"KP_Home":          oswin.KeyPadHome,
		"KP_Up":            oswin.KeyPadUp,
		"KP_Prior":         oswin.KeyPadPrior,
		"Caps_Lock":        oswin.KeyCapsLock,
		"Terminate_Server": oswin.KeyBackspace,
	}

	for l := byte('a'); l <= byte('z'); l++ {
		codeKeys[string(l)] = string(l)
		k := l + 'A' - 'a'
		codeKeys[string(k)] = string(l)
	}

}
