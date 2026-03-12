package platform

/*
#cgo linux pkg-config: x11
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
*/
import "C"

import (
	"fmt"
	"os/exec"
	"strings"
)

// FindWindowID finds the X11 window ID by title using wmctrl.
func FindWindowID(title string) uint64 {
	out, err := exec.Command("wmctrl", "-l").Output()
	if err != nil {
		return 0
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, title) {
			fields := strings.Fields(line)
			if len(fields) >= 1 {
				var wid uint64
				if _, err := fmt.Sscanf(fields[0], "0x%x", &wid); err == nil {
					return wid
				}
			}
		}
	}
	return 0
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
