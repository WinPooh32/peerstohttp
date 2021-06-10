package render

import (
	"context"
	"net/http"

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

func ContentTypeFromString(s string) ContentType {
	switch s {
	case "html":
		return ContentTypeHTML
	case "json":
		return ContentTypeJSON
	case "m3u":
		return ContentTypeM3U
	default:
		return ContentTypeUnknown
	}
}

func GetAcceptedContentType(r *http.Request) ContentType {
	if contentType, ok := r.Context().Value(render.ContentTypeCtxKey).(ContentType); ok {
		return contentType
	}

	var p = chi.URLParam(r, ParamContentType)
	var contentType = ContentTypeFromString(p)

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
