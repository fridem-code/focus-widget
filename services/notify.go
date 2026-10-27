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
	// focusToday counts today's finished focus sessions (T25).
	focusToday func() int
	onError    func(error)
}

// NewPhaseNotifier subscribes to the timer's transitions. Create it after the
// StatsService: listeners run in the order they subscribed, so the session
// that just ended is already counted by focusToday.
func NewPhaseNotifier(t *timer.Timer, sender ToastSender, settings *SettingsService, focusToday func() int, onError func(error)) *PhaseNotifier {
	n := &PhaseNotifier{sender: sender, settings: settings, focusToday: focusToday, onError: onError}
	t.OnTransition(func(tr timer.Transition) {
		if tr.Cause != timer.CauseExpire {
			return
		}
		// Read now, in the listener: by the time a goroutine runs, the user
		// may have changed the label for the next session.
		s := settings.Get()
		day := PhaseDay{Label: s.Timer.Label, Goal: s.Timer.DailyGoal}
		if tr.From.Phase == timer.Focus && focusToday != nil {
			day.Done = focusToday()
		}
		// Off the timer's goroutine: showing a toast must not delay timer events.
		go n.notify(tr, day)
	})
	return n
}

func (n *PhaseNotifier) notify(tr timer.Transition, day PhaseDay) {
	title, body, ok := PhaseEndMessage(tr, day)
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

// PhaseDay is what the toast says beyond the timer: the label of the focus
// (T24) and the day's progress towards the goal (T25, Goal 0 is no goal).
type PhaseDay struct {
	Label string
	Done  int
	Goal  int
}

// PhaseEndMessage returns the toast for a transition; ok is false for
// anything but a phase running out (Start, Skip, Reset, … need no toast).
// The focus that reaches the daily goal announces that instead of itself.
func PhaseEndMessage(tr timer.Transition, day PhaseDay) (title, body string, ok bool) {
	if tr.Cause != timer.CauseExpire {
		return "", "", false
	}
	if tr.From.Phase == timer.Focus {
		title = "Фокус завершён"
		if day.Label != "" {
			title += " · " + day.Label
		}
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

	if tr.From.Phase == timer.Focus && day.Goal > 0 && day.Done == day.Goal {
		title = "Цель на день выполнена"
		body = fmt.Sprintf("%d из %d %s за сегодня. %s", day.Done, day.Goal, pomodorosOf(day.Goal), body)
	}
	return title, body, true
}

// pomodorosOf is "помидор" after "из N": "из 1 помидора", "из 8 помидоров".
func pomodorosOf(n int) string {
	if n%10 == 1 && n%100 != 11 {
		return "помидора"
	}
	return "помидоров"
}
