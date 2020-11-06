FROM golang

WORKDIR ${GOPATH}/src/github.com/WinPooh32/peerstohttp/

COPY . .
RUN go mod vendor -v
RUN \
  cd cmd && \
  go build -mod=vendor -o peerstohttp

ENTRYPOINT [ "cmd/peerstohttp" ]
