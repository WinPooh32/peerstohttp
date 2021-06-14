package app

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"sync"

	"github.com/anacrolix/torrent"
	"github.com/rs/zerolog/log"
	"go.etcd.io/bbolt"

	"github.com/WinPooh32/peerstohttp/settings"

	"github.com/anacrolix/torrent/metainfo"
)

const (
	dbName       = ".app.bolt.db"
	dbBucketInfo = "torrent_info"
)

var trackers = [][]string{
	{"udp://opentor.org:2710", "https://bt.t-ru.org/ann?magnet", "http://bt.t-ru.org/ann?magnet"},
	{"udp://tracker.coppersurfer.tk:6969/announce"},
	{"http://retracker.local/announce"},
}

// TODO add torrent management(disk cache size control, start/stop and etc.).
type App struct {
	client *torrent.Client

	torrents map[string]*torrent.Torrent
	mu       sync.RWMutex

	db *bbolt.DB

	// Path to temporary data folder.
	tmp string
	cwd string
}

func New(service *settings.Settings) (*App, error) {
	var err error
	var tmp string
	var cwd string

	var client *torrent.Client
	var store *bbolt.DB

	// Working directory.
	if *service.DownloadDir == "" {
		tmp, err = ioutil.TempDir("", "peerstohttp")
		if err != nil {
			return nil, fmt.Errorf("create temp. folder: %w", err)
		}
		cwd = tmp
	} else {
		cwd = *service.DownloadDir
	}

	err = os.MkdirAll(cwd, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("failed to create dir: %w", err)
	}

	client, err = p2p(service, cwd)
	if err != nil {
		return nil, fmt.Errorf("new torrent client: %w", err)
	}

	store, err = db(fmt.Sprintf("%s/%s", cwd, dbName))
	if err != nil {
		return nil, fmt.Errorf("new db: %w", err)
	}

	var app = &App{
		torrents: map[string]*torrent.Torrent{},
		client:   client,
		db:       store,
		tmp:      tmp,
		cwd:      cwd,
	}

	err = app.Load()
	if err != nil {
		return nil, fmt.Errorf("load app state from db: %w", err)
	}

	log.Info().Msg("app loaded")

	return app, nil
}

func (app *App) Load() error {
	app.mu.Lock()
	defer app.mu.Unlock()

	return app.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte(dbBucketInfo))

		return b.ForEach(func(k, v []byte) error {
			var err error
			var mi *metainfo.MetaInfo
			var t *torrent.Torrent

			mi, err = metainfo.Load(bytes.NewReader(v))
			if err != nil {
				log.Warn().Msgf("read meta info: %s", err)
				return nil
			}

			t, err = app.client.AddTorrent(mi)
			if err != nil {
				log.Warn().Msgf("add torrent: %s", err)
				return nil
			}

			app.torrents[t.InfoHash().String()] = t

			return nil
		})
	})
}

func (app *App) Client() *torrent.Client {
	return app.client
}

func (app *App) Track(t *torrent.Torrent) (*torrent.Torrent, error) {
	return app.TrackContext(context.Background(), t)
}

func (app *App) TrackHash(hash metainfo.Hash) (*torrent.Torrent, error) {
	return app.TrackHashContext(context.Background(), hash)
}

func (app *App) TrackMagnet(magnet *metainfo.Magnet) (*torrent.Torrent, error) {
	return app.TrackMagnetContext(context.Background(), magnet)
}

func (app *App) TrackContext(ctx context.Context, t *torrent.Torrent) (*torrent.Torrent, error) {
	return t, app.trackContext(ctx, t)
}

func (app *App) TrackHashContext(ctx context.Context, hash metainfo.Hash) (*torrent.Torrent, error) {
	var t, _ = app.client.AddTorrentInfoHash(hash)

	if t == nil {
		return nil, fmt.Errorf("torrent is nil")
	}

	return t, app.trackContext(ctx, t)
}

func (app *App) TrackMagnetContext(ctx context.Context, magnet *metainfo.Magnet) (*torrent.Torrent, error) {
	var err error
	var t *torrent.Torrent

	var spec = &torrent.TorrentSpec{
		Trackers:    [][]string{magnet.Trackers},
		DisplayName: magnet.DisplayName,
		InfoHash:    magnet.InfoHash,
	}

	t, _, err = app.client.AddTorrentSpec(spec)
	if err != nil {
		return nil, fmt.Errorf("torrent add magnet: %w", err)
	}

	return t, app.trackContext(ctx, t)
}

func (app *App) Torrent(hash string) (*torrent.Torrent, bool) {
	app.mu.RLock()
	defer app.mu.RUnlock()

	t, ok := app.torrents[hash]
	return t, ok
}

func (app *App) Close() error {
	var err error

	// Remove temporary data folder if required.
	if app.tmp != "" {
		err = os.RemoveAll(app.tmp)
		if err != nil {
			return fmt.Errorf("remove temp. dir: %w", err)
		}
	}

	// Close database.
	if app.db != nil {
		err = app.db.Close()
		if err != nil {
			return fmt.Errorf("close db: %w", err)
		}
	}

	return nil
}

func (app *App) track(t *torrent.Torrent) error {
	app.mu.Lock()
	defer app.mu.Unlock()

	var err error
	var mi = t.Metainfo()
	var buf = bytes.NewBuffer(nil)

	err = mi.Write(buf)
	if err != nil {
		return fmt.Errorf("write metaInfo: %w", err)
	}

	err = app.db.Update(func(tx *bbolt.Tx) error {
		var b = tx.Bucket([]byte(dbBucketInfo))
		return b.Put(t.InfoHash().Bytes(), buf.Bytes())
	})
	if err != nil {
		return fmt.Errorf("put to db: %w", err)
	}

	t.AddTrackers(trackers)

	app.torrents[t.InfoHash().String()] = t

	return nil
}

func (app *App) trackContext(ctx context.Context, t *torrent.Torrent) error {

	select {
	case <-ctx.Done():
		return ctx.Err()

	case <-t.GotInfo():
	}

	var err = app.track(t)
	if err != nil {
		return fmt.Errorf("track torrent: %w", err)
	}

	return nil
}
