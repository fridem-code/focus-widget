package main

import (
	"syscall"

	"github.com/wailsapp/wails/v3/pkg/application"
)

var procGetWindowLongW = syscall.NewLazyDLL("user32.dll").NewProc("GetWindowLongW")

// isTopmost reads WS_EX_TOPMOST straight from Win32 to verify SetAlwaysOnTop.
func isTopmost(win *application.WebviewWindow) bool {
	gwlExStyle := int32(-20)
	const wsExTopmost = 0x8
	exStyle, _, _ := procGetWindowLongW.Call(uintptr(win.NativeWindow()), uintptr(gwlExStyle))
	return exStyle&wsExTopmost != 0
}
