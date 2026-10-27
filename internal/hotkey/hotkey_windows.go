package hotkey

import (
	"errors"
	"fmt"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	wmQuit      = 0x0012
	wmHotkey    = 0x0312
	modNoRepeat = 0x4000 // holding the keys down fires once
	pmNoRemove  = 0x0000

	errorHotkeyAlreadyRegistered = windows.Errno(1409)
)

var (
	user32                = windows.NewLazySystemDLL("user32.dll")
	procRegisterHotKey    = user32.NewProc("RegisterHotKey")
	procUnregisterHotKey  = user32.NewProc("UnregisterHotKey")
	procGetMessage        = user32.NewProc("GetMessageW")
	procPeekMessage       = user32.NewProc("PeekMessageW")
	procPostThreadMessage = user32.NewProc("PostThreadMessageW")
)

// ErrTaken: another application (or Windows) already has the shortcut.
var ErrTaken = errors.New("taken by another application")

type msg struct {
	hwnd    uintptr
	message uint32
	wParam  uintptr
	lParam  uintptr
	time    uint32
	pt      struct{ x, y int32 }
	private uint32
}

// Watch registers the shortcuts under their ids and calls fired(id), on a
// goroutine of its own, whenever one is pressed, until stop is called. A
// shortcut that can't be registered is reported in failed and the others
// still work. stop returns once the shortcuts are free again, so the next
// Watch may take the same ones.
//
// The shortcuts belong to a locked OS thread without a window: Windows posts
// WM_HOTKEY to the thread's queue.
func Watch(combos map[int]Combo, fired func(id int)) (stop func(), failed map[int]error) {
	type result struct {
		threadID uint32
		failed   map[int]error
	}
	started := make(chan result, 1)
	done := make(chan struct{})
	go func() {
		defer close(done)
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		// The queue must exist before stop can post to it.
		var m msg
		_, _, _ = procPeekMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0, pmNoRemove)

		res := result{threadID: windows.GetCurrentThreadId(), failed: map[int]error{}}
		var registered []int
		for id, c := range combos {
			r, _, callErr := procRegisterHotKey.Call(0, uintptr(id), uintptr(c.Mods|modNoRepeat), uintptr(c.Key))
			switch {
			case r != 0:
				registered = append(registered, id)
			case errors.Is(callErr, errorHotkeyAlreadyRegistered):
				res.failed[id] = ErrTaken
			default:
				res.failed[id] = fmt.Errorf("RegisterHotKey %s: %w", c, callErr)
			}
		}
		started <- res
		for {
			if r, _, _ := procGetMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0); int32(r) <= 0 {
				break
			}
			if m.message == wmHotkey {
				go fired(int(m.wParam))
			}
		}
		for _, id := range registered {
			_, _, _ = procUnregisterHotKey.Call(0, uintptr(id))
		}
	}()
	res := <-started
	return func() {
		_, _, _ = procPostThreadMessage.Call(uintptr(res.threadID), wmQuit, 0, 0)
		<-done
	}, res.failed
}
