package main

import (
	"encoding/json"
	"errors"
	"flag"
	"log"
	"os"
	"runtime"

	"../.."
)

func chk(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	var out string
	var infile string
	arch := runtime.GOARCH
	switch arch {
	case "amd64":
		arch = "x86_64"
	case "386":
		arch = "x86"
	}
	flag.StringVar(&infile, "in", "", "input file to generate from")
	flag.StringVar(&out, "out", "", "file to output to")
	flag.Parse()
	r, err := panuxpackager.ParseFile(infile)
	chk(err)
	if r.SrcPath == "" {
		panic(errors.New("missing SrcPath"))
	}
	r.Arch = arch
	pg, err := r.Preprocess()
	chk(err)
	debug, err := json.Marshal(pg)
	chk(err)
	log.Println(string(debug))
	f, err := os.OpenFile(out, os.O_CREATE|os.O_WRONLY, 0600)
	chk(err)
	defer func() { chk(f.Close()) }()
	chk(pg.GenPkgSrc(f))
}
