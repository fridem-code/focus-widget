//go:build !production

package app

import (
	"os"
	"strconv"
)

// devBrowserArgs opens the WebView2 DevTools protocol on FOCUSWIDGET_DEVTOOLS_PORT
// so tests and scripts can drive the page. Dev builds only: WebView2's own
// WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS is ignored once Wails passes arguments.
func devBrowserArgs() []string {
	port, err := strconv.Atoi(os.Getenv("FOCUSWIDGET_DEVTOOLS_PORT"))
	if err != nil || port <= 0 {
		return nil
	}
	return []string{"--remote-debugging-port=" + strconv.Itoa(port)}
}

// logToConsole mirrors the log to stderr in dev builds.
const logToConsole = true
