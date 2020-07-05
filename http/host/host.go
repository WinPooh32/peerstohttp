package host

import (
	"context"
	"net/http"
)

type contextKey int

const (
	ContextKeyHost contextKey = iota
)

func Host(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var scheme string

		if r.TLS != nil {
			scheme = "https://"
		} else {
			scheme = "http://"
		}

		var host = scheme + r.Host

		var ctx = context.WithValue(r.Context(), ContextKeyHost, host)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
