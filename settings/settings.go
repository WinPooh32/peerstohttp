package settings

import "flag"

type Settings struct {
	Host            *string
	Port            *int
	TorrPort        *int
	DownloadDir     *string
	DownloadRate    *int
	UploadRate      *int
	MaxConnections  *int
	NoDHT           *bool
	NoUTP           *bool
	ForceEncryption *bool
	JsonLogs        *bool
	TorrentDebug    *bool
	Profile         *bool
}

func (s *Settings) parse() {
	// HTTP

	*s = Settings{
		Host: flag.String("host", "0.0.0.0", "listening server ip"),
		Port: flag.Int("port", 80, "listening port"),

		// Torrent
		TorrPort:        flag.Int("port-torr", 0, "listening port for torrent"),
		DownloadDir:     flag.String("dir", "", "where files will be downloaded to"),
		DownloadRate:    flag.Int("down-rate", 0, "download speed rate in kib/s"),
		UploadRate:      flag.Int("up-rate", 0, "upload speed rate in kib/s"),
		MaxConnections:  flag.Int("max-connections", 50, "max connections per torrent"),
		NoDHT:           flag.Bool("no-dht", false, "disable dht"),
		NoUTP:           flag.Bool("no-utp", false, "disable utp"),
		ForceEncryption: flag.Bool("force-encryption", false, "force encryption"),

		// Debug
		JsonLogs:     flag.Bool("json-logs", false, "json logs output"),
		TorrentDebug: flag.Bool("torr-debug", false, "enable torrent backend verbosity"),
		Profile:      flag.Bool("profile", false, "enable service profiling"),
	}

	flag.Parse()
}

var Service *Settings

func init() {
	Service = &Settings{}
	Service.parse()
}
