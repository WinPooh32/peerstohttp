package app

import (
	"fmt"
	"io/ioutil"
	"os"
	"sync"

	anacrolixlog "github.com/anacrolix/log"
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/mse"
	"golang.org/x/time/rate"

	"github.com/WinPooh32/peerstohttp/settings"
)

// TODO add torrent management(disk cache size control, start/stop and etc.).
type App struct {
	client *torrent.Client

	torrents map[string]*torrent.Torrent
	mu       sync.RWMutex

	// Path to temporary data folder.
	tmp string
}

func New(service *settings.Settings) (*App, error) {
	var err error
	var tmp string
	var cfg *torrent.ClientConfig = torrent.NewDefaultClientConfig()

	// Bind port.
	cfg.ListenPort = *service.TorrPort

	// Download directory.
	if *service.DownloadDir == "" {
		tmp, err = ioutil.TempDir("", "peerstohttp")
		if err != nil {
			return nil, fmt.Errorf("create temp folder: %w", err)
		}

		cfg.DataDir = tmp
	} else {
		cfg.DataDir = *service.DownloadDir
	}

	// Rate limits.
	const kib = 1 << 10

	if *service.DownloadRate != 0 {
		cfg.DownloadRateLimiter = limit(*service.DownloadRate * kib)
	}

	if *service.UploadRate != 0 {
		cfg.UploadRateLimiter = limit(*service.UploadRate * kib)
	}

	cfg.EstablishedConnsPerTorrent = *service.MaxConnections
	cfg.TorrentPeersLowWater = *service.MaxConnections

	cfg.NoDHT = *service.NoDHT
	cfg.DisableUTP = *service.NoUTP
	cfg.Seed = true

	// Header obfuscation.
	cfg.HeaderObfuscationPolicy = torrent.HeaderObfuscationPolicy{
		Preferred:        true,
		RequirePreferred: *service.ForceEncryption,
	}
	// Force encryption.
	if *service.ForceEncryption {
		cfg.CryptoProvides = mse.CryptoMethodRC4
	}

	// Torrent debug.
	cfg.Debug = false
	if !*service.TorrentDebug {
		cfg.Logger = anacrolixlog.Discard
	}

	cfg.DefaultRequestStrategy = torrent.RequestStrategyFastest()

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
	app.mu.RLock()
	t, ok := app.torrents[hash]
	app.mu.Unlock()

	return t, ok
}

func (app *App) Close() error {
	var err error

	// Remove temporary data folder if required.
	if app.tmp != "" {
		err = os.RemoveAll(app.tmp)
		if err != nil {
			return err
		}
	}

	return nil
}

// https://gitlab.com/axet/libtorrent/-/blob/master/libtorrent.go
func limit(kbps int) *rate.Limiter {
	var l = rate.NewLimiter(rate.Inf, 0)

	if kbps > 0 {
		b := kbps
		if b < 16*1024 {
			b = 16 * 1024
		}
		l = rate.NewLimiter(rate.Limit(kbps), b)
	}

	return l
}
