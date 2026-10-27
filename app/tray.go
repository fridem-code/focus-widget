package app

import (
	"github.com/wailsapp/wails/v3/pkg/application"

	"focuswidget/internal/trayicon"
	"focuswidget/services"
)

// buildTray creates the tray icon and menu (docs/SPEC.md §3.5) and hands them
// to the tray controller, which keeps labels and tooltip up to date.
// Idle, the icon is the exe's own (build/windows/icon.ico), which Wails picks
// up when none is set; while a phase runs it shows the time left (T21).
func buildTray(app *application.App, tray *services.Tray) {
	menu := application.NewMenu()
	visibility := menu.Add("Скрыть").OnClick(func(*application.Context) { tray.ToggleVisible() })
	menu.AddSeparator()
	timerItem := menu.Add("Старт").OnClick(func(*application.Context) { tray.ToggleTimer() })
	menu.Add("Пропустить фазу").OnClick(func(*application.Context) { tray.Skip() })
	menu.Add("Следующий жанр").OnClick(func(*application.Context) { tray.NextGenre() })
	menu.Add("Сбросить цикл").OnClick(func(*application.Context) { tray.ResetCycle() })
	menu.AddSeparator()
	mode := menu.Add("Компактный режим").OnClick(func(*application.Context) { tray.ToggleMode() })
	onTop := menu.AddCheckbox("Всегда поверх", true).OnClick(func(*application.Context) { tray.ToggleAlwaysOnTop() })
	menu.AddSeparator()
	menu.Add("Выход").OnClick(func(*application.Context) { tray.Quit() })

	icon := app.SystemTray.New()
	icon.SetMenu(menu)
	// Left click shows/hides the window; the menu opens on right click.
	icon.OnClick(tray.ToggleVisible)

	tray.Attach(trayUI{icon: icon, menu: menu}, services.TrayItems{
		Visibility:  visibility,
		Timer:       timerItem,
		Mode:        mode,
		AlwaysOnTop: onTop,
	})
}

type trayUI struct {
	icon *application.SystemTray
	menu *application.Menu
}

func (u trayUI) SetTooltip(s string) { u.icon.SetTooltip(s) }
func (u trayUI) UpdateMenu()         { u.menu.Update() }

func (u trayUI) SetRing(r trayicon.Ring) { u.icon.SetIcon(trayicon.PNG(r, trayIconSize())) }
