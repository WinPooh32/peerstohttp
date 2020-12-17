package http

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/anacrolix/torrent/metainfo"
	"github.com/asaskevich/govalidator"
	"github.com/go-chi/chi"
)

func hash(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var hash = strings.ToLower(chi.URLParam(r, paramHash))

		if !govalidator.IsSHA1(hash) {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		ctx := context.WithValue(r.Context(), paramHash, hash)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func magnet(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var parts = strings.Split(r.URL.RawQuery, "&")

		if len(parts) == 0 {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		parts[0] = strings.ToLower(parts[0])

		var magnetURI = chi.URLParam(r, "*") + "?" + strings.Join(parts, "&")
		var magnet, err = metainfo.ParseMagnetUri(magnetURI)

		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		ctx := context.WithValue(r.Context(), paramMagnet, &magnet)
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

		if !govalidator.IsRequestURI("/" + path) {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}

		ctx := context.WithValue(r.Context(), paramPath, path)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func whitelist(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var whitelist = map[string]struct{}{}
		var p = chi.URLParam(r, paramWhitelist)

		if p != "-" {
			var args = strings.Split(strings.ToLower(p), ",")
			for _, v := range args {
				whitelist["."+v] = struct{}{}
			}
		}

		ctx := context.WithValue(r.Context(), paramWhitelist, whitelist)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func ignoretags(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var ignoretags = map[string]struct{}{}
		var p = chi.URLParam(r, paramIgnoretags)

		if p != "-" {
			var args = strings.Split(strings.ToLower(p), ",")
			for _, v := range args {
				ignoretags[v] = struct{}{}
			}
		}

		ctx := context.WithValue(r.Context(), paramIgnoretags, ignoretags)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
