package stats

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"
)

var quiet = slog.New(slog.NewTextHandler(io.Discard, nil))

// Moscow keeps no DST; the DST test below uses Berlin.
var msk = time.FixedZone("MSK", 3*60*60)

func at(day, hour, minute int) time.Time { return time.Date(2026, 10, day, hour, minute, 0, 0, msk) }

func TestLogKeepsSessionsAcrossOpens(t *testing.T) {
	dir := t.TempDir()
	l := Open(dir, quiet)
	if got := l.Sessions(); len(got) != 0 {
		t.Fatalf("new log has %v", got)
	}
	a := Session{End: at(27, 10, 25), Minutes: 25, Label: "Отчёт"}
	b := Session{End: at(28, 9, 50), Minutes: 50}
	for _, s := range []Session{a, b} {
		if err := l.Add(s); err != nil {
			t.Fatal(err)
		}
	}
	got := Open(dir, quiet).Sessions()
	if len(got) != 2 || !got[0].End.Equal(a.End) || got[0].Label != "Отчёт" || got[1].Minutes != 50 {
		t.Fatalf("reopened log = %+v", got)
	}
}

func TestLogSkipsUnreadableLines(t *testing.T) {
	dir := t.TempDir()
	content := `{"end":"2026-10-27T10:25:00+03:00","minutes":25}
not json
{"end":"2026-10-27T11:00:00+03:00","minutes":0}

{"minutes":25}
{"end":"2026-10-27T12:00:00+03:00","minutes":25,"label":"ок"}
`
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	l := Open(dir, quiet)
	if got := l.Sessions(); len(got) != 2 || got[1].Label != "ок" {
		t.Fatalf("sessions = %+v, want the two good lines", got)
	}
	// Adding keeps the bad lines in the file: nothing is rewritten.
	if err := l.Add(Session{End: at(28, 9, 0), Minutes: 25}); err != nil {
		t.Fatal(err)
	}
	if got := Open(dir, quiet).Sessions(); len(got) != 3 {
		t.Fatalf("after add: %d sessions, want 3", len(got))
	}
}

func TestSummarize(t *testing.T) {
	sessions := []Session{
		{End: at(20, 10, 0), Minutes: 25}, // eight days ago: outside the week
		{End: at(25, 10, 0), Minutes: 25},
		{End: at(26, 23, 59), Minutes: 25},
		{End: at(27, 9, 0), Minutes: 50},
		{End: at(28, 15, 0), Minutes: 25, Label: "второй"},
		{End: at(28, 0, 30), Minutes: 25, Label: "первый"},
		// 23:30 UTC on the 27th is 02:30 on the 28th in Moscow.
		{End: time.Date(2026, 10, 27, 23, 30, 0, 0, time.UTC), Minutes: 25},
	}
	s := Summarize(sessions, at(28, 18, 0))

	if s.Today != (Day{Date: "2026-10-28", Focus: 3, Minutes: 75}) {
		t.Errorf("Today = %+v", s.Today)
	}
	wantWeek := []Day{
		{Date: "2026-10-22"}, {Date: "2026-10-23"}, {Date: "2026-10-24"},
		{Date: "2026-10-25", Focus: 1, Minutes: 25},
		{Date: "2026-10-26", Focus: 1, Minutes: 25},
		{Date: "2026-10-27", Focus: 1, Minutes: 50},
		{Date: "2026-10-28", Focus: 3, Minutes: 75},
	}
	for i, d := range wantWeek {
		if s.Week[i] != d {
			t.Errorf("Week[%d] = %+v, want %+v", i, s.Week[i], d)
		}
	}
	if s.WeekMinutes != 175 {
		t.Errorf("WeekMinutes = %d, want 175", s.WeekMinutes)
	}
	if s.Streak != 4 {
		t.Errorf("Streak = %d, want 4 (25th to 28th)", s.Streak)
	}
	if len(s.Sessions) != 3 || s.Sessions[0].Label != "первый" || s.Sessions[2].Label != "второй" {
		t.Errorf("Sessions = %+v, want today's, oldest first", s.Sessions)
	}
}

func TestStreakWaitsForToday(t *testing.T) {
	sessions := []Session{{End: at(26, 10, 0), Minutes: 25}, {End: at(27, 10, 0), Minutes: 25}}
	if got := Summarize(sessions, at(28, 8, 0)).Streak; got != 2 {
		t.Errorf("morning without a session yet: Streak = %d, want 2", got)
	}
	if got := Summarize(sessions, at(29, 8, 0)).Streak; got != 0 {
		t.Errorf("after a day without sessions: Streak = %d, want 0", got)
	}
}

func TestSummarizeEmpty(t *testing.T) {
	s := Summarize(nil, at(28, 8, 0))
	if s.Today.Focus != 0 || s.Streak != 0 || len(s.Week) != 7 || s.Sessions == nil {
		t.Errorf("empty summary = %+v", s)
	}
}

func TestWeekAcrossDST(t *testing.T) {
	berlin, err := time.LoadLocation("Europe/Berlin")
	if err != nil {
		t.Skip("no time zone data:", err)
	}
	// Clocks went back on 2026-10-25 in Berlin; at midnight a 24-hour step
	// back from the 26th lands on 23:00 of the 24th.
	now := time.Date(2026, 10, 26, 0, 30, 0, 0, berlin)
	s := Summarize(nil, now)
	want := []string{"2026-10-20", "2026-10-21", "2026-10-22", "2026-10-23", "2026-10-24", "2026-10-25", "2026-10-26"}
	for i, d := range s.Week {
		if d.Date != want[i] {
			t.Errorf("Week[%d].Date = %s, want %s", i, d.Date, want[i])
		}
	}
}
