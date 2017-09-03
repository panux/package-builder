FROM alpine:3.6

ADD . /build

WORKDIR /build

RUN apk add --no-cache git make bash curl go musl-dev gcc

RUN make -j20 install

ENTRYPOINT ["/bin/buildpkg"]
