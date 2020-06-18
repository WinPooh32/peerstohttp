package http

import (
	"fmt"
	"net/http"
	"strings"

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
	var err error

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

	if t.Info().IsDir() && strings.Count(path, "/") == 0 {
		err = serveTorrentDir(w, r, t, path)
	} else {
		err = serveTorrentFile(w, r, t, path)
	}

	if err != nil {
		log.Warn().Err(err).Msg("serve content")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

//func fileInfoHeader(fi *torrent.File) (*zip.FileHeader, error) {

//}
