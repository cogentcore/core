// Copyright 2026 Cogent Core. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build linux && (x11 || !wayland)

// Drag-and-drop with other windows and apps on X11, using the XDND
// protocol (https://www.freedesktop.org/wiki/Specifications/XDND/).
// GLFW only supports file drops, without any feedback during the drag,
// so we use our own X connection, on its own thread, with an invisible
// proxy window: each GLFW window has the XdndProxy property set to it,
// so that drags over the windows are sent to us instead of GLFW, and
// we start drags from it, as the owner of the XdndSelection. Drags
// within the app use the same protocol.

#include <X11/Xlib.h>
#include <X11/Xatom.h>
#include <X11/cursorfont.h>
#include <X11/keysym.h>
#include <pthread.h>
#include <poll.h>
#include <stdbool.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <time.h>
#include <unistd.h>

#include "_cgo_export.h"

static Display* xd;
static Window root, proxy;
static int wakeFds[2] = {-1, -1};
static pthread_mutex_t mu = PTHREAD_MUTEX_INITIALIZER;

static Atom aAware, aProxy, aEnter, aPosition, aStatus, aLeave, aDrop, aFinished;
static Atom aSelection, aTypeList, aCopy, aMove, aLink;
static Atom aUriList, aMime, aUTF8, aTextPlain, aTargets, aProp;

// the modifier bits passed to Go
enum { modShift = 1, modControl = 2, modAlt = 4, modMeta = 8 };

static int stateMods(unsigned int state) {
	int m = 0;
	if (state & ShiftMask) m |= modShift;
	if (state & ControlMask) m |= modControl;
	if (state & Mod1Mask) m |= modAlt;
	if (state & Mod4Mask) m |= modMeta;
	return m;
}

static double now() {
	struct timespec ts;
	clock_gettime(CLOCK_MONOTONIC, &ts);
	return ts.tv_sec + ts.tv_nsec * 1e-9;
}

////////  Errors

// X errors for windows that go away during a drag are expected,
// and must not exit the app, as the default handler does.

static int ignoreErrors(Display* d, XErrorEvent* e) { return 0; }

static int (*prevHandler)(Display*, XErrorEvent*);

static void grabErrors() {
	prevHandler = XSetErrorHandler(ignoreErrors);
}

static void releaseErrors() {
	XSync(xd, False);
	XSetErrorHandler(prevHandler);
}

////////  Commands

// Commands from other threads are run on the X thread,
// which is woken up by writing to wakeFds.

typedef struct cmd {
	int typ;
	Window win;
	struct cmd* next;
} cmd;

enum { cmdAdd, cmdRemove, cmdDrag };

static cmd* cmds;

static void postCmd(int typ, Window win) {
	cmd* c = calloc(1, sizeof(cmd));
	c->typ = typ;
	c->win = win;
	pthread_mutex_lock(&mu);
	cmd** p = &cmds;
	while (*p != NULL) p = &(*p)->next;
	*p = c;
	pthread_mutex_unlock(&mu);
	write(wakeFds[1], "x", 1);
}

////////  Drag source

// the data of a drag, with uri-list (files), mime and text data
typedef struct {
	Window window; // the GLFW window the drag is from
	char* uris;
	int urislen;
	char* mime;
	int mimelen;
	char* text;
	int textlen;
} dragData;

static void freeDragData(dragData* d) {
	free(d->uris);
	free(d->mime);
	free(d->text);
	memset(d, 0, sizeof(dragData));
}

// pending is the data for the next drag, set by startDrag
static dragData pending;

static struct {
	bool active; // pointer grabbed, dragging
	bool waiting; // dropped, waiting for XdndFinished
	double deadline; // for waiting
	dragData data;
	Window target; // the current target window, or None
	Window targetProxy; // where to send messages for the target
	int version; // XDND version of the target
	bool accepted; // the target accepts the drop
	Atom action;
	Cursor cursor;
} src;

// sendXdnd sends the given XDND client message about window win to the window to.
static void sendXdnd(Window to, Window win, Atom type, long l0, long l1, long l2, long l3, long l4) {
	XEvent e;
	memset(&e, 0, sizeof(e));
	e.xclient.type = ClientMessage;
	e.xclient.display = xd;
	e.xclient.window = win;
	e.xclient.message_type = type;
	e.xclient.format = 32;
	e.xclient.data.l[0] = l0;
	e.xclient.data.l[1] = l1;
	e.xclient.data.l[2] = l2;
	e.xclient.data.l[3] = l3;
	e.xclient.data.l[4] = l4;
	XSendEvent(xd, to, False, NoEventMask, &e);
}

// windowProperty returns the first item of the given property of
// the given 32-bit type on the given window, or 0.
static unsigned long windowProperty(Window w, Atom prop, Atom type) {
	Atom actual;
	int format;
	unsigned long n, after;
	unsigned char* data = NULL;
	unsigned long v = 0;
	if (XGetWindowProperty(xd, w, prop, 0, 1, False, type, &actual, &format, &n, &after, &data) == Success && data != NULL) {
		if (actual == type && format == 32 && n > 0) {
			v = ((unsigned long*)data)[0];
		}
		XFree(data);
	}
	return v;
}

// findTarget returns the XDND aware window at the given root position,
// with its XDND version and the window to send messages to.
static Window findTarget(int rx, int ry, int* version, Window* to) {
	Window w = root;
	for (int depth = 0; depth < 16; depth++) {
		int x, y;
		Window child = None;
		if (!XTranslateCoordinates(xd, root, w, rx, ry, &x, &y, &child) || child == None) {
			return None;
		}
		int v = (int)windowProperty(child, aAware, XA_ATOM);
		if (v > 0) {
			*version = v < 5 ? v : 5;
			Window p = (Window)windowProperty(child, aProxy, XA_WINDOW);
			*to = p != None ? p : child;
			return child;
		}
		w = child;
	}
	return None;
}

// sourceAction returns the action for a drag from this app to the
// current target, as in events.DefaultModBits within the app, and
// copy for other apps, which do not move files out from under us.
static Atom sourceAction(unsigned int state) {
	if (src.targetProxy != proxy) return aCopy;
	if (state & ControlMask) return aCopy;
	if (state & ShiftMask) return aMove;
	if (state & Mod1Mask) return aLink;
	return aCopy;
}

static void sourceLeave() {
	if (src.target != None) {
		sendXdnd(src.targetProxy, src.target, aLeave, proxy, 0, 0, 0, 0);
	}
	src.target = None;
	src.accepted = false;
}

static void sourceMove(int rx, int ry, unsigned int state, Time t) {
	grabErrors();
	int version = 0;
	Window to = None;
	Window target = findTarget(rx, ry, &version, &to);
	if (target != src.target) {
		sourceLeave();
		if (target != None) {
			src.target = target;
			src.targetProxy = to;
			src.version = version;
			Atom types[3] = {None, None, None};
			if (src.data.urislen > 0) {
				types[0] = aUriList;
			} else {
				int n = 0;
				if (src.data.mimelen > 0) types[n++] = aMime;
				if (src.data.textlen > 0) {
					types[n++] = aUTF8;
					types[n++] = aTextPlain;
				}
			}
			sendXdnd(to, target, aEnter, proxy, (long)version << 24, types[0], types[1], types[2]);
		}
	}
	if (src.target != None) {
		sendXdnd(src.targetProxy, src.target, aPosition, proxy, 0, (rx << 16) | (ry & 0xFFFF), t, sourceAction(state));
	}
	releaseErrors();
}

// sendButtonRelease sends the release of the mouse button to the GLFW
// window the drag is from, which does not get it during the drag,
// and which ends the drag in Go.
static void sendButtonRelease(int rx, int ry) {
	XEvent e;
	memset(&e, 0, sizeof(e));
	e.xbutton.type = ButtonRelease;
	e.xbutton.display = xd;
	e.xbutton.window = src.data.window;
	e.xbutton.root = root;
	e.xbutton.time = CurrentTime;
	e.xbutton.x_root = rx;
	e.xbutton.y_root = ry;
	Window child;
	grabErrors();
	XTranslateCoordinates(xd, root, src.data.window, rx, ry, &e.xbutton.x, &e.xbutton.y, &child);
	e.xbutton.state = Button1Mask;
	e.xbutton.button = Button1;
	e.xbutton.same_screen = True;
	XSendEvent(xd, src.data.window, True, ButtonReleaseMask, &e);
	releaseErrors();
}

// sourceEnd ends the pointer grab of a drag, sending the drop to the
// target if drop is true and it accepts it, or else leaving it.
static void sourceEnd(bool drop, int rx, int ry, Time t) {
	XUngrabPointer(xd, CurrentTime);
	XUngrabKeyboard(xd, CurrentTime);
	src.active = false;
	grabErrors();
	if (drop && src.target != None && src.accepted) {
		sendXdnd(src.targetProxy, src.target, aDrop, proxy, 0, t, 0, 0);
		src.waiting = true;
		src.deadline = now() + 5;
	} else {
		sourceLeave();
	}
	releaseErrors();
	sendButtonRelease(rx, ry);
	if (!src.waiting) {
		freeDragData(&src.data);
	}
	XFlush(xd);
}

// sourceStart starts dragging the pending data.
static void sourceStart() {
	pthread_mutex_lock(&mu);
	freeDragData(&src.data);
	src.data = pending;
	memset(&pending, 0, sizeof(pending));
	pthread_mutex_unlock(&mu);
	src.target = None;
	src.waiting = false;
	XSetSelectionOwner(xd, aSelection, proxy, CurrentTime);
	if (src.cursor == None) {
		src.cursor = XCreateFontCursor(xd, XC_hand2);
	}
	// the GLFW window releases its grab of the pointer first (ungrabGLFW),
	// which can take a moment to be processed
	int res = AlreadyGrabbed;
	for (int i = 0; i < 20 && res != GrabSuccess; i++) {
		res = XGrabPointer(xd, root, False, ButtonReleaseMask | PointerMotionMask, GrabModeAsync, GrabModeAsync, None, src.cursor, CurrentTime);
		if (res != GrabSuccess) usleep(10000);
	}
	Window r, c;
	int rx = 0, ry = 0, x, y;
	unsigned int state = 0;
	XQueryPointer(xd, root, &r, &c, &rx, &ry, &x, &y, &state);
	if (res != GrabSuccess) {
		fprintf(stderr, "cogentcore: drag-and-drop could not grab the pointer: %d\n", res);
		sendButtonRelease(rx, ry);
		freeDragData(&src.data);
		return;
	}
	XGrabKeyboard(xd, root, False, GrabModeAsync, GrabModeAsync, CurrentTime);
	src.active = true;
	sourceMove(rx, ry, state, CurrentTime);
}

static void setProperty(Window w, Atom prop, Atom type, char* data, int len) {
	XChangeProperty(xd, w, prop, type, 8, PropModeReplace, (unsigned char*)data, len);
}

// selectionRequest provides the drag data to the target.
static void selectionRequest(XSelectionRequestEvent* r) {
	XEvent ev;
	memset(&ev, 0, sizeof(ev));
	ev.xselection.type = SelectionNotify;
	ev.xselection.display = xd;
	ev.xselection.requestor = r->requestor;
	ev.xselection.selection = r->selection;
	ev.xselection.target = r->target;
	ev.xselection.time = r->time;
	ev.xselection.property = None;
	Atom prop = r->property != None ? r->property : r->target;
	dragData* d = &src.data;
	grabErrors();
	if (r->target == aTargets) {
		Atom ts[5];
		int n = 0;
		ts[n++] = aTargets;
		if (d->urislen > 0) ts[n++] = aUriList;
		if (d->mimelen > 0) ts[n++] = aMime;
		if (d->textlen > 0) {
			ts[n++] = aUTF8;
			ts[n++] = aTextPlain;
		}
		XChangeProperty(xd, r->requestor, prop, XA_ATOM, 32, PropModeReplace, (unsigned char*)ts, n);
		ev.xselection.property = prop;
	} else if (r->target == aUriList && d->urislen > 0) {
		setProperty(r->requestor, prop, aUriList, d->uris, d->urislen);
		ev.xselection.property = prop;
	} else if (r->target == aMime && d->mimelen > 0) {
		setProperty(r->requestor, prop, aMime, d->mime, d->mimelen);
		ev.xselection.property = prop;
	} else if ((r->target == aUTF8 || r->target == aTextPlain || r->target == XA_STRING) && d->textlen > 0) {
		setProperty(r->requestor, prop, r->target, d->text, d->textlen);
		ev.xselection.property = prop;
	}
	XSendEvent(xd, r->requestor, False, NoEventMask, &ev);
	releaseErrors();
}

////////  Drop target

static struct {
	Window source; // the source of the current drag, or None
	Window target; // the GLFW window the drag is over
	Atom type; // the type of the data that we take, or None
	bool fromApp; // the drag is from this app
	int x, y, mods; // the last position, in the target
	bool waiting; // dropped, waiting for the data
	Atom action;
} dst;

// hasType returns whether the given types include the given type.
static bool hasType(Atom* types, int n, Atom t) {
	for (int i = 0; i < n; i++) {
		if (types[i] == t) return true;
	}
	return false;
}

static void dstEnter(XClientMessageEvent* e) {
	dst.source = e->data.l[0];
	dst.target = e->window;
	dst.fromApp = dst.source == proxy;
	dst.waiting = false;
	Atom* types = NULL;
	Atom msgTypes[3] = {e->data.l[2], e->data.l[3], e->data.l[4]};
	unsigned long n = 3;
	unsigned char* data = NULL;
	if (e->data.l[1] & 1) { // more than 3 types: in XdndTypeList
		Atom actual;
		int format;
		unsigned long after;
		grabErrors();
		if (XGetWindowProperty(xd, dst.source, aTypeList, 0, 1024, False, XA_ATOM, &actual, &format, &n, &after, &data) != Success || data == NULL) {
			n = 0;
		}
		releaseErrors();
		types = (Atom*)data;
	} else {
		types = msgTypes;
	}
	dst.type = None;
	if (hasType(types, n, aUriList)) {
		dst.type = aUriList;
	} else if (hasType(types, n, aMime)) {
		dst.type = aMime;
	}
	if (data != NULL) XFree(data);
}

static Atom dstAction(Atom proposed) {
	if (!dst.fromApp) return aCopy;
	if (dst.mods & modControl) return aCopy;
	if (dst.mods & modShift) return aMove;
	if (dst.mods & modAlt) return aLink;
	return aCopy;
}

static void dstPosition(XClientMessageEvent* e) {
	if ((Window)e->data.l[0] != dst.source) return;
	dst.target = e->window;
	int rx = (e->data.l[2] >> 16) & 0xFFFF, ry = e->data.l[2] & 0xFFFF;
	Window r, c;
	int qx, qy, x, y;
	unsigned int state = 0;
	grabErrors();
	XQueryPointer(xd, root, &r, &c, &qx, &qy, &x, &y, &state);
	XTranslateCoordinates(xd, root, dst.target, rx, ry, &dst.x, &dst.y, &c);
	dst.mods = stateMods(state);
	bool accept = dst.type != None;
	dst.action = accept ? dstAction(e->data.l[4]) : None;
	// accept, and send a position for every move
	sendXdnd(dst.source, dst.source, aStatus, dst.target, accept ? 3 : 2, 0, 0, dst.action);
	releaseErrors();
	if (accept) {
		x11DragMove(dst.target, false, dst.x, dst.y, dst.mods, dst.fromApp);
	}
}

static void dstLeave(XClientMessageEvent* e) {
	if ((Window)e->data.l[0] != dst.source) return;
	if (dst.type != None) {
		x11DragMove(dst.target, true, 0, 0, 0, dst.fromApp);
	}
	dst.source = None;
}

static void dstFinished(bool ok) {
	grabErrors();
	sendXdnd(dst.source, dst.source, aFinished, dst.target, ok ? 1 : 0, ok ? dst.action : None, 0, 0);
	releaseErrors();
	dst.source = None;
	dst.waiting = false;
}

static void dstDrop(XClientMessageEvent* e) {
	if ((Window)e->data.l[0] != dst.source) return;
	if (dst.type == None) {
		dstFinished(false);
		return;
	}
	Time t = e->data.l[2] != 0 ? (Time)e->data.l[2] : CurrentTime;
	XConvertSelection(xd, aSelection, dst.type, aProp, proxy, t);
	dst.waiting = true;
}

// dstData reads the dropped data, and sends the drop to Go.
static void dstData(XSelectionEvent* e) {
	if (!dst.waiting) return;
	if (e->property == None) {
		x11DragMove(dst.target, true, 0, 0, 0, dst.fromApp);
		dstFinished(false);
		return;
	}
	Atom actual;
	int format;
	unsigned long n = 0, after;
	unsigned char* data = NULL;
	XGetWindowProperty(xd, proxy, aProp, 0, 0x7FFFFFFF, True, AnyPropertyType, &actual, &format, &n, &after, &data);
	bool ok = data != NULL && format == 8;
	if (ok) {
		if (dst.type == aUriList) {
			x11DragDrop(dst.target, dst.x, dst.y, dst.mods, dst.fromApp, (char*)data, (int)n, NULL, 0);
		} else {
			x11DragDrop(dst.target, dst.x, dst.y, dst.mods, dst.fromApp, NULL, 0, (char*)data, (int)n);
		}
	} else {
		x11DragMove(dst.target, true, 0, 0, 0, dst.fromApp);
	}
	if (data != NULL) XFree(data);
	dstFinished(ok);
}

////////  Event loop

static void clientMessage(XClientMessageEvent* e) {
	Atom t = e->message_type;
	if (t == aEnter) {
		dstEnter(e);
	} else if (t == aPosition) {
		dstPosition(e);
	} else if (t == aLeave) {
		dstLeave(e);
	} else if (t == aDrop) {
		dstDrop(e);
	} else if (t == aStatus) { // to us as the source
		if (src.active && (Window)e->data.l[0] == src.target) {
			src.accepted = e->data.l[1] & 1;
			src.action = e->data.l[4];
		}
	} else if (t == aFinished) {
		if (src.waiting) {
			src.waiting = false;
			freeDragData(&src.data);
		}
	}
}

static void handleEvent(XEvent* e) {
	switch (e->type) {
	case ClientMessage:
		clientMessage(&e->xclient);
		break;
	case SelectionRequest:
		selectionRequest(&e->xselectionrequest);
		break;
	case SelectionNotify:
		dstData(&e->xselection);
		break;
	case MotionNotify:
		if (src.active) {
			// only the last motion matters
			while (XCheckTypedEvent(xd, MotionNotify, e));
			sourceMove(e->xmotion.x_root, e->xmotion.y_root, e->xmotion.state, e->xmotion.time);
		}
		break;
	case ButtonRelease:
		if (src.active) {
			sourceEnd(true, e->xbutton.x_root, e->xbutton.y_root, e->xbutton.time);
		}
		break;
	case KeyPress:
		if (src.active && XLookupKeysym(&e->xkey, 0) == XK_Escape) {
			sourceEnd(false, e->xkey.x_root, e->xkey.y_root, e->xkey.time);
		}
		break;
	}
}

static void runCmds() {
	pthread_mutex_lock(&mu);
	cmd* c = cmds;
	cmds = NULL;
	pthread_mutex_unlock(&mu);
	while (c != NULL) {
		switch (c->typ) {
		case cmdAdd:
			grabErrors();
			XChangeProperty(xd, c->win, aProxy, XA_WINDOW, 32, PropModeReplace, (unsigned char*)&proxy, 1);
			releaseErrors();
			break;
		case cmdRemove:
			break; // the window is destroyed with its properties
		case cmdDrag:
			sourceStart();
			break;
		}
		cmd* next = c->next;
		free(c);
		c = next;
	}
	XFlush(xd);
}

static void* eventLoop(void* arg) {
	struct pollfd fds[2] = {{ConnectionNumber(xd), POLLIN, 0}, {wakeFds[0], POLLIN, 0}};
	for (;;) {
		while (XPending(xd)) {
			XEvent e;
			XNextEvent(xd, &e);
			handleEvent(&e);
		}
		XFlush(xd);
		if (poll(fds, 2, 250) > 0 && (fds[1].revents & POLLIN)) {
			char b[64];
			read(wakeFds[0], b, sizeof(b));
			runCmds();
		}
		if (src.waiting && now() > src.deadline) { // target never finished
			src.waiting = false;
			freeDragData(&src.data);
		}
	}
	return NULL;
}

// initDrag starts drag-and-drop, returning false if it is not available.
static bool initDrag() {
	static int inited = 0; // 1 = ok, -1 = failed
	if (inited != 0) return inited > 0;
	inited = -1;
	xd = XOpenDisplay(NULL);
	if (xd == NULL) return false;
	if (pipe(wakeFds) != 0) return false;
	root = DefaultRootWindow(xd);
	char* names[] = {"XdndAware", "XdndProxy", "XdndEnter", "XdndPosition", "XdndStatus", "XdndLeave", "XdndDrop", "XdndFinished",
		"XdndSelection", "XdndTypeList", "XdndActionCopy", "XdndActionMove", "XdndActionLink",
		"text/uri-list", "application/x-cogentcore-mimedata", "UTF8_STRING", "text/plain;charset=utf-8", "TARGETS", "COGENT_XDND"};
	Atom* atoms[] = {&aAware, &aProxy, &aEnter, &aPosition, &aStatus, &aLeave, &aDrop, &aFinished,
		&aSelection, &aTypeList, &aCopy, &aMove, &aLink,
		&aUriList, &aMime, &aUTF8, &aTextPlain, &aTargets, &aProp};
	int n = sizeof(names) / sizeof(names[0]);
	Atom vals[sizeof(names) / sizeof(names[0])];
	XInternAtoms(xd, names, n, False, vals);
	for (int i = 0; i < n; i++) *atoms[i] = vals[i];

	XSetWindowAttributes attr;
	memset(&attr, 0, sizeof(attr));
	attr.override_redirect = True;
	proxy = XCreateWindow(xd, root, -10, -10, 1, 1, 0, 0, InputOnly, CopyFromParent, CWOverrideRedirect, &attr);
	long version = 5;
	XChangeProperty(xd, proxy, aAware, XA_ATOM, 32, PropModeReplace, (unsigned char*)&version, 1);
	XChangeProperty(xd, proxy, aProxy, XA_WINDOW, 32, PropModeReplace, (unsigned char*)&proxy, 1);
	XFlush(xd);

	pthread_t th;
	if (pthread_create(&th, NULL, eventLoop, NULL) != 0) return false;
	pthread_detach(th);
	inited = 1;
	return true;
}

// registerDrag sets up drag-and-drop for the given GLFW window.
void registerDrag(Window win) {
	if (initDrag()) postCmd(cmdAdd, win);
}

// unregisterDrag removes drag-and-drop for the given GLFW window.
void unregisterDrag(Window win) {
	if (initDrag()) postCmd(cmdRemove, win);
}

// ungrabGLFW releases the pointer grab of the GLFW display,
// from the mouse press that started a drag, so that we can grab it.
// It must be called on the main thread.
void ungrabGLFW(Display* gd) {
	XUngrabPointer(gd, CurrentTime);
	XSync(gd, False);
}

static char* copyBytes(void* b, int n) {
	if (n <= 0) return NULL;
	char* c = malloc(n);
	memcpy(c, b, n);
	return c;
}

// startDrag starts dragging the given uri-list of files, or else mime
// data and text, from the given GLFW window, with the mouse button down,
// after ungrabGLFW. It returns false if drag-and-drop is not available.
bool startDrag(Window win, void* uris, int urislen, void* mime, int mimelen, void* text, int textlen) {
	if (!initDrag()) return false;
	pthread_mutex_lock(&mu);
	freeDragData(&pending);
	pending.window = win;
	pending.uris = copyBytes(uris, urislen);
	pending.urislen = urislen;
	pending.mime = copyBytes(mime, mimelen);
	pending.mimelen = mimelen;
	pending.text = copyBytes(text, textlen);
	pending.textlen = textlen;
	pthread_mutex_unlock(&mu);
	postCmd(cmdDrag, win);
	return true;
}
