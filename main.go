package main

import (
	"embed"
	"log"

	"focuswidget/app"
)

// frontend/dist is produced by the frontend build (`wails3 task build`).
//
//go:embed all:frontend/dist
var assets embed.FS

func main() {
	if err := app.New(assets).Run(); err != nil {
		log.Fatal(err)
	}
}
