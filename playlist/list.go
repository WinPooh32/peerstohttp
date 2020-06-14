package playlist

import (
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/anacrolix/torrent"
)

type Header struct {
	Hash string `json:"hash"`
	Name string `json:"name"`
}

type Video struct {
	Container string `json:"container"`
	Codecs    string `json:"codecs"`
	Bitrate   int    `json:"bitrate"`
	Duration  int64  `json:"length"`

	Width  int `json:"width"`
	Height int `json:"height"`

	Lang string `json:"lang"`

	PreviewIMG string    `json:"preview_img"`
	Season     int       `json:"season"`
	Episode    int       `json:"episode"`
	Genres     []string  `json:"genres"`
	ReleasedAt time.Time `json:"released_at"`
}

type Music struct {
	Codec    string `json:"codec"`
	Bitrate  int    `json:"bitrate"`
	Duration int64  `json:"length"`

	Lang string `json:"lang"`

	CoverIMG   string    `json:"cover_img"`
	TrackNum   int       `json:"track_num"`
	Genre      string    `json:"genre"`
	Artists    []string  `json:"artist"`
	Album      string    `json:"album"`
	ReleasedAt time.Time `json:"released_at"`
}

type Image struct {
	Container string `json:"container"`
	Codec     string `json:"codec"`

	Width  int `json:"width"`
	Height int `json:"height"`
}

type Item struct {
	Name string   `json:"name"`
	Ext  string   `json:"ext"`
	MIME string   `json:"mime"`
	Size int64    `json:"size"`
	Path []string `json:"path"`

	Tags []string `json:"tags"`

	Video *Video `json:"video"`
	Music *Music `json:"audio"`
	Image *Image `json:"image"`
}

type PlayList struct {
	Header  Header `json:"header"`
	Content []Item `json:"content"`

	Torr      *torrent.Torrent    `json:"-"`
	Whitelist map[string]struct{} `json:"-"`
}

func (p *PlayList) Render(w http.ResponseWriter, r *http.Request) error {
	var files = p.Torr.Files()
	var name = p.Torr.Name()

	var content = make([]Item, 0, len(files))

	for _, f := range files {
		var path = f.FileInfo().Path
		var base string

		if size := len(path); size > 1 {
			base = path[len(path)-1]
		} else if size == 1 {
			base = path[0]
		} else {
			base = f.DisplayPath()
			path = append(path, base)
		}

		var ext = filepath.Ext(base)

		if len(p.Whitelist) != 0 {
			if _, ok := p.Whitelist[ext]; !ok {
				continue
			}
		}

		content = append(content, makeItem(f, path, base, ext))
	}

	p.Header.Name = name
	p.Header.Hash = p.Torr.InfoHash().String()
	p.Content = content

	return nil
}

func makeItem(file *torrent.File, path []string, base, ext string) Item {
	var name = strings.TrimSuffix(base, ext)

	var item = Item{
		Name: name,
		Ext:  ext,
		MIME: mime.TypeByExtension(ext),
		Size: file.Length(),
		Path: path,
		Tags: ExtractPathTags(path),
	}

	switch strings.Split(item.MIME, "/")[0] {
	case "audio":
		// TODO
		var music Music
		music.Duration = -1
		music.Artists = make([]string, 0, 0)

		item.Music = &music

	case "video":
		// TODO
		var video Video
		video.Duration = -1

		item.Video = &video
		item.Video.Genres = make([]string, 0, 0)

	case "image":
		// TODO
		var image Image

		item.Image = &image
	}

	return item
}
