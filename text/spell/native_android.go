// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build android

package spell

/*
#include <jni.h>
#include <stdlib.h>

// The system spell checker is used through the org.golang.app.SpellChecker
// Java class (system/driver/android/SpellChecker.java), which wraps the
// asynchronous SpellCheckerSession API with synchronous methods.

static jobject checker;
static jmethodID misspelledMethod;
static jmethodID suggestMethod;

// spellInit loads the SpellChecker class through the class loader of
// the app context (FindClass only sees system classes on a native thread),
// and creates the checker, returning 0 if there is none.
static int spellInit(uintptr_t jniEnv, uintptr_t ctx) {
	JNIEnv *env = (JNIEnv *)jniEnv;
	jobject jctx = (jobject)ctx;
	jclass ctxClass = (*env)->GetObjectClass(env, jctx);
	jmethodID getClassLoader = (*env)->GetMethodID(env, ctxClass, "getClassLoader", "()Ljava/lang/ClassLoader;");
	jobject loader = (*env)->CallObjectMethod(env, jctx, getClassLoader);
	jclass loaderClass = (*env)->GetObjectClass(env, loader);
	jmethodID loadClass = (*env)->GetMethodID(env, loaderClass, "loadClass", "(Ljava/lang/String;)Ljava/lang/Class;");
	jstring name = (*env)->NewStringUTF(env, "org.golang.app.SpellChecker");
	jclass cls = (jclass)(*env)->CallObjectMethod(env, loader, loadClass, name);
	if ((*env)->ExceptionCheck(env) || cls == NULL) {
		// the app was built with an older core tool
		(*env)->ExceptionClear(env);
		return 0;
	}
	jmethodID create = (*env)->GetStaticMethodID(env, cls, "create", "(Landroid/content/Context;)Lorg/golang/app/SpellChecker;");
	misspelledMethod = (*env)->GetMethodID(env, cls, "misspelled", "(Ljava/lang/String;)[I");
	suggestMethod = (*env)->GetMethodID(env, cls, "suggest", "(Ljava/lang/String;)[Ljava/lang/String;");
	if ((*env)->ExceptionCheck(env) || create == NULL || misspelledMethod == NULL || suggestMethod == NULL) {
		(*env)->ExceptionClear(env);
		return 0;
	}
	jobject sc = (*env)->CallStaticObjectMethod(env, cls, create, jctx);
	if ((*env)->ExceptionCheck(env) || sc == NULL) {
		(*env)->ExceptionClear(env);
		return 0;
	}
	checker = (*env)->NewGlobalRef(env, sc);
	return 1;
}

// spellMisspelled returns the UTF-16 offsets of the misspelled words
// in the UTF-16 text, allocated with malloc, with their count in n,
// which is -1 if the spell checker did not respond.
static jint *spellMisspelled(uintptr_t jniEnv, const jchar *text, int textLen, int *n) {
	JNIEnv *env = (JNIEnv *)jniEnv;
	*n = -1;
	jstring str = (*env)->NewString(env, text, textLen);
	jintArray arr = (jintArray)(*env)->CallObjectMethod(env, checker, misspelledMethod, str);
	(*env)->DeleteLocalRef(env, str);
	if ((*env)->ExceptionCheck(env) || arr == NULL) {
		(*env)->ExceptionClear(env);
		return NULL;
	}
	int len = (*env)->GetArrayLength(env, arr);
	jint *offs = NULL;
	*n = 0;
	if (len > 0) {
		offs = (jint *)malloc(len * sizeof(jint));
		(*env)->GetIntArrayRegion(env, arr, 0, len, offs);
		*n = len;
	}
	(*env)->DeleteLocalRef(env, arr);
	return offs;
}

// spellSuggest returns the suggestions for the UTF-16 word as a Java
// String array, with its length in n.
static jobjectArray spellSuggest(uintptr_t jniEnv, const jchar *word, int wordLen, int *n) {
	JNIEnv *env = (JNIEnv *)jniEnv;
	*n = 0;
	jstring str = (*env)->NewString(env, word, wordLen);
	jobjectArray arr = (jobjectArray)(*env)->CallObjectMethod(env, checker, suggestMethod, str);
	(*env)->DeleteLocalRef(env, str);
	if ((*env)->ExceptionCheck(env) || arr == NULL) {
		(*env)->ExceptionClear(env);
		return NULL;
	}
	*n = (*env)->GetArrayLength(env, arr);
	return arr;
}

// spellString returns the UTF-16 chars of element i of the String array,
// with its length in n, to be released with spellReleaseString.
static const jchar *spellString(uintptr_t jniEnv, jobjectArray arr, int i, jstring *str, int *n) {
	JNIEnv *env = (JNIEnv *)jniEnv;
	*str = (jstring)(*env)->GetObjectArrayElement(env, arr, i);
	*n = (*env)->GetStringLength(env, *str);
	return (*env)->GetStringChars(env, *str, NULL);
}

static void spellReleaseArray(uintptr_t jniEnv, jobjectArray arr) {
	JNIEnv *env = (JNIEnv *)jniEnv;
	if (arr != NULL) {
		(*env)->DeleteLocalRef(env, arr);
	}
}

static void spellReleaseString(uintptr_t jniEnv, jstring str, const jchar *chars) {
	JNIEnv *env = (JNIEnv *)jniEnv;
	(*env)->ReleaseStringChars(env, str, chars);
	(*env)->DeleteLocalRef(env, str);
}
*/
import "C"

import (
	"bufio"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"unicode/utf16"
	"unsafe"

	"cogentcore.org/core/system/driver/mobile/mobileinit"
)

func init() {
	newNative = newAndroidChecker
}

// androidChecker uses the system spell checker (the one selected in the
// system settings, such as Gboard or the AOSP checker) through
// SpellCheckerSession. Apps cannot add words to the Android user
// dictionary (only keyboards and spell checkers can, since Android 6),
// so learned words are kept in a file in the app files directory.
type androidChecker struct {
	// mu protects learned.
	mu sync.Mutex

	// learned are the words learned in this app, in lowercase.
	learned map[string]bool

	// learnedFile is the file that learned is saved to, if any.
	learnedFile string
}

func newAndroidChecker() nativeChecker {
	ok := false
	err := mobileinit.RunOnJVM(func(vm, env, ctx uintptr) error {
		ok = ctx != 0 && C.spellInit(C.uintptr_t(env), C.uintptr_t(ctx)) != 0
		return nil
	})
	if err != nil || !ok {
		return nil
	}
	ac := &androidChecker{learned: map[string]bool{}}
	if dir := os.Getenv("FILESDIR"); dir != "" {
		ac.learnedFile = filepath.Join(dir, "spell-learned.txt")
		ac.openLearned()
	}
	return ac
}

// openLearned reads the learned words from learnedFile.
func (ac *androidChecker) openLearned() {
	f, err := os.Open(ac.learnedFile)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if w := strings.TrimSpace(sc.Text()); w != "" {
			ac.learned[w] = true
		}
	}
}

// saveLearned saves the learned words to learnedFile, under mu.
func (ac *androidChecker) saveLearned() {
	if ac.learnedFile == "" {
		return
	}
	words := make([]string, 0, len(ac.learned))
	for w := range ac.learned {
		words = append(words, w)
	}
	slices.Sort(words)
	os.WriteFile(ac.learnedFile, []byte(strings.Join(words, "\n")+"\n"), 0666)
}

// utf16Ptr returns a pointer to the UTF-16 chars, which must not be empty.
func utf16Ptr(u []uint16) *C.jchar {
	return (*C.jchar)(unsafe.Pointer(&u[0]))
}

func (ac *androidChecker) misspelled(words []string) ([]int, bool) {
	// Check all of the words in one string, recording the UTF-16
	// offset of each word to map the error locations back.
	starts := make([]int, len(words))
	var text []uint16
	for i, w := range words {
		if i > 0 {
			text = append(text, ' ')
		}
		starts[i] = len(text)
		for _, r := range w {
			text = utf16.AppendRune(text, r)
		}
	}
	if len(text) == 0 {
		return nil, true
	}
	var offs []int
	ok := false
	mobileinit.RunOnJVM(func(vm, env, ctx uintptr) error {
		var n C.int
		res := C.spellMisspelled(C.uintptr_t(env), utf16Ptr(text), C.int(len(text)), &n)
		ok = n >= 0
		if res == nil {
			return nil
		}
		defer C.free(unsafe.Pointer(res))
		for _, o := range unsafe.Slice(res, int(n)) {
			offs = append(offs, int(o))
		}
		return nil
	})
	ac.mu.Lock()
	defer ac.mu.Unlock()
	var bad []int
	for _, o := range offs {
		i, found := slices.BinarySearch(starts, o)
		if !found {
			i--
		}
		if i < 0 || (len(bad) > 0 && bad[len(bad)-1] == i) || ac.learned[strings.ToLower(words[i])] {
			continue
		}
		bad = append(bad, i)
	}
	return bad, ok
}

func (ac *androidChecker) suggest(word string) []string {
	w := utf16.Encode([]rune(word))
	if len(w) == 0 {
		return nil
	}
	var sugs []string
	mobileinit.RunOnJVM(func(vm, env, ctx uintptr) error {
		var n C.int
		arr := C.spellSuggest(C.uintptr_t(env), utf16Ptr(w), C.int(len(w)), &n)
		for i := range int(n) {
			var str C.jstring
			var sn C.int
			chars := C.spellString(C.uintptr_t(env), arr, C.int(i), &str, &sn)
			s := unsafe.Slice((*uint16)(unsafe.Pointer(chars)), int(sn))
			sugs = append(sugs, string(utf16.Decode(s)))
			C.spellReleaseString(C.uintptr_t(env), str, chars)
		}
		C.spellReleaseArray(C.uintptr_t(env), arr)
		return nil
	})
	return sugs
}

func (ac *androidChecker) learn(word string) {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.learned[strings.ToLower(word)] = true
	ac.saveLearned()
}

func (ac *androidChecker) unlearn(word string) {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	delete(ac.learned, strings.ToLower(word))
	ac.saveLearned()
}
