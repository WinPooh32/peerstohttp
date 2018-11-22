package main

import (
	"bytes"
	"encoding/base64"
	"log"
	"net/http"
	"path/filepath"
	"sort"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/storage"
)

//list of active torrents
var torrents map[string]*torrent.Torrent

//connection counters
var fileClients map[string]int
var fileStopping map[*torrent.File]chan bool

//torrent client
var torrClient *torrent.Client

func init() {
	torrents = make(map[string]*torrent.Torrent)
	fileClients = make(map[string]int)
	fileStopping = make(map[*torrent.File]chan bool)

	cfg := torrent.NewDefaultClientConfig()
	cfg.DefaultStorage = storage.NewMMap("./")

	cl, err := torrent.NewClient(cfg)
	torrClient = cl

	if err != nil {
		panic(err)
	}
}

func incFileClients(path string) int {
	if v, ok := fileClients[path]; ok {
		v++
		fileClients[path] = v
		return v
	} else {
		fileClients[path] = 1
		return 1
	}
}

func decFileClients(path string) int {
	if v, ok := fileClients[path]; ok {
		v--
		fileClients[path] = v
		return v
	} else {
		fileClients[path] = 0
		return 0
	}
}

func addMagnet(uri string) *torrent.Torrent {
	spec, err := torrent.TorrentSpecFromMagnetURI(uri)
	if err != nil {
		log.Println(err)
		return nil
	}

	infoHash := spec.InfoHash.String()
	if t, ok := torrents[infoHash]; ok {
		return t
	}

	if t, err := torrClient.AddMagnet(uri); err != nil {
		log.Panicln(err)
		return nil
	} else {
		<-t.GotInfo()

		torrents[t.InfoHash().String()] = t
		return t
	}
}

func stopDownloadFile(file *torrent.File) {
	if file != nil {
		file.SetPriority(torrent.PiecePriorityNone)
	}
}

func sortFiles(files []*torrent.File) {
	sort.Slice(files, func(i, j int) bool {
		return files[i].DisplayPath() < files[j].DisplayPath()
	})
}

func appendString(buf *bytes.Buffer, strs ...string) {
	for _, s := range strs {
		buf.WriteString(s)
	}
}

func m3uFilesList(address string, files []*torrent.File) string {
	sortFiles(files)

	var playlist bytes.Buffer

	appendString(&playlist, "#EXTM3U\r\n")

	for _, f := range files {
		path := f.DisplayPath()
		name := filepath.Base(path)
		encoded := base64.StdEncoding.EncodeToString([]byte(path))
		appendString(&playlist, "#EXTINF:-1,", name, "\r\n",
			"http://", address, "/api/infohash/", f.Torrent().InfoHash().String(), "/", encoded, "\r\n")
	}

	return playlist.String()
}

func htmlFilesList(address string, files []*torrent.File) string {
	sortFiles(files)

	var list bytes.Buffer

	for _, f := range files {
		path := f.DisplayPath()

		appendString(&list,
			"<a href=\"http://", address, "/api/infohash/",
			f.Torrent().InfoHash().String(), "/",
			base64.StdEncoding.EncodeToString([]byte(path)),
			"\">", path, "</a>\n</br>")
	}

	return list.String()
}

func jsonFilesList(address string, files []*torrent.File) string {
	sortFiles(files)

	var list bytes.Buffer

	firstLine := true

	appendString(&list, "[")

	for _, f := range files {
		path := f.DisplayPath()

		if firstLine {
			firstLine = false
		} else {
			appendString(&list, ",\n")
		}

		appendString(&list, "[\"", path, "\", \"http://", address, "/api/infohash/",
			f.Torrent().InfoHash().String(), "/",
			base64.StdEncoding.EncodeToString([]byte(path)), "\"]")
	}

	appendString(&list, "]")

	return list.String()
}

func getFileByPath(search string, files []*torrent.File) int {

	for i, f := range files {
		if search == f.DisplayPath() {
			return i
		}
	}

	return -1
}

func serveTorrentFile(w http.ResponseWriter, r *http.Request, file *torrent.File) {
	reader := file.NewReader()

	// Only the first 512 bytes are used to sniff the content type.
	buffer := make([]byte, 512)
	_, err := reader.Read(buffer)
	if err != nil {
		return
	}
	reader.Seek(0, 0)

	// Always returns a valid content-type and "application/octet-stream" if no others seemed to match.
	contentType := http.DetectContentType(buffer)

	path := file.FileInfo().Path
	fname := ""
	if len(path) == 0 {
		fname = file.DisplayPath()
	} else {
		fname = path[len(path)-1]
	}

	w.Header().Set("Content-Disposition", "filename="+fname)
	w.Header().Set("Content-Type", contentType)

	http.ServeContent(w, r, fname, time.Unix(0, 0), reader)
}
