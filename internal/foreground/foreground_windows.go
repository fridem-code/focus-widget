// Package foreground reports foreground window changes on Windows and pins a
// window to the top of the topmost band without activating it. The widget
// uses both to stay above apps that open over always-on-top windows (Paint
// did, docs/BUGS.md B2).
package foreground

import (
	"fmt"
	"runtime"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	eventSystemForeground  = 0x0003
	winEventOutOfContext   = 0x0000
	winEventSkipOwnProcess = 0x0002
	wmQuit                 = 0x0012

	swpNoSize     = 0x0001
	swpNoMove     = 0x0002
	swpNoActivate = 0x0010
)

var hwndTopmost = ^uintptr(0) // (HWND)-1

var (
	user32                       = windows.NewLazySystemDLL("user32.dll")
	procSetWinEventHook          = user32.NewProc("SetWinEventHook")
	procUnhookWinEvent           = user32.NewProc("UnhookWinEvent")
	procGetMessage               = user32.NewProc("GetMessageW")
	procPostThreadMessage        = user32.NewProc("PostThreadMessageW")
	procSetWindowPos             = user32.NewProc("SetWindowPos")
	procGetForeground            = user32.NewProc("GetForegroundWindow")
	procSetForeground            = user32.NewProc("SetForegroundWindow")
	procGetWindowThreadProcessID = user32.NewProc("GetWindowThreadProcessId")
)

// msg mirrors the Win32 MSG structure; only its size matters here.
type msg struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      struct{ x, y int32 }
	private uint32
}

// Watch calls changed whenever a window of another process becomes the
// foreground window, until stop is called. The hook lives on its own locked
// OS thread, which pumps messages as out-of-context WinEvent hooks require;
// changed runs on that thread.
func Watch(changed func()) (stop func(), err error) {
	started := make(chan error, 1)
	var threadID uint32
	go func() {
		runtime.LockOSThread()
		threadID = windows.GetCurrentThreadId()
		callback := windows.NewCallback(func(_, _, _, _, _, _, _ uintptr) uintptr {
			changed()
			return 0
		})
		hook, _, callErr := procSetWinEventHook.Call(eventSystemForeground, eventSystemForeground,
			0, callback, 0, 0, winEventOutOfContext|winEventSkipOwnProcess)
		if hook == 0 {
			started <- fmt.Errorf("SetWinEventHook: %w", callErr)
			return
		}
		started <- nil
		var m msg
		for {
			// The hook's callbacks are delivered while this thread waits here.
			if r, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0); int32(r) <= 0 {
				break
			}
		}
		_, _, _ = procUnhookWinEvent.Call(hook)
	}()
	if err := <-started; err != nil {
		return nil, err
	}
	return func() { _, _, _ = procPostThreadMessage.Call(uintptr(threadID), wmQuit, 0, 0) }, nil
}

// Pin moves hwnd to the top of the topmost band without activating it, so the
// window the user just switched to keeps the keyboard focus.
func Pin(hwnd uintptr) error {
	if r, _, err := procSetWindowPos.Call(hwnd, hwndTopmost, 0, 0, 0, 0,
		swpNoMove|swpNoSize|swpNoActivate); r == 0 {
		return fmt.Errorf("SetWindowPos: %w", err)
	}
	return nil
}

// Keep runs show and hands the keyboard focus back to the window that had it.
// Showing a window activates it, so a widget started by Windows (T19) would
// otherwise take the focus from what the user is doing. While this process is
// in the foreground Windows lets it pass the focus on. WebView2 can take the
// focus again a moment after the show, so it is handed back once more later,
// unless the user has moved on to another app by then.
func Keep(show func()) {
	prev, _, _ := procGetForeground.Call()
	show()
	if prev == 0 || ownedByUs(prev) {
		return
	}
	giveBack := func() {
		if cur, _, _ := procGetForeground.Call(); cur != prev && ownedByUs(cur) {
			_, _, _ = procSetForeground.Call(prev)
		}
	}
	giveBack()
	time.AfterFunc(keepRetry, giveBack)
}

// keepRetry: when WebView2 has usually finished taking the focus after a show.
const keepRetry = 600 * time.Millisecond

func ownedByUs(hwnd uintptr) bool {
	var pid uint32
	_, _, _ = procGetWindowThreadProcessID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	return pid == windows.GetCurrentProcessId()
}
