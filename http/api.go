package http

import (
	"fmt"
	"net/http"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"github.com/rs/zerolog/log"

	"github.com/WinPooh32/peerstohttp/app"
	"github.com/WinPooh32/peerstohttp/http/host"
	list_render "github.com/WinPooh32/peerstohttp/http/render"
	"github.com/WinPooh32/peerstohttp/playlist"
)

const (
	paramMagnet    = "magnet"
	paramHash      = "hash"
	paramPath      = "path"
	paramWhitelist = "whitelist"
)

var (
	patternList = fmt.Sprintf("%s:[json,m3u,html]+", list_render.ParamContentType)
)

type handle struct {
	app *app.App
}

func RouteApp(r chi.Router, app *app.App) {
	var h = handle{
		app: app,
	}

	r.Route("/list/{"+patternList+"}/{"+paramWhitelist+"}", func(r chi.Router) {
		r.Use(
			whitelist,
			host.Host,
			list_render.ListContentType,
			list_render.SetListResponder,
		)

		r.With(hash).Get("/hash/{hash}", h.hash)
		r.With(magnet).Get("/magnet/*", h.magnet)
	})

	r.With(hash, path).Get("/content/{"+paramHash+"}/*", h.content)
}

func (h *handle) hash(w http.ResponseWriter, r *http.Request) {
	var hash = r.Context().Value(paramHash).(string)
	var whitelist = r.Context().Value(paramWhitelist).(map[string]struct{})

	t, ok := addNewTorrentHash(r.Context(), h.app, hash)
	if !ok {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	render.Render(w, r, &playlist.PlayList{Torr: t, Whitelist: whitelist})
}

func (h *handle) magnet(w http.ResponseWriter, r *http.Request) {
	var magnet = r.Context().Value(paramMagnet).(string)
	var whitelist = r.Context().Value(paramWhitelist).(map[string]struct{})

	var t, ok = addNewTorrentMagnet(r.Context(), h.app, magnet)
	if !ok {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	render.Render(w, r, &playlist.PlayList{Torr: t, Whitelist: whitelist})
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

	//TODO stream folder as zip file.

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

	var err = serveTorrentContent(w, r, file)
	if err != nil {
		log.Warn().Err(err).Msg("write content")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
