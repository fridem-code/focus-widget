// Spike T0.5: does a Windows 11 Acrylic backdrop give the "glass" look from the
// design, together with a frameless, always-on-top, resizable widget window and
// a playing YouTube iframe?
//
//	t05acrylic.exe -variant acrylic|transparent|solid [-compact]
//
// A second, normal window ("wallpaper") with coloured blobs sits under the
// widget so screenshots show what the backdrop does with content behind it.
// WebView2 DevTools protocol: port 9334 (widget page reports window.__state).
package main

import (
	"embed"
	"flag"
	"io/fs"
	"log"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed www
var www embed.FS

func main() {
	variant := flag.String("variant", "acrylic", "acrylic | transparent | solid")
	compact := flag.Bool("compact", false, "start the widget at the compact size")
	blurAfter := flag.Int("blurafter", 0, "seconds after start to focus the wallpaper window (0: never)")
	resizeAfter := flag.Int("resizeafter", 0, "seconds after start to shrink the widget to 300x96 at runtime (0: never)")
	wallOnly := flag.Bool("wallonly", false, "open only the wallpaper window (backdrop for checking the real app)")
	flag.Parse()

	assets, err := fs.Sub(www, "www")
	if err != nil {
		log.Fatal(err)
	}
	app := application.New(application.Options{
		Name:   "t05acrylic",
		Assets: application.AssetOptions{Handler: application.AssetFileServerFS(assets)},
		Windows: application.WindowsOptions{
			AdditionalBrowserArgs: []string{
				"--autoplay-policy=no-user-gesture-required",
				"--remote-debugging-port=9334",
			},
		},
	})

	wallpaper := app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "t05 wallpaper",
		Width:            760,
		Height:           720,
		X:                60,
		Y:                60,
		BackgroundColour: application.NewRGB(12, 17, 40),
		URL:              "/wall.html",
	})

	if *wallOnly {
		if err := app.Run(); err != nil {
			log.Fatal(err)
		}
		return
	}

	opts := application.WebviewWindowOptions{
		Title:         "t05 widget",
		Width:         380,
		Height:        540,
		X:             250,
		Y:             150,
		Frameless:     true,
		AlwaysOnTop:   true,
		DisableResize: true,
		URL:           "/widget.html?variant=" + *variant,
	}
	switch *variant {
	case "acrylic":
		opts.BackgroundType = application.BackgroundTypeTranslucent
		opts.Windows.BackdropType = application.Acrylic
	case "transparent":
		// Per-pixel transparency: CSS draws the rounded card, no system blur and no
		// DWM corners/shadow.
		opts.BackgroundType = application.BackgroundTypeTransparent
		opts.Windows.DisableFramelessWindowDecorations = true
	default:
		opts.BackgroundColour = application.NewRGB(31, 33, 41)
	}
	if *compact {
		opts.Width, opts.Height = 300, 96
		opts.URL += "&compact=1"
	}
	widget := app.Window.NewWithOptions(opts)

	if *resizeAfter > 0 {
		go func() {
			time.Sleep(time.Duration(*resizeAfter) * time.Second)
			widget.SetSize(300, 96)
		}()
	}
	if *blurAfter > 0 {
		go func() {
			time.Sleep(time.Duration(*blurAfter) * time.Second)
			wallpaper.Focus()
			time.Sleep(4 * time.Second)
			widget.Focus()
		}()
	}

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
