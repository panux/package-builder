FROM golang:1.8-alpine3.6

ADD . /build

WORKDIR /build

RUN apk add --no-cache git make bash curl

RUN make install

ENTRYPOINT ["/bin/buildpkg"]
