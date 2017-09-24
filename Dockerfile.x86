FROM multiarch/alpine:x86-v3.6

ADD . /build

WORKDIR /build

RUN apk add --no-cache git make bash curl wget go musl-dev gcc

RUN make install

ENTRYPOINT ["/bin/buildpkg"]
