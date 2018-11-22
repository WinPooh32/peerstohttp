package main

import (
	"encoding/base64"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/anacrolix/torrent"
	"github.com/gorilla/mux"
)

const (
	urlAPI = "/api/"
)

func renderList(address string, torr *torrent.Torrent, listType string) string {
	switch listType {
	case "m3u":
		return m3uFilesList(address, torr.Files())
	case "html":
		return htmlFilesList(address, torr.Files())
	case "json":
		return jsonFilesList(address, torr.Files())
	default:
		return ""
	}
}

func sendList(w http.ResponseWriter, torr *torrent.Torrent, listType, host string) {
	// w.Header().Set("Content-Disposition", "filename=playlist.m3u8")
	// w.Header().Set("Content-Type", "application/mpegurl")

	io.WriteString(w, renderList(host, torr, listType))
}

func handleAPI() {
	routerAPI := mux.NewRouter()
	routerAPI.SkipClean(true)

	routerAPI.HandleFunc(urlAPI+"{playlist:[m3u,html,json]+}/magnet/{magnet}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		magnet := "magnet" + strings.TrimLeft(r.URL.Path+"?"+r.URL.RawQuery, "/api/"+vars["playlist"]+"/magnet/")

		if t := addMagnet(magnet); t != nil {
			sendList(w, t, vars["playlist"], r.Host)
		}
	})

	routerAPI.HandleFunc(urlAPI+"{playlist:[m3u,html,json]+}/hash/{hash}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		magnet := "magnet:?xt=urn:btih:" + vars["hash"]

		if t := addMagnet(magnet); t != nil {
			sendList(w, t, vars["playlist"], r.Host)
		}
	})

	routerAPI.HandleFunc(urlAPI+"infohash/{infohash}/{base64path}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		if d, err := base64.StdEncoding.DecodeString(vars["base64path"]); err == nil {
			if t, ok := torrents[vars["infohash"]]; ok {
				idx := getFileByPath(string(d), t.Files())
				file := t.Files()[idx]

				path := file.DisplayPath()

				incFileClients(path)

				serveTorrentFile(w, r, file)
				//stop downloading the file when no connections left
				if decFileClients(path) <= 0 {
					stopDownloadFile(file)
				}
			} else {
				log.Println("Unknown torrent with infohash: ", vars["infohash"])
				return
			}
		} else {
			log.Println(err)
			return
		}
	})

	http.Handle(urlAPI, routerAPI)
}

func startHTTPServer(addr string) *http.Server {
	srv := &http.Server{
		Addr: addr,
	}

	handleAPI()

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			// cannot panic, because this probably is an intentional close
			log.Printf("Httpserver: ListenAndServe() error: %s", err)
		}
	}()

	// returning reference so caller can call Shutdown()
	return srv
}
