// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build linux && !android && cgo

package spell

/*
#cgo LDFLAGS: -ldl
#include <dlfcn.h>
#include <stdlib.h>
#include <sys/types.h>

// Enchant-2 is loaded at run time with dlopen, so that building
// does not need its headers, and running does not need it installed.

typedef void *(*brokerInitFunc)(void);
typedef void *(*requestDictFunc)(void *broker, const char *tag);
typedef int (*dictCheckFunc)(void *dict, const char *word, ssize_t len);
typedef char **(*dictSuggestFunc)(void *dict, const char *word, ssize_t len, size_t *n);
typedef void (*dictFreeListFunc)(void *dict, char **list);
typedef void (*dictWordFunc)(void *dict, const char *word, ssize_t len);

static brokerInitFunc brokerInit;
static requestDictFunc requestDict;
static dictCheckFunc dictCheck;
static dictSuggestFunc dictSuggest;
static dictFreeListFunc dictFreeList;
static dictWordFunc dictAdd;
static dictWordFunc dictRemove;
static void *broker;

static int enchantLoad(void) {
	void *lib = dlopen("libenchant-2.so.2", RTLD_NOW);
	if (lib == NULL) {
		lib = dlopen("libenchant-2.so", RTLD_NOW);
	}
	if (lib == NULL) {
		return 0;
	}
	brokerInit = (brokerInitFunc)dlsym(lib, "enchant_broker_init");
	requestDict = (requestDictFunc)dlsym(lib, "enchant_broker_request_dict");
	dictCheck = (dictCheckFunc)dlsym(lib, "enchant_dict_check");
	dictSuggest = (dictSuggestFunc)dlsym(lib, "enchant_dict_suggest");
	dictFreeList = (dictFreeListFunc)dlsym(lib, "enchant_dict_free_string_list");
	dictAdd = (dictWordFunc)dlsym(lib, "enchant_dict_add");
	dictRemove = (dictWordFunc)dlsym(lib, "enchant_dict_remove");
	if (!brokerInit || !requestDict || !dictCheck || !dictSuggest || !dictFreeList || !dictAdd || !dictRemove) {
		return 0;
	}
	broker = brokerInit();
	return broker != NULL;
}

static void *enchantDict(const char *tag) {
	return requestDict(broker, tag);
}

static int enchantCheck(void *dict, const char *word, int len) {
	return dictCheck(dict, word, len);
}

static char **enchantSuggest(void *dict, const char *word, int len, size_t *n) {
	return dictSuggest(dict, word, len, n);
}

static void enchantFreeList(void *dict, char **list) {
	dictFreeList(dict, list);
}

static void enchantAdd(void *dict, const char *word, int len) {
	dictAdd(dict, word, len);
}

static void enchantRemove(void *dict, const char *word, int len) {
	dictRemove(dict, word, len);
}
*/
import "C"

import (
	"os"
	"strings"
	"sync"
	"unsafe"
)

func init() {
	newNative = newEnchantChecker
}

// enchantChecker uses the Enchant-2 library, which uses whatever
// spell checking backends and dictionaries are installed (Hunspell,
// Nuspell, Aspell, etc), and the user's personal dictionary in
// ~/.config/enchant, shared with other apps that use Enchant.
type enchantChecker struct {
	// mu serializes access to the Enchant dictionary.
	mu   sync.Mutex
	dict unsafe.Pointer
}

// enchantOnce loads Enchant at most once.
var enchantOnce = sync.OnceValue(func() bool {
	return C.enchantLoad() != 0
})

func newEnchantChecker() nativeChecker {
	if !enchantOnce() {
		return nil
	}
	for _, tag := range localeTags() {
		ct := C.CString(tag)
		dict := C.enchantDict(ct)
		C.free(unsafe.Pointer(ct))
		if dict != nil {
			return &enchantChecker{dict: dict}
		}
	}
	return nil
}

// localeTags returns the language tags to try, from the
// locale environment variables, with English as a fallback.
func localeTags() []string {
	var tags []string
	for _, env := range []string{"LC_ALL", "LC_MESSAGES", "LANG"} {
		v := os.Getenv(env)
		if i := strings.IndexAny(v, ".@"); i >= 0 {
			v = v[:i]
		}
		if v == "" || v == "C" || v == "POSIX" {
			continue
		}
		tags = append(tags, v)
		if i := strings.IndexByte(v, '_'); i > 0 {
			tags = append(tags, v[:i])
		}
		break
	}
	return append(tags, "en_US", "en")
}

func (ec *enchantChecker) misspelled(words []string) ([]int, bool) {
	var bad []int
	ec.mu.Lock()
	defer ec.mu.Unlock()
	for i, w := range words {
		cw := C.CString(w)
		if C.enchantCheck(ec.dict, cw, C.int(len(w))) > 0 {
			bad = append(bad, i)
		}
		C.free(unsafe.Pointer(cw))
	}
	return bad, true
}

func (ec *enchantChecker) suggest(word string) []string {
	cw := C.CString(word)
	defer C.free(unsafe.Pointer(cw))
	ec.mu.Lock()
	defer ec.mu.Unlock()
	var n C.size_t
	list := C.enchantSuggest(ec.dict, cw, C.int(len(word)), &n)
	if list == nil {
		return nil
	}
	defer C.enchantFreeList(ec.dict, list)
	sugs := make([]string, int(n))
	for i, s := range unsafe.Slice(list, int(n)) {
		sugs[i] = C.GoString(s)
	}
	return sugs
}

func (ec *enchantChecker) learn(word string) {
	cw := C.CString(word)
	defer C.free(unsafe.Pointer(cw))
	ec.mu.Lock()
	defer ec.mu.Unlock()
	C.enchantAdd(ec.dict, cw, C.int(len(word)))
}

func (ec *enchantChecker) unlearn(word string) {
	cw := C.CString(word)
	defer C.free(unsafe.Pointer(cw))
	ec.mu.Lock()
	defer ec.mu.Unlock()
	C.enchantRemove(ec.dict, cw, C.int(len(word)))
}
