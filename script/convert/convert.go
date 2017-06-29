package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"

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
	var arch string
	flag.StringVar(&infile, "in", "", "input file to generate from")
	flag.StringVar(&out, "out", "", "file to output to")
	flag.StringVar(&arch, "arch", "x86_64", "cpu architecture")
	flag.Parse()
	dat, err := ioutil.ReadFile(infile)
	chk(err)
	r, err := panuxpackager.ParseRaw(dat)
	chk(err)
	r.Arch = arch
	pg, err := r.Preprocess()
	chk(err)
	debug, err := json.Marshal(pg)
	chk(err)
	log.Println(string(debug))
	f, err := os.OpenFile(out, os.O_CREATE|os.O_WRONLY, 0600)
	chk(err)
	defer func() { chk(f.Close()) }()
	chk(pg.GenMake(f))
}
