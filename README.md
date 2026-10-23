# Focus Widget

Десктопный Pomodoro-виджет для Windows: при старте фокус-сессии включает музыку для концентрации с YouTube.
Go + [Wails v3](https://v3.wails.io) (WebView2), интерфейс на Vue 3 + TypeScript.

- ТЗ: [docs/SPEC.md](docs/SPEC.md)
- Задачи: [docs/TASKS.md](docs/TASKS.md)
- Результаты спайка (YouTube в WebView2): [docs/spike-results.md](docs/spike-results.md)

## Требования

| Инструмент | Версия | Установка |
|---|---|---|
| Go | 1.26+ | https://go.dev/dl/ |
| Node.js + npm | 22+ | https://nodejs.org/ |
| Wails CLI | v3.0.0-beta.34 | `go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.34` |
| golangci-lint | v2 | `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest` |
| WebView2 Runtime | — | есть в Windows 11; проверка: `wails3 doctor` |

Отдельный Task CLI не нужен: таски запускаются раннером, встроенным в Wails (`wails3 task <имя>`).

## Команды

| Команда | Что делает |
|---|---|
| `wails3 task dev` | dev-режим с hot reload фронтенда |
| `wails3 task build` | production-сборка в `bin/focuswidget.exe` |
| `wails3 task test` | `go test ./...` + vitest |
| `wails3 task lint` | golangci-lint + ESLint + Prettier check + vue-tsc |
| `wails3 task fmt` | форматирование Go и фронтенда |
| `wails3 task bindings` | перегенерировать TS-биндинги Go-сервисов в `frontend/bindings` |
| `wails3 task package` | установщик (нужен NSIS, задача T13) |

`test` и `lint` сначала собирают фронтенд: Go встраивает `frontend/dist` через `go:embed`, а vue-tsc нужны сгенерированные биндинги.

## Отладка

В dev-сборках (`wails3 task dev`, `wails3 build DEV=true`):
- в консоли DevTools (F12) доступен `window.__fw`: `await __fw.TimerService.Start()`, `await __fw.SettingsService.Get()`, а в `__fw.log` копятся полученные события;
- `FOCUSWIDGET_DEVTOOLS_PORT=9333` открывает порт протокола DevTools WebView2, чтобы скрипты могли управлять страницей (`http://127.0.0.1:9333/json`).

В production-сборку ни то ни другое не попадает.

Настройки лежат в `%APPDATA%\FocusWidget\config.json`.

## Структура

```
main.go            точка входа, go:embed фронтенда
app/               сборка Wails-приложения: опции, окно, трей
services/          Wails-сервисы (с T4)
internal/          логика без зависимости от Wails: timer, config, catalog (с T2/T3)
frontend/          Vue + TS (src/components, src/composables, src/lib)
build/             иконки, манифесты, NSIS, платформенные Taskfile
docs/              ТЗ, задачи, результаты спайка
spike/t0-youtube/  код спайка T0, в сборку не входит
```
