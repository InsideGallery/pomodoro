//go:build linux && !android

package platform

/*
#cgo linux pkg-config: x11
#include <stdlib.h>
#include <X11/Xlib.h>
#include <X11/Xatom.h>
#include <string.h>

static void x11_hide(unsigned long wid) {
    Display *dpy = XOpenDisplay(NULL);
    if (!dpy) return;

    Window win = (Window)wid;
    Atom wmState = XInternAtom(dpy, "_NET_WM_STATE", False);
    Atom skipTaskbar = XInternAtom(dpy, "_NET_WM_STATE_SKIP_TASKBAR", False);
    Atom skipPager = XInternAtom(dpy, "_NET_WM_STATE_SKIP_PAGER", False);

    // Add SKIP_TASKBAR + SKIP_PAGER
    XEvent ev;
    memset(&ev, 0, sizeof(ev));
    ev.xclient.type = ClientMessage;
    ev.xclient.window = win;
    ev.xclient.message_type = wmState;
    ev.xclient.format = 32;
    ev.xclient.data.l[0] = 1; // _NET_WM_STATE_ADD
    ev.xclient.data.l[1] = (long)skipTaskbar;
    ev.xclient.data.l[2] = (long)skipPager;
    ev.xclient.data.l[3] = 1;
    XSendEvent(dpy, DefaultRootWindow(dpy), False,
        SubstructureRedirectMask | SubstructureNotifyMask, &ev);

    XUnmapWindow(dpy, win);
    XFlush(dpy);
    XCloseDisplay(dpy);
}

static void x11_set_above(unsigned long wid, int enable) {
    Display *dpy = XOpenDisplay(NULL);
    if (!dpy) return;

    Window win = (Window)wid;
    Atom wmState = XInternAtom(dpy, "_NET_WM_STATE", False);
    Atom above = XInternAtom(dpy, "_NET_WM_STATE_ABOVE", False);

    XEvent ev;
    memset(&ev, 0, sizeof(ev));
    ev.xclient.type = ClientMessage;
    ev.xclient.window = win;
    ev.xclient.message_type = wmState;
    ev.xclient.format = 32;
    ev.xclient.data.l[0] = enable ? 1 : 0;
    ev.xclient.data.l[1] = (long)above;
    ev.xclient.data.l[2] = 0;
    ev.xclient.data.l[3] = 1;
    XSendEvent(dpy, DefaultRootWindow(dpy), False,
        SubstructureRedirectMask | SubstructureNotifyMask, &ev);

    XFlush(dpy);
    XCloseDisplay(dpy);
}

static void x11_show(unsigned long wid) {
    Display *dpy = XOpenDisplay(NULL);
    if (!dpy) return;

    Window win = (Window)wid;
    XMapWindow(dpy, win);

    Atom wmState = XInternAtom(dpy, "_NET_WM_STATE", False);
    Atom skipTaskbar = XInternAtom(dpy, "_NET_WM_STATE_SKIP_TASKBAR", False);
    Atom skipPager = XInternAtom(dpy, "_NET_WM_STATE_SKIP_PAGER", False);

    // Remove SKIP_TASKBAR + SKIP_PAGER
    XEvent ev;
    memset(&ev, 0, sizeof(ev));
    ev.xclient.type = ClientMessage;
    ev.xclient.window = win;
    ev.xclient.message_type = wmState;
    ev.xclient.format = 32;
    ev.xclient.data.l[0] = 0; // _NET_WM_STATE_REMOVE
    ev.xclient.data.l[1] = (long)skipTaskbar;
    ev.xclient.data.l[2] = (long)skipPager;
    ev.xclient.data.l[3] = 1;
    XSendEvent(dpy, DefaultRootWindow(dpy), False,
        SubstructureRedirectMask | SubstructureNotifyMask, &ev);

    XFlush(dpy);
    XCloseDisplay(dpy);
}

static void x11_raise(unsigned long wid) {
    Display *dpy = XOpenDisplay(NULL);
    if (!dpy) return;

    Window win = (Window)wid;
    XRaiseWindow(dpy, win);
    XSetInputFocus(dpy, win, RevertToParent, CurrentTime);
    XFlush(dpy);
    XCloseDisplay(dpy);
}

static unsigned long x11_find_window(const char *title) {
    Display *dpy = XOpenDisplay(NULL);
    if (!dpy) return 0;

    Window root = DefaultRootWindow(dpy);
    Atom netClientList = XInternAtom(dpy, "_NET_CLIENT_LIST", False);
    Atom netWmName = XInternAtom(dpy, "_NET_WM_NAME", False);
    Atom utf8String = XInternAtom(dpy, "UTF8_STRING", False);

    Atom actualType;
    int actualFormat;
    unsigned long nItems, bytesAfter;
    unsigned char *data = NULL;
    unsigned long result = 0;

    if (XGetWindowProperty(dpy, root, netClientList, 0, 1024, False,
            XA_WINDOW, &actualType, &actualFormat, &nItems, &bytesAfter, &data) == Success && data) {
        Window *windows = (Window *)data;
        for (unsigned long i = 0; i < nItems; i++) {
            unsigned char *name = NULL;
            Atom nameType;
            int nameFormat;
            unsigned long nameItems, nameBytesAfter;

            if (XGetWindowProperty(dpy, windows[i], netWmName, 0, 256, False,
                    utf8String, &nameType, &nameFormat, &nameItems, &nameBytesAfter, &name) == Success && name) {
                if (strstr((char *)name, title) != NULL) {
                    result = (unsigned long)windows[i];
                    XFree(name);
                    break;
                }
                XFree(name);
            }
        }
        XFree(data);
    }

    XCloseDisplay(dpy);
    return result;
}
*/
import "C"

import "unsafe"

// FindWindowID finds the X11 window ID by title using _NET_CLIENT_LIST.
func FindWindowID(title string) uint64 {
	cs := C.CString(title)
	defer C.free(unsafe.Pointer(cs))
	return uint64(C.x11_find_window(cs))
}

var savedWID uint64

// HideWindow unmaps a window and removes it from the taskbar.
func HideWindow(title string) {
	wid := FindWindowID(title)
	if wid == 0 {
		return
	}
	savedWID = wid
	C.x11_hide(C.ulong(wid))
}

// SetAlwaysOnTop sets or clears the always-on-top state for the window.
func SetAlwaysOnTop(title string, enable bool) {
	wid := savedWID
	if wid == 0 {
		wid = FindWindowID(title)
	}

	if wid == 0 {
		return
	}

	flag := C.int(0)
	if enable {
		flag = 1
	}

	C.x11_set_above(C.ulong(wid), flag)
}

// RaiseWindow brings the window to the front and gives it focus.
func RaiseWindow(title string) {
	wid := savedWID
	if wid == 0 {
		wid = FindWindowID(title)
	}

	if wid == 0 {
		return
	}

	C.x11_raise(C.ulong(wid))
}

// ShowWindow maps a window and restores it on the taskbar.
func ShowWindow(title string) {
	wid := savedWID
	if wid == 0 {
		wid = FindWindowID(title)
	}
	if wid == 0 {
		return
	}
	C.x11_show(C.ulong(wid))
}
