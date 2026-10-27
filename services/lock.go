package services

import (
	"fmt"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/services/notifications"

	"focuswidget/internal/timer"
)

// ActionToastSender sends a toast with buttons.
type ActionToastSender interface {
	SendNotificationWithActions(notifications.NotificationOptions) error
}

const (
	// FocusPausedToastID and FocusPausedCategory: the toast offered after an
	// unlock, with its "Продолжить" button (ResumeAction).
	FocusPausedToastID  = "focus-paused"
	FocusPausedCategory = "focus-paused"
	ResumeAction        = "resume"
)

// FocusPausedToastCategory is registered with the notification service at
// start, so the toast can carry its button.
func FocusPausedToastCategory() notifications.NotificationCategory {
	return notifications.NotificationCategory{
		ID:      FocusPausedCategory,
		Actions: []notifications.NotificationAction{{ID: ResumeAction, Title: "Продолжить"}},
	}
}

// LockPauser pauses a running focus phase while the PC is locked (T20): time
// away from the desk is not focus. A break goes on, since resting away from
// the PC is the point of it. After the unlock a toast offers to resume. Not a
// Wails service: app wires it to internal/session.
type LockPauser struct {
	timer    *TimerService
	settings *SettingsService
	toasts   ActionToastSender
	onError  func(error)

	mu           sync.Mutex
	pausedByLock bool
}

func NewLockPauser(t *TimerService, settings *SettingsService, toasts ActionToastSender, onError func(error)) *LockPauser {
	return &LockPauser{timer: t, settings: settings, toasts: toasts, onError: onError}
}

// SessionChanged handles a lock (true) or unlock (false); it runs on the
// session watcher's thread.
func (p *LockPauser) SessionChanged(locked bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	st := p.timer.State()
	if locked {
		if p.settings.Get().Timer.PauseOnLock && st.Phase == timer.Focus && st.Status == timer.Running {
			p.pausedByLock = p.timer.Pause() == nil
		}
		return
	}
	if !p.pausedByLock {
		return
	}
	p.pausedByLock = false
	// Resumed or reset meanwhile (from the tray, say): nothing to offer.
	if st.Phase != timer.Focus || st.Status != timer.Paused || !p.settings.Get().Notify.Toast {
		return
	}
	err := p.toasts.SendNotificationWithActions(notifications.NotificationOptions{
		ID:         FocusPausedToastID,
		Title:      "Фокус на паузе",
		Body:       fmt.Sprintf("Пока компьютер был заблокирован. Осталось %d мин", (st.RemainingMs+59_999)/60_000),
		CategoryID: FocusPausedCategory,
	})
	if err != nil && p.onError != nil {
		p.onError(err)
	}
}
