package http

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"

	"github.com/WinPooh32/peerstohttp/app"
)

func addNewTorrentHash(ctx context.Context, app *app.App, hash string) (*torrent.Torrent, bool) {
	var t, new = app.Client().AddTorrentInfoHash(metainfo.NewHashFromHex(hash))

	if t == nil {
		return nil, false
	}

	if new {
		app.Track(t)
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

	app.Track(t)

	select {
	case <-ctx.Done():
		return nil, false

	case <-t.GotInfo():
	}

	return t, true
}

func serveTorrentContent(w http.ResponseWriter, r *http.Request, file *torrent.File) error {
	var err error
	var name string
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

	// Only the first 512 bytes are used to sniff the content type.
	var buffer = make([]byte, 512)

	// Don't wait for pieces to complete and be verified.
	reader.SetResponsive()
	// Read ahead 10% of file.
	reader.SetReadahead((file.Length() * 10) / 100)

	_, err = reader.Read(buffer)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Disposition", `filename="`+url.PathEscape(name)+`"`)

	_, err = reader.Seek(0, 0)
	if err != nil {
		return err
	}

	http.ServeContent(w, r, "", time.Unix(0, 0), reader)
	return nil
}
