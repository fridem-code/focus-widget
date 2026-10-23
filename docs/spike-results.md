# Спайк T0 — результаты

Дата: 2026-10-23 · Wails **v3.0.0-beta.34** · WebView2 **155.0.4283.62** · Windows 11 Pro 25H2 · Go 1.27.1 · Node 24.19

Код спайка: [spike/t0-youtube/t0spike](../spike/t0-youtube/t0spike). Сырые отчёты: `spike/t0-youtube/report-*.json`.

Как устроен самотест: `t0spike.exe -selftest` открывает два окна с одной и той же страницей: с origin Wails (`http://wails.localhost`) и с запасного origin `http://127.0.0.1:<port>` (локальный Go-сервер). Страница сама управляет IFrame-плеером без участия пользователя и отправляет результаты на локальный сервер. Go параллельно проверяет API окна, трея, toast и single instance. Запускали трижды: с ID из ТЗ, с актуальным стримом и с флагом autoplay.

## Итог

**Идём по основному пути:** фронтенд отдаётся встроенным asset-сервером Wails с `http://wails.localhost`. Запасной путь через `127.0.0.1` не нужен, он даже хуже (см. п. 2).

Что из этого следует для ТЗ:
1. Обязателен флаг WebView2 `--autoplay-policy=no-user-gesture-required` (п. 3).
2. Любую ошибку `150` считать сигналом «источник недоступен» и пропускать его (п. 4).
3. ID live-стримов со временем перестают работать, каталог нужно проверять в самом плеере (п. 5).

## Ответы по пунктам T0

### 1. Воспроизведение в WebView2, ошибка 153 — ✅
- Live-стрим (`0sfDth4lMLc`, Lofi Girl) и список видео через `loadPlaylist([...])` играют. `nextVideo()`, `setVolume()`, `pauseVideo()`, `setShuffle()` работают.
- **Ошибка 153 не появилась ни разу**, ни на одном origin.
- `window.location.origin` = `http://wails.localhost`. При кросс-доменном запросе WebView2 шлёт `Referer: http://wails.localhost/` (политика strict-origin-when-cross-origin). Для YouTube этого достаточно.
- Плейлист с настоящим `list=PL…` не проверяли (под рукой не было надёжного ID). Механика `loadPlaylist`/`nextVideo` та же, проверка переносится в T6 вместе с подбором каталога.

### 2. Запасной origin `http://127.0.0.1` — ❌ не нужен
Во всех трёх прогонах `dQw4w9WgXcQ` с origin `127.0.0.1` отдавал ошибку `150`, а с `wails.localhost` играл. YouTube относится к origin `127.0.0.1` строже. Запасной путь из ТЗ (§7) снимаем.

### 3. Autoplay без жеста пользователя — ⚠️ нужен флаг браузера
| Условие | Результат |
|---|---|
| По умолчанию, `playVideo()` со звуком без клика | ❌ Сработало событие `onAutoplayBlocked`, плеер в состоянии `-1` |
| По умолчанию, `mute()` + `playVideo()` | ✅ играет; `unMute()` после этого тоже сработал, но полагаться на это нельзя |
| `AdditionalBrowserArgs: ["--autoplay-policy=no-user-gesture-required"]` | ✅ звук сразу, без клика, на обоих origin |

Без флага не будут работать сценарии без клика по странице: «Старт» из меню трея, автостарт фокуса после перерыва, продолжение после выхода из компактного режима. **Решение: в приложении всегда включаем этот флаг.** Синхронный `playVideo()` в обработчике клика оставляем как дополнительную страховку.

### 4. Коды ошибок плеера
- Несуществующий ID (`invalid0000`) отдаёт **`150`**, а не `2`/`100`, как обещает документация.
- Устаревший live из ТЗ (`jfKfPfyJRdk`) тоже отдаёт `150`, хотя oEmbed по нему отвечает 200 (с пустым названием).
- Вывод: `150` значит «по любой причине не играет здесь». Обработка из §3.2 подходит, просто основным будет код 150.

### 5. Каталог: ID стримов не вечные
- `jfKfPfyJRdk` из ТЗ больше не воспроизводится. Сейчас `https://www.youtube.com/@LofiGirl/live` ведёт на `0sfDth4lMLc` (сезонный хэллоуинский стрим).
- **oEmbed для проверки не годится:** по мёртвому `jfKfPfyJRdk` он отвечает 200. Проверять каталог нужно в самом IFrame-плеере, например режимом `-selftest` по всем источникам каталога (входит в T6).
- Для T6 стоит проверить источник вида `embed/live_stream?channel=<CHANNEL_ID>`: он всегда ведёт на текущий стрим канала. Через `loadVideoById` такой источник не загрузить, понадобится пересоздавать iframe.

### 6. API Wails v3 — ✅ всё есть
| Возможность | API (beta.34) | Проверка |
|---|---|---|
| Окно без рамки / поверх / без ресайза | `WebviewWindowOptions{Frameless, AlwaysOnTop, DisableResize}` | ✅ |
| Смена размера в рантайме | `win.SetSize(w, h)`, `win.Size()` | ✅ 380×540 → 300×96 → 380×540 |
| Always-on-top в рантайме | `win.SetAlwaysOnTop(bool)` | ✅ сверено с флагом Win32 `WS_EX_TOPMOST` |
| Позиция | `win.SetPosition(x, y)`, `win.Position()`, событие `events.Windows.WindowDidMove` (debounce 50 мс по умолчанию) | ✅ |
| Перетаскивание frameless-окна | CSS `--wails-draggable: drag` | нужна ручная проверка |
| Трей | `app.SystemTray.New()`, `SetMenu`, `SetTooltip`, `OnClick`, `AttachWindow`; меню `application.NewMenu()`, `Add`, `AddCheckbox`, `AddSeparator` | создаётся без ошибок; видимость и клик — ручная проверка |
| Single instance | `Options.SingleInstance{UniqueID, OnSecondInstanceLaunch}` | ✅ второй процесс завершается за 55 мс, в колбэк приходят его аргументы |
| Toast | `pkg/services/notifications`: `notifications.New()`, `SendNotification(NotificationOptions{ID, Title, Body})` | ✅ отправка без ошибок; видимость — ручная проверка |
| Аргументы WebView2 | `Options.Windows.AdditionalBrowserArgs` | ✅ |
| События Go → JS | `app.Event.Emit(name, data)`, типизация через `application.RegisterEvent[T](name)` | есть в шаблоне |

Побочный эффект toast на Windows: сервис регистрирует в **HKCU** ключи `Software\Classes\AppUserModelId\<AppName>` и `Software\Classes\CLSID\{GUID}\LocalServer32`, а иконку кладёт в `%TEMP%`. Деинсталлятор (T13) должен их удалять.

### 7. Прочее
- Размер production-exe спайка: **13.6 МБ** (требование ≤ 20 МБ).
- В шаблоне `wails3 init` по умолчанию лежат таски для Android/iOS/macOS/Linux. В T1 лишнее можно удалить.

## Что осталось проверить вручную (около 2 минут)
Запустить `spike/t0-youtube/t0spike/bin/t0spike.exe` без флагов и проверить:
- [ ] окно перетаскивается за верхнюю полосу
- [ ] иконка в трее видна, левый клик скрывает и показывает окно, пункты меню работают
- [ ] плеер играет по кнопкам Play / Live / Array playlist, громкость меняется слайдером

Toast проверяется только в режиме `-selftest` (появляется примерно через 3 с после запуска).
