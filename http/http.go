package http

import (
	"bytes"
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
	var info = t.Info()

	var hash = t.InfoHash().String()
	var name = t.Name()

	switch list {
	case listM3U:
		w.Header().Set("Content-Type", "application/mpegurl")
		w.WriteHeader(http.StatusOK)
		w.Write(renderM3Ulinks(r.URL, hash, name, info))

	case listHTML:
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write(renderHTMLlinks(hash, name, info))

	case listJSON:
		fallthrough

	default:
		render.JSON(w, r, info.Files)
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

	if t.Info().IsDir() {
		for _, f := range t.Files() {
			var p = f.Path()
			if p == path {
				file = f
				ok = true
				break
			}
		}
	} else if ok = (path == t.Info().Name); ok {
		file = t.Files()[0]
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

func renderHTMLlinks(hash string, torrName string, info *metainfo.Info) []byte {
	var files = info.Files
	var playlist bytes.Buffer

	playlist.WriteString(` <!DOCTYPE html>
<html>
<head>
<title>Title of the document</title>
</head>

<body>
`)
	if !info.IsDir() {
		formatLinkHTML(hash, torrName, nil, &playlist)
	}

	for _, fi := range files {
		formatLinkHTML(hash, torrName, fi.Path, &playlist)
	}

	playlist.WriteString(`</body>
</html>`)

	return playlist.Bytes()
}

func renderM3Ulinks(req *url.URL, hash string, torrName string, info *metainfo.Info) []byte {
	var playlist bytes.Buffer
	var files = info.Files

	playlist.WriteString("#EXTM3U\r\n")
	playlist.WriteString("#EXTENC: UTF-8\r\n")

	if !info.IsDir() {
		formatLinkM3U(req.Scheme, req.Host, hash, torrName, nil, &playlist)
	}

	for _, fi := range files {
		formatLinkM3U(req.Scheme, req.Host, hash, torrName, fi.Path, &playlist)
	}

	return playlist.Bytes()
}

func formatLinkHTML(hash string, torrName string, path []string, out *bytes.Buffer) {
	var name string
	if len(path) > 0 {
		name = path[len(path)-1]
	} else {
		name = torrName
	}

	var link = url.URL{
		Path: "/content/" + joinEscapedPath(append([]string{hash, torrName}, path...)),
	}
	out.WriteString(`<a href="` + link.String() + `">` + name + "</a><br>\n")
}

func formatLinkM3U(scheme, host, hash string, torrName string, path []string, out *bytes.Buffer) {
	var name string
	if len(path) > 0 {
		name = path[len(path)-1]
	} else {
		name = torrName
	}

	var link = url.URL{
		Scheme: scheme,
		Host:   host,
		Path:   "/content/" + joinEscapedPath(append([]string{hash, torrName}, path...)),
	}
	out.WriteString("#EXTINF:-1," + name + "\r\n")
	out.WriteString(link.RequestURI() + "\r\n")
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
