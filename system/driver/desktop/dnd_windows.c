// Copyright 2026 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows

// Drag-and-drop with other windows and apps, using OLE. GLFW only
// supports file drops (WM_DROPFILES), without any feedback during the
// drag, so we register an OLE drop target for each window, which takes
// precedence, and start drags with DoDragDrop.

#define COBJMACROS
#define CINTERFACE
#include <windows.h>
#include <ole2.h>
#include <shlobj.h>
#include <shellapi.h>
#include <stdbool.h>
#include <stdlib.h>

#include "_cgo_export.h"

// the clipboard format for mimedata.Mimes from Cogent Core apps,
// as multipart MIME data
static CLIPFORMAT cfMime;

// dragActive is set during a drag started by startDrag,
// for drags from this app.
static bool dragActive = false;

// the modifier bits passed to Go
enum { modShift = 1, modControl = 2, modAlt = 4, modMeta = 8 };

static int keyMods(DWORD keys) {
	int m = 0;
	if (keys & MK_SHIFT) m |= modShift;
	if (keys & MK_CONTROL) m |= modControl;
	if (keys & MK_ALT) m |= modAlt;
	if ((GetKeyState(VK_LWIN) | GetKeyState(VK_RWIN)) & 0x8000) m |= modMeta;
	return m;
}

static bool hasFormat(IDataObject* obj, CLIPFORMAT cf) {
	FORMATETC fe = {cf, NULL, DVASPECT_CONTENT, -1, TYMED_HGLOBAL};
	return IDataObject_QueryGetData(obj, &fe) == S_OK;
}

// dropEffect returns the drop effect for the given keys, as in
// events.DefaultModBits for drags from this app, and copy for drags
// from other apps, as allowed by the source.
static DWORD dropEffect(DWORD keys, DWORD allowed) {
	DWORD eff = DROPEFFECT_COPY;
	if (dragActive) {
		if (keys & MK_CONTROL) {
			eff = DROPEFFECT_COPY;
		} else if (keys & MK_SHIFT) {
			eff = DROPEFFECT_MOVE;
		} else if (keys & MK_ALT) {
			eff = DROPEFFECT_LINK;
		}
	}
	if (allowed & eff) return eff;
	if (allowed & DROPEFFECT_COPY) return DROPEFFECT_COPY;
	return allowed & (DROPEFFECT_MOVE | DROPEFFECT_LINK);
}

////////  Drop target

typedef struct {
	IDropTarget iface;
	LONG refs;
	HWND hwnd;
	IDropTargetHelper* helper; // shows the drag image over the window
	bool accept; // the current drag has data that we accept
} DropTarget;

static HRESULT STDMETHODCALLTYPE dtQueryInterface(IDropTarget* this, REFIID riid, void** ppv) {
	if (IsEqualIID(riid, &IID_IUnknown) || IsEqualIID(riid, &IID_IDropTarget)) {
		*ppv = this;
		IDropTarget_AddRef(this);
		return S_OK;
	}
	*ppv = NULL;
	return E_NOINTERFACE;
}

static ULONG STDMETHODCALLTYPE dtAddRef(IDropTarget* this) {
	return InterlockedIncrement(&((DropTarget*)this)->refs);
}

static ULONG STDMETHODCALLTYPE dtRelease(IDropTarget* this) {
	DropTarget* dt = (DropTarget*)this;
	LONG n = InterlockedDecrement(&dt->refs);
	if (n == 0) {
		if (dt->helper != NULL) IDropTargetHelper_Release(dt->helper);
		free(dt);
	}
	return n;
}

static POINT clientPoint(HWND hwnd, POINTL pt) {
	POINT p = {pt.x, pt.y};
	ScreenToClient(hwnd, &p);
	return p;
}

static HRESULT STDMETHODCALLTYPE dtDragEnter(IDropTarget* this, IDataObject* obj, DWORD keys, POINTL pt, DWORD* effect) {
	DropTarget* dt = (DropTarget*)this;
	dt->accept = hasFormat(obj, CF_HDROP) || hasFormat(obj, cfMime);
	if (!dt->accept) {
		*effect = DROPEFFECT_NONE;
		return S_OK;
	}
	*effect = dropEffect(keys, *effect);
	POINT sp = {pt.x, pt.y};
	if (dt->helper != NULL) IDropTargetHelper_DragEnter(dt->helper, dt->hwnd, obj, &sp, *effect);
	POINT p = clientPoint(dt->hwnd, pt);
	winDragMove(dt->hwnd, false, p.x, p.y, keyMods(keys), dragActive);
	return S_OK;
}

static HRESULT STDMETHODCALLTYPE dtDragOver(IDropTarget* this, DWORD keys, POINTL pt, DWORD* effect) {
	DropTarget* dt = (DropTarget*)this;
	if (!dt->accept) {
		*effect = DROPEFFECT_NONE;
		return S_OK;
	}
	*effect = dropEffect(keys, *effect);
	POINT sp = {pt.x, pt.y};
	if (dt->helper != NULL) IDropTargetHelper_DragOver(dt->helper, &sp, *effect);
	POINT p = clientPoint(dt->hwnd, pt);
	winDragMove(dt->hwnd, false, p.x, p.y, keyMods(keys), dragActive);
	return S_OK;
}

static HRESULT STDMETHODCALLTYPE dtDragLeave(IDropTarget* this) {
	DropTarget* dt = (DropTarget*)this;
	if (!dt->accept) return S_OK;
	if (dt->helper != NULL) IDropTargetHelper_DragLeave(dt->helper);
	winDragMove(dt->hwnd, true, 0, 0, 0, dragActive);
	return S_OK;
}

// utf8 returns a malloc'd UTF-8 copy of the given wide string.
static char* utf8(const WCHAR* ws) {
	int n = WideCharToMultiByte(CP_UTF8, 0, ws, -1, NULL, 0, NULL, NULL);
	char* s = malloc(n);
	WideCharToMultiByte(CP_UTF8, 0, ws, -1, s, n, NULL, NULL);
	return s;
}

static HRESULT STDMETHODCALLTYPE dtDrop(IDropTarget* this, IDataObject* obj, DWORD keys, POINTL pt, DWORD* effect) {
	DropTarget* dt = (DropTarget*)this;
	if (!dt->accept) {
		*effect = DROPEFFECT_NONE;
		return S_OK;
	}
	*effect = dropEffect(keys, *effect);
	POINT sp = {pt.x, pt.y};
	if (dt->helper != NULL) IDropTargetHelper_Drop(dt->helper, obj, &sp, *effect);
	POINT p = clientPoint(dt->hwnd, pt);

	char** files = NULL;
	UINT nfiles = 0;
	void* mime = NULL;
	SIZE_T mimelen = 0;
	FORMATETC fe = {CF_HDROP, NULL, DVASPECT_CONTENT, -1, TYMED_HGLOBAL};
	STGMEDIUM stg;
	if (IDataObject_GetData(obj, &fe, &stg) == S_OK) {
		HDROP hd = (HDROP)GlobalLock(stg.hGlobal);
		if (hd != NULL) {
			nfiles = DragQueryFileW(hd, 0xFFFFFFFF, NULL, 0);
			files = calloc(nfiles + 1, sizeof(char*));
			for (UINT i = 0; i < nfiles; i++) {
				UINT n = DragQueryFileW(hd, i, NULL, 0);
				WCHAR* ws = calloc(n + 1, sizeof(WCHAR));
				DragQueryFileW(hd, i, ws, n + 1);
				files[i] = utf8(ws);
				free(ws);
			}
			GlobalUnlock(stg.hGlobal);
		}
		ReleaseStgMedium(&stg);
	} else {
		fe.cfFormat = cfMime;
		if (IDataObject_GetData(obj, &fe, &stg) == S_OK) {
			void* d = GlobalLock(stg.hGlobal);
			if (d != NULL) {
				mimelen = GlobalSize(stg.hGlobal);
				mime = malloc(mimelen);
				memcpy(mime, d, mimelen);
				GlobalUnlock(stg.hGlobal);
			}
			ReleaseStgMedium(&stg);
		}
	}
	winDragDrop(dt->hwnd, p.x, p.y, keyMods(keys), dragActive, files, nfiles, mime, (int)mimelen);
	for (UINT i = 0; i < nfiles; i++) free(files[i]);
	free(files);
	free(mime);
	return S_OK;
}

static IDropTargetVtbl dropTargetVtbl = {
	dtQueryInterface, dtAddRef, dtRelease,
	dtDragEnter, dtDragOver, dtDragLeave, dtDrop,
};

// registerDrag sets up drag-and-drop for the given window.
// It must be called on the main thread.
void registerDrag(HWND hwnd) {
	static bool inited = false;
	if (!inited) {
		inited = true;
		OleInitialize(NULL);
		cfMime = (CLIPFORMAT)RegisterClipboardFormatW(L"org.cogentcore.mimedata");
	}
	DropTarget* dt = calloc(1, sizeof(DropTarget));
	dt->iface.lpVtbl = &dropTargetVtbl;
	dt->refs = 1;
	dt->hwnd = hwnd;
	CoCreateInstance(&CLSID_DragDropHelper, NULL, CLSCTX_INPROC_SERVER, &IID_IDropTargetHelper, (void**)&dt->helper);
	RegisterDragDrop(hwnd, &dt->iface); // keeps a reference
	IDropTarget_Release(&dt->iface);
}

// unregisterDrag removes drag-and-drop for the given window.
void unregisterDrag(HWND hwnd) {
	RevokeDragDrop(hwnd);
}

////////  Drag source

static HRESULT STDMETHODCALLTYPE dsQueryInterface(IDropSource* this, REFIID riid, void** ppv) {
	if (IsEqualIID(riid, &IID_IUnknown) || IsEqualIID(riid, &IID_IDropSource)) {
		*ppv = this;
		return S_OK;
	}
	*ppv = NULL;
	return E_NOINTERFACE;
}

// the drop source is static, so it is not reference counted
static ULONG STDMETHODCALLTYPE dsAddRef(IDropSource* this) { return 1; }
static ULONG STDMETHODCALLTYPE dsRelease(IDropSource* this) { return 1; }

static HRESULT STDMETHODCALLTYPE dsQueryContinueDrag(IDropSource* this, BOOL escape, DWORD keys) {
	if (escape) return DRAGDROP_S_CANCEL;
	if (!(keys & MK_LBUTTON)) return DRAGDROP_S_DROP;
	// DoDragDrop runs its own event loop on the main thread until the
	// drop, so we run the functions that are waiting for the main thread.
	winDragPoll();
	return S_OK;
}

static HRESULT STDMETHODCALLTYPE dsGiveFeedback(IDropSource* this, DWORD effect) {
	return DRAGDROP_S_USEDEFAULTCURSORS;
}

static IDropSourceVtbl dropSourceVtbl = {
	dsQueryInterface, dsAddRef, dsRelease,
	dsQueryContinueDrag, dsGiveFeedback,
};

static IDropSource dropSource = {&dropSourceVtbl};

// setData sets the given global memory as the data of the given format.
static void setData(IDataObject* obj, CLIPFORMAT cf, HGLOBAL hg) {
	FORMATETC fe = {cf, NULL, DVASPECT_CONTENT, -1, TYMED_HGLOBAL};
	STGMEDIUM stg;
	memset(&stg, 0, sizeof(stg));
	stg.tymed = TYMED_HGLOBAL;
	stg.hGlobal = hg;
	if (IDataObject_SetData(obj, &fe, &stg, TRUE) != S_OK) {
		GlobalFree(hg);
	}
}

// wideLen returns the length of the given UTF-8 string as UTF-16,
// including the terminating null.
static int wideLen(const char* s, int len) {
	return MultiByteToWideChar(CP_UTF8, 0, s, len, NULL, 0);
}

// setDragImage sets the drag image from the given top-down,
// premultiplied BGRA pixels, with its top left at the mouse.
static void setDragImage(IDataObject* obj, void* bgra, int w, int h) {
	IDragSourceHelper* helper = NULL;
	if (CoCreateInstance(&CLSID_DragDropHelper, NULL, CLSCTX_INPROC_SERVER, &IID_IDragSourceHelper, (void**)&helper) != S_OK) {
		return;
	}
	BITMAPINFO bi = {0};
	bi.bmiHeader.biSize = sizeof(BITMAPINFOHEADER);
	bi.bmiHeader.biWidth = w;
	bi.bmiHeader.biHeight = -h; // top down
	bi.bmiHeader.biPlanes = 1;
	bi.bmiHeader.biBitCount = 32;
	bi.bmiHeader.biCompression = BI_RGB;
	void* bits = NULL;
	HBITMAP hbm = CreateDIBSection(NULL, &bi, DIB_RGB_COLORS, &bits, NULL, 0);
	if (hbm != NULL) {
		memcpy(bits, bgra, (size_t)w * h * 4);
		SHDRAGIMAGE di = {0};
		di.sizeDragImage.cx = w;
		di.sizeDragImage.cy = h;
		di.hbmpDragImage = hbm;
		di.crColorKey = CLR_NONE;
		if (IDragSourceHelper_InitializeFromBitmap(helper, &di, obj) != S_OK) {
			DeleteObject(hbm); // otherwise owned by the helper
		}
	}
	IDragSourceHelper_Release(helper);
}

// startDrag drags the given files, or else the given multipart mime data
// and text, from the given window, until the drop, with the given drag image.
// It must be called on the main thread, with the left mouse button down.
void startDrag(HWND hwnd, char** files, int nfiles, void* mime, int mimelen, char* text, int textlen, void* bgra, int imgw, int imgh) {
	IDataObject* obj = NULL;
	if (SHCreateDataObject(NULL, 0, NULL, NULL, &IID_IDataObject, (void**)&obj) != S_OK) {
		return;
	}
	DWORD allowed = DROPEFFECT_COPY | DROPEFFECT_MOVE;
	if (nfiles > 0) {
		// other apps only copy: they do not move files out from under us
		allowed = DROPEFFECT_COPY;
		size_t n = 1; // final null
		for (int i = 0; i < nfiles; i++) n += wideLen(files[i], -1);
		HGLOBAL hg = GlobalAlloc(GMEM_MOVEABLE | GMEM_ZEROINIT, sizeof(DROPFILES) + n * sizeof(WCHAR));
		DROPFILES* df = GlobalLock(hg);
		df->pFiles = sizeof(DROPFILES);
		df->fWide = TRUE;
		WCHAR* p = (WCHAR*)(df + 1);
		int left = (int)n;
		for (int i = 0; i < nfiles; i++) {
			int k = MultiByteToWideChar(CP_UTF8, 0, files[i], -1, p, left);
			p += k;
			left -= k;
		}
		GlobalUnlock(hg);
		setData(obj, CF_HDROP, hg);
	}
	if (mimelen > 0) {
		HGLOBAL hg = GlobalAlloc(GMEM_MOVEABLE, mimelen);
		memcpy(GlobalLock(hg), mime, mimelen);
		GlobalUnlock(hg);
		setData(obj, cfMime, hg);
	}
	if (textlen > 0) {
		int n = wideLen(text, textlen);
		HGLOBAL hg = GlobalAlloc(GMEM_MOVEABLE | GMEM_ZEROINIT, (n + 1) * sizeof(WCHAR));
		MultiByteToWideChar(CP_UTF8, 0, text, textlen, GlobalLock(hg), n);
		GlobalUnlock(hg);
		setData(obj, CF_UNICODETEXT, hg);
	}
	if (bgra != NULL) {
		setDragImage(obj, bgra, imgw, imgh);
	}

	dragActive = true;
	DWORD effect = DROPEFFECT_NONE;
	DoDragDrop(obj, &dropSource, allowed, &effect);
	dragActive = false;
	IDataObject_Release(obj);

	// GLFW does not get the mouse button release, which ends the drag in
	// Go, so we send it.
	POINT pt;
	GetCursorPos(&pt);
	ScreenToClient(hwnd, &pt);
	PostMessageW(hwnd, WM_LBUTTONUP, 0, MAKELPARAM(pt.x, pt.y));
}
