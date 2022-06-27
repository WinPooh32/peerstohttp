FROM golang:alpine AS build-env
RUN apk --no-cache add build-base git

WORKDIR ${GOPATH}/src/github.com/WinPooh32/peerstohttp/

COPY . .
RUN \
    cd cmd && \
    go build -v -mod=vendor -o /peerstohttp


FROM alpine
WORKDIR /app
RUN apk add --no-cache libstdc++ libgcc
COPY --from=build-env /peerstohttp /app/peerstohttp

ENTRYPOINT [ "./peerstohttp" ]
