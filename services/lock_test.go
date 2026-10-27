package services

import (
	"testing"

	"github.com/wailsapp/wails/v3/pkg/services/notifications"

	"focuswidget/internal/timer"
)

type fakeActionToasts struct {
	sent []notifications.NotificationOptions
}

func (f *fakeActionToasts) SendNotificationWithActions(o notifications.NotificationOptions) error {
	f.sent = append(f.sent, o)
	return nil
}

func newLockPauser(t *testing.T) (*LockPauser, *TimerService, *SettingsService, *fakeActionToasts) {
	t.Helper()
	settings, _, tm, emit := newSettingsService(t)
	timerSvc := NewTimerService(tm, emit)
	toasts := &fakeActionToasts{}
	return NewLockPauser(timerSvc, settings, toasts, func(err error) { t.Error(err) }), timerSvc, settings, toasts
}

func TestLockPausesARunningFocusAndOffersToResume(t *testing.T) {
	p, timerSvc, _, toasts := newLockPauser(t)
	if err := timerSvc.Start(); err != nil {
		t.Fatal(err)
	}
	p.SessionChanged(true)
	if st := timerSvc.State(); st.Status != timer.Paused {
		t.Fatalf("after lock: %+v, want paused", st)
	}
	p.SessionChanged(false)
	if len(toasts.sent) != 1 {
		t.Fatalf("toasts = %+v, want one", toasts.sent)
	}
	got := toasts.sent[0]
	if got.CategoryID != FocusPausedCategory || got.Title != "Фокус на паузе" || got.Body != "Пока компьютер был заблокирован. Осталось 25 мин" {
		t.Fatalf("toast = %+v", got)
	}
	// The pause stays: the user resumes from the toast or the window.
	if st := timerSvc.State(); st.Status != timer.Paused {
		t.Fatalf("after unlock: %+v, want still paused", st)
	}
	p.SessionChanged(false) // a second unlock event offers nothing more
	if len(toasts.sent) != 1 {
		t.Fatal("offered twice")
	}
}

func TestLockLeavesEverythingButARunningFocusAlone(t *testing.T) {
	t.Run("break goes on", func(t *testing.T) {
		p, timerSvc, _, toasts := newLockPauser(t)
		timerSvc.Skip() // to the short break
		if err := timerSvc.Start(); err != nil {
			t.Fatal(err)
		}
		p.SessionChanged(true)
		p.SessionChanged(false)
		if st := timerSvc.State(); st.Status != timer.Running || len(toasts.sent) != 0 {
			t.Fatalf("break: %+v, toasts %d", st, len(toasts.sent))
		}
	})
	t.Run("focus paused by the user", func(t *testing.T) {
		p, timerSvc, _, toasts := newLockPauser(t)
		_ = timerSvc.Start()
		_ = timerSvc.Pause()
		p.SessionChanged(true)
		p.SessionChanged(false)
		if len(toasts.sent) != 0 {
			t.Fatal("offered to resume a pause the user made")
		}
	})
	t.Run("setting off", func(t *testing.T) {
		p, timerSvc, settings, _ := newLockPauser(t)
		if _, err := settings.Patch(SettingsPatch{PauseOnLock: new(false)}); err != nil {
			t.Fatal(err)
		}
		_ = timerSvc.Start()
		p.SessionChanged(true)
		if st := timerSvc.State(); st.Status != timer.Running {
			t.Fatalf("paused with the setting off: %+v", st)
		}
	})
}

func TestUnlockAfterAResetOrWithToastsOffOffersNothing(t *testing.T) {
	p, timerSvc, settings, toasts := newLockPauser(t)
	_ = timerSvc.Start()
	p.SessionChanged(true)
	timerSvc.Reset() // e.g. from the tray, while locked
	p.SessionChanged(false)
	if len(toasts.sent) != 0 {
		t.Fatal("offered to resume after a reset")
	}

	if _, err := settings.Patch(SettingsPatch{Toast: new(false)}); err != nil {
		t.Fatal(err)
	}
	_ = timerSvc.Start()
	p.SessionChanged(true)
	p.SessionChanged(false)
	if len(toasts.sent) != 0 {
		t.Fatal("toast sent with toasts off")
	}
}
