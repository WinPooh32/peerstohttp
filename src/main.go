package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/anacrolix/torrent"
)

type serviceSettings struct {
	Host            *string
	Port            *int
	DownloadDir     *string
	DownloadRate    *int
	UploadRate      *int
	MaxConnections  *int
	NoDHT           *bool
	ForceEncryption *bool
}

var procQuit chan bool
var procError chan string

func quit(cl *torrent.Client, srv *http.Server) {
	log.Println("Quitting")

	srv.Close()
	cl.Close()

	//Wait active connections
	// if err := srv.Shutdown(context.Background()); err != nil {
	// 	log.Println(err)
	// }
}

func handleSignals() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go func() {
		for _ = range c {
			// ^C, handle it
			procQuit <- true
		}
	}()
}

func main() {
	procQuit = make(chan bool)
	procError = make(chan string)

	var settings serviceSettings

	settings.Host = flag.String("host", "", "listening server ip")
	settings.Port = flag.Int("port", 8080, "listening port")
	settings.DownloadDir = flag.String("dir", "./", "where files will be downloaded to")
	settings.DownloadRate = flag.Int("drate", 0, "download speed rate in kib/s")
	settings.UploadRate = flag.Int("urate", 0, "upload speed rate in kib/s")
	settings.MaxConnections = flag.Int("maxconn", 20, "max connections per torrent")
	settings.NoDHT = flag.Bool("noDht", false, "disable dht")
	settings.ForceEncryption = flag.Bool("force-encryption", false, "force encryption")

	flag.Parse()

	handleSignals()

	cl := startTorrent(settings)
	srv := startHTTPServer(fmt.Sprintf("%s:%d", *settings.Host, *settings.Port), cl)

	//wait
	select {
	case err := <-procError:
		log.Println(err)
		quit(cl, srv)

	case <-procQuit:
		quit(cl, srv)
	}
}
