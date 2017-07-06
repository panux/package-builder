FROM alpine:3.6

ADD . /build

WORKDIR /build

RUN apk add --no-cache git make bash curl go

RUN make install

ENTRYPOINT ["/bin/buildpkg"]
