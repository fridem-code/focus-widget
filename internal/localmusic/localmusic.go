// Package localmusic lists the music files in the user's folder for the
// "Мои файлы" genre (T22). The page plays them through the asset server, which
// serves only the files of the last scan, by their number in it.
package localmusic

import (
	"errors"
	"io/fs"
	"path/filepath"
	"strings"
)

// Types are the playable extensions and their media types; WebView2 plays all
// of them natively.
var Types = map[string]string{
	".mp3":  "audio/mpeg",
	".m4a":  "audio/mp4",
	".aac":  "audio/aac",
	".ogg":  "audio/ogg",
	".oga":  "audio/ogg",
	".opus": "audio/ogg",
	".flac": "audio/flac",
	".wav":  "audio/wav",
}

// MaxTracks bounds a scan, so a whole drive picked by mistake stays quick.
const MaxTracks = 5000

// Track is one music file.
type Track struct {
	Path  string // absolute
	Title string // the file name without its extension
}

var errEnough = errors.New("enough tracks")

// Scan returns the music files in dir and its subfolders, in path order.
// Unreadable subfolders are skipped; only an unreadable dir is an error.
func Scan(dir string) ([]Track, error) {
	var tracks []Track
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if path == dir {
				return err
			}
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if _, ok := Types[ext]; !ok {
			return nil
		}
		tracks = append(tracks, Track{Path: path, Title: strings.TrimSuffix(d.Name(), filepath.Ext(d.Name()))})
		if len(tracks) == MaxTracks {
			return errEnough
		}
		return nil
	})
	if err != nil && !errors.Is(err, errEnough) {
		return nil, err
	}
	return tracks, nil
}
