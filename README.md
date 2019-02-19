# Go-PeersToHTTP
[![CircleCI](https://circleci.com/gh/WinPooh32/peerstohttp.svg?style=svg)](https://circleci.com/gh/WinPooh32/peerstohttp) [![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2FWinPooh32%2Fpeerstohttp.svg?type=shield)](https://app.fossa.io/projects/git%2Bgithub.com%2FWinPooh32%2Fpeerstohttp?ref=badge_shield)

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
Get htlm links list by Sintel torrent hash:
```
http://127.0.0.1:8080/api/html/hash/08ada5a7a6183aae1e09d831df6748d566095a10
```

Then watch Sintel.mp4 using video player:
```
$ vlc http://127.0.0.1:8080/api/infohash/08ada5a7a6183aae1e09d831df6748d566095a10/U2ludGVsLm1wNA==
```

Or open m3u playlist in player:
```
$ vlc http://127.0.0.1:8080/api/m3u/hash/08ada5a7a6183aae1e09d831df6748d566095a10
```

## Build steps
```
$ git clone https://github.com/WinPooh32/peerstohttp.git
$ cd peerstohttp
$ go install .
```
Run:
```
$GOPATH/bin/peerstohttp
```

By default $GOPATH is "~/go"

## License
[![FOSSA Status](https://app.fossa.io/api/projects/git%2Bgithub.com%2FWinPooh32%2Fpeerstohttp.svg?type=large)](https://app.fossa.io/projects/git%2Bgithub.com%2FWinPooh32%2Fpeerstohttp?ref=badge_large)
