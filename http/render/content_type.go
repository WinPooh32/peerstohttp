package render

import (
	"context"
	"mime"
	"net/http"
	"strings"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
)

const ParamContentType = "ContentType"

// ContentType is an enumeration of common HTTP content types.
type ContentType int

// ContentTypes handled by this package.
const (
	ContentTypeUnknown = iota
	ContentTypeHTML
	ContentTypeJSON
	ContentTypeM3U
)

func GetContentType(s string) ContentType {
	switch strings.Split(s, ";")[0] {
	case "text/html", "application/xhtml+xml":
		return ContentTypeHTML
	case "application/json", "text/javascript":
		return ContentTypeJSON
	case "application/mpegURL", "application/x-mpegurl", "audio/mpegurl", "audio/x-mpegurl":
		return ContentTypeM3U
	default:
		return ContentTypeUnknown
	}
}

func GetAcceptedContentType(r *http.Request) ContentType {
	if contentType, ok := r.Context().Value(render.ContentTypeCtxKey).(ContentType); ok {
		return contentType
	}

	var contentType = GetContentType(mime.TypeByExtension("." + chi.URLParam(r, ParamContentType)))

	if contentType == ContentTypeUnknown {
		contentType = ContentTypeJSON
	}
	return contentType
}

func ListContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var ctx = context.WithValue(r.Context(), render.ContentTypeCtxKey, GetAcceptedContentType(r))
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
