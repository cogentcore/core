// Copyright (c) 2026, Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows

#include <windows.h>
#include <imm.h>
#include <commctrl.h>
#include <stdlib.h>
#include "_cgo_export.h"

// Input method (IME) support, with IMM32. GLFW does not handle the IME
// messages, so Windows shows its own composition window at the top left of
// the window. The window procedure of GLFW is subclassed here to send the
// composition to Go instead, which shows it in the focused text widget,
// and to place the candidate window at its caret. GLFW already ignores
// the keys that the input method processes (VK_PROCESSKEY).

// WM_COGENT_IME is posted by setTextInput to update the state on the
// window thread, with an imeUpdate in lParam.
#define WM_COGENT_IME (WM_APP + 0x1E)

// imeState is the input method state of a window.
typedef struct {
	BOOL active;    // whether a text widget is focused
	RECT caret;     // in client pixels
	BOOL composing; // whether a composition is being sent
} imeState;

typedef struct {
	BOOL active;
	RECT caret;
} imeUpdate;

// utf8 returns the given UTF-16 text as UTF-8, allocated with malloc.
static char* utf8(const WCHAR* s, int n) {
	int len = WideCharToMultiByte(CP_UTF8, 0, s, n, NULL, 0, NULL, NULL);
	char* r = (char*)malloc(len + 1);
	WideCharToMultiByte(CP_UTF8, 0, s, n, r, len, NULL, NULL);
	r[len] = 0;
	return r;
}

// runeIndex returns the number of runes in the first i UTF-16 units of s.
static int runeIndex(const WCHAR* s, int i) {
	int n = 0;
	for (int j = 0; j < i; j++) {
		if (s[j] < 0xDC00 || s[j] > 0xDFFF) { // not a low surrogate
			n++;
		}
	}
	return n;
}

// placeWindows places the candidate and composition windows at the caret.
static void placeWindows(HWND hwnd, imeState* st) {
	HIMC imc = ImmGetContext(hwnd);
	if (imc == NULL) {
		return;
	}
	CANDIDATEFORM cf = {0};
	cf.dwIndex = 0;
	cf.dwStyle = CFS_EXCLUDE;
	cf.ptCurrentPos.x = st->caret.left;
	cf.ptCurrentPos.y = st->caret.top;
	cf.rcArea = st->caret;
	ImmSetCandidateWindow(imc, &cf);
	COMPOSITIONFORM comp = {0};
	comp.dwStyle = CFS_POINT;
	comp.ptCurrentPos.x = st->caret.left;
	comp.ptCurrentPos.y = st->caret.top;
	ImmSetCompositionWindow(imc, &comp);
	ImmReleaseContext(hwnd, imc);
}

// endComposition sends the end of the composition, if it was sent.
static void endComposition(HWND hwnd, imeState* st) {
	if (st->composing) {
		st->composing = FALSE;
		winIMEComposition(hwnd, "", 0, 0);
	}
}

// compositionString returns the given composition string of the input
// method context, allocated with malloc, with its length in UTF-16 units.
static WCHAR* compositionString(HIMC imc, DWORD index, int* n) {
	LONG size = ImmGetCompositionStringW(imc, index, NULL, 0);
	*n = 0;
	if (size <= 0) {
		return NULL;
	}
	WCHAR* s = (WCHAR*)malloc(size);
	ImmGetCompositionStringW(imc, index, s, size);
	*n = size / sizeof(WCHAR);
	return s;
}

// sendComposition sends the current composition string and its selected
// range, which is the target clause being converted, or else the cursor.
static void sendComposition(HWND hwnd, HIMC imc, imeState* st) {
	int n;
	WCHAR* s = compositionString(imc, GCS_COMPSTR, &n);
	if (s == NULL || n == 0) {
		free(s);
		endComposition(hwnd, st);
		return;
	}
	int start = -1, end = -1;
	LONG na = ImmGetCompositionStringW(imc, GCS_COMPATTR, NULL, 0);
	if (na > 0) {
		BYTE* attrs = (BYTE*)malloc(na);
		ImmGetCompositionStringW(imc, GCS_COMPATTR, attrs, na);
		for (int i = 0; i < na && i < n; i++) {
			if (attrs[i] == ATTR_TARGET_CONVERTED || attrs[i] == ATTR_TARGET_NOTCONVERTED) {
				if (start < 0) {
					start = i;
				}
				end = i + 1;
			}
		}
		free(attrs);
	}
	if (start < 0) {
		LONG cur = ImmGetCompositionStringW(imc, GCS_CURSORPOS, NULL, 0);
		start = end = (cur < 0 || cur > n) ? n : (int)cur;
	}
	char* text = utf8(s, n);
	st->composing = TRUE;
	winIMEComposition(hwnd, text, runeIndex(s, start), runeIndex(s, end));
	free(text);
	free(s);
}

static LRESULT CALLBACK imeProc(HWND hwnd, UINT msg, WPARAM wParam, LPARAM lParam, UINT_PTR id, DWORD_PTR ref) {
	imeState* st = (imeState*)ref;
	switch (msg) {
	case WM_COGENT_IME: {
		imeUpdate* up = (imeUpdate*)lParam;
		if (!up->active && st->composing) {
			HIMC imc = ImmGetContext(hwnd);
			if (imc != NULL) {
				ImmNotifyIME(imc, NI_COMPOSITIONSTR, CPS_CANCEL, 0);
				ImmReleaseContext(hwnd, imc);
			}
			endComposition(hwnd, st);
		}
		st->active = up->active;
		st->caret = up->caret;
		free(up);
		if (st->active) {
			placeWindows(hwnd, st);
		}
		return 0;
	}
	case WM_IME_SETCONTEXT:
		if (st->active) { // the composition is shown in the widget
			lParam &= ~ISC_SHOWUICOMPOSITIONWINDOW;
		}
		break;
	case WM_IME_STARTCOMPOSITION:
		if (st->active) {
			placeWindows(hwnd, st);
			return 0; // no default composition window
		}
		break;
	case WM_IME_COMPOSITION: {
		if (!st->active) {
			break;
		}
		HIMC imc = ImmGetContext(hwnd);
		if (imc == NULL) {
			break;
		}
		if (lParam & GCS_RESULTSTR) {
			int n;
			WCHAR* s = compositionString(imc, GCS_RESULTSTR, &n);
			endComposition(hwnd, st);
			if (s != NULL && n > 0) {
				char* text = utf8(s, n);
				winIMECommit(hwnd, text);
				free(text);
			}
			free(s);
		}
		if (lParam & GCS_COMPSTR) {
			sendComposition(hwnd, imc, st);
		}
		placeWindows(hwnd, st);
		ImmReleaseContext(hwnd, imc);
		return 0; // the result is not also sent as WM_IME_CHAR / WM_CHAR
	}
	case WM_IME_ENDCOMPOSITION:
		endComposition(hwnd, st);
		break;
	case WM_NCDESTROY:
		RemoveWindowSubclass(hwnd, imeProc, id);
		free(st);
		break;
	}
	return DefSubclassProc(hwnd, msg, wParam, lParam);
}

// initIME sets up input methods for the given window.
// It must be called on the main thread.
void initIME(HWND hwnd) {
	imeState* st = (imeState*)calloc(1, sizeof(imeState));
	SetWindowSubclass(hwnd, imeProc, 1, (DWORD_PTR)st);
}

// setTextInput sets whether text input is active in the given window,
// and the caret, in client pixels. It can be called from any thread:
// the state is updated on the window thread. When text input stops,
// any composition is canceled.
void setTextInput(HWND hwnd, int active, int x, int y, int w, int h) {
	imeUpdate* up = (imeUpdate*)malloc(sizeof(imeUpdate));
	up->active = active != 0;
	up->caret.left = x;
	up->caret.top = y;
	up->caret.right = x + w;
	up->caret.bottom = y + h;
	if (!PostMessageW(hwnd, WM_COGENT_IME, 0, (LPARAM)up)) {
		free(up);
	}
}
