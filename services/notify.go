package services

import (
	"fmt"

	"github.com/wailsapp/wails/v3/pkg/services/notifications"

	"focuswidget/internal/timer"
)

// ToastSender is the part of notifications.NotificationService used here.
type ToastSender interface {
	SendNotification(notifications.NotificationOptions) error
}

// PhaseEndToastID identifies our toasts. It does not replace earlier ones:
// Windows keeps each phase-end toast in the Action Center (checked in T11).
const PhaseEndToastID = "phase-end"

// PhaseNotifier shows a Windows toast when a phase runs out (docs/SPEC.md
// §3.1). The timer produces exactly one expire transition per phase, even
// after the PC slept, so there is exactly one toast per phase end. The chime
// is played by the frontend.
type PhaseNotifier struct {
	sender   ToastSender
	settings *SettingsService
	onError  func(error)
}

// NewPhaseNotifier subscribes to the timer's transitions.
func NewPhaseNotifier(t *timer.Timer, sender ToastSender, settings *SettingsService, onError func(error)) *PhaseNotifier {
	n := &PhaseNotifier{sender: sender, settings: settings, onError: onError}
	t.OnTransition(func(tr timer.Transition) {
		// Off the timer's goroutine: showing a toast must not delay timer events.
		go n.notify(tr)
	})
	return n
}

func (n *PhaseNotifier) notify(tr timer.Transition) {
	title, body, ok := PhaseEndMessage(tr)
	if !ok || !n.settings.Get().Notify.Toast {
		return
	}
	err := n.sender.SendNotification(notifications.NotificationOptions{
		ID:    PhaseEndToastID,
		Title: title,
		Body:  body,
	})
	if err != nil && n.onError != nil {
		n.onError(err)
	}
}

// PhaseEndMessage returns the toast for a transition; ok is false for
// anything but a phase running out (Start, Skip, Reset, … need no toast).
func PhaseEndMessage(tr timer.Transition) (title, body string, ok bool) {
	if tr.Cause != timer.CauseExpire {
		return "", "", false
	}
	if tr.From.Phase == timer.Focus {
		title = "Фокус завершён"
	} else {
		title = "Перерыв окончен"
	}

	minutes := (tr.To.DurationMs + 59_999) / 60_000
	var next string
	switch tr.To.Phase {
	case timer.ShortBreak:
		next = fmt.Sprintf("короткий перерыв, %d мин", minutes)
	case timer.LongBreak:
		next = fmt.Sprintf("длинный перерыв, %d мин", minutes)
	default:
		next = fmt.Sprintf("фокус, сессия %d из %d", min(tr.To.CompletedFocus+1, tr.To.LongBreakEvery), tr.To.LongBreakEvery)
	}
	if tr.To.Status == timer.Running {
		body = "Начался " + next
	} else {
		body = "Дальше " + next + ". Нажмите «Старт», когда будете готовы"
	}
	return title, body, true
}
