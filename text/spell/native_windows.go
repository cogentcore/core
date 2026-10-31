// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package spell

import (
	"runtime"
	"slices"
	"strings"
	"syscall"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

// This uses the Windows Spell Checking API (spellcheck.h), available
// since Windows 8, through its COM interfaces, without cgo.

func init() {
	newNative = newWinChecker
}

var (
	ole32                        = windows.NewLazySystemDLL("ole32.dll")
	procCoCreateInstance         = ole32.NewProc("CoCreateInstance")
	kernel32                     = windows.NewLazySystemDLL("kernel32.dll")
	procGetUserDefaultLocaleName = kernel32.NewProc("GetUserDefaultLocaleName")

	clsidSpellCheckerFactory = guid("{7AB36653-1796-484B-BDFA-E74F1DB7C1DC}")
	iidISpellCheckerFactory  = guid("{8E018A9D-2415-4677-BF08-794EA61F94BB}")
	iidISpellChecker2        = guid("{E7ED1C71-87F7-4378-A840-C9200DACEE47}")
)

// guid parses a GUID string, which must be valid.
func guid(s string) windows.GUID {
	g, err := windows.GUIDFromString(s)
	if err != nil {
		panic(err)
	}
	return g
}

// COM vtable method indexes, in the order declared in spellcheck.h
// (and objidl.h for IEnumString), after the IUnknown methods.
const (
	vtQueryInterface = 0
	vtRelease        = 2

	// ISpellCheckerFactory
	vtFactoryIsSupported        = 4
	vtFactoryCreateSpellChecker = 5

	// ISpellChecker
	vtCheck   = 4
	vtSuggest = 5
	vtAdd     = 6

	// ISpellChecker2
	vtRemove = 17

	// IEnumSpellingError and IEnumString
	vtNext = 3

	// ISpellingError
	vtStartIndex       = 3
	vtCorrectiveAction = 5
)

// correctiveActionDelete is CORRECTIVE_ACTION_DELETE, which is
// reported for a repeated word, not a misspelling.
const correctiveActionDelete = 3

// comObject is a pointer to a COM object, whose first field
// is a pointer to its vtable.
type comObject = *struct {
	vtbl *[32]uintptr
}

// comCall calls the vtable method at index i with the object and args,
// returning the HRESULT.
func comCall(o comObject, i int, args ...uintptr) uintptr {
	r, _, _ := syscall.SyscallN(o.vtbl[i], append([]uintptr{uintptr(unsafe.Pointer(o))}, args...)...)
	return r
}

// comRelease releases the object if it is not nil.
func comRelease(o comObject) {
	if o != nil {
		comCall(o, vtRelease)
	}
}

// winChecker uses the Windows ISpellChecker. All COM calls are made on a
// single locked OS thread that has initialized COM, through funcs sent
// on the calls channel.
type winChecker struct {
	calls   chan func()
	checker comObject
	// checker2 is the ISpellChecker2 interface, nil before Windows 10.
	checker2 comObject
}

func newWinChecker() nativeChecker {
	wc := &winChecker{calls: make(chan func())}
	ok := make(chan bool)
	go wc.run(ok)
	if !<-ok {
		return nil
	}
	return wc
}

// run initializes COM and the spell checker, sending the
// result to ok, and then serves calls.
func (wc *winChecker) run(ok chan bool) {
	runtime.LockOSThread()
	if err := windows.CoInitializeEx(0, windows.COINIT_MULTITHREADED); err != nil {
		ok <- false
		return
	}
	if !wc.init() {
		windows.CoUninitialize()
		ok <- false
		return
	}
	ok <- true
	for f := range wc.calls {
		f()
	}
}

// init creates the spell checker for the user language, or English.
func (wc *winChecker) init() bool {
	var factory comObject
	hr, _, _ := procCoCreateInstance.Call(uintptr(unsafe.Pointer(&clsidSpellCheckerFactory)), 0,
		windows.CLSCTX_INPROC_SERVER, uintptr(unsafe.Pointer(&iidISpellCheckerFactory)), uintptr(unsafe.Pointer(&factory)))
	if hr != 0 || factory == nil {
		return false
	}
	defer comRelease(factory)
	for _, lang := range []string{userLocale(), "en-US"} {
		if lang == "" {
			continue
		}
		lp, err := windows.UTF16PtrFromString(lang)
		if err != nil {
			continue
		}
		var supported int32
		if comCall(factory, vtFactoryIsSupported, uintptr(unsafe.Pointer(lp)), uintptr(unsafe.Pointer(&supported))) != 0 || supported == 0 {
			continue
		}
		if comCall(factory, vtFactoryCreateSpellChecker, uintptr(unsafe.Pointer(lp)), uintptr(unsafe.Pointer(&wc.checker))) != 0 || wc.checker == nil {
			continue
		}
		comCall(wc.checker, vtQueryInterface, uintptr(unsafe.Pointer(&iidISpellChecker2)), uintptr(unsafe.Pointer(&wc.checker2)))
		return true
	}
	return false
}

// userLocale returns the user default locale name, such as en-US.
func userLocale() string {
	buf := make([]uint16, 85) // LOCALE_NAME_MAX_LENGTH
	n, _, _ := procGetUserDefaultLocaleName.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if n == 0 {
		return ""
	}
	return windows.UTF16ToString(buf)
}

// do runs f on the COM thread and waits for it to finish.
func (wc *winChecker) do(f func()) {
	done := make(chan struct{})
	wc.calls <- func() {
		f()
		close(done)
	}
	<-done
}

func (wc *winChecker) misspelled(words []string) []int {
	// Check all of the words in one string, recording the UTF-16
	// offset of each word to map the error locations back.
	starts := make([]int, len(words))
	var b strings.Builder
	u16 := 0
	for i, w := range words {
		if i > 0 {
			b.WriteByte(' ')
			u16++
		}
		starts[i] = u16
		b.WriteString(w)
		for _, r := range w {
			u16 += utf16.RuneLen(r)
		}
	}
	text, err := windows.UTF16PtrFromString(b.String())
	if err != nil {
		return nil
	}
	var bad []int
	wc.do(func() {
		var errs comObject
		if comCall(wc.checker, vtCheck, uintptr(unsafe.Pointer(text)), uintptr(unsafe.Pointer(&errs))) != 0 || errs == nil {
			return
		}
		defer comRelease(errs)
		for {
			var se comObject
			if comCall(errs, vtNext, uintptr(unsafe.Pointer(&se))) != 0 || se == nil { // S_FALSE at end
				return
			}
			var start, action uint32
			comCall(se, vtStartIndex, uintptr(unsafe.Pointer(&start)))
			comCall(se, vtCorrectiveAction, uintptr(unsafe.Pointer(&action)))
			comRelease(se)
			if action == correctiveActionDelete {
				continue
			}
			i, found := slices.BinarySearch(starts, int(start))
			if !found {
				i--
			}
			if i >= 0 && (len(bad) == 0 || bad[len(bad)-1] != i) {
				bad = append(bad, i)
			}
		}
	})
	return bad
}

func (wc *winChecker) suggest(word string) []string {
	wp, err := windows.UTF16PtrFromString(word)
	if err != nil {
		return nil
	}
	var sugs []string
	wc.do(func() {
		var enum comObject
		if comCall(wc.checker, vtSuggest, uintptr(unsafe.Pointer(wp)), uintptr(unsafe.Pointer(&enum))) != 0 || enum == nil {
			return
		}
		defer comRelease(enum)
		for {
			var s *uint16
			var fetched uint32
			if comCall(enum, vtNext, 1, uintptr(unsafe.Pointer(&s)), uintptr(unsafe.Pointer(&fetched))) != 0 || fetched == 0 {
				return
			}
			sugs = append(sugs, windows.UTF16PtrToString(s))
			windows.CoTaskMemFree(unsafe.Pointer(s))
		}
	})
	return sugs
}

func (wc *winChecker) learn(word string) {
	wp, err := windows.UTF16PtrFromString(word)
	if err != nil {
		return
	}
	wc.do(func() {
		comCall(wc.checker, vtAdd, uintptr(unsafe.Pointer(wp)))
	})
}

func (wc *winChecker) unlearn(word string) {
	wp, err := windows.UTF16PtrFromString(word)
	if err != nil || wc.checker2 == nil {
		return
	}
	wc.do(func() {
		comCall(wc.checker2, vtRemove, uintptr(unsafe.Pointer(wp)))
	})
}
