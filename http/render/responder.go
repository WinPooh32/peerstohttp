package render

import (
	"bufio"
	"html"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/render"
	"github.com/rs/zerolog/log"

	"github.com/WinPooh32/peerstohttp/http/host"
	"github.com/WinPooh32/peerstohttp/playlist"
)

func M3U(w http.ResponseWriter, r *http.Request, list *playlist.PlayList) {
	var err error
	var buf = bufio.NewWriter(w)
	var items = list.Content
	var host = r.Context().Value(host.ContextKeyHost).(string)

	w.Header().Set("Content-Disposition", "filename=\""+url.PathEscape(list.Header.Name)+".m3u8\"")
	w.Header().Set("Content-Type", "application/x-mpegURL; charset=utf-8")
	if status, ok := r.Context().Value(render.StatusCtxKey).(int); ok {
		w.WriteHeader(status)
	}

	_, err = buf.WriteString(
		"#EXTM3U\r\n" +
			"#EXTENC: UTF-8\r\n",
	)
	if err != nil {
		log.Error().Err(err).Msg("responder m3u header")
		return
	}

	for _, itm := range items {
		var name = list.Header.Name
		var hash = list.Header.Hash
		var path = strings.Join(itm.Path, "/")
		var duration int64

		var contentURL = hash
		if len(path) == 1 {
			contentURL += "/" + url.PathEscape(name)
		} else {
			contentURL += "/" + url.PathEscape(name) + "/" + url.PathEscape(path)
		}

		var displayName string
		if len(itm.Path) > 1 {
			displayName = itm.Path[len(itm.Path)-2] + "/" + itm.Name
		} else {
			displayName = itm.Name
		}

		if itm.Music != nil {
			duration = itm.Music.Duration
		} else if itm.Video != nil {
			duration = itm.Video.Duration
		} else {
			continue
		}

		_, err = buf.WriteString(
			"#EXTINF:" + strconv.FormatInt(duration, 10) + "," + displayName + "\r\n" +
				host + "/content/" + contentURL + "\r\n",
		)
		if err != nil {
			log.Error().Err(err).Msg("responder m3u item")
			return
		}
	}

	err = buf.Flush()
	if err != nil {
		log.Error().Err(err).Msg("responder m3u: flush buffer")
		return
	}
}

func HTML(w http.ResponseWriter, r *http.Request, list *playlist.PlayList) {
	var err error
	var buf = bufio.NewWriter(w)
	var items = list.Content

	w.Header().Set("Content-Disposition", "filename=\""+url.PathEscape(list.Header.Name)+".html\"")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if status, ok := r.Context().Value(render.StatusCtxKey).(int); ok {
		w.WriteHeader(status)
	}

	_, err = buf.WriteString(` <!DOCTYPE html>
<html>
<head>
<title>` + html.EscapeString(list.Header.Name) + `</title>
<meta http-equiv="Content-Type" content="text/html; charset=utf-8" />
</head>

<body>
`)
	if err != nil {
		log.Error().Err(err).Msg("responder html header")
		return
	}

	for _, itm := range items {
		var name = list.Header.Name
		var hash = list.Header.Hash
		var path = strings.Join(itm.Path, "/")

		var contentURL = hash
		if len(itm.Path) == 1 {
			contentURL += "/" + url.PathEscape(name)
		} else {
			contentURL += "/" + url.PathEscape(name) + "/" + url.PathEscape(path)
		}

		_, err = buf.WriteString(
			`<a href="/content/` + contentURL + `">` + path + `</a></br>`,
		)
		if err != nil {
			log.Error().Err(err).Msg("responder html item")
			return
		}
	}

	buf.WriteString(`</body>
</html>`)
	if err != nil {
		log.Error().Err(err).Msg("responder html")
		return
	}

	err = buf.Flush()
	if err != nil {
		log.Error().Err(err).Msg("responder html: flush buffer")
		return
	}
}

func Responder(w http.ResponseWriter, r *http.Request, v interface{}) {
	var list, ok = v.(*playlist.PlayList)
	if !ok {
		log.Error().Msg("responder: expected playlist.PlayList")
		return
	}

	// Format response based on request Accept header.
	switch GetAcceptedContentType(r) {
	case ContentTypeJSON:
		render.JSON(w, r, list)
	case ContentTypeHTML:
		HTML(w, r, list)
	case ContentTypeM3U:
		M3U(w, r, list)
	default:
		render.JSON(w, r, list)
	}
}

// SetListResponder is a middleware that switches default chi responder to list responder.
func SetListResponder(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		render.Respond = Responder
		defer func() {
			render.Respond = render.DefaultResponder
		}()
		next.ServeHTTP(w, r)
	})
}
