package http

import (
	"context"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/asaskevich/govalidator"
	"github.com/go-chi/chi"
)

const (
	magnetURI = "^magnet:\\?xt=urn:[a-zA-Z0-9]+:[a-zA-Z0-9]{32,40}((&dn=.+&tr=.+)|(&tr=.+&dn=.+))$"
)

var (
	regMagnetURI = regexp.MustCompile(magnetURI)
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

		var magnet = chi.URLParam(r, "*") + "?" + strings.Join(parts, "&")

		if !regMagnetURI.Match([]byte(magnet)) {
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

		if chi.URLParam(r, paramWhitelist) != "any" {
			var args = strings.Split(strings.ToLower(chi.URLParam(r, paramWhitelist)), ",")
			for _, v := range args {
				whitelist["."+v] = struct{}{}
			}
		}

		ctx := context.WithValue(r.Context(), paramWhitelist, whitelist)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
