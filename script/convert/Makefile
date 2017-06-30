godeps:
	go get github.com/hashicorp/go-version gopkg.in/yaml.v2 github.com/panux/encoding-sh

pkgenconvert: godeps
	go build -o pkgenconvert

install: pkgenconvert
	install -m 0700 pkgenconvert /bin/pkgenconvert
