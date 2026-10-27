// Package stats keeps the log of finished focus sessions in sessions.jsonl and
// sums it up for the statistics screen (T23).
package stats

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"
)

// FileName is the log in the app's data folder, one JSON session per line.
const FileName = "sessions.jsonl"

// Session is a focus phase that ran to its end.
type Session struct {
	End     time.Time `json:"end"`
	Minutes int       `json:"minutes"`
	Label   string    `json:"label,omitempty"`
}

// Log is the session log: read once at start, then appended to. Lines are
// only ever added, so a crash can cost at most the line being written.
type Log struct {
	path string

	mu       sync.Mutex
	sessions []Session
}

// Open reads the log in dir. A missing file is an empty log; lines that can't
// be read are skipped (and counted in the app log), never rewritten.
func Open(dir string, logger *slog.Logger) *Log {
	l := &Log{path: filepath.Join(dir, FileName)}
	data, err := os.ReadFile(l.path)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			logger.Warn("read the session log", "err", err)
		}
		return l
	}
	bad := 0
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		var s Session
		if json.Unmarshal(line, &s) != nil || s.End.IsZero() || s.Minutes <= 0 {
			bad++
			continue
		}
		l.sessions = append(l.sessions, s)
	}
	if bad > 0 {
		logger.Warn("skipped unreadable lines in the session log", "lines", bad)
	}
	return l
}

// Add records a session. It counts from now on even if the file can't be
// written; the error says so.
func (l *Log) Add(s Session) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	// In memory once the file has it (or failed to), so whoever sees the
	// session can also read it back from the file.
	defer func() { l.sessions = append(l.sessions, s) }()

	line, err := json.Marshal(s)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return fmt.Errorf("write the session log: %w", err)
	}
	f, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("write the session log: %w", err)
	}
	_, err = f.Write(append(line, '\n'))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return fmt.Errorf("write the session log: %w", err)
	}
	return nil
}

// Sessions returns every session, in the order they were added.
func (l *Log) Sessions() []Session {
	l.mu.Lock()
	defer l.mu.Unlock()
	return slices.Clone(l.sessions)
}

// Day sums one calendar day.
type Day struct {
	Date    string `json:"date"`  // YYYY-MM-DD, local time
	Focus   int    `json:"focus"` // sessions
	Minutes int    `json:"minutes"`
}

// Summary is what the statistics screen, the header and the tray show.
type Summary struct {
	Today Day `json:"today"`
	// Week is the last seven days, oldest first; the last one is today.
	Week        []Day `json:"week"`
	WeekMinutes int   `json:"weekMinutes"`
	// Streak counts the days in a row with at least one session, up to today.
	// A today without sessions yet does not break it: the count then ends
	// yesterday.
	Streak int `json:"streak"`
	// Sessions are today's, oldest first.
	Sessions []Session `json:"sessions"`
}

const dateLayout = "2006-01-02"

// Summarize sums the sessions up as of now, by calendar days in now's time zone.
func Summarize(sessions []Session, now time.Time) Summary {
	loc := now.Location()
	byDate := map[string]Day{}
	var today []Session
	todayKey := now.Format(dateLayout)
	for _, s := range sessions {
		key := s.End.In(loc).Format(dateLayout)
		d := byDate[key]
		d.Focus++
		d.Minutes += s.Minutes
		byDate[key] = d
		if key == todayKey {
			today = append(today, s)
		}
	}
	slices.SortStableFunc(today, func(a, b Session) int { return a.End.Compare(b.End) })

	// Calendar days are stepped with time.Date, not 24-hour steps, so a DST
	// change can't skip or repeat a day.
	day := func(back int) string {
		return time.Date(now.Year(), now.Month(), now.Day()-back, 12, 0, 0, 0, loc).Format(dateLayout)
	}
	sum := Summary{Week: make([]Day, 7), Sessions: today}
	if sum.Sessions == nil {
		sum.Sessions = []Session{}
	}
	for i := range 7 {
		key := day(6 - i)
		d := byDate[key]
		d.Date = key
		sum.Week[i] = d
		sum.WeekMinutes += d.Minutes
	}
	sum.Today = sum.Week[6]

	back := 0
	if sum.Today.Focus == 0 {
		back = 1
	}
	for byDate[day(back)].Focus > 0 {
		sum.Streak++
		back++
	}
	return sum
}
