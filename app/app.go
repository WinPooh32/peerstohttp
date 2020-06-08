package app

import (
	"fmt"
	"io/ioutil"
	"os"
	"sync"

	anacrolixlog "github.com/anacrolix/log"
	"github.com/anacrolix/torrent"

	"github.com/WinPooh32/peerstohttp/settings"
)

type App struct {
	client *torrent.Client

	torrents map[string]*torrent.Torrent
	mu       sync.RWMutex

	tmp string
}

func New(service *settings.Settings) (*App, error) {
	var err error
	var tmp string
	var cfg *torrent.ClientConfig = torrent.NewDefaultClientConfig()

	if !*service.TorrentDebug {
		cfg.Logger = anacrolixlog.Discard
	}

	if *service.DownloadDir == "" {
		tmp, err = ioutil.TempDir("", "peerstohttp")
		if err != nil {
			return nil, fmt.Errorf("create temp folder: %w", err)
		}

		cfg.DataDir = tmp
	}

	// Bind any free port
	cfg.ListenPort = 0

	client, err := torrent.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("new torrent client: %w", err)
	}

	return &App{
		torrents: map[string]*torrent.Torrent{},
		client:   client,
		tmp:      tmp,
	}, nil
}

func (app *App) Client() *torrent.Client {
	return app.client
}

func (app *App) Track(torrent *torrent.Torrent) {
	app.mu.Lock()

	app.torrents[torrent.InfoHash().String()] = torrent

	app.mu.Unlock()
}

func (app *App) Torrent(hash string) (*torrent.Torrent, bool) {
	var torrent *torrent.Torrent
	var ok bool

	app.mu.RLock()
	torrent, ok = app.torrents[hash]
	app.mu.Unlock()

	return torrent, ok
}

func (app *App) Cleanup() error {
	var err error

	if app.tmp != "" {
		err = os.RemoveAll(app.tmp)
		if err != nil {
			return err
		}
	}

	return nil
}