// Package session reports when the user's Windows session is locked and
// unlocked (Win+L, the screen saver's lock, …), for pausing focus while the
// user is away (T20).
package session

import (
	"fmt"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	wmQuit               = 0x0012
	wmWTSSessionChange   = 0x02B1
	wtsSessionLock       = 0x7
	wtsSessionUnlock     = 0x8
	notifyForThisSession = 0
)

var hwndMessage = ^uintptr(2) // (HWND)-3: a message-only window

var (
	user32                = windows.NewLazySystemDLL("user32.dll")
	procRegisterClassEx   = user32.NewProc("RegisterClassExW")
	procCreateWindowEx    = user32.NewProc("CreateWindowExW")
	procDestroyWindow     = user32.NewProc("DestroyWindow")
	procDefWindowProc     = user32.NewProc("DefWindowProcW")
	procGetMessage        = user32.NewProc("GetMessageW")
	procDispatchMessage   = user32.NewProc("DispatchMessageW")
	procPostThreadMessage = user32.NewProc("PostThreadMessageW")
	wtsapi32              = windows.NewLazySystemDLL("wtsapi32.dll")
	procRegister          = wtsapi32.NewProc("WTSRegisterSessionNotification")
	procUnregister        = wtsapi32.NewProc("WTSUnRegisterSessionNotification")
)

type wndClassEx struct {
	size       uint32
	style      uint32
	wndProc    uintptr
	clsExtra   int32
	wndExtra   int32
	instance   windows.Handle
	icon       uintptr
	cursor     uintptr
	background uintptr
	menuName   *uint16
	className  *uint16
	iconSm     uintptr
}

type msg struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      struct{ x, y int32 }
	private uint32
}

// Watch calls changed(true) when the session gets locked and changed(false)
// when it gets unlocked, until stop is called. The notifications go to a
// message-only window on a locked OS thread of its own; changed runs there.
func Watch(changed func(locked bool)) (stop func(), err error) {
	started := make(chan error, 1)
	var threadID uint32
	go func() {
		runtime.LockOSThread()
		threadID = windows.GetCurrentThreadId()
		hwnd, err := createWindow(changed)
		if err != nil {
			started <- err
			return
		}
		if r, _, callErr := procRegister.Call(hwnd, notifyForThisSession); r == 0 {
			_, _, _ = procDestroyWindow.Call(hwnd)
			started <- fmt.Errorf("WTSRegisterSessionNotification: %w", callErr)
			return
		}
		started <- nil
		var m msg
		for {
			if r, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0); int32(r) <= 0 {
				break
			}
			_, _, _ = procDispatchMessage.Call(uintptr(unsafe.Pointer(&m)))
		}
		_, _, _ = procUnregister.Call(hwnd)
		_, _, _ = procDestroyWindow.Call(hwnd)
	}()
	if err := <-started; err != nil {
		return nil, err
	}
	return func() { _, _, _ = procPostThreadMessage.Call(uintptr(threadID), wmQuit, 0, 0) }, nil
}

// The window class is registered once per process, so its procedure reads the
// handler of the current Watch from here. One Watch at a time.
var (
	classRegistered bool
	handler         func(locked bool)
)

func createWindow(changed func(locked bool)) (uintptr, error) {
	handler = changed
	className, _ := windows.UTF16PtrFromString("FocusWidgetSessionWatcher")
	var instance windows.Handle // this executable
	if err := windows.GetModuleHandleEx(0, nil, &instance); err != nil {
		return 0, fmt.Errorf("GetModuleHandle: %w", err)
	}
	proc := windows.NewCallback(func(hwnd, message, wParam, lParam uintptr) uintptr {
		if message == wmWTSSessionChange {
			switch wParam {
			case wtsSessionLock:
				handler(true)
			case wtsSessionUnlock:
				handler(false)
			}
			return 0
		}
		r, _, _ := procDefWindowProc.Call(hwnd, message, wParam, lParam)
		return r
	})
	if !classRegistered {
		wc := wndClassEx{wndProc: proc, instance: instance, className: className}
		wc.size = uint32(unsafe.Sizeof(wc))
		if r, _, callErr := procRegisterClassEx.Call(uintptr(unsafe.Pointer(&wc))); r == 0 {
			return 0, fmt.Errorf("RegisterClassEx: %w", callErr)
		}
		classRegistered = true
	}
	hwnd, _, callErr := procCreateWindowEx.Call(0, uintptr(unsafe.Pointer(className)), 0, 0,
		0, 0, 0, 0, hwndMessage, 0, uintptr(instance), 0)
	if hwnd == 0 {
		return 0, fmt.Errorf("CreateWindowEx: %w", callErr)
	}
	return hwnd, nil
}
