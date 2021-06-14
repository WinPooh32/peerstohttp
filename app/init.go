package app

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	anacrolixlog "github.com/anacrolix/log"
	"github.com/anacrolix/missinggo/v2/filecache"
	"github.com/anacrolix/missinggo/v2/resource"
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/mse"
	"github.com/anacrolix/torrent/storage"
	"go.etcd.io/bbolt"
	"golang.org/x/time/rate"

	"github.com/WinPooh32/peerstohttp/settings"
)

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

func p2p(service *settings.Settings, cwd string) (*torrent.Client, error) {
	var cfg *torrent.ClientConfig = torrent.NewDefaultClientConfig()

	// Bind port.
	cfg.ListenPort = *service.TorrPort

	// Download dir.
	cfg.DataDir = cwd

	// Rate limits.
	const kib = 1 << 10

	if *service.DownloadRate != 0 {
		cfg.DownloadRateLimiter = limit(*service.DownloadRate * kib)
	}

	if *service.UploadRate != 0 {
		cfg.UploadRateLimiter = limit(*service.UploadRate * kib)
	}

	// Connections limits.
	cfg.EstablishedConnsPerTorrent = *service.MaxConnections
	cfg.TorrentPeersLowWater = *service.MaxConnections

	// Discovery services.
	cfg.NoDHT = *service.NoDHT
	cfg.DisableUTP = *service.NoUTP
	cfg.DisableTCP = *service.NoTCP

	cfg.DisableIPv4 = *service.NoIPv4
	cfg.DisableIPv6 = *service.NoIPv6

	if *service.ProxyHTTP != "" {
		var u, err = url.Parse(*service.ProxyHTTP)
		if err != nil {
			return nil, fmt.Errorf("parse http proxy url: %w", err)
		}

		cfg.HTTPProxy = http.ProxyURL(u)
	}

	// Enable seeding.
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

	cfg.DefaultRequestStrategy = torrent.RequestStrategyFastest()

	// Torrent debug.
	cfg.Debug = false

	if !*service.TorrentDebug {
		cfg.Logger = anacrolixlog.Discard
	}

	// File cache.
	var err error
	var res resource.Provider
	var capacity int64

	if *service.CacheCapacity > 0 {
		capacity = *service.CacheCapacity
	} else {
		capacity = -1
	}

	res, err = makeResourceProvider(cwd, capacity)
	if err != nil {
		return nil, fmt.Errorf("make resource provider: %w", err)
	}

	cfg.DefaultStorage = makeStorageProvider(res)

	return torrent.NewClient(cfg)
}

func db(path string) (*bbolt.DB, error) {
	var db, err = bbolt.Open(path, 0600, &bbolt.Options{
		Timeout: 5 * time.Second,
		NoSync:  false,
	})
	if err != nil {
		return nil, err
	}

	// Create buckets.
	err = db.Update(func(tx *bbolt.Tx) error {
		var _, err = tx.CreateBucketIfNotExists([]byte(dbBucketInfo))
		if err != nil {
			return fmt.Errorf("create bucket: %w", err)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return db, nil
}

func makeResourceProvider(dir string, capacity int64) (resource.Provider, error) {
	var err error
	var fc *filecache.Cache

	fc, err = filecache.NewCache(dir)
	if err != nil {
		return nil, fmt.Errorf("new file cache: %w", err)
	}

	fc.SetCapacity(capacity)

	return fc.AsResourceProvider(), nil
}

func makeStorageProvider(res resource.Provider) storage.ClientImpl {
	return storage.NewResourcePieces(res)
}
