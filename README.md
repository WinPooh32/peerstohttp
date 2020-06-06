# Go-PeersToHTTP
[![CircleCI](https://circleci.com/gh/WinPooh32/peerstohttp.svg?style=svg)](https://circleci.com/gh/WinPooh32/peerstohttp) [![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2FWinPooh32%2Fpeerstohttp.svg?type=shield)](https://app.fossa.io/projects/git%2Bgithub.com%2FWinPooh32%2Fpeerstohttp?ref=badge_shield)
[![Go Report Card](https://goreportcard.com/badge/github.com/WinPooh32/peerstohttp)](https://goreportcard.com/report/github.com/WinPooh32/peerstohttp)

Simple torrent proxy to http stream controlled over REST-like api

## Http API
Get list of files by magnet url:
```
http://127.0.0.1:8080/api/{playlist:[m3u,html,json]+}/magnet/{magnet}
```

Get list of files by infoHash:
```
http://127.0.0.1:8080/api/{playlist:[m3u,html,json]+}/hash/{hash}
```

Download file:
```
http://127.0.0.1:8080/api/infohash/{infohash}/{base64path}
```

## Example
Get HTML links list for Sintel by torrent hash:
```
http://127.0.0.1:8080/api/html/hash/08ada5a7a6183aae1e09d831df6748d566095a10
```
or by magnet URI:
```
http://127.0.0.1:8080/api/html/magnet/magnet:?xt=urn:btih:08ada5a7a6183aae1e09d831df6748d566095a10&dn=Sintel&tr=udp%3A%2F%2Fexplodie.org%3A6969&tr=udp%3A%2F%2Ftracker.coppersurfer.tk%3A6969&tr=udp%3A%2F%2Ftracker.empire-js.us%3A1337&tr=udp%3A%2F%2Ftracker.leechers-paradise.org%3A6969&tr=udp%3A%2F%2Ftracker.opentrackr.org%3A1337&tr=wss%3A%2F%2Ftracker.btorrent.xyz&tr=wss%3A%2F%2Ftracker.fastcast.nz&tr=wss%3A%2F%2Ftracker.openwebtorrent.com&ws=https%3A%2F%2Fwebtorrent.io%2Ftorrents%2F&xs=https%3A%2F%2Fwebtorrent.io%2Ftorrents%2Fsintel.torrent
```

Then watch Sintel.mp4 using VLC video player:
```
$ vlc http://127.0.0.1:8080/api/infohash/08ada5a7a6183aae1e09d831df6748d566095a10/U2ludGVsLm1wNA==
```

Or open m3u playlist in VLC video player:
```
$ vlc http://127.0.0.1:8080/api/m3u/hash/08ada5a7a6183aae1e09d831df6748d566095a10
```

## Build steps
Build in vendor mode:
```
go build -mod=vendor
```

Using GOPATH:
```
go get -u github.com/WinPooh32/peerstohttp
```

Run:
```
$GOPATH/bin/peerstohttp
```

or:
```
go run github.com/WinPooh32/peerstohttp
```
By default $GOPATH is "~/go"

## License
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2FWinPooh32%2Fpeerstohttp.svg?type=large)](https://app.fossa.io/projects/git%2Bgithub.com%2FWinPooh32%2Fpeerstohttp?ref=badge_large)
