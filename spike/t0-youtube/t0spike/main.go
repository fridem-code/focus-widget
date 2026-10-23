// Spike T0: checks that the YouTube IFrame Player works inside Wails v3 / WebView2
// and that the window, tray, single-instance and toast APIs behave as SPEC.md expects.
//
// Manual mode:   t0spike.exe
// Self-test:     t0spike.exe -selftest -out report.json [-browserargs --autoplay-policy=no-user-gesture-required]
package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/services/notifications"
)

//go:embed all:frontend/dist
var assets embed.FS

var variants = []string{"wails", "localhttp"}

type report struct {
	mu          sync.Mutex
	Started     time.Time                   `json:"started"`
	BrowserArgs []string                    `json:"browserArgs"`
	Go          map[string]any              `json:"go"`
	Pages       map[string][]map[string]any `json:"pages"`
	Done        map[string]bool             `json:"done"`
	Referers    map[string]string           `json:"referersSeenByLocalServer"`
}

func (r *report) setGo(key string, value any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Go[key] = value
}

func (r *report) allDone() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, v := range variants {
		if !r.Done[v] {
			return false
		}
	}
	return true
}

func (r *report) write(path string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func main() {
	selftest := flag.Bool("selftest", false, "run automated checks and quit")
	out := flag.String("out", "spike-report.json", "self-test report path")
	browserArgs := flag.String("browserargs", "", "comma-separated extra WebView2 browser args")
	liveID := flag.String("live", "0sfDth4lMLc", "live stream video id to test")
	// Set only on the copy runSecondInstanceProbe launches; it must exit inside application.New.
	flag.Bool("second-instance-probe", false, "internal")
	flag.Parse()

	rep := &report{
		Started:  time.Now(),
		Go:       map[string]any{},
		Pages:    map[string][]map[string]any{},
		Done:     map[string]bool{},
		Referers: map[string]string{},
	}
	var extraArgs []string
	if *browserArgs != "" {
		extraArgs = strings.Split(*browserArgs, ",")
	}
	rep.BrowserArgs = extraArgs

	secondInstance := make(chan application.SecondInstanceData, 1)
	notifier := notifications.New()

	app := application.New(application.Options{
		Name:        "t0spike",
		Description: "Spike T0: YouTube IFrame in Wails v3",
		Services: []application.Service{
			application.NewService(notifier),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
		SingleInstance: &application.SingleInstanceOptions{
			UniqueID: "com.focuswidget.t0spike",
			OnSecondInstanceLaunch: func(data application.SecondInstanceData) {
				select {
				case secondInstance <- data:
				default:
				}
			},
		},
		Windows: application.WindowsOptions{
			AdditionalBrowserArgs: extraArgs,
		},
	})

	// The local server serves the same frontend from a plain http://127.0.0.1 origin
	// (the fallback variant) and collects reports from both pages.
	port, err := startLocalServer(rep)
	if err != nil {
		log.Fatal(err)
	}
	query := fmt.Sprintf("port=%d&selftest=%t&live=%s", port, *selftest, *liveID)

	mainWin := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "t0spike (wails origin)",
		Width:            380,
		Height:           540,
		Frameless:        true,
		AlwaysOnTop:      true,
		DisableResize:    true,
		BackgroundColour: application.NewRGB(18, 18, 24),
		URL:              "/?variant=wails&" + query,
	})
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "t0spike (127.0.0.1 origin)",
		Width:            380,
		Height:           540,
		X:                420,
		BackgroundColour: application.NewRGB(18, 18, 24),
		URL:              fmt.Sprintf("http://127.0.0.1:%d/?variant=localhttp&%s", port, query),
	})

	trayMenu := application.NewMenu()
	trayMenu.Add("Показать/Скрыть").OnClick(func(*application.Context) {
		if mainWin.IsVisible() {
			mainWin.Hide()
		} else {
			mainWin.Show()
		}
	})
	trayMenu.AddCheckbox("Всегда поверх", true).OnClick(func(ctx *application.Context) {
		mainWin.SetAlwaysOnTop(ctx.ClickedMenuItem().Checked())
	})
	trayMenu.AddSeparator()
	trayMenu.Add("Выход").OnClick(func(*application.Context) { app.Quit() })
	tray := app.SystemTray.New()
	tray.SetTooltip("t0spike · 25:00 фокус")
	tray.SetMenu(trayMenu)
	tray.OnClick(func() {
		if mainWin.IsVisible() {
			mainWin.Hide()
		} else {
			mainWin.Show()
		}
	})

	if *selftest {
		go runGoChecks(app, mainWin, notifier, secondInstance, rep)
		go func() {
			deadline := time.Now().Add(150 * time.Second)
			for !rep.allDone() && time.Now().Before(deadline) {
				time.Sleep(time.Second)
			}
			rep.setGo("finishedAllPages", rep.allDone())
			if err := rep.write(*out); err != nil {
				log.Println("write report:", err)
			}
			app.Quit()
		}()
	}

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

func runGoChecks(app *application.App, win *application.WebviewWindow, notifier *notifications.NotificationService,
	secondInstance chan application.SecondInstanceData, rep *report) {
	time.Sleep(3 * time.Second)

	w, h := win.Size()
	rep.setGo("window.initialSize", []int{w, h})
	rep.setGo("window.topmostAtStart", isTopmost(win))

	win.SetSize(300, 96)
	time.Sleep(700 * time.Millisecond)
	w, h = win.Size()
	rep.setGo("window.sizeAfterCompact", []int{w, h})
	win.SetSize(380, 540)
	time.Sleep(700 * time.Millisecond)
	w, h = win.Size()
	rep.setGo("window.sizeAfterExpand", []int{w, h})

	win.SetAlwaysOnTop(false)
	time.Sleep(300 * time.Millisecond)
	rep.setGo("window.topmostAfterOff", isTopmost(win))
	win.SetAlwaysOnTop(true)
	time.Sleep(300 * time.Millisecond)
	rep.setGo("window.topmostAfterOn", isTopmost(win))

	win.SetPosition(200, 150)
	time.Sleep(300 * time.Millisecond)
	x, y := win.Position()
	rep.setGo("window.positionAfterSet", []int{x, y})

	err := notifier.SendNotification(notifications.NotificationOptions{
		ID:    "t0-phase-end",
		Title: "Фокус завершён",
		Body:  "Спайк T0: проверка toast-уведомления",
	})
	rep.setGo("toast.sendError", errString(err))

	exe, err := os.Executable()
	if err != nil {
		rep.setGo("singleInstance.error", err.Error())
		return
	}
	cmd := exec.Command(exe, "-second-instance-probe")
	start := time.Now()
	if err := cmd.Start(); err != nil {
		rep.setGo("singleInstance.error", err.Error())
		return
	}
	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()
	select {
	case err := <-exited:
		rep.setGo("singleInstance.secondExitedAfterMs", time.Since(start).Milliseconds())
		rep.setGo("singleInstance.secondExitError", errString(err))
	case <-time.After(15 * time.Second):
		_ = cmd.Process.Kill()
		rep.setGo("singleInstance.secondExited", false)
	}
	select {
	case data := <-secondInstance:
		rep.setGo("singleInstance.callbackArgs", data.Args)
	case <-time.After(3 * time.Second):
		rep.setGo("singleInstance.callbackFired", false)
	}
}

func startLocalServer(rep *report) (int, error) {
	dist, err := fs.Sub(assets, "frontend/dist")
	if err != nil {
		return 0, err
	}
	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.FS(dist)))
	mux.HandleFunc("/report", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		var entry map[string]any
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &entry); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		variant, _ := entry["variant"].(string)
		rep.mu.Lock()
		rep.Pages[variant] = append(rep.Pages[variant], entry)
		if entry["key"] == "done" {
			rep.Done[variant] = true
		}
		rep.mu.Unlock()
	})
	// A cross-origin request from each page lands here, so the report shows which
	// Referer WebView2 actually sends from that origin.
	mux.HandleFunc("/echo", func(w http.ResponseWriter, r *http.Request) {
		rep.mu.Lock()
		rep.Referers[r.URL.Query().Get("variant")] = r.Header.Get("Referer")
		rep.mu.Unlock()
		_, _ = w.Write([]byte("ok"))
	})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	go func() { _ = http.Serve(ln, mux) }()
	return ln.Addr().(*net.TCPAddr).Port, nil
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
