package settings

import "flag"

type Settings struct {
	Host            *string
	Port            *int
	DownloadDir     *string
	DownloadRate    *int
	UploadRate      *int
	MaxConnections  *int
	NoDHT           *bool
	ForceEncryption *bool
	JsonLogs        *bool
	TorrentDebug    *bool
}

func (s *Settings) parse() {
	// HTTP
	s.Host = flag.String("host", "0.0.0.0", "listening server ip")
	s.Port = flag.Int("port", 80, "listening port")

	// Torrent
	s.DownloadDir = flag.String("dir", "", "where files will be downloaded to")
	s.DownloadRate = flag.Int("up-rate", 0, "download speed rate in kib/s")
	s.UploadRate = flag.Int("down-rate", 0, "upload speed rate in kib/s")
	s.MaxConnections = flag.Int("max-connections", 20, "max connections per torrent")
	s.NoDHT = flag.Bool("no-dht", false, "disable dht")
	s.ForceEncryption = flag.Bool("force-encryption", false, "force encryption")

	// Debug
	s.JsonLogs = flag.Bool("json-logs", false, "json logs output")
	s.TorrentDebug = flag.Bool("torr-debug", false, "enable torrent backend verbosity")

	flag.Parse()
}

var Service *Settings

func init() {
	Service = &Settings{}
	Service.parse()
}
