package http

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/rs/zerolog/log"

	valid "github.com/asaskevich/govalidator"

	"github.com/WinPooh32/peerstohttp/app"
)

type listType string

const (
	listJSON listType = "json"
	listM3U           = "m3u"
	listHTML          = "html"
)

const (
	paramList   = "list"
	paramMagnet = "magnet"
	paramHash   = "hash"
	paramPath   = "path"
)

var (
	patternList = fmt.Sprintf("[%s,%s,%s]+", listJSON, listM3U, listHTML)
)

type handle struct {
	app *app.App
}

func RouteApp(r chi.Router, app *app.App) {
	var h = handle{
		app: app,
	}

	r.Route("/list/{"+paramList+":"+patternList+"}", func(r chi.Router) {
		r.Use(list)
		r.With(hash).Get("/hash/{"+paramHash+"}", h.hash)
		r.With(magnet).Get("/magnet/*", h.magnet)
	})

	r.With(hash).With(path).Get("/content/{"+paramHash+"}/*", h.content)
}

func (h *handle) hash(w http.ResponseWriter, r *http.Request) {
	var hash = r.Context().Value(paramHash).(string)
	var listType = r.Context().Value(paramList).(listType)

	t, ok := addNewTorrentHash(r.Context(), h.app, hash)
	if !ok {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	renderList(w, r, t, listType)
}

func renderList(w http.ResponseWriter, r *http.Request, t *torrent.Torrent, list listType) {
	var files = t.Info().Files

	switch list {
	case listHTML:
		w.Write(renderHTMLlinks(t.InfoHash().String(), t.Name(), files))
	case listJSON:
		fallthrough
	default:
		render.JSON(w, r, files)
	}
}

func (h *handle) magnet(w http.ResponseWriter, r *http.Request) {
	var magnet = r.Context().Value(paramMagnet).(string)
	var listType = r.Context().Value(paramList).(listType)

	var t, ok = addNewTorrentMagnet(r.Context(), h.app, magnet)
	if !ok {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	renderList(w, r, t, listType)
}

func (h *handle) content(w http.ResponseWriter, r *http.Request) {
	var hash = r.Context().Value(paramHash).(string)
	var path = r.Context().Value(paramPath).(string)

	var t, ok = h.app.Client().Torrent(metainfo.NewHashFromHex(hash))

	if !ok {
		t, ok = addNewTorrentHash(r.Context(), h.app, hash)
		if !ok {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
	}

	select {
	case <-r.Context().Done():
		http.Error(w, http.StatusText(http.StatusRequestTimeout), http.StatusRequestTimeout)
		return

	case <-t.GotInfo():
	}

	var file *torrent.File
	ok = false
	for _, f := range t.Files() {
		var p = f.Path()
		if p == path {
			file = f
			ok = true
			break
		}
	}

	if !ok || file == nil {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	var err = serveTorrentFile(w, r, file)
	if err != nil {
		log.Warn().Err(err).Msg("write content")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func list(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var list = chi.URLParam(r, paramList)
		var ctx = context.WithValue(r.Context(), paramList, listType(list))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func hash(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var hash = chi.URLParam(r, paramHash)

		if !valid.IsSHA1(hash) {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		ctx := context.WithValue(r.Context(), paramHash, hash)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func magnet(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var magnet = chi.URLParam(r, "*") + "?" + r.URL.RawQuery

		if !valid.IsMagnetURI(magnet) {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		ctx := context.WithValue(r.Context(), paramMagnet, magnet)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func path(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var path, err = url.QueryUnescape(chi.URLParam(r, "*"))

		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		if !valid.IsRequestURI("/" + path) {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		ctx := context.WithValue(r.Context(), paramPath, path)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func addNewTorrentHash(ctx context.Context, app *app.App, hash string) (*torrent.Torrent, bool) {
	var t, new = app.Client().AddTorrentInfoHash(metainfo.NewHashFromHex(hash))

	if new {
		app.Track(t)
	}

	select {
	case <-ctx.Done():
		return nil, false

	case <-t.GotInfo():
	}

	return t, t != nil
}

func addNewTorrentMagnet(ctx context.Context, app *app.App, magnetURI string) (*torrent.Torrent, bool) {
	var t, err = app.Client().AddMagnet(magnetURI)
	if err != nil {
		return nil, false
	}

	app.Track(t)

	select {
	case <-ctx.Done():
		return nil, false

	case <-t.GotInfo():
	}

	return t, t != nil
}

func renderHTMLlinks(hash string, torrName string, files []metainfo.FileInfo) []byte {
	var b []byte

	b = append(b, []byte(` <!DOCTYPE html>
<html>
<head>
<title>Title of the document</title>
</head>

<body>

`)...)

	for _, fi := range files {
		var escaped = joinEscapedPath(append([]string{torrName}, fi.Path...))
		//var escaped = url.PathEscape(torrName + "/" + path)
		var link = fmt.Sprintf("<a href=\"/content/%s/%s\">%s</a><br>\n", hash, escaped, strings.Join(fi.Path, "/"))

		b = append(b, []byte(link)...)
	}

	return append(b, []byte(`</body>
</html>`)...)
}

func joinEscapedPath(elems []string) string {
	var sep = "/"

	switch len(elems) {
	case 0:
		return ""
	case 1:
		return elems[0]
	}
	n := len(sep) * (len(elems) - 1)
	for i := 0; i < len(elems); i++ {
		n += len(elems[i])
	}

	var b strings.Builder
	b.Grow(n)
	b.WriteString(url.QueryEscape(elems[0]))
	for _, s := range elems[1:] {
		b.WriteString(sep)
		b.WriteString(url.QueryEscape(s))
	}
	return b.String()
}

func serveTorrentFile(w http.ResponseWriter, r *http.Request, file *torrent.File) error {
	reader := file.NewReader()

	// Don't wait for pieces to complete and be verified.
	reader.SetResponsive()
	// Preload 10% of file.
	reader.SetReadahead((file.Length() * 10) / 100)

	// Only the first 512 bytes are used to sniff the content type.
	buffer := make([]byte, 512)
	_, err := reader.Read(buffer)
	if err != nil {
		return err
	}

	_, err = reader.Seek(0, 0)
	if err != nil {
		return err
	}

	// Always returns a valid content-type and "application/octet-stream" if no others seemed to match.
	contentType := http.DetectContentType(buffer)

	path := file.FileInfo().Path
	fname := ""
	if len(path) == 0 {
		fname = file.DisplayPath()
	} else {
		fname = path[len(path)-1]
	}

	w.Header().Set("Content-Disposition", "filename="+fname)
	w.Header().Set("Content-Type", contentType)

	http.ServeContent(w, r, fname, time.Unix(0, 0), reader)
	return nil
}
