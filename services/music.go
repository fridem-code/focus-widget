package services

import "focuswidget/internal/catalog"

// MusicService exposes the genre catalog to the frontend. Playback itself
// lives in the frontend (the YouTube player is a DOM object); T7 adds saving
// the selected genre and volume.
type MusicService struct {
	catalog catalog.Catalog
}

func NewMusicService(c catalog.Catalog) *MusicService {
	return &MusicService{catalog: c}
}

// Genres returns the catalog: the user's genres.json override if valid,
// otherwise the built-in genres.
func (s *MusicService) Genres() catalog.Catalog { return s.catalog }
