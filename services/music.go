package services

import (
	"fmt"

	"focuswidget/internal/catalog"
	"focuswidget/internal/config"
)

// MusicService exposes the genre catalog and remembers the music choices.
// Playback itself lives in the frontend (the YouTube player is a DOM object).
type MusicService struct {
	catalog  catalog.Catalog
	settings *SettingsService
}

func NewMusicService(c catalog.Catalog, settings *SettingsService) *MusicService {
	return &MusicService{catalog: c, settings: settings}
}

// Genres returns the catalog: the user's genres.json override if valid,
// otherwise the built-in genres.
func (s *MusicService) Genres() catalog.Catalog { return s.catalog }

// Select remembers the current genre and source. An unknown genre or a source
// index outside it is rejected.
func (s *MusicService) Select(genreID string, sourceIndex int) error {
	g, ok := s.catalog.Genre(genreID)
	if !ok {
		return fmt.Errorf("unknown genre %q", genreID)
	}
	if sourceIndex < 0 || sourceIndex >= len(g.Sources) {
		return fmt.Errorf("source index %d out of range for genre %q (%d sources)", sourceIndex, genreID, len(g.Sources))
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
