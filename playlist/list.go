package playlist

import (
	"mime"
	"net/http"
	"path/filepath"
	"strings"

	name_reader "github.com/WinPooh32/peerstohttp/playlist/name"

	"github.com/anacrolix/torrent"
)

type Header struct {
	Hash  string `json:"hash"`
	Name  string `json:"name"`
	Files int    `json:"files"`
}

type Item struct {
	Name     string   `json:"name"`
	NameOrig string   `json:"name_orig"`
	Ext      string   `json:"ext"`
	MIME     string   `json:"mime"`
	Size     int64    `json:"size"`
	Path     []string `json:"path"`

	Tags []string `json:"tags"`
}

type PlayList struct {
	Header  Header `json:"header"`
	Content []Item `json:"content"`

	Torr       *torrent.Torrent    `json:"-"`
	Whitelist  map[string]struct{} `json:"-"`
	IgnoreTags map[string]struct{} `json:"-"`
}

func (p *PlayList) Render(w http.ResponseWriter, r *http.Request) error {
	var files = p.Torr.Files()
	var name = p.Torr.Name()

	var content = make([]Item, 0, len(files))

	for _, f := range files {
		var path = f.FileInfo().Path
		var base string

		var tags = ExtractPathTags(path)
		if Overlap(tags, p.IgnoreTags) {
			continue
		}

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

		content = append(content, makeItem(f, path, tags, base, ext))
	}

	p.Header.Name = name
	p.Header.Hash = p.Torr.InfoHash().String()
	p.Header.Files = len(p.Torr.Files())
	p.Content = content

	return nil
}

func makeItem(file *torrent.File, path, tags []string, base, ext string) Item {
	var mime = mime.TypeByExtension(ext)
	var name = strings.TrimSuffix(base, ext)

	if mime != "" {
		switch strings.Split(mime, "/")[0] {
		case "audio":
			name = name_reader.Music(name).Name()
		case "video":
			// TODO
		case "image":
			// TODO
		}
	}

	var item = Item{
		Name:     name,
		NameOrig: base,
		Ext:      ext,
		MIME:     mime,
		Size:     file.Length(),
		Path:     path,
		Tags:     tags,
	}

	return item
}
