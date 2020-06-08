package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"

	"github.com/go-chi/chi"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	application "github.com/WinPooh32/peerstohttp/app"
	peershttp "github.com/WinPooh32/peerstohttp/http"
	"github.com/WinPooh32/peerstohttp/settings"

	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"
)

func newCors(origins []string) *cors.Cors {
	return cors.New(cors.Options{
		AllowedOrigins:   origins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300, // Maximum value not ignored by any of major browsers
	})
}

func init() {
	if !*settings.Service.JsonLogs {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	}
}

func main() {
	var err error

	var app *application.App

	var server http.Server
	var router chi.Router

	var wg sync.WaitGroup

	var ctx, cancel = context.WithCancel(context.Background())

	app, err = application.New(settings.Service)
	if err != nil {
		log.Fatal().Err(err).Msg("new app")
	}
	defer func() {
		err = app.Cleanup()
		if err != nil {
			log.Fatal().Err(err).Msg("app clean up")
		}
	}()

	// Init router
	router = chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)
	router.Use(newCors([]string{"*"}).Handler)

	peershttp.RouteApp(router, app)

	// Init HTTP server
	server.Addr = fmt.Sprintf("%s:%d", *settings.Service.Host, *settings.Service.Port)
	server.Handler = router

	// Serve
	handleStopSignals(ctx, cancel, &wg, &server, app.Client())
	serveHTTP(cancel, &wg, &server)

	wg.Wait()
}
