package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/rs/zerolog/log"
)

const defaultShutdownTimeout = 60 * time.Second

func stopSignal() <-chan os.Signal {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, os.Kill)
	return c
}

func graceShutdownHTTP(server *http.Server, client *torrent.Client) {
	var ctx, cancel = context.WithTimeout(context.Background(), defaultShutdownTimeout)
	defer cancel()

	var err = server.Shutdown(ctx)
	if err != nil {
		log.Err(err).Msg("http shutdown")
	}

	client.Close()
}

func handleStopSignals(ctx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, server *http.Server, client *torrent.Client) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancel()

		select {
		case <-ctx.Done():
			log.Warn().Err(ctx.Err())

		case s := <-stopSignal():
			log.Warn().Msgf("got os signal: %s", s)
		}

		graceShutdownHTTP(server, client)
	}()
}

func serveHTTP(cancel context.CancelFunc, wg *sync.WaitGroup, server *http.Server) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancel()

		log.Info().Msg("http start")

		var err = server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.Err(err).Msg("http shutdown")
		}

		log.Info().Msg("http stop")
	}()
}
