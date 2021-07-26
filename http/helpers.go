package http

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"path/filepath"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/rs/zerolog/log"

	"github.com/WinPooh32/peerstohttp/app"
)

var errFileNotFound = errors.New("file not found")

func addNewTorrentHash(ctx context.Context, app *app.App, hash string) (*torrent.Torrent, bool) {
	var t, new = app.Client().AddTorrentInfoHash(metainfo.NewHashFromHex(hash))

	if t == nil {
		return nil, false
	}

	if new {
		app.TrackContext(ctx, t)
	}

	select {
	case <-ctx.Done():
		return nil, false

	case <-t.GotInfo():
	}

	return t, true
}

func addNewTorrentMagnet(ctx context.Context, app *app.App, magnetURI string) (*torrent.Torrent, bool) {
	var t, err = app.Client().AddMagnet(magnetURI)
	if err != nil {
		return nil, false
	}

	if t == nil {
		return nil, false
	}

	app.TrackContext(ctx, t)

	select {
	case <-ctx.Done():
		return nil, false

	case <-t.GotInfo():
	}

	return t, true
}

func serveTorrentDir(w http.ResponseWriter, r *http.Request, t *torrent.Torrent, path string) error {
	var err error
	//var name string
	//
	//if path == "" {
	//	name = t.Name()
	//} else {
	//	name = filepath.Base(path)
	//}
	//
	//w.Header().Set("Content-Disposition", `filename="`+url.PathEscape(name+".zip")+`"`)
	//w.Header().Set("Content-Type", "application/zip")
	//w.WriteHeader(http.StatusOK)
	//
	//var reader = t.NewReader()
	//defer reader.Close()
	//
	//// TODO адаптер из файлов торрента
	//
	//var zipWriter io.WriteCloser
	//zipWriter = zip.NewWriter(w)
	//
	//for _, f := range t.Files() {
	//
	//	header, err := zip.FileInfoHeader(f)
	//	if err != nil {
	//		return err
	//	}
	//
	//
	//}
	//
	//_, err = io.Copy(zipWriter, reader)
	return err
}

func serveTorrentFile(w http.ResponseWriter, r *http.Request, t *torrent.Torrent, path string) error {
	var name string

	var file *torrent.File
	var ok bool

	log.Info().Msgf("file path = %s", path)

	file, ok = findFile(t, path)

	if !ok || file == nil {
		return errFileNotFound
	}

	var reader = file.NewReader()
	defer reader.Close()

	fip := file.FileInfo().Path
	if len(fip) == 0 {
		name = file.DisplayPath()
	} else if len(fip) == 1 {
		name = fip[0]
	} else {
		name = fip[len(fip)-1]
	}

	return serveContent(w, r, file.Length(), reader, name)
}

func findFile(t *torrent.Torrent, path string) (*torrent.File, bool) {
	var file *torrent.File

	if !t.Info().IsDir() {
		if filepath.Base(path) != t.Info().Name {
			return nil, false
		}
		file = t.Files()[0]
	} else {
		for _, f := range t.Files() {
			var p = f.Path()
			if p == path {
				file = f
				break
			}
		}
		if file == nil {
			return nil, false
		}
	}

	return file, true
}

func serveContent(w http.ResponseWriter, r *http.Request, size int64, reader torrent.Reader, name string) error {
	var err error

	// Don't wait for pieces to complete and be verified.
	//reader.SetResponsive()

	if size > 0 {
		// Read ahead 10% of file.
		reader.SetReadahead((size * 10) / 100)
	}

	w.Header().Set("Content-Disposition", `filename="`+url.PathEscape(name)+`"`)

	_, err = reader.Seek(0, 0)
	if err != nil {
		return err
	}

	http.ServeContent(w, r, "", time.Unix(0, 0), reader)
	return nil
}
