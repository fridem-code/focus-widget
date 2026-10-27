package services

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"

	"focuswidget/internal/catalog"
	"focuswidget/internal/config"
	"focuswidget/internal/localmusic"
)

const (
	// LocalGenreID is the genre of the user's music folder (T22).
	LocalGenreID   = "local"
	localGenreName = "Мои файлы"
	// LocalMusicPath is where the page fetches a folder file by its number.
	LocalMusicPath = "/local-music/"
)

// MusicService exposes the genre catalog and remembers the music choices.
// Playback itself lives in the frontend (the YouTube player is a DOM object).
type MusicService struct {
	catalog  catalog.Catalog
	settings *SettingsService
	// pickFolder asks the user for a folder; "" when they cancel.
	pickFolder func() (string, error)

	mu     sync.Mutex
	tracks []localmusic.Track // the music folder's last scan
}

func NewMusicService(c catalog.Catalog, settings *SettingsService, pickFolder func() (string, error)) *MusicService {
	return &MusicService{catalog: c, settings: settings, pickFolder: pickFolder}
}

// Genres returns the catalog — the user's genres.json override if valid,
// otherwise the built-in genres — and, when a music folder is chosen, the
// "Мои файлы" genre with the folder's files, scanned now. A folder that is
// empty or gone gives the genre no sources, which the page reports.
func (s *MusicService) Genres() catalog.Catalog {
	c := s.catalog
	folder := s.settings.Get().Music.LocalFolder
	if folder == "" {
		s.setTracks(nil)
		return c
	}
	tracks, _ := localmusic.Scan(folder) // unreadable: no tracks, as for an empty folder
	s.setTracks(tracks)
	local := catalog.Genre{ID: LocalGenreID, Name: localGenreName, Sources: make([]catalog.Source, len(tracks))}
	for i, t := range tracks {
		local.Sources[i] = catalog.Source{Type: catalog.SourceFile, ID: strconv.Itoa(i), Title: t.Title}
	}
	c.Genres = append(slices.Clone(c.Genres), local)
	return c
}

func (s *MusicService) setTracks(tracks []localmusic.Track) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tracks = tracks
}

// Select remembers the current genre and source. An unknown genre or a source
// index outside it is rejected. "Мои файлы" is checked against the last scan,
// so stepping through the tracks does not rescan the folder.
func (s *MusicService) Select(genreID string, sourceIndex int) error {
	var n int
	if genreID == LocalGenreID {
		s.mu.Lock()
		n = len(s.tracks)
		s.mu.Unlock()
	} else {
		g, ok := s.catalog.Genre(genreID)
		if !ok {
			return fmt.Errorf("unknown genre %q", genreID)
		}
		n = len(g.Sources)
	}
	if sourceIndex < 0 || sourceIndex >= n {
		return fmt.Errorf("source index %d out of range for genre %q (%d sources)", sourceIndex, genreID, n)
	}
	_, err := s.settings.modify(func(cur *config.Settings) {
		cur.Music.GenreID = genreID
		cur.Music.SourceIndex = sourceIndex
	})
	return err
}

// SaveVolume remembers volume (0–100) and mute.
func (s *MusicService) SaveVolume(volume int, muted bool) error {
	_, err := s.settings.modify(func(cur *config.Settings) {
		cur.Music.Volume = volume
		cur.Music.Muted = muted
	})
	return err
}

// ChooseFolder asks the user for their music folder and remembers it. It
// returns the folder, or "" when the user cancels.
func (s *MusicService) ChooseFolder() (string, error) {
	folder, err := s.pickFolder()
	if err != nil || folder == "" {
		return "", err
	}
	if _, err := s.settings.modify(func(cur *config.Settings) { cur.Music.LocalFolder = folder }); err != nil {
		return "", err
	}
	return folder, nil
}

// ClearFolder forgets the music folder; "Мои файлы" leaves the genre list.
func (s *MusicService) ClearFolder() error {
	_, err := s.settings.modify(func(cur *config.Settings) { cur.Music.LocalFolder = "" })
	return err
}

// LocalMusic serves the music folder's files to the page at
// LocalMusicPath<number>, the number from the last scan; nothing else on disk
// is reachable. Other requests go to next. A function rather than a method,
// so it is not bound for the page like the service's methods.
func LocalMusic(s *MusicService, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rest, ok := strings.CutPrefix(r.URL.Path, LocalMusicPath)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}
		i, err := strconv.Atoi(rest)
		s.mu.Lock()
		var track localmusic.Track
		found := err == nil && i >= 0 && i < len(s.tracks)
		if found {
			track = s.tracks[i]
		}
		s.mu.Unlock()
		if !found {
			http.NotFound(w, r)
			return
		}
		f, err := os.Open(track.Path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer func() { _ = f.Close() }()
		info, err := f.Stat()
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", localmusic.Types[strings.ToLower(filepath.Ext(track.Path))])
		// Ranges included, so the page can seek and start long files fast.
		http.ServeContent(w, r, "", info.ModTime(), f)
	})
}
